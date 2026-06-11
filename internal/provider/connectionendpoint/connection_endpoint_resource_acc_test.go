package connectionendpoint_test

import (
	"fmt"
	"github.com/jarcoal/httpmock"
	"net/http"
	"terraform-provider-solacecloud/internal"
	"terraform-provider-solacecloud/missioncontrol"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/labstack/gommon/random"
)

// tests against mock api
func TestAccConnectionEndpointResource_Create(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	serviceName := "CE_Test_Service_" + randomName
	// this is not realistic, but it lets us reuse some of the existing mocks
	endpointName := "Default Public"

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          serviceName,
		ConnectionEndpointId: "newep123456",
		ServiceId:            "6q1p55o6ovr", // Used only in mocked mode
	}
	instance.Init(params)

	// Setup connection endpoint mocks if in mocked mode
	if !instance.IsMocked() {
		return
	}

	httpmock.RegisterResponder("POST", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints",
		func(r *http.Request) (*http.Response, error) {
			// Return a successful response for the connection endpoint update
			return internal.JsonResponder(202, `{"data": {
					"id": "connection-endpoint-update-operation-123",
					"type": "operation",
					"operationType": "createConnectionEndpoint",
					"createdBy": "67tr8tkuel",
					"createdTime": "2025-02-19T01:33:04Z",
					"completedTime": "2025-02-19T01:33:04Z",
					"resourceId": "`+params.ServiceId+`/connectionEndpoints/80dx8er674q" ,
					"resourceType": "connectionEndpoint",
					"status": "PENDING",
					"error": null
				}}`)(r)
		})

	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/operations/connection-endpoint-update-operation-123",
		func(r *http.Request) (*http.Response, error) {
			return internal.JsonResponder(200, `{"data": {
					"id": "connection-endpoint-update-operation-123",
					"type": "operation",
					"operationType": "createConnectionEndpoint",
					"createdBy": "67tr8tkuel",
					"createdTime": "2025-02-19T01:33:04Z",
					"completedTime": "2025-02-19T01:33:04Z",
					"resourceId": "80dx8er674q",
					"resourceId": "`+params.ServiceId+`/connectionEndpoints/80dx8er674q" ,
					"status": "SUCCEEDED",
					"error": null
				}}`)(r)
		})

	httpmock.RegisterResponder("DELETE", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/80dx8er674q",
		func(r *http.Request) (*http.Response, error) {
			return internal.JsonResponder(202, `{"data": {
					"id": "connection-endpoint-delete-operation-123",
					"type": "operation",
					"operationType": "deleteConnectionEndpoint",
					"createdBy": "67tr8tkuel",
					"createdTime": "2025-02-19T01:33:04Z",
					"completedTime": "2025-02-19T01:33:04Z",
					"resourceId": "`+params.ServiceId+`/connectionEndpoints/80dx8er674q",
					"resourceType": "connectionEndpoint",
					"status": "PENDING",
					"error": null
				}}`)(r)
		})

	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/operations/connection-endpoint-delete-operation-123",
		func(r *http.Request) (*http.Response, error) {
			return internal.JsonResponder(200, `{"data": {
					"id": "connection-endpoint-delete-operation-123",
					"type": "operation",
					"operationType": "deleteConnectionEndpoint",
					"createdBy": "67tr8tkuel",
					"createdTime": "2025-02-19T01:33:04Z",
					"completedTime": "2025-02-19T01:33:04Z",
					"resourceId": "`+params.ServiceId+`/connectionEndpoints/80dx8er674q",
					"resourceType": "connectionEndpoint",
					"status": "SUCCEEDED",
					"error": null
				}}`)(r)
		})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create service and connection endpoint together
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "test" {
  name             = "%s"
  datacenter_id    = "gke-gcp-us-central1-a"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint" "test" {
    service_id  = solacecloud_service.test.id
    name        = "%s"
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
`, serviceName, params.ServiceClass, endpointName),
				Check: resource.ComposeTestCheckFunc(
					// Verify service was created
					resource.TestCheckResourceAttrSet("solacecloud_service.test", "id"),
					resource.TestCheckResourceAttr("solacecloud_service.test", "name", serviceName),
					// Verify connection endpoint attributes
					resource.TestCheckResourceAttrSet("solacecloud_connection_endpoint.test", "id"),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.test", "name", endpointName),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.test", "description", ""),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.test", "access_type", "PUBLIC"),
					resource.TestCheckResourceAttrSet("solacecloud_connection_endpoint.test", "service_id"),
					// Verify computed attributes
					resource.TestCheckResourceAttrSet("solacecloud_connection_endpoint.test", "k8s_service_type"),
					resource.TestCheckResourceAttrSet("solacecloud_connection_endpoint.test", "k8s_service_id"),
				),
			},
		},
	})
}

func TestE2eResourceCreate(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	serviceName := "CE_Test_Service_" + randomName
	endpointName := "Default Public"

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          serviceName,
		ConnectionEndpointId: "newep123456",
	}
	instance.Init(params)

	if instance.IsMocked() {
		t.Skip("Skipping E2E test in mocked mode")
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create service and connection endpoint together
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "test" {
  name             = "%s"
  datacenter_id    = "pcorg-jacks-special-dc"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint" "test" {
	service_id  = solacecloud_service.test.id
	name        = "%s"
	access_type = "PRIVATE"
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
`, serviceName, params.ServiceClass, endpointName),
				Check: resource.ComposeTestCheckFunc(
					// Verify service was created
					resource.TestCheckResourceAttrSet("solacecloud_service.test", "id"),
					resource.TestCheckResourceAttr("solacecloud_service.test", "name", serviceName),
					// Verify connection endpoint attributes
					resource.TestCheckResourceAttrSet("solacecloud_connection_endpoint.test", "id"),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.test", "name", endpointName),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.test", "description", ""),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.test", "access_type", "PRIVATE"),
					resource.TestCheckResourceAttrSet("solacecloud_connection_endpoint.test", "service_id"),
					// Verify computed attributes
					resource.TestCheckResourceAttrSet("solacecloud_connection_endpoint.test", "k8s_service_type"),
					resource.TestCheckResourceAttrSet("solacecloud_connection_endpoint.test", "k8s_service_id"),
				),
			},
		},
	})
}

func TestE2eCRUD(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	serviceName := "CE_Test_Service_" + randomName

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          serviceName,
		ConnectionEndpointId: "newep123456",
	}
	instance.Init(params)

	if instance.IsMocked() {
		t.Skip("Skipping E2E test in mocked mode")
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create service and connection endpoint together
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
# Create the service first
resource "solacecloud_service" "broker_service" {
  name                 = "%s"
  datacenter_id        = "pcorg-jacks-special-dc"
  service_class_id     = "ENTERPRISE_1K_STANDALONE"
  event_broker_version = "10.25.0.276-48"
  locked               = false
  connection_endpoint = {
    name        = "main-endpoint"
    access_type = "PUBLIC"
    description = "Primary"
    ports = {
      mqtt_tls       = { port = 12345 },
      smf_tls        = { port = 54321 },
      amqp_tls       = { port = 5671 },
      management_tls = { port = 12311 }
    }
  }
}
resource "solacecloud_connection_endpoint" "myendpoint" {
  access_type = "PRIVATE"
  name        = "my-privte-endpoint"
  ports = {
    mqtt_tls = { port = 1823 },
    smf_tls  = { port = 1234 },
  }
  service_id = solacecloud_service.broker_service.id
}

data "solacecloud_connection_endpoints" "test2" {
  service_id = solacecloud_service.broker_service.id
  name       = "main-endpoint"
}

data "solacecloud_connection_endpoint_dns_names" "test" {
  service_id             = solacecloud_service.broker_service.id
  connection_endpoint_id = data.solacecloud_connection_endpoints.test2.endpoints[0].id
}

output "mqtt_fqdn" {
  value = "https://${data.solacecloud_connection_endpoint_dns_names.test.dns_names[0].dns_name}:${data.solacecloud_connection_endpoints.test2.endpoints[0].ports.mqtt_tls.port}"
}`, serviceName),
			},
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
# Create the service first
resource "solacecloud_service" "broker_service" {
  name                 = "%s"
  datacenter_id        = "pcorg-jacks-special-dc"
  service_class_id     = "ENTERPRISE_1K_STANDALONE"
  event_broker_version = "10.25.0.276-48"
  locked               = false
  connection_endpoint = {
    name        = "main-endpoint"
    access_type = "PUBLIC"
    description = "Primary"
    ports = {
      mqtt_tls       = { port = 12345 },
      smf_tls        = { port = 54321 },
      amqp_tls       = { port = 5671 },
      management_tls = { port = 12311 }
    }
  }
}
resource "solacecloud_connection_endpoint" "myendpoint" {
  access_type = "PRIVATE"
  name        = "my-privte-endpoint"
  ports = {
    smf_tls  = { port = 1234 },
	management_tls = { port = 1111 }
  }
  service_id = solacecloud_service.broker_service.id
}

data "solacecloud_connection_endpoints" "test2" {
  service_id = solacecloud_service.broker_service.id
  name       = "main-endpoint"
}

data "solacecloud_connection_endpoint_dns_names" "test" {
  service_id             = solacecloud_service.broker_service.id
  connection_endpoint_id = data.solacecloud_connection_endpoints.test2.endpoints[0].id
}

output "mqtt_fqdn" {
  value = "https://${data.solacecloud_connection_endpoint_dns_names.test.dns_names[0].dns_name}:${data.solacecloud_connection_endpoints.test2.endpoints[0].ports.mqtt_tls.port}"
}`, serviceName),
			},
		},
	})
}

func Int32Ptr(i int32) *int32 {
	return &i
}

func StringPtr(s string) *string {
	return &s
}

func EndpointK8sServiceType(s missioncontrol.GetConnectionEndpointK8sServiceType) *missioncontrol.GetConnectionEndpointK8sServiceType {
	return &s
}

// uses custom mocks to execute a mocked test similar to TestE2eCRUD
func TestCRUDUnit(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	serviceName := "CE_Test_Service_" + randomName

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          serviceName,
		ConnectionEndpointId: "80dx8er674q",
		ServiceId:            "6q1p55o6ovr",
	}
	instance.Init(params)

	if !instance.IsMocked() {
		t.Skip("Skipping unit test in non-mocked mode")
	}

	currentState := missioncontrol.ConnectionEndpointList{}
	defaultEndpoint := &missioncontrol.GetConnectionEndpoint{
		Id:             StringPtr("default-public-endpoint-id"),
		Name:           "main-endpoint",
		Description:    StringPtr("Primary"),
		AccessType:     "PUBLIC",
		K8sServiceType: EndpointK8sServiceType(missioncontrol.GetConnectionEndpointK8sServiceTypeLOADBALANCER),
		Ports: []missioncontrol.ServiceConnectionEndpointPort{
			{Protocol: missioncontrol.ServiceManagementTlsListenPort, Port: Int32Ptr(12311)},
			{Protocol: missioncontrol.ServiceSmfTlsListenPort, Port: Int32Ptr(54321)},
			{Protocol: missioncontrol.ServiceAmqpTlsListenPort, Port: Int32Ptr(5671)},
			{Protocol: missioncontrol.ServiceMqttTlsListenPort, Port: Int32Ptr(12345)},
		},
	}

	newId := "private-endpoint-id"
	new_endpoint := missioncontrol.GetConnectionEndpoint{
		Id:             StringPtr(newId),
		Name:           "my-privte-endpoint",
		Description:    StringPtr(""),
		AccessType:     "PRIVATE",
		K8sServiceType: EndpointK8sServiceType(missioncontrol.GetConnectionEndpointK8sServiceTypeLOADBALANCER),
		Ports: []missioncontrol.ServiceConnectionEndpointPort{
			{Protocol: missioncontrol.ServiceMqttTlsListenPort, Port: Int32Ptr(1823)},
			{Protocol: missioncontrol.ServiceSmfTlsListenPort, Port: Int32Ptr(1234)},
		},
	}

	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints",
		func(r *http.Request) (*http.Response, error) {
			currentState.Data = []missioncontrol.GetConnectionEndpoint{
				// public endpoint created as part of the service
				*defaultEndpoint,
			}
			return httpmock.NewJsonResponse(200,
				currentState)
		})

	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+*defaultEndpoint.Id,
		func(request *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(200,
				missioncontrol.GetConnectionEndpointResponseInternal{
					Data: *defaultEndpoint,
				})
		},
	)
	operationState := missioncontrol.Operation{
		Id: StringPtr("operation-123"),
		OperationType: func() *missioncontrol.JobOperationTypes {
			t := missioncontrol.JobOperationTypesCreateConnectionEndpoint
			return &t
		}(),
		Status: func() *missioncontrol.OperationStatus {
			t := missioncontrol.OperationStatusPENDING
			return &t
		}(),
		ResourceId: StringPtr(params.ServiceId + "/connectionEndpoints/" + newId),
	}

	httpmock.RegisterResponder("POST", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints",
		func(r *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(202,
				missioncontrol.OperationResponse{Data: operationState})
		})

	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/operations/operation-123",
		func(r *http.Request) (*http.Response, error) {
			operationState.Status = func() *missioncontrol.OperationStatus {
				t := missioncontrol.OperationStatusSUCCEEDED
				return &t
			}()
			return httpmock.NewJsonResponse(200,
				missioncontrol.OperationResponse{Data: operationState})
		})
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+newId,
		func(r *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(200,
				missioncontrol.GetConnectionEndpointResponseInternal{
					Data: new_endpoint,
				})
		})

	// Register UPDATE operation
	updateOperationState := missioncontrol.Operation{
		Id: StringPtr("operation-update-123"),
		OperationType: func() *missioncontrol.JobOperationTypes {
			t := missioncontrol.JobOperationTypesUpdateConnectionEndpoint
			return &t
		}(),
		Status: func() *missioncontrol.OperationStatus {
			t := missioncontrol.OperationStatusPENDING
			return &t
		}(),
		ResourceId: StringPtr(params.ServiceId + "/connectionEndpoints/" + newId),
	}

	httpmock.RegisterResponder("PATCH", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+newId,
		func(r *http.Request) (*http.Response, error) {
			// Update the endpoint state to reflect new ports
			new_endpoint.Ports = []missioncontrol.ServiceConnectionEndpointPort{
				{Protocol: missioncontrol.ServiceSmfTlsListenPort, Port: Int32Ptr(1234)},
				{Protocol: missioncontrol.ServiceManagementTlsListenPort, Port: Int32Ptr(1111)},
			}
			return httpmock.NewJsonResponse(202, missioncontrol.OperationResponse{Data: updateOperationState})
		})

	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/operations/operation-update-123",
		func(r *http.Request) (*http.Response, error) {
			updateOperationState.Status = func() *missioncontrol.OperationStatus {
				t := missioncontrol.OperationStatusSUCCEEDED
				return &t
			}()
			return httpmock.NewJsonResponse(200, missioncontrol.OperationResponse{Data: updateOperationState})
		})

	// Register DELETE operation
	deleteOperationState := missioncontrol.Operation{
		Id: StringPtr("operation-delete-123"),
		OperationType: func() *missioncontrol.JobOperationTypes {
			t := missioncontrol.JobOperationTypesDeleteConnectionEndpoint
			return &t
		}(),
		Status: func() *missioncontrol.OperationStatus {
			t := missioncontrol.OperationStatusPENDING
			return &t
		}(),
		ResourceId: StringPtr(params.ServiceId + "/connectionEndpoints/" + newId),
	}

	httpmock.RegisterResponder("DELETE", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+newId,
		func(r *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(202, missioncontrol.OperationResponse{Data: deleteOperationState})
		})

	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/operations/operation-delete-123",
		func(r *http.Request) (*http.Response, error) {
			deleteOperationState.Status = func() *missioncontrol.OperationStatus {
				t := missioncontrol.OperationStatusSUCCEEDED
				return &t
			}()
			return httpmock.NewJsonResponse(200, missioncontrol.OperationResponse{Data: deleteOperationState})
		})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create service with main endpoint and an additional private endpoint
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "broker_service" {
  name                 = "%s"
  datacenter_id        = "gke-gcp-us-central1-a"
  service_class_id     = "ENTERPRISE_1K_STANDALONE"
  event_broker_version = "10.25.0.276-48"
  locked               = false
  connection_endpoint = {
    name        = "main-endpoint"
    access_type = "PUBLIC"
    description = "Primary"
    ports = {
      mqtt_tls       = { port = 12345 },
      smf_tls        = { port = 54321 },
      amqp_tls       = { port = 5671 },
      management_tls = { port = 12311 }
    }
  }
}

resource "solacecloud_connection_endpoint" "myendpoint" {
  access_type = "PRIVATE"
  name        = "my-privte-endpoint"
  ports = {
    mqtt_tls = { port = 1823 },
    smf_tls  = { port = 1234 },
  }
  service_id = solacecloud_service.broker_service.id
}
`, serviceName),
				Check: resource.ComposeTestCheckFunc(
					// Verify service was created
					resource.TestCheckResourceAttrSet("solacecloud_service.broker_service", "id"),
					resource.TestCheckResourceAttr("solacecloud_service.broker_service", "name", serviceName),
					// Verify additional connection endpoint
					resource.TestCheckResourceAttrSet("solacecloud_connection_endpoint.myendpoint", "id"),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.myendpoint", "name", "my-privte-endpoint"),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.myendpoint", "access_type", "PRIVATE"),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.myendpoint", "ports.mqtt_tls.port", "1823"),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.myendpoint", "ports.smf_tls.port", "1234"),
				),
			},
			// Step 2: Update the additional endpoint by changing ports (remove mqtt_tls, add management_tls)
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "broker_service" {
  name                 = "%s"
  datacenter_id        = "gke-gcp-us-central1-a"
  service_class_id     = "ENTERPRISE_1K_STANDALONE"
  event_broker_version = "10.25.0.276-48"
  locked               = false
  connection_endpoint = {
    name        = "main-endpoint"
    access_type = "PUBLIC"
    description = "Primary"
    ports = {
      mqtt_tls       = { port = 12345 },
      smf_tls        = { port = 54321 },
      amqp_tls       = { port = 5671 },
      management_tls = { port = 12311 }
    }
  }
}

resource "solacecloud_connection_endpoint" "myendpoint" {
  access_type = "PRIVATE"
  name        = "my-privte-endpoint"
  ports = {
    smf_tls        = { port = 1234 },
    management_tls = { port = 1111 }
  }
  service_id = solacecloud_service.broker_service.id
}
`, serviceName),
				Check: resource.ComposeTestCheckFunc(
					// Verify the endpoint was updated with new ports
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.myendpoint", "ports.smf_tls.port", "1234"),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint.myendpoint", "ports.management_tls.port", "1111"),
				),
			},
		},
	})
}
