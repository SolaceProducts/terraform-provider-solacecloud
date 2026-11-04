package provider_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
	"github.com/labstack/gommon/random"
)

// Runs against a real API only when SOLACE_TEST_MATE_LINK_TRANSITIONS=true.

const (
	attrMateLinkEncryption   = "mate_link_encryption"
	hclMateLinkTrue          = "  mate_link_encryption = true\n"
	hclMateLinkFalse         = "  mate_link_encryption = false\n"
	serviceClassHa           = "ENTERPRISE_1K_HIGHAVAILABILITY"
	mateLinkRefusalNotHa     = "Request not supported for this service"
	mateLinkDidNotTakeEffect = "Mate-link encryption configuration did not take effect"
	mateLinkChangeFailed     = "Changing mate_link_encryption failed: "
)

func e2eHaServiceClassId() string {
	if override := os.Getenv("SOLACE_TEST_HA_SERVICE_CLASS_ID"); override != "" {
		return override
	}
	return serviceClassHa
}

func mateLinkServiceId() string {
	return "mlk" + strings.ToLower(random.String(8, random.Alphanumeric))
}

func mateLinkServiceHcl(instance *internal.TestInstance, params internal.ConfigurableParams, arguments string) string {
	return instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "%s"
  service_class_id = "%s"
%s%s}
`, params.ServiceName, params.ServiceName, e2eDatacenterId(), params.ServiceClass, e2eEventBrokerVersionHcl(), arguments)
}

func mateLinkPatchCall(instance *internal.TestInstance, serviceId string) string {
	return "PATCH " + instance.GetBaseURL() + "/api/v2/missionControl/eventBrokerServices/" + serviceId + "/mateLinkEncryption"
}

func expectMateLinkRequests(call string, want int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if got := httpmock.GetCallCountInfo()[call]; got != want {
			return fmt.Errorf("expected %d mate-link request(s) in total, got %d", want, got)
		}
		return nil
	}
}

// wrappedText matches text as Terraform prints it, which may wrap a long message across lines.
func wrappedText(text string) *regexp.Regexp {
	return regexp.MustCompile(strings.Join(strings.Fields(regexp.QuoteMeta(text)), `\s+`))
}

func TestAccMateLinkLifecycle(t *testing.T) {
	instance := internal.NewTestInstance()
	serviceName := "E2E_MateLink_Lifecycle_" + random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: e2eHaServiceClassId(),
		ServiceName:  serviceName,
		ServiceId:    mateLinkServiceId(),
	}
	instance.Init(params)
	if instance.IsMocked() {
		instance.SetupMateLinkMocks(params, "")
	} else if os.Getenv("SOLACE_TEST_MATE_LINK_TRANSITIONS") != "true" {
		t.Skip("set SOLACE_TEST_MATE_LINK_TRANSITIONS=true to run the mate-link transitions against a real API")
	}

	resourceName := resourceTypePrefix + serviceName

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mateLinkServiceHcl(instance, params, hclMateLinkFalse),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, attrId),
					resource.TestCheckResourceAttr(resourceName, attrMateLinkEncryption, valueFalse),
				),
			},
			{
				Config: mateLinkServiceHcl(instance, params, hclMateLinkTrue),
				Check:  resource.TestCheckResourceAttr(resourceName, attrMateLinkEncryption, valueTrue),
			},
			{
				Config: mateLinkServiceHcl(instance, params, hclMateLinkFalse),
				Check:  resource.TestCheckResourceAttr(resourceName, attrMateLinkEncryption, valueFalse),
			},
			{
				Config:   mateLinkServiceHcl(instance, params, hclMateLinkFalse),
				PlanOnly: true,
			},
		},
	})
}

func TestServiceWithoutMateLinkArgumentPlansClean(t *testing.T) {
	for _, mateLinkOff := range []bool{false, true} {
		t.Run(fmt.Sprintf("service reports encryption %t", !mateLinkOff), func(t *testing.T) {
			instance := internal.NewTestInstance()
			params := internal.ConfigurableParams{
				ServiceClass: serviceClassHa,
				ServiceName:  "Test_MateLink_Omitted_" + random.String(8),
				ServiceId:    mateLinkServiceId(),
				MateLinkOff:  mateLinkOff,
			}
			instance.Init(params)
			if !instance.IsMocked() {
				return
			}
			instance.SetupMateLinkMocks(params, "")

			resourceName := resourceTypePrefix + params.ServiceName
			patchCall := mateLinkPatchCall(instance, params.ServiceId)

			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: mateLinkServiceHcl(instance, params, ""),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(resourceName, attrMateLinkEncryption, fmt.Sprint(!mateLinkOff)),
							expectMateLinkRequests(patchCall, 0),
						),
					},
					{
						Config:   mateLinkServiceHcl(instance, params, ""),
						PlanOnly: true,
					},
				},
			})
		})
	}
}

func TestMateLinkEncryptionConvergesAtCreate(t *testing.T) {
	cases := []struct {
		name         string
		createdOff   bool
		arguments    string
		wantValue    string
		wantRequests int
	}{
		{name: "created on, configured off", createdOff: false, arguments: hclMateLinkFalse, wantValue: valueFalse, wantRequests: 1},
		{name: "created off, configured on", createdOff: true, arguments: hclMateLinkTrue, wantValue: valueTrue, wantRequests: 1},
		{name: "created on, configured on", createdOff: false, arguments: hclMateLinkTrue, wantValue: valueTrue, wantRequests: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instance := internal.NewTestInstance()
			params := internal.ConfigurableParams{
				ServiceClass: serviceClassHa,
				ServiceName:  "Test_MateLink_Create_" + random.String(8),
				ServiceId:    mateLinkServiceId(),
				MateLinkOff:  tc.createdOff,
			}
			instance.Init(params)
			if !instance.IsMocked() {
				return
			}
			instance.SetupMateLinkMocks(params, "")

			resourceName := resourceTypePrefix + params.ServiceName
			patchCall := mateLinkPatchCall(instance, params.ServiceId)

			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: mateLinkServiceHcl(instance, params, tc.arguments),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(resourceName, attrMateLinkEncryption, tc.wantValue),
							expectMateLinkRequests(patchCall, tc.wantRequests),
						),
					},
					{
						Config:   mateLinkServiceHcl(instance, params, tc.arguments),
						PlanOnly: true,
						Check:    expectMateLinkRequests(patchCall, tc.wantRequests),
					},
				},
			})
		})
	}
}

func standaloneMateLinkRefusal(serviceName string) (*internal.TestInstance, internal.ConfigurableParams, bool) {
	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{
		ServiceClass: serviceClass1K,
		ServiceName:  serviceName + random.String(8),
		ServiceId:    mateLinkServiceId(),
		MateLinkOff:  true,
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return nil, params, false
	}
	httpmock.RegisterResponder("PATCH",
		instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/mateLinkEncryption",
		internal.JsonResponder(400, `{"message": "`+mateLinkRefusalNotHa+`", "errorId": "mock", "validationDetails": {"type": ["must support this request"]}}`))
	return instance, params, true
}

// Terraform prints the mate_link_encryption line just above the error detail.
var mateLinkRefusalOnArgument = wrappedText(strings.TrimSpace(hclMateLinkTrue) + " " + mateLinkChangeFailed + mateLinkRefusalNotHa)

func TestMateLinkEncryptionOnNonHaServiceSurfacesApiRefusal(t *testing.T) {
	instance, params, mocked := standaloneMateLinkRefusal("Test_MateLink_Standalone_")
	if !mocked {
		return
	}
	resourceName := resourceTypePrefix + params.ServiceName

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: mateLinkServiceHcl(instance, params, ""),
				Check:  resource.TestCheckResourceAttr(resourceName, attrMateLinkEncryption, valueFalse),
			},
			{
				Config:      mateLinkServiceHcl(instance, params, hclMateLinkTrue),
				ExpectError: mateLinkRefusalOnArgument,
			},
		},
	})
}

func TestMateLinkEncryptionOnNewNonHaServiceSurfacesApiRefusal(t *testing.T) {
	instance, params, mocked := standaloneMateLinkRefusal("Test_MateLink_Standalone_Create_")
	if !mocked {
		return
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      mateLinkServiceHcl(instance, params, hclMateLinkTrue),
				ExpectError: mateLinkRefusalOnArgument,
			},
		},
	})
}

func TestMateLinkEncryptionReadBackMismatchIsReported(t *testing.T) {
	steps := map[string]func(instance *internal.TestInstance, params internal.ConfigurableParams) []resource.TestStep{
		"at update": func(instance *internal.TestInstance, params internal.ConfigurableParams) []resource.TestStep {
			return []resource.TestStep{
				{
					Config: mateLinkServiceHcl(instance, params, ""),
					Check:  resource.TestCheckResourceAttr(resourceTypePrefix+params.ServiceName, attrMateLinkEncryption, valueTrue),
				},
				{
					Config:      mateLinkServiceHcl(instance, params, hclMateLinkFalse),
					ExpectError: wrappedText(mateLinkDidNotTakeEffect),
				},
			}
		},
		"at create": func(instance *internal.TestInstance, params internal.ConfigurableParams) []resource.TestStep {
			return []resource.TestStep{
				{
					Config:      mateLinkServiceHcl(instance, params, hclMateLinkFalse),
					ExpectError: wrappedText(mateLinkDidNotTakeEffect),
				},
			}
		},
	}

	for name, buildSteps := range steps {
		t.Run(name, func(t *testing.T) {
			instance := internal.NewTestInstance()
			params := internal.ConfigurableParams{
				ServiceClass: serviceClassHa,
				ServiceName:  "Test_MateLink_Mismatch_" + random.String(8),
				ServiceId:    mateLinkServiceId(),
			}
			instance.Init(params)
			if !instance.IsMocked() {
				return
			}
			const operationId = "mate-link-noop-operation"
			serviceUrl := instance.GetBaseURL() + "/api/v2/missionControl/eventBrokerServices/" + params.ServiceId
			operation := `{"data": {"id": "` + operationId + `", "type": "operation", "operationType": "updateMateLinkEncryption",
		"resourceId": "` + params.ServiceId + `", "resourceType": "service", "status": "%s", "error": null}}`
			// Accepted and reported as succeeded, but the mocked service keeps reporting encryption on.
			httpmock.RegisterResponder("PATCH", serviceUrl+"/mateLinkEncryption",
				internal.JsonResponder(202, fmt.Sprintf(operation, "PENDING")))
			httpmock.RegisterResponder("GET", serviceUrl+"/operations/"+operationId,
				internal.JsonResponder(200, fmt.Sprintf(operation, "SUCCEEDED")))

			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps:                    buildSteps(instance, params),
			})
		})
	}
}
