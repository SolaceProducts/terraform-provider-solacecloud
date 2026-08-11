package provider_test

import (
	"fmt"
	"regexp"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/labstack/gommon/random"
)

func service200KHcl(instance *internal.TestInstance, params internal.ConfigurableParams) string {
	return instance.GetBaseHcl() + fmt.Sprintf(`
resource "solacecloud_service" "%s" {
  name             = "%s"
  datacenter_id    = "gke-gcp-us-central1-a"
  service_class_id = "%s"
}
`, params.ServiceName, params.ServiceName, params.ServiceClass)
}

func TestServiceCreationWith200KServiceClasses(t *testing.T) {
	testCases := map[string]string{
		"standalone":        "ENTERPRISE_200K_STANDALONE",
		"high availability": "ENTERPRISE_200K_HIGHAVAILABILITY",
	}

	for name, serviceClass := range testCases {
		t.Run(name, func(t *testing.T) {
			instance := internal.NewTestInstance()
			params := internal.ConfigurableParams{
				ServiceClass: serviceClass,
				ServiceName:  "Test_200K_" + random.String(8),
			}
			instance.Init(params)
			if !instance.IsMocked() {
				return
			}

			resourceName := "solacecloud_service." + params.ServiceName

			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: service200KHcl(instance, params),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttrSet(resourceName, "id"),
							resource.TestCheckResourceAttr(resourceName, "name", params.ServiceName),
							resource.TestCheckResourceAttr(resourceName, "service_class_id", serviceClass),
							resource.TestCheckResourceAttr(resourceName, "datacenter_id", "gke-gcp-us-central1-a"),
						),
					},
				},
			})
		})
	}
}

func TestServiceClassIsImmutableFor200KUpgrade(t *testing.T) {
	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_100K_STANDALONE",
		ServiceName:  "Test_200K_Immutable_" + random.String(8),
	}
	instance.Init(params)
	if !instance.IsMocked() {
		return
	}

	upgraded := params
	upgraded.ServiceClass = "ENTERPRISE_200K_STANDALONE"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: service200KHcl(instance, params),
				Check: resource.TestCheckResourceAttr(
					"solacecloud_service."+params.ServiceName, "service_class_id", params.ServiceClass),
			},
			{
				Config:      service200KHcl(instance, upgraded),
				ExpectError: regexp.MustCompile("Immutable Attribute Change"),
			},
		},
	})
}
