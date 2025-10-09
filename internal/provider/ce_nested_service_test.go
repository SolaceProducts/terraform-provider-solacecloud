package provider_test

import (
	"fmt"
	"net/http"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/labstack/gommon/random"
)

// TestServiceCreationSuccess tests successful service creation with mocked API
func TestServiceCreationSceSuccess(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Success_Test_" + randomName,
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
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
  connection_endpoint = {
    name        = "Default Public"
    access_type = "PUBLIC"
    description = ""
    ports = {
      mqtt_tls = { port = 8883 },
      mqtt_websocket_tls = { port = 8443 },
      smf_tls = { port = 55443 },
      amqp_tls = { port = 5671 },
      management_tls = { port = 943 },
      rest_incoming_tls = { port = 9443 },
      ssh_tls = { port = 22 },
      web_tls = { port = 443 }
    }
  }
}
`, params.ServiceName, params.ServiceName, params.ServiceClass),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "id"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "name", params.ServiceName),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "service_class_id", params.ServiceClass),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "datacenter_id", "eks-us-east-1"),
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "event_broker_version"),
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "message_vpn_name"),
					// check the default connection endpoint
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "connection_endpoint.id"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "connection_endpoint.name", "Default Public"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "connection_endpoint.access_type", "PUBLIC"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "connection_endpoint.ports.mqtt_tls.port", "8883"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "connection_endpoint.ports.management_tls.port", "943"),
				),
			},
		},
	})
}

// TestServiceCreationSuccess tests successful service creation with mocked API
// Create a service with an SCE. update it then Remove it from spec. Should not be deleted for real
func TestServiceCRUDSceSuccess(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Success_Test_" + randomName,
		ServiceId:    "6q1p55o6ovr",
	}
	instance.Init(params)
	if instance.IsMocked() {

		// mock the patch method for connection endpoint updates
		httpmock.RegisterResponder("PATCH", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/80dx8er674q",
			func(r *http.Request) (*http.Response, error) {
				// Return a successful response for the connection endpoint update
				return internal.JsonResponder(202, `{"data": {
					"id": "connection-endpoint-update-operation-123",
					"type": "operation",
					"operationType": "updateConnectionEndpoint",
					"createdBy": "67tr8tkuel",
					"createdTime": "2025-02-19T01:33:04Z",
					"completedTime": "2025-02-19T01:33:04Z",
					"resourceId": "80dx8er674q",
					"resourceType": "connectionEndpoint",
					"status": "PENDING",
					"error": null
				}}`)(r)
			})

		// Mock the operation status check endpoint to show completed operation
		httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/operations/connection-endpoint-update-operation-123",

			func(r *http.Request) (*http.Response, error) {
				// mock the get service to return the connection endpoint with updated ssh port
				// also mock the getConnectionEndpoint to return the updated ssh port

				updatedParams := internal.ConfigurableParams{
					ServiceClass: "ENTERPRISE_1K_STANDALONE",
					ServiceName:  "Success_Test_" + randomName,
					ServiceId:    "6q1p55o6ovr",
					SshTlsPort:   224,
				}
				instance.SetupDefaultMocks(updatedParams)

				return internal.JsonResponder(200, `{
					"data": {
						"id": "connection-endpoint-update-operation-123",
						"type": "operation",
						"operationType": "updateConnectionEndpoint",
						"createdBy": "67tr8tkuel",
						"createdTime": "2025-02-19T01:33:04Z",
						"completedTime": "2025-02-19T01:33:04Z",
						"resourceId": "`+params.ServiceId+`/connectionEndpoints/80dx8er674q",
						"resourceType": "connectionEndpoint",
						"status": "SUCCEEDED",
						"error": null
					}
				}`)(r)
			})
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
  connection_endpoint = {
    name        = "Default Public"
    access_type = "PUBLIC"
    description = ""
    ports = {
      mqtt_tls = { port = 8883 },
      mqtt_websocket_tls = { port = 8443 },
      smf_tls = { port = 55443 },
      amqp_tls = { port = 5671 },
      management_tls = { port = 943 },
      rest_incoming_tls = { port = 9443 },
      ssh_tls = { port = 22 },
      web_tls = { port = 443 }
    }
  }
}
`, params.ServiceName, params.ServiceName, params.ServiceClass),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "id"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "name", params.ServiceName),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "service_class_id", params.ServiceClass),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "datacenter_id", "eks-us-east-1"),
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "event_broker_version"),
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "message_vpn_name"),
				),
			},
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
  connection_endpoint = {
    name        = "Default Public"
    access_type = "PUBLIC"
    description = ""
    ports = {
      mqtt_tls = { port = 8883 },
      mqtt_websocket_tls = { port = 8443 },
      smf_tls = { port = 55443 },
      amqp_tls = { port = 5671 },
      management_tls = { port = 943 },
      rest_incoming_tls = { port = 9443 },
      ssh_tls = { port = 224 },
      web_tls = { port = 443 }
    }
  }
}
`, params.ServiceName, params.ServiceName, params.ServiceClass),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "id"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "name", params.ServiceName),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "connection_endpoint.ports.ssh_tls.port", "224"),
				),
			},
			{
				// nothing happens when endpoint is removed. Endpoint is not returned anymore
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "id"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "name", params.ServiceName),
				),
			},
			{
				Config: instance.GetBaseHcl(),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}
