package provider_test

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/labstack/gommon/random"
)

const (
	attrEditorPassword  = "message_vpn.editor_management_credential.password"
	attrManagerPassword = "message_vpn.manager_management_credential.password"
	attrViewerPassword  = "message_vpn.viewer_management_credential.password"
	attrRotationTrigger = "broker_password_rotation.trigger"
	attrRotationUsers   = "broker_password_rotation.users.#"
)

// An empty trigger omits the block.
func rotationServiceHcl(instance *internal.TestInstance, params internal.ConfigurableParams, trigger string, users ...string) string {
	block := ""
	if trigger != "" {
		quoted := make([]string, 0, len(users))
		for _, u := range users {
			quoted = append(quoted, fmt.Sprintf("%q", u))
		}
		block = fmt.Sprintf(`  broker_password_rotation = {
    users   = [%s]
    trigger = %q
  }
`, strings.Join(quoted, ", "), trigger)
	}
	return instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "%s"
  service_class_id = "%s"
%s}
`, params.ServiceName, params.ServiceName, internal.TestDatacenterId(), params.ServiceClass, block)
}

func passwordsAre(resourceName string, params internal.ConfigurableParams, generations map[string]int) resource.TestCheckFunc {
	p := params
	p.PasswordGenerations = generations
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, attrEditorPassword, internal.MockedPassword(p, internal.PasswordUserEditor)),
		resource.TestCheckResourceAttr(resourceName, attrViewerPassword, internal.MockedPassword(p, internal.PasswordUserViewer)),
	}
	if !p.NoManagerCredential {
		checks = append(checks, resource.TestCheckResourceAttr(resourceName, attrManagerPassword, internal.MockedPassword(p, internal.PasswordUserManager)))
	}
	return resource.ComposeTestCheckFunc(checks...)
}

func resetRequestsAre(rec *internal.PasswordResetRecorder, want [][]string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got := rec.Requests()
		if len(got) == 0 && len(want) == 0 {
			return nil
		}
		// Users within a request are unordered.
		if !reflect.DeepEqual(sortedUsers(got), sortedUsers(want)) {
			return fmt.Errorf("expected reset requests %v, got %v", want, got)
		}
		return nil
	}
}

func sortedUsers(requests [][]string) [][]string {
	out := make([][]string, len(requests))
	for i, users := range requests {
		out[i] = append([]string(nil), users...)
		sort.Strings(out[i])
	}
	return out
}

func newRotationTest(t *testing.T, noManager bool) (*internal.TestInstance, internal.ConfigurableParams, string) {
	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{
		ServiceClass:        serviceClass1K,
		ServiceName:         "Test_Rotation_" + random.String(8),
		NoManagerCredential: noManager,
	}
	instance.Init(params)
	return instance, params, resourceTypePrefix + params.ServiceName
}

func TestBrokerPasswordRotationRefusesManagerWithoutManagerCredential(t *testing.T) {
	instance, params, _ := newRotationTest(t, true)
	if !instance.IsMocked() {
		return
	}
	rec := instance.SetupPasswordResetMocks(params, 0, "")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: rotationServiceHcl(instance, params, "v1", internal.PasswordUserViewer)},
			{
				Config:      rotationServiceHcl(instance, params, "v2", internal.PasswordUserManager),
				ExpectError: regexp.MustCompile(`Cannot rotate password for\s+mc-manager`),
			},
			{
				// Check does not run on an ExpectError step, so this no-op apply confirms the
				// refusal happened at plan time, before any reset request was sent.
				Config: rotationServiceHcl(instance, params, "v1", internal.PasswordUserViewer),
				Check:  resetRequestsAre(rec, nil),
			},
		},
	})
}

// A refused or timed-out reset fails the apply, isn't retried, and leaves the old state.
func TestBrokerPasswordRotationSurfacesApiRefusals(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		wantError string
	}{
		{"another operation in progress", 409,
			`{"message":"Job 4blh8jqu78r is still in progress.","errorId":"abc"}`,
			`Job 4blh8jqu78r is still in progress`},
		{"gateway timeout", 503,
			`{"error":"Gateway Timeout","message":"Timeout waiting 28000 ms for...."}`,
			`Password rotation may have`},
		{"shared event broker service", 400,
			`{"message":"You cannot reset Broker Manager passwords on event broker service 6q1p55o6ovr because it is a shared event broker service.","errorId":"abc",` +
				`"validationDetails":{"serviceId":["must not be a shared event broker service"]}}`,
			`because it is a shared event broker service`},
		{"forbidden", 403,
			`{"message":"You do not have permissions to perform this operation.","errorId":"abc"}`,
			`You do not have permissions`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instance, params, resourceName := newRotationTest(t, false)
			if !instance.IsMocked() {
				return
			}
			rec := instance.SetupPasswordResetMocks(params, tc.status, tc.body)

			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: rotationServiceHcl(instance, params, "v1", internal.PasswordUserViewer)},
					{
						Config:      rotationServiceHcl(instance, params, "v2", internal.PasswordUserViewer),
						ExpectError: regexp.MustCompile(tc.wantError),
					},
					{
						// State was not advanced: still the old trigger and passwords, one request.
						// This is a real (no-op) apply, because Check does not run on PlanOnly steps.
						Config: rotationServiceHcl(instance, params, "v1", internal.PasswordUserViewer),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(resourceName, attrRotationTrigger, "v1"),
							passwordsAre(resourceName, params, nil),
							resetRequestsAre(rec, [][]string{{internal.PasswordUserViewer}}),
						),
					},
				},
			})
		})
	}
}
