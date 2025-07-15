package provider_test

import (
	"fmt"
	"regexp"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/labstack/gommon/random"
)

// TestServiceCreationSuccess tests successful service creation with mocked API
func TestServiceCreationSuccess(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Success_Test_" + randomName,
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
		},
	})
}

func TestServiceCreationValidationFailure(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "INVALID_CLASS",
		ServiceName:  "Validation_Test_" + randomName,
	}
	// Force mock mode and setup failure mocks
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}
	instance.SetupCreationFailureMocks()
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
`, params.ServiceName, params.ServiceName, params.ServiceClass),
				ExpectError: regexp.MustCompile("Attribute service_class_id value must be one of"),
			},
		},
	})
}

// TestServiceCreationValidationFailure tests creation with invalid parameters
func TestServiceCreationValidationBrokerVersionFailure(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "INVALID_CLASS",
		ServiceName:  "Validation_Test_" + randomName,
	}

	// Force mock mode and setup failure mocks
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	versionTestCases := []struct {
		version     string
		description string
		shouldFail  bool
	}{
		{"10.11", "major.minor format", true},
		{"10.11.1", "major.minor.patch format", true},
		{"10.10.1.112-3", "major.minor.patch-build format", false},
		{"9.12.0", "older version format", true},
		{"11.0", "newer major version", true},
		{"10.11.1-rc1", "release candidate format", true},
		{"invalid", "invalid version string", true},
		{"", "empty version string", true},
		{"10", "incomplete version", true},
		{"10.11.1.2.3", "too many version components", true},
		{"v10.11.1", "version with v prefix", true},
		{"10.11-", "trailing dash", true},
		{"10.11.", "trailing dot", true},
	}

	for _, tc := range versionTestCases {
		t.Run(tc.description, func(t *testing.T) {
			instance := internal.NewTestInstance()
			params := internal.ConfigurableParams{
				ServiceClass: "ENTERPRISE_1K_STANDALONE",
				ServiceName:  "Version_Test_" + random.String(8),
			}

			instance.Init(params)
			if !instance.IsMocked() {
				return
			}

			if tc.shouldFail {
				instance.SetupCreationFailureMocks()
			}

			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name                    = "%s"
  datacenter_id          = "eks-us-east-1"
  service_class_id       = "%s"
  event_broker_version   = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass, tc.version),
						ExpectError: func() *regexp.Regexp {
							if tc.shouldFail {
								return regexp.MustCompile("event broker version format is")
							}
							return nil
						}(),
						Check: func() resource.TestCheckFunc {
							if !tc.shouldFail {
								return resource.ComposeTestCheckFunc(
									resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "id"),
									resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "event_broker_version", tc.version),
								)
							}
							return nil
						}(),
					},
				},
			})
		})
	}
}

// TestServiceCreationAsyncPattern tests async creation pattern with operation polling
func TestServiceCreationAsyncPattern(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_HIGHAVAILABILITY",
		ServiceName:  "Async_Test_" + randomName,
	}

	// Force mock mode and setup async mocks
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}
	instance.SetupAsyncCreationMocks(params)

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
`, params.ServiceName, params.ServiceName, params.ServiceClass),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "id"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "name", params.ServiceName),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "service_class_id", params.ServiceClass),
				),
			},
		},
	})
}

// TestServiceCreationRateLimit tests rate limiting scenarios
func TestServiceCreationRateLimit(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "RateLimit_Test_" + randomName,
	}

	// Force mock mode and setup rate limit mocks
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}
	instance.SetupRateLimitMocks()

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
`, params.ServiceName, params.ServiceName, params.ServiceClass),
				// Should eventually succeed after rate limit retry
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("solacecloud_service."+params.ServiceName, "id"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "name", params.ServiceName),
				),
			},
		},
	})
}

// TestServiceCreationConflict tests resource conflict scenarios
func TestServiceCreationConflict(t *testing.T) {
	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "duplicate-service",
	}

	// Force mock mode and setup conflict mocks
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}
	instance.SetupConflictMocks()

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
`, params.ServiceName, params.ServiceName, params.ServiceClass),
				ExpectError: regexp.MustCompile("HTTP 409"),
			},
		},
	})
}

// TestServiceCreationPartialFailure tests partial creation scenarios
func TestServiceCreationPartialFailure(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Partial_Test_" + randomName,
	}

	// Force mock mode and setup partial creation mocks
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}
	instance.SetupPartialCreationMocks(params)

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
`, params.ServiceName, params.ServiceName, params.ServiceClass),
				ExpectError: regexp.MustCompile("Resource Creation FAILED"),
			},
		},
	})

}
