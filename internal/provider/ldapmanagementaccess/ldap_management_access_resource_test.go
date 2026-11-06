package ldapmanagementaccess_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"terraform-provider-solacecloud/internal"
	"terraform-provider-solacecloud/internal/provider"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"solacecloud": providerserver.NewProtocol6WithError(provider.New("test")()),
}

const resourceName = "solacecloud_ldap_management_access.test"

func accessHcl(instance *internal.TestInstance, serviceId, groups string) string {
	return instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_ldap_management_access" "test" {
  service_id           = %q
  enabled              = true
  group_attribute_name = "memberOf"
%s
}
`, serviceId, groups)
}

// Runs against a real API only when SOLACE_TEST_LDAP_SERVICE_ID names a service with a 'management' profile.
func TestLdapManagementAccessLifecycle(t *testing.T) {
	instance := internal.NewTestInstance()
	instance.Init(internal.ConfigurableParams{})
	serviceId := "ldap-lifecycle-service"
	var mock *internal.LdapManagementAccessMock
	if instance.IsMocked() {
		mock = instance.SetupLdapManagementAccessMocks(serviceId)
	} else if serviceId = os.Getenv("SOLACE_TEST_LDAP_SERVICE_ID"); serviceId == "" {
		t.Skip("set SOLACE_TEST_LDAP_SERVICE_ID to a service with a 'management' LDAP authentication profile to run against a real API")
	}

	configured := func() bool {
		if mock != nil {
			return mock.Configured()
		}
		apiResp, err := instance.GetClient().GetLdapManagementAccessWithResponse(context.Background(), serviceId)
		if err != nil {
			t.Fatal(err)
		}
		return apiResp.StatusCode() != 404
	}
	removeOutsideTerraform := func() {
		if mock != nil {
			mock.Remove()
			return
		}
		if _, err := instance.GetClient().DeleteLdapManagementAccessWithResponse(context.Background(), serviceId); err != nil {
			t.Fatal(err)
		}
		for i := 0; configured(); i++ {
			if i == 60 {
				t.Fatal("LDAP management access was not removed within 5 minutes")
			}
			time.Sleep(5 * time.Second)
		}
	}

	viewers := `  read_only_group_names = ["cn=zeta,dc=example,dc=org", "cn=alpha,dc=example,dc=org"]`
	viewersReordered := `  read_only_group_names = ["cn=alpha,dc=example,dc=org", "cn=zeta,dc=example,dc=org"]`
	editorsOnly := `  read_write_group_names = ["cn=editors,dc=example,dc=org"]`
	withEditors := viewers + "\n" + editorsOnly

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create; omitted group lists are recorded as empty sets.
			{
				Config: accessHcl(instance, serviceId, viewers),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", serviceId),
					resource.TestCheckResourceAttr(resourceName, "management_profile_name", "management"),
					resource.TestCheckResourceAttr(resourceName, "read_only_group_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "read_only_group_names.*", "cn=alpha,dc=example,dc=org"),
					resource.TestCheckResourceAttr(resourceName, "read_write_group_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "mesh_manager_group_names.#", "0"),
				),
			},
			// An unchanged or reordered configuration plans no change.
			{Config: accessHcl(instance, serviceId, viewers), PlanOnly: true},
			{Config: accessHcl(instance, serviceId, viewersReordered), PlanOnly: true},
			// An update sends only the field that changed.
			{
				Config: accessHcl(instance, serviceId, withEditors),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "read_write_group_names.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "read_only_group_names.#", "2"),
					checkPatched(mock, 1, "readWriteGroupNames", `["cn=editors,dc=example,dc=org"]`),
				),
			},
			// Removed outside Terraform: planned for re-creation instead of failing, then re-created.
			{
				PreConfig:          removeOutsideTerraform,
				Config:             accessHcl(instance, serviceId, withEditors),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: accessHcl(instance, serviceId, withEditors),
				Check:  resource.TestCheckResourceAttr(resourceName, "read_write_group_names.#", "1"),
			},
			// Removing a list from the configuration revokes those groups: the PATCH sends it empty.
			{
				Config: accessHcl(instance, serviceId, editorsOnly),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "read_only_group_names.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "read_write_group_names.#", "1"),
					checkPatched(mock, 2, "readOnlyGroupNames", `[]`),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     serviceId,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: func(*terraform.State) error {
			if configured() {
				return fmt.Errorf("LDAP management access still configured on service %s after destroy", serviceId)
			}
			return nil
		},
	})
}

// Request bodies are only visible when mocked, so this passes against a real API.
func checkPatched(mock *internal.LdapManagementAccessMock, count int, field, value string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if mock == nil {
			return nil
		}
		patches := mock.Patches()
		if len(patches) != count {
			return fmt.Errorf("expected %d PATCH requests, got %d", count, len(patches))
		}
		last := patches[count-1]
		if len(last) != 1 || string(last[field]) != value {
			return fmt.Errorf("expected the last PATCH to send only %s = %s, it sent %v", field, value, last)
		}
		return nil
	}
}

func TestLdapManagementAccessRejected(t *testing.T) {
	instance := internal.NewTestInstance()
	instance.Init(internal.ConfigurableParams{})
	if !instance.IsMocked() {
		t.Skip("rejections are only simulated against mocks")
	}

	url := func(serviceId string) string {
		return instance.GetBaseURL() + "/api/v2/missionControl/eventBrokerServices/" + serviceId + "/ldapManagementAccess"
	}
	httpmock.RegisterResponder("POST", url("ldap-no-profile-service"), internal.JsonResponder(400, `{
  "message": "Event broker service ldap-no-profile-service has no 'management' LDAP authentication profile. Create it first.",
  "errorId": "mock",
  "validationDetails": {"ldapAuthenticationProfile": ["the 'management' LDAP authentication profile must exist on this event broker service"]}
}`))
	httpmock.RegisterResponder("POST", url("ldap-configured-service"), internal.JsonResponder(409, `{
  "message": "LDAP management access is already configured for event broker service ldap-configured-service. Use the PATCH API to change it, or the DELETE API to remove it first.",
  "errorId": "mock",
  "validationDetails": {"ldapManagementAccess": ["is already configured for this event broker service"]}
}`))
	httpmock.RegisterResponder("POST", url("ldap-configured-spec-service"), internal.JsonResponder(409, `{
  "message": "LDAP management access is already configured for event broker service ldap-configured-spec-service. Use the PATCH API to change it, or the DELETE API to remove it first.",
  "errorId": "550e8400-e29b-41d4-a716-446655440001"
}`))
	httpmock.RegisterResponder("POST", url("ldap-busy-service"), internal.JsonResponder(409,
		`{"message": "Another operation is already in progress.", "errorId": "mock"}`))
	httpmock.RegisterResponder("POST", url("ldap-shared-service"), internal.JsonResponder(400,
		`{"message": "Request not supported for this service", "errorId": "mock"}`))

	viewers := `  read_only_group_names = ["cn=viewers,dc=example,dc=org"]`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      accessHcl(instance, "ldap-no-profile-service", viewers),
				ExpectError: regexp.MustCompile(`no\s+'management'\s+LDAP\s+authentication\s+profile`),
			},
			{
				Config:      accessHcl(instance, "ldap-configured-service", viewers),
				ExpectError: regexp.MustCompile(`LDAP management access is already configured(.|\n)*Import it to manage it with\s+Terraform:\s+terraform\s+import`),
			},
			{
				Config:      accessHcl(instance, "ldap-configured-spec-service", viewers),
				ExpectError: regexp.MustCompile(`LDAP management access is already configured(.|\n)*Import it to manage it with\s+Terraform`),
			},
			{
				Config:      accessHcl(instance, "ldap-busy-service", viewers),
				ExpectError: regexp.MustCompile(`Another\s+operation\s+is\s+already\s+in\s+progress`),
			},
			{
				Config:      accessHcl(instance, "ldap-shared-service", viewers),
				ExpectError: regexp.MustCompile(`Request\s+not\s+supported\s+for\s+this\s+service`),
			},
		},
	})
}

func TestLdapManagementAccessErrorPaths(t *testing.T) {
	instance := internal.NewTestInstance()
	instance.Init(internal.ConfigurableParams{})
	if !instance.IsMocked() {
		t.Skip("API errors are only simulated against mocks")
	}

	serviceId := "ldap-errors-service"
	instance.SetupLdapManagementAccessMocks(serviceId)
	url := instance.GetBaseURL() + "/api/v2/missionControl/eventBrokerServices/" + serviceId
	respond := func(method, path string, status int, body string) func() {
		return func() { httpmock.RegisterResponder(method, url+path, internal.JsonResponder(status, body)) }
	}
	acceptedAs := func(operationId string) string {
		return `{"data": {"id": "` + operationId + `", "type": "operation", "status": "PENDING"}}`
	}

	viewers := `  read_only_group_names = ["cn=viewers,dc=example,dc=org"]`
	withEditors := viewers + "\n" + `  read_write_group_names = ["cn=editors,dc=example,dc=org"]`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: accessHcl(instance, serviceId, viewers)},
			{
				PreConfig:   respond("PATCH", "/ldapManagementAccess", 409, `{"message": "A job is already in progress for this service", "errorId": "mock"}`),
				Config:      accessHcl(instance, serviceId, withEditors),
				ExpectError: regexp.MustCompile(`A\s+job\s+is\s+already\s+in\s+progress\s+for\s+this\s+service`),
			},
			{
				PreConfig: respond("PATCH", "/ldapManagementAccess", 400, `{
  "message": "readWriteGroupNames[1] 'cn=admins,dc=example,dc=org' duplicates readWriteGroupNames[0].",
  "errorId": "mock",
  "validationDetails": {"readWriteGroupNames[1]": ["duplicates an earlier group name in readWriteGroupNames"]}
}`),
				Config:      accessHcl(instance, serviceId, withEditors),
				ExpectError: regexp.MustCompile(`duplicates\s+readWriteGroupNames\[0\]`),
			},
			{
				PreConfig:   respond("PATCH", "/ldapManagementAccess", 202, `{"data": {"type": "operation"}}`),
				Config:      accessHcl(instance, serviceId, withEditors),
				ExpectError: regexp.MustCompile(`accepted\s+without\s+an\s+operation\s+identifier`),
			},
			{
				PreConfig: func() {
					respond("PATCH", "/ldapManagementAccess", 202, acceptedAs("ldap-failed-operation"))()
					respond("GET", "/operations/ldap-failed-operation", 200,
						`{"data": {"id": "ldap-failed-operation", "type": "operation", "status": "FAILED", "error": {"message": "The LDAP server could not be reached"}}}`)()
				},
				Config:      accessHcl(instance, serviceId, withEditors),
				ExpectError: regexp.MustCompile(`The\s+LDAP\s+server\s+could\s+not\s+be\s+reached`),
			},
			{
				PreConfig:   respond("GET", "/ldapManagementAccess", 403, `{"message": "You are not allowed to view this service", "errorId": "mock"}`),
				Config:      accessHcl(instance, serviceId, viewers),
				ExpectError: regexp.MustCompile(`You\s+are\s+not\s+allowed\s+to\s+view\s+this\s+service`),
			},
			{
				PreConfig: func() {
					respond("GET", "/ldapManagementAccess", 200, `{"data": {"enabled": true, "groupAttributeName": "memberOf",
  "readOnlyGroupNames": ["cn=viewers,dc=example,dc=org"], "readWriteGroupNames": [], "meshManagerGroupNames": [],
  "managementProfileName": "management"}, "meta": {}}`)()
					respond("DELETE", "/ldapManagementAccess", 404, `{"message": "Could not find LDAP management access configuration for event broker service `+serviceId+`.", "errorId": "mock"}`)()
				},
				Config: instance.GetBaseHcl(),
			},
		},
	})
}

func TestLdapManagementAccessFailedCreateIsReplaced(t *testing.T) {
	instance := internal.NewTestInstance()
	instance.Init(internal.ConfigurableParams{})
	if !instance.IsMocked() {
		t.Skip("API errors are only simulated against mocks")
	}

	serviceId := "ldap-failed-create-service"
	mock := instance.SetupLdapManagementAccessMocks(serviceId)
	operationUrl := instance.GetBaseURL() + "/api/v2/missionControl/eventBrokerServices/" + serviceId + "/operations/ldap-management-access-operation-123"
	operationEndsAs := func(status, errorBlock string) func() {
		return func() {
			httpmock.RegisterResponder("GET", operationUrl, internal.JsonResponder(200,
				`{"data": {"id": "ldap-management-access-operation-123", "type": "operation", "status": "`+status+`", "error": `+errorBlock+`}}`))
		}
	}

	config := accessHcl(instance, serviceId, `  read_only_group_names = ["cn=viewers,dc=example,dc=org"]`)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig:   operationEndsAs("FAILED", `{"message": "The LDAP server could not be reached"}`),
				Config:      config,
				ExpectError: regexp.MustCompile(`The\s+LDAP\s+server\s+could\s+not\s+be\s+reached`),
			},
			{
				PreConfig: operationEndsAs("SUCCEEDED", "null"),
				Config:    config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate)},
				},
				Check: func(*terraform.State) error {
					if !mock.Configured() {
						return fmt.Errorf("expected the configuration to be re-created")
					}
					return nil
				},
			},
		},
	})
}
