package connectionendpoint_test

import (
	"fmt"
	"net/http"
	"regexp"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/labstack/gommon/random"
)

// TestConnectionEndpointDataSourceList tests listing all connection endpoints
func TestConnectionEndpointDataSourceList(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Test_" + randomName,
		ServiceId:    "6q1p55o6ovr",
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
data "solacecloud_connection_endpoints" "test" {
  service_id = "%s"
}
`, params.ServiceId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.test", "service_id", params.ServiceId),
					resource.TestCheckResourceAttrSet("data.solacecloud_connection_endpoints.test", "endpoints.#"),
				),
			},
		},
	})
}

// TestConnectionEndpointDataSourceById tests getting a connection endpoint by ID
func TestConnectionEndpointDataSourceById(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Test_" + randomName,
		ServiceId:    "6q1p55o6ovr",
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
data "solacecloud_connection_endpoints" "test" {
  service_id = "%s"
  id         = "80dx8er674q"
}
`, params.ServiceId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.test", "service_id", params.ServiceId),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.test", "id", "80dx8er674q"),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.test", "endpoints.#", "1"),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.test", "endpoints.0.id", "80dx8er674q"),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.test", "endpoints.0.name", "Default Public"),
				),
			},
		},
	})
}

// TestConnectionEndpointDataSourceByName tests getting a connection endpoint by name
func TestConnectionEndpointDataSourceByName(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Test_" + randomName,
		ServiceId:    "6q1p55o6ovr",
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
data "solacecloud_connection_endpoints" "test" {
  service_id = "%s"
  name       = "Default Public"
}
`, params.ServiceId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.test", "service_id", params.ServiceId),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.test", "name", "Default Public"),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.test", "endpoints.#", "1"),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.test", "endpoints.0.name", "Default Public"),
				),
			},
		},
	})
}

// TestConnectionEndpointDataSourceNotFound tests error handling when endpoint not found
func TestConnectionEndpointDataSourceNotFound(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Test_" + randomName,
		ServiceId:    "6q1p55o6ovr",
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Mock a 404 response for a non-existent endpoint
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/nonexistent",
		func(r *http.Request) (*http.Response, error) {
			return internal.JsonResponder(404, `{
				"error": {
					"code": "NOT_FOUND",
					"message": "Connection endpoint not found"
				}
			}`)(r)
		})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
data "solacecloud_connection_endpoints" "test" {
  service_id = "%s"
  id         = "nonexistent"
}
`, params.ServiceId),
				ExpectError: regexp.MustCompile("404|NOT_FOUND|not found"),
			},
		},
	})
}

// TestAccConnectionEndpointDataSource_Real is a real (non-mocked) acceptance test
// It requires a real service to exist and will query its connection endpoints
func TestAccConnectionEndpointDataSource_Real(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Test_" + randomName,
	}
	instance.Init(params)

	// Skip if mocked - this is a real acceptance test
	if instance.IsMocked() {
		t.Skip("Skipping real acceptance test in mocked mode")
		return
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// First create a service
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "id"),
				),
			},
			// Then list all connection endpoints for the service
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

data "solacecloud_connection_endpoints" "all" {
  service_id = solacecloud_service.%s.id
}
`, params.ServiceName, params.ServiceName, params.ServiceClass, params.ServiceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.solacecloud_connection_endpoints.all", "service_id"),
					resource.TestCheckResourceAttrSet("data.solacecloud_connection_endpoints.all", "endpoints.#"),
					resource.TestCheckResourceAttrSet("data.solacecloud_connection_endpoints.all", "endpoints.0.id"),
					resource.TestCheckResourceAttrSet("data.solacecloud_connection_endpoints.all", "endpoints.0.name"),
					resource.TestCheckResourceAttrSet("data.solacecloud_connection_endpoints.all", "endpoints.0.access_type"),
				),
			},
			// Test getting by name
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

data "solacecloud_connection_endpoints" "by_name" {
  service_id = solacecloud_service.%s.id
  name       = "Default Public"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass, params.ServiceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.by_name", "name", "Default Public"),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.by_name", "endpoints.#", "1"),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoints.by_name", "endpoints.0.name", "Default Public"),
				),
			},
			// Cleanup - destroy the service
			{
				Config: instance.GetBaseHcl(),
			},
		},
	})
}
