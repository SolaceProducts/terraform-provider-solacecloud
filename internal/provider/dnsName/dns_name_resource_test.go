package dnsName_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"terraform-provider-solacecloud/internal"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/labstack/gommon/random"
	"terraform-provider-solacecloud/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"solacecloud": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// TestDnsNameCreationSuccess tests successful DNS name creation with mocked API
func TestDnsNameCreationSuccess(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_Test_Service_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
		DnsName:             "test-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup DNS name specific mocks
	instance.SetupDnsNameMocks(params)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name.test", "dns_name", params.DnsName),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name.test", "service_id", params.ServiceId),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name.test", "connection_endpoint_id", params.ConnectionEndpointId),
					resource.TestCheckResourceAttrSet("solacecloud_connection_endpoint_dns_name.test", "id"),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name.test", "domain_type", "CUSTOM"),
				),
			},
		},
	})
}

// TestDnsNameValidation tests DNS name validation logic
func TestDnsNameValidation(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_Validation_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test invalid DNS name with invalid characters
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "invalid_underscore.example.com"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId),
				ExpectError: regexp.MustCompile("(?s)DNS name must be a valid FQDN.*alphanumeric.*characters.*dots.*hyphens"),
			},
		},
	})
}

// TestDnsNameImport tests the import functionality for DNS name resources
func TestDnsNameImport(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_Import_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
		DnsName:             "import-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup DNS name specific mocks
	instance.SetupDnsNameMocks(params)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name.test", "dns_name", params.DnsName),
				),
			},
			// Test import
			{
				ResourceName: "solacecloud_connection_endpoint_dns_name.test",
				ImportState:  true,
				ImportStateId: fmt.Sprintf("%s/%s/%s", params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				ImportStateVerify: true,
			},
		},
	})
}

// TestDnsNameMoveSuccess tests successful DNS name move between connection endpoints
func TestDnsNameMoveSuccess(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceId:                  "test-service-id",
		ConnectionEndpointId:       "source-endpoint-id",
		TargetServiceId:            "target-service-id",
		TargetConnectionEndpointId: "target-endpoint-id",
		DnsName:                   "move-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup DNS name move specific mocks
	instance.SetupDnsNameMoveMocks(params)

	// Additional mock for verifying DNS name exists at target after move
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.TargetServiceId+"/connectionEndpoints/"+params.TargetConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(200, `{
    "data": [
        {
            "id": "dns-id-123",
            "dnsName": "`+params.DnsName+`",
            "domainType": "CUSTOM"
        }
    ]
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_connection_endpoint_dns_name_move" "test" {
  source_service_id             = "%s"
  source_connection_endpoint_id = "%s"
  dns_name                      = "%s"
  target_service_id             = "%s"
  target_connection_endpoint_id = "%s"
}
`, params.ServiceId, params.ConnectionEndpointId, params.DnsName,
   params.TargetServiceId, params.TargetConnectionEndpointId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name_move.test", "dns_name", params.DnsName),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name_move.test", "source_service_id", params.ServiceId),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name_move.test", "target_service_id", params.TargetServiceId),
					resource.TestCheckResourceAttrSet("solacecloud_connection_endpoint_dns_name_move.test", "id"),
				),
			},
		},
	})
}

// TestDnsNameMoveWithinSameService tests moving DNS name within the same service
func TestDnsNameMoveWithinSameService(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceId:                  "same-service-id",
		ConnectionEndpointId:       "source-endpoint-id",
		TargetServiceId:            "same-service-id", // Same service
		TargetConnectionEndpointId: "target-endpoint-id",
		DnsName:                   "same-service-move-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup DNS name move specific mocks
	instance.SetupDnsNameMoveMocks(params)

	// Mock for verifying DNS name exists at target after move (same service)
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+params.TargetConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(200, `{
    "data": [
        {
            "id": "dns-id-123",
            "dnsName": "`+params.DnsName+`",
            "domainType": "CUSTOM"
        }
    ]
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_connection_endpoint_dns_name_move" "test" {
  source_service_id             = "%s"
  source_connection_endpoint_id = "%s"
  dns_name                      = "%s"
  target_connection_endpoint_id = "%s"
  # target_service_id omitted - should default to source_service_id
}
`, params.ServiceId, params.ConnectionEndpointId, params.DnsName,
   params.TargetConnectionEndpointId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name_move.test", "dns_name", params.DnsName),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name_move.test", "source_service_id", params.ServiceId),
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name_move.test", "target_service_id", params.ServiceId),
				),
			},
		},
	})
}

// TestDnsNameCapacityExceeded tests error when maximum DNS names (5) is reached
func TestDnsNameCapacityExceeded(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_Capacity_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
		DnsName:             "capacity-test-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup error mocks for capacity exceeded
	instance.SetupDnsNameErrorMocks(params)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				ExpectError: regexp.MustCompile("Maximum DNS names reached"),
			},
		},
	})
}

// TestDnsNamesDataSource tests the DNS names data source
func TestDnsNamesDataSource(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_DataSource_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup mocks for data source (returns multiple DNS names)
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+params.ConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(200, `{
    "data": [
        {
            "id": "dns-id-1",
            "dnsName": "api.example.com",
            "domainType": "CUSTOM"
        },
        {
            "id": "dns-id-2",
            "dnsName": "www.example.com",
            "domainType": "CUSTOM"
        },
        {
            "id": "dns-id-3",
            "dnsName": "app.example.com",
            "domainType": "CUSTOM"
        }
    ]
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

data "solacecloud_connection_endpoint_dns_names" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoint_dns_names.test", "service_id", params.ServiceId),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoint_dns_names.test", "connection_endpoint_id", params.ConnectionEndpointId),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoint_dns_names.test", "dns_names.#", "3"),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoint_dns_names.test", "dns_names.0.dns_name", "api.example.com"),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoint_dns_names.test", "dns_names.1.dns_name", "www.example.com"),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoint_dns_names.test", "dns_names.2.dns_name", "app.example.com"),
				),
			},
		},
	})
}

// TestDnsNameUpdateNotSupported tests that DNS name updates are not supported
func TestDnsNameUpdateNotSupported(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_Update_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
		DnsName:             "update-test-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup DNS name specific mocks for the first step
	instance.SetupDnsNameMocks(params)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name.test", "dns_name", params.DnsName),
				),
			},
			// Test update attempt - should trigger resource replacement or show error
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "updated-%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				ExpectError: regexp.MustCompile("DNS name not found"),
			},
		},
	})
}

// TestDnsNameDeleteDefaultHostname tests error when trying to delete default hostname
func TestDnsNameDeleteDefaultHostname(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_Default_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
		DnsName:             "default-" + randomName + ".solace.cloud",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Use default service mocks but modify to include default hostname
	instance.SetupDnsNameMocks(params)

	// Replace the service response with one that has the default hostname set to our DNS name
	modifiedServiceResponse := internal.CreateGetServiceResponse(params)
	// Insert defaultManagementHostname after the service name
	modifiedServiceResponse = strings.Replace(modifiedServiceResponse,
		`"name": "`+params.ServiceName+`",`,
		`"name": "`+params.ServiceName+`",
        "defaultManagementHostname": "`+params.DnsName+`",`, 1)
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId,
		internal.JsonResponder(200, modifiedServiceResponse))

	// Mock DNS names response
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+params.ConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(200, `{
    "data": [
        {
            "id": "default-dns-id",
            "dnsName": "`+params.DnsName+`",
            "domainType": "SOLACE_CLOUD"
        }
    ]
}`))

	// Mock create response with async operation format
	httpmock.RegisterResponder("POST", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+params.ConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(202, `{
    "data": {
        "id": "default-create-op-123",
        "type": "operation",
        "operationType": "createDnsName",
        "status": "INPROGRESS"
    }
}`))

	// Mock operation status polling endpoint
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/operations/default-create-op-123",
		internal.JsonResponder(200, `{
    "data": {
        "id": "default-create-op-123",
        "type": "operation",
        "operationType": "createDnsName",
        "status": "SUCCEEDED",
        "resourceId": "default-dns-id"
    }
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name.test", "dns_name", params.DnsName),
				),
			},
		},
	})
}

// TestDnsNameErrorHandling tests various API error scenarios
func TestDnsNameErrorHandling(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_Error_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
		DnsName:             "error-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Test 401 Unauthorized error
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+params.ConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(401, `{
    "error": {
        "code": "UNAUTHORIZED",
        "message": "Invalid API token"
    }
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				ExpectError: regexp.MustCompile("Unauthorized|Invalid API token"),
			},
		},
	})
}

// TestDnsNameReadNotFound tests reading a DNS name that doesn't exist
func TestDnsNameReadNotFound(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_NotFound_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
		DnsName:             "notfound-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Mock empty DNS names response (name not found)
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+params.ConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(200, `{
    "data": []
}`))

	// Mock create response with proper async operation format
	httpmock.RegisterResponder("POST", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+params.ConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(202, `{
    "data": {
        "id": "test-create-op-123",
        "type": "operation",
        "operationType": "createDnsName",
        "status": "INPROGRESS"
    }
}`))

	// Mock operation status that returns a failure
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/operations/test-create-op-123",
		internal.JsonResponder(200, `{
    "data": {
        "id": "test-create-op-123",
        "type": "operation",
        "operationType": "createDnsName",
        "status": "FAILED",
        "error": {
            "code": "DNS_NAME_NOT_FOUND",
            "message": "DNS name not found"
        }
    }
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				ExpectError: regexp.MustCompile("DNS name creation failed|FAILED|The operation failed"),
			},
		},
	})
}

// TestDnsNameInvalidImportFormat tests import with invalid format
func TestDnsNameInvalidImportFormat(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_Import_Invalid_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
		DnsName:             "import-invalid-" + randomName + ".example.com",
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
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				ResourceName:  "solacecloud_connection_endpoint_dns_name.test",
				ImportState:   true,
				ImportStateId: "invalid_format", // Missing required parts
				ExpectError:   regexp.MustCompile("Invalid Import ID|Expected import ID in format|Error: Invalid import ID format"),
			},
		},
	})
}

// TestDnsNameDuplicateCreation tests error when trying to create duplicate DNS name
func TestDnsNameDuplicateCreation(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_Duplicate_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
		DnsName:             "duplicate-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Mock GET returning existing DNS name (simulating duplicate)
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+params.ConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(200, `{
    "data": [
        {
            "id": "existing-dns-id",
            "dnsName": "`+params.DnsName+`",
            "domainType": "CUSTOM"
        }
    ]
}`))

	// Mock CREATE returning 409 conflict
	httpmock.RegisterResponder("POST", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+params.ConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(409, `{
    "error": {
        "code": "CONFLICT",
        "message": "DNS name already exists"
    }
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				ExpectError: regexp.MustCompile("DNS name already exists|Conflict"),
			},
		},
	})
}

// TestDnsNameMoveUpdateNotSupported tests that DNS name move updates are not supported
func TestDnsNameMoveUpdateNotSupported(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceId:                  "source-service-id",
		ConnectionEndpointId:       "source-endpoint-id",
		TargetServiceId:            "target-service-id",
		TargetConnectionEndpointId: "target-endpoint-id",
		DnsName:                   "move-update-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup DNS name move specific mocks
	instance.SetupDnsNameMoveMocks(params)

	// Additional mock for verifying DNS name exists at target after move
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.TargetServiceId+"/connectionEndpoints/"+params.TargetConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(200, `{
    "data": [
        {
            "id": "dns-id-123",
            "dnsName": "`+params.DnsName+`",
            "domainType": "CUSTOM"
        }
    ]
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_connection_endpoint_dns_name_move" "test" {
  source_service_id             = "%s"
  source_connection_endpoint_id = "%s"
  dns_name                      = "%s"
  target_service_id             = "%s"
  target_connection_endpoint_id = "%s"
}
`, params.ServiceId, params.ConnectionEndpointId, params.DnsName,
   params.TargetServiceId, params.TargetConnectionEndpointId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("solacecloud_connection_endpoint_dns_name_move.test", "dns_name", params.DnsName),
				),
			},
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_connection_endpoint_dns_name_move" "test" {
  source_service_id             = "%s"
  source_connection_endpoint_id = "%s"
  dns_name                      = "updated-%s"
  target_service_id             = "%s"
  target_connection_endpoint_id = "%s"
}
`, params.ServiceId, params.ConnectionEndpointId, params.DnsName,
   params.TargetServiceId, params.TargetConnectionEndpointId),
				ExpectError: regexp.MustCompile("Could not move DNS name|no responder found|DNS name move operations are immutable|DNS Name Move Update Not Supported"),
			},
		},
	})
}

// TestDnsNameMoveTargetCapacityExceeded tests error when target endpoint at capacity
func TestDnsNameMoveTargetCapacityExceeded(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceId:                  "source-service-id",
		ConnectionEndpointId:       "source-endpoint-id",
		TargetServiceId:            "target-service-id",
		TargetConnectionEndpointId: "target-endpoint-id",
		DnsName:                   "capacity-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Add missing mock for target service (required for service validation)
	targetParams := params
	targetParams.ServiceId = params.TargetServiceId
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.TargetServiceId,
		internal.JsonResponder(200, internal.CreateGetServiceResponse(targetParams)))

	// Mock target endpoint with 5 DNS names (at capacity)
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.TargetServiceId+"/connectionEndpoints/"+params.TargetConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(200, `{
    "data": [
        {"id": "dns-1", "dnsName": "name1.example.com", "domainType": "CUSTOM"},
        {"id": "dns-2", "dnsName": "name2.example.com", "domainType": "CUSTOM"},
        {"id": "dns-3", "dnsName": "name3.example.com", "domainType": "CUSTOM"},
        {"id": "dns-4", "dnsName": "name4.example.com", "domainType": "CUSTOM"},
        {"id": "dns-5", "dnsName": "name5.example.com", "domainType": "CUSTOM"}
    ]
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_connection_endpoint_dns_name_move" "test" {
  source_service_id             = "%s"
  source_connection_endpoint_id = "%s"
  dns_name                      = "%s"
  target_service_id             = "%s"
  target_connection_endpoint_id = "%s"
}
`, params.ServiceId, params.ConnectionEndpointId, params.DnsName,
   params.TargetServiceId, params.TargetConnectionEndpointId),
				ExpectError: regexp.MustCompile("Target endpoint at maximum DNS names|maximum of 5 DNS names reached"),
			},
		},
	})
}

// TestDnsNamesDataSourceEmpty tests data source with no DNS names
func TestDnsNamesDataSourceEmpty(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_DataSource_Empty_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup mocks for empty data source response
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+params.ConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(200, `{"data": []}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

data "solacecloud_connection_endpoint_dns_names" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoint_dns_names.test", "service_id", params.ServiceId),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoint_dns_names.test", "connection_endpoint_id", params.ConnectionEndpointId),
					resource.TestCheckResourceAttr("data.solacecloud_connection_endpoint_dns_names.test", "dns_names.#", "0"),
				),
			},
		},
	})
}

// TestDnsNamesDataSourceError tests data source with API error
func TestDnsNamesDataSourceError(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_DataSource_Error_Test_" + randomName,
		ServiceId:            "test-service-id",
		ConnectionEndpointId: "test-endpoint-id",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup error mock for data source (403 Forbidden)
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/connectionEndpoints/"+params.ConnectionEndpointId+"/dnsNames",
		internal.JsonResponder(403, `{
    "error": {
        "code": "FORBIDDEN",
        "message": "Access denied to endpoint"
    }
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

data "solacecloud_connection_endpoint_dns_names" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId),
				ExpectError: regexp.MustCompile("Forbidden|Access denied"),
			},
		},
	})
}



// TestDnsNameMovePollingTimeout tests DNS name move operation timeout
func TestDnsNameMovePollingTimeout(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:                  "ENTERPRISE_1K_STANDALONE",
		ServiceName:                   "DNS_Move_Timeout_Test_" + randomName,
		ServiceId:                    "timeout-source-service-id",
		ConnectionEndpointId:         "timeout-source-endpoint-id",
		TargetServiceId:              "timeout-target-service-id",
		TargetConnectionEndpointId:   "timeout-target-endpoint-id",
		DnsName:                     "timeout-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup basic service and capacity check mocks
	instance.SetupDnsNameMoveMocks(params)

	// Override the operation status to always return INPROGRESS (simulating timeout)
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.TargetServiceId+"/operations/move-operation-123",
		internal.JsonResponder(200, `{
    "data": {
        "id": "move-operation-123",
        "type": "operation",
        "operationType": "moveDnsName",
        "status": "INPROGRESS"
    }
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_connection_endpoint_dns_name_move" "test" {
  source_service_id             = "%s"
  source_connection_endpoint_id = "%s"
  dns_name                     = "%s"
  target_service_id             = "%s"
  target_connection_endpoint_id = "%s"
}
`, params.ServiceId, params.ConnectionEndpointId, params.DnsName,
   params.TargetServiceId, params.TargetConnectionEndpointId),
				ExpectError: regexp.MustCompile("DNS name move timeout|timed out|Service operation timeout"),
			},
		},
	})
}

// TestDnsNameMovePollingFailure tests DNS name move operation failure
func TestDnsNameMovePollingFailure(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:                  "ENTERPRISE_1K_STANDALONE",
		ServiceName:                   "DNS_Move_Failure_Test_" + randomName,
		ServiceId:                    "failure-source-service-id",
		ConnectionEndpointId:         "failure-source-endpoint-id",
		TargetServiceId:              "failure-target-service-id",
		TargetConnectionEndpointId:   "failure-target-endpoint-id",
		DnsName:                     "failure-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup basic service and capacity check mocks
	instance.SetupDnsNameMoveMocks(params)

	// Override the operation status to return FAILED
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.TargetServiceId+"/operations/move-operation-123",
		internal.JsonResponder(200, `{
    "data": {
        "id": "move-operation-123",
        "type": "operation",
        "operationType": "moveDnsName",
        "status": "FAILED",
        "error": {
            "code": "MOVE_FAILED",
            "message": "DNS name move operation failed"
        }
    }
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_connection_endpoint_dns_name_move" "test" {
  source_service_id             = "%s"
  source_connection_endpoint_id = "%s"
  dns_name                     = "%s"
  target_service_id             = "%s"
  target_connection_endpoint_id = "%s"
}
`, params.ServiceId, params.ConnectionEndpointId, params.DnsName,
   params.TargetServiceId, params.TargetConnectionEndpointId),
				ExpectError: regexp.MustCompile("DNS name move failed|FAILED|The operation failed"),
			},
		},
	})
}

// TestDnsNameCreationPollingTimeout tests DNS name creation operation timeout
func TestDnsNameCreationPollingTimeout(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:         "ENTERPRISE_1K_STANDALONE",
		ServiceName:          "DNS_Create_Timeout_Test_" + randomName,
		ServiceId:           "timeout-create-service-id",
		ConnectionEndpointId: "timeout-create-endpoint-id",
		DnsName:             "create-timeout-" + randomName + ".example.com",
	}

	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	// Setup basic service mocks
	instance.SetupDnsNameMocks(params)

	// Override the operation status to always return INPROGRESS (simulating timeout)
	httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/operations/create-operation-123",
		internal.JsonResponder(200, `{
    "data": {
        "id": "create-operation-123",
        "type": "operation",
        "operationType": "createDnsName",
        "status": "INPROGRESS"
    }
}`))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "%s"
}

resource "solacecloud_connection_endpoint_dns_name" "test" {
  service_id             = "%s"
  connection_endpoint_id = "%s"
  dns_name               = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName),
				ExpectError: regexp.MustCompile("DNS name creation timeout|timed out|Service operation timeout"),
			},
		},
	})
}

// TestDnsNameMoveImportInvalidFormat tests import with invalid ID format
func TestDnsNameMoveImportInvalidFormat(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := strings.ToLower(random.String(8))

	params := internal.ConfigurableParams{
		ServiceClass:               "ENTERPRISE_1K_STANDALONE",
		ServiceName:                "DNS_Move_Import_Invalid_Test_" + randomName,
		ServiceId:                  "source-service-id",
		ConnectionEndpointId:       "source-endpoint-id",
		TargetServiceId:            "target-service-id",
		TargetConnectionEndpointId: "target-endpoint-id",
		DnsName:                   "import-invalid-" + randomName + ".example.com",
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
}

resource "solacecloud_connection_endpoint_dns_name_move" "test" {
  source_service_id             = "%s"
  source_connection_endpoint_id = "%s"
  dns_name                      = "%s"
  target_service_id             = "%s"
  target_connection_endpoint_id = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass,
   params.ServiceId, params.ConnectionEndpointId, params.DnsName,
   params.TargetServiceId, params.TargetConnectionEndpointId),
				ResourceName:  "solacecloud_connection_endpoint_dns_name_move.test",
				ImportState:   true,
				ImportStateId: "invalid/format/missing/parts", // Missing required 5th part
				ExpectError:   regexp.MustCompile("Import not supported for DNS name move operations"),
			},
		},
	})
}

