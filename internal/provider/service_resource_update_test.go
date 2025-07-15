package provider_test

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/labstack/gommon/random"
)

func TestUpdateWorksMocked(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Azure_TF_Service" + randomName,
		ServiceId:    "myid",
		OwnerId:      "ownerid",
	}
	// only run against mocks
	if !instance.IsMocked() {
		return
	}
	instance.Init(params)

	if instance.IsMocked() {
		httpmock.RegisterResponder("PATCH", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId,
			func(r *http.Request) (*http.Response, error) {

				// Clone the params struct to avoid modifying the original
				updatedParams := internal.ConfigurableParams{
					ServiceClass: params.ServiceClass,
					ServiceName:  params.ServiceName + "New",
					ServiceId:    params.ServiceId,
					OwnerId:      "owneridnew",
				}
				instance.SetupDefaultMocks(updatedParams)
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader(internal.CreateGetServiceResponse(params))),
					Status:     "200 OK",
					StatusCode: http.StatusOK,
				}, nil
			})
	}

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
  owner_id         = "` + params.OwnerId + `"
}
`,
			},
			{
				Config: instance.GetBaseHcl() + `
resource "solacecloud_service" "` + params.ServiceName + `" {
  name             = "` + params.ServiceName + `New"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "` + params.ServiceClass + `"
  owner_id         = "owneridnew"
}
`,
			},
		},
	})
}

func TestUpdateWorksNotMocked(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Azure_TF_Service" + randomName,
		ServiceId:    "myid",
	}
	instance.Init(params)

	if instance.IsMocked() {
		return
	}

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
			{
				Config: instance.GetBaseHcl() + `
resource "solacecloud_service" "` + params.ServiceName + `" {
  name             = "` + params.ServiceName + `New"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "` + params.ServiceClass + `"
}
`,
			},
		},
	})
}

func TestUpdateSpoolWorks(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass:  "ENTERPRISE_1K_STANDALONE",
		ServiceName:   "Azure_TF_ScaleUp_Service" + randomName,
		ServiceId:     "myid",
		MaxSpoolUsage: 200,
	}
	instance.Init(params)

	if instance.IsMocked() {
		// Mock the PATCH request for scaling up max_spool_usage
		httpmock.RegisterResponder("PATCH", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/messageSpool",
			func(r *http.Request) (*http.Response, error) {

				updatedParams := params
				updatedParams.MaxSpoolUsage = 400
				instance.SetupDefaultMocks(updatedParams)
				return internal.JsonResponder(202, `{
				"data": {
					"id": "update-operation-123",
					"type": "operation",
					"operationType": "updateService",
					"createdBy": "67tr8tkuel",
					"createdTime": "2025-02-19T01:33:04Z",
					"completedTime": "2025-02-19T01:33:04Z",
					"resourceId": "`+params.ServiceId+`",
					"resourceType": "service",
					"status": "INPROGRESS",
					"error": null
				}
			}`)(r)
			})

		// Mock the operation status check endpoint
		httpmock.RegisterResponder("GET", instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/operations/update-operation-123",
			func(r *http.Request) (*http.Response, error) {

				return internal.JsonResponder(200, `{
					"data": {
						"id": "update-operation-123",
						"type": "operation",
						"operationType": "updateService",
						"createdBy": "67tr8tkuel",
						"createdTime": "2025-02-19T01:33:04Z",
						"completedTime": "2025-02-19T01:33:06Z",
						"resourceId": "`+params.ServiceId+`",
						"resourceType": "service",
						"status": "SUCCEEDED",
						"error": null
					}
				}`)(r)
			})
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create service with initial max_spool_usage
			{
				Config: instance.GetBaseHcl() + `
resource "solacecloud_service" "` + params.ServiceName + `" {
  name             = "` + params.ServiceName + `"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "` + params.ServiceClass + `"
  max_spool_usage  = 200
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "max_spool_usage", "200"),
				),
			},
			// Scale up max_spool_usage
			{
				Config: instance.GetBaseHcl() + `
resource "solacecloud_service" "` + params.ServiceName + `" {
  name             = "` + params.ServiceName + `"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "` + params.ServiceClass + `"
  max_spool_usage  = 400
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "max_spool_usage", "400"),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "name", params.ServiceName),
					resource.TestCheckResourceAttr("solacecloud_service."+params.ServiceName, "service_class_id", params.ServiceClass),
				),
			},
		},
	})
}

func TestUpdateImmutableValueFails(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Azure_TF_Service" + randomName,
		ServiceId:    "myid",
		OwnerId:      "ownerid",
	}
	instance.Init(params)

	// only run against mocks
	if !instance.IsMocked() {
		return
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create service with initial values
			{
				Config: instance.GetBaseHcl() + `
resource "solacecloud_service" "` + params.ServiceName + `" {
  name             = "` + params.ServiceName + `"
  datacenter_id    = "eks-us-east-1"
  service_class_id = "` + params.ServiceClass + `"
}
`,
			},
			// Attempt to update immutable field (datacenter_id) - should fail
			{
				Config: instance.GetBaseHcl() + `
resource "solacecloud_service" "` + params.ServiceName + `" {
  name             = "` + params.ServiceName + `"
  datacenter_id    = "eks-us-west-1"
  service_class_id = "` + params.ServiceClass + `"
}
`,
				ExpectError: regexp.MustCompile("You cannot change"),
			},
		},
	})
}
