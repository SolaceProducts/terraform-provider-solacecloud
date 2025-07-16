package environment_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"

	"terraform-provider-solacecloud/internal"
	"terraform-provider-solacecloud/internal/provider"
)

func TestAccEnvironmentDataSource_Basic(t *testing.T) {
	// Create a test instance
	instance := internal.NewTestInstance()
	instance.Init(internal.ConfigurableParams{})

	// Set up mocks for the environment API calls
	if instance.IsMocked() {
		// Mock the search environments API call
		httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/platform/environments?name=Default",
			internal.JsonResponder(http.StatusOK, `{
				"data": [
					{
						"id": "env-123456",
						"name": "Default",
						"type": "environment"
					}
				]
			}`))

		// Mock the get environment by ID API call
		httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/platform/environments/env-123456",
			internal.JsonResponder(http.StatusOK, `{
				"data": {
					"id": "env-123456",
					"name": "Default",
					"type": "environment"
				}
			}`))
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { /* TODO: Add any pre-check logic if needed */ },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: instance.GetBaseHcl() + testAccEnvironmentDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.solacecloud_environment.test", "name", "Default"),
					resource.TestCheckResourceAttrSet("data.solacecloud_environment.test", "id"),
					resource.TestCheckResourceAttrSet("data.solacecloud_environment.test", "type"),
				),
			},
		},
	})
}

func testAccEnvironmentDataSourceConfig() string {
	return `
data "solacecloud_environment" "test" {
  name = "Default"
}
`
}

// testAccProtoV6ProviderFactories is a shared provider factory for all tests
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"solacecloud": providerserver.NewProtocol6WithError(provider.New("test")()),
}
