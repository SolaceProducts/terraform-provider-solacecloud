package provider_test

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"terraform-provider-solacecloud/internal"
	"terraform-provider-solacecloud/internal/provider"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
	"github.com/labstack/gommon/random"
)

func TestOrderResource(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Azure_TF_Service" + randomName,
	}
	instance.Init(params)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: instance.GetBaseHcl() + `
resource "solacecloud_service" "` + params.ServiceName + `" {
  name             = "` + params.ServiceName + `"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "` + params.ServiceClass + `"
}
`,
			},
		},
	})
}

func TestResourceDeletedExternally(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Azure_TF_Service" + randomName,
	}
	instance.Init(params)

	var serviceID string

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: instance.GetBaseHcl() + `
resource "solacecloud_service" "` + params.ServiceName + `" {
  name             = "` + params.ServiceName + `"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "` + params.ServiceClass + `"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "id"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "name", params.ServiceName),
					// Capture the service ID for deletion
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["solacecloud_service."+params.ServiceName]
						if !ok {
							return fmt.Errorf("resource not found: solacecloud_service.%s", params.ServiceName)
						}
						serviceID = rs.Primary.ID
						t.Logf("Captured service ID for deletion test: %s", serviceID)
						return nil
					},
				),
			},
			// Simulate external deletion by deleting the service outside of Terraform
			{
				PreConfig: func() {
					// Delete the service using the API directly (simulating external deletion)
					if serviceID != "" {
						t.Logf("Simulating external deletion of service ID: %s", serviceID)
						// Use the test instance to delete the service directly
						if instance.IsMocked() {
							httpmock.RegisterResponder("DELETE", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/6q1p55o6ovr",
								internal.JsonResponder(404, "{\n    \"message\": \"Could not find event broker service with id 6q1p55o6ovr\",\n    \"errorId\": \"cd77f668-ad6d-4ea9-9ee3-e72bc18de62b\"\n}"))

							httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/6q1p55o6ovr",
								internal.JsonResponder(404, "{\n    \"message\": \"Could not find event broker service with id 6q1p55o6ovr\",\n    \"errorId\": \"cd77f668-ad6d-4ea9-9ee3-e72bc18de62b\"\n}"))

							newParamsService := internal.ConfigurableParams{
								ServiceClass: "ENTERPRISE_1K_STANDALONE",
								ServiceName:  "Azure_TF_Service" + randomName,
								ServiceId:    "newserviceid",
							}
							instance.SetupDefaultMocks(newParamsService)

						} else {
							res, err := instance.GetClient().DeleteService(context.Background(), serviceID)
							if err == nil && res.StatusCode != http.StatusAccepted {
								t.Log("Could not delete service. Test could not run")
								t.FailNow()
							}
						}
					}
				},
				Config: instance.GetBaseHcl() + `
resource "solacecloud_service" "` + params.ServiceName + `" {
  name             = "` + params.ServiceName + `"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "` + params.ServiceClass + `"
}
`,
				Check: resource.ComposeTestCheckFunc(
					// Verify the resource was recreated with a new ID
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "id"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "name", params.ServiceName),
					// Verify the new ID is different from the original
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["solacecloud_service."+params.ServiceName]
						if !ok {
							return fmt.Errorf("resource not found after recreation: solacecloud_service.%s", params.ServiceName)
						}
						newServiceID := rs.Primary.ID
						if newServiceID == serviceID {
							return fmt.Errorf("service ID should have changed after recreation, but got same ID: %s", newServiceID)
						}
						t.Logf("Service successfully recreated with new ID: %s (old ID: %s)", newServiceID, serviceID)
						return nil
					},
				),
			},
		},
	})
}

// TestServiceResourceImport tests the import functionality for the service resource
func TestServiceResourceImport(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	serviceName := "Import_Test_Service_" + randomName
	var capturedServiceID string

	params := internal.ConfigurableParams{
		ServiceClass:     "ENTERPRISE_1K_HIGHAVAILABILITY",
		ServiceName:      serviceName,
		CustomRouterName: "myrouter",
	}
	instance.Init(params)

	// Configuration that will be reused across steps
	baseConfig := instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
  custom_router_name = "%s"
}
`, serviceName, serviceName, params.ServiceClass, params.CustomRouterName)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create the resource first and capture the ID
			{
				Config: baseConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(fmt.Sprintf("solacecloud_service.%s", serviceName), "id"),
					// Capture the service ID for use in import step
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[fmt.Sprintf("solacecloud_service.%s", serviceName)]
						if !ok {
							return fmt.Errorf("resource not found: solacecloud_service.%s", serviceName)
						}
						capturedServiceID = rs.Primary.ID
						t.Logf("Captured service ID: %s", capturedServiceID)
						return nil
					},
				),
			},
			// Import the resource using the captured ID
			{
				Config:       baseConfig,
				ResourceName: fmt.Sprintf("solacecloud_service.%s", serviceName),
				ImportState:  true,
				ImportStateIdFunc: func(*terraform.State) (string, error) {
					return capturedServiceID, nil
				},
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"custom_router_name",
					"locked",
				},
			},
		},
	})
}

func TestNameValidation(t *testing.T) {
	// Create a simple test provider factory that mocks the provider behavior
	testAccProviderFactories := map[string]func() (tfprotov6.ProviderServer, error){
		"solacecloud": providerserver.NewProtocol6WithError(provider.New("test")()),
	}

	// Test case 1: Empty name (should fail LengthAtLeast(1) validation)
	t.Run("empty_name", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: `
provider "solacecloud" {}

resource "solacecloud_service" "test_service" {
  name             = ""
  datacenter_id    = "mock-datacenter"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}`,
					ExpectError: regexp.MustCompile("Attribute name string length must be at least 1"),
				},
			},
		})
	})

	// Test case 2: Name too long (should fail LengthAtMost(50) validation)
	t.Run("name_too_long", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: `
provider "solacecloud" {}

resource "solacecloud_service" "test_service" {
  name             = "this_is_a_very_long_name_that_exceeds_fifty_characters_limit_for_testing"
  datacenter_id    = "mock-datacenter"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}`,
					ExpectError: regexp.MustCompile("Attribute name string length must be at most 50"),
				},
			},
		})
	})

	// Test case 3: Name is 'default' (should fail NameNotDefaultValidator)
	t.Run("name_default", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: `
provider "solacecloud" {}

resource "solacecloud_service" "test_service" {
  message_vpn_name = "default"
  datacenter_id    = "mock-datacenter"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
  name             = "valid-name"
}`,
					ExpectError: regexp.MustCompile("Name cannot be 'default'"),
				},
			},
		})
	})

	// Test case 4: Valid name (should pass validation)
	t.Run("valid_name", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProviderFactories,
			Steps: []resource.TestStep{
				{
					// This will fail at the API call stage but pass validation
					Config: `
provider "solacecloud" {
  api_token = "mock-token"
  base_url = "https://api-mock.solace.cloud"
}

resource "solacecloud_service" "test_service" {
  name             = "valid-name"
  datacenter_id    = "mock-datacenter"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}`,
					// We expect this to fail after validation passes, due to API errors
					ExpectError: regexp.MustCompile("Error calling Solace Cloud API"),
				},
			},
		})
	})
}

func TestAccServiceResource_Delete(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "TF_Delete_Service_" + randomName,
	}
	instance.Init(params)

	resourceName := "solacecloud_service." + params.ServiceName

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + `
resource "solacecloud_service" "` + params.ServiceName + `" {
  name             = "` + params.ServiceName + `"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "` + params.ServiceClass + `"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				ResourceName: resourceName,
				Config:       instance.GetBaseHcl(), // no service block = delete
			},
		},
	})
}

func TestAccServiceResource_DeleteLockedFails(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "TF_Locked_Service_" + randomName,
		Locked:       true,
		ServiceId:    "myid",
	}
	instance.Init(params)

	if instance.IsMocked() {
		httpmock.RegisterResponder("PATCH", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId,
			func(r *http.Request) (res *http.Response, err error) {

				params.Locked = false
				instance.SetupDefaultMocks(params)

				return internal.JsonResponder(200, `{
			"id": "`+params.ServiceId+`",
			"name": "`+params.ServiceName+`",
			"locked": false,
			"serviceClassId": "`+params.ServiceClass+`",
			"datacenterId": "eks-us-east-1"
		}`)(r)
			},
		)

		httpmock.RegisterResponder("DELETE", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId,
			internal.JsonResponder(400, `{
			"message": "You cannot delete a service with deletion protection enabled.",
			"errorId": "service-locked-error"
		}`))

	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + `
 resource "solacecloud_service" "` + params.ServiceName + `" {
   name             = "` + params.ServiceName + `"
   datacenter_id    = "eks-us-east-1"
   service_class_id = "` + params.ServiceClass + `"
   locked           = true
 }
 `,
			},
			{
				Config:      instance.GetBaseHcl(), // No service block: triggers deletion
				ExpectError: regexp.MustCompile("You cannot delete a service with deletion protection enabled."),
			},
			{
				Config: instance.GetBaseHcl() + `
 resource "solacecloud_service" "` + params.ServiceName + `" {
   name             = "` + params.ServiceName + `"
   datacenter_id    = "eks-us-east-1"
   service_class_id = "` + params.ServiceClass + `"
   locked           = false
 }
 `,
			},
			{
				Config:  instance.GetBaseHcl(),
				Destroy: true, // Now we can delete it
			},
		},
	})
}
