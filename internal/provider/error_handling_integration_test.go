package provider_test

import (
	"net/http"
	"regexp"
	"strings"
	"terraform-provider-solacecloud/internal"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/missioncontrol"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/labstack/gommon/random"
)

func TestErrorHandling_ServiceCreation_AuthenticationFailure(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Auth_Fail_Service_" + randomName,
	}
	instance.Init(params)

	// Setup mock to return 401 Unauthorized
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(
		"POST",
		instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices",
		httpmock.NewStringResponder(401, `{
		"message": "Invalid API token",
		"errorId": "auth-error-123"
		}`))

	resource.UnitTest(t, resource.TestCase{
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
				ExpectError: regexp.MustCompile("Authentication Failed"),
			},
		},
	})
}

func TestErrorHandling_ServiceCreation_BadRequest(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_HIGHAVAILABILITY",
		ServiceName:  "Bad_Request_Service_" + randomName,
	}
	instance.Init(params)

	// Setup mock to return 400 Bad Request
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(
		"POST",
		instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices",
		mockBadRequestResponder,
	)

	resource.UnitTest(t, resource.TestCase{
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
				ExpectError: regexp.MustCompile("Invalid service class specified"),
			},
		},
	})
}

func mockBadRequestResponder(req *http.Request) (*http.Response, error) {
	resp := httpmock.NewStringResponse(400, `{
	"message": "Invalid service class specified",
	"errorId": "validation-error-456"
	}`)
	resp.Header.Set("Content-Type", "application/json")
	return resp, nil
}

func TestErrorHandling_ServiceCreation_ServiceUnavailable(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Unavailable_Service_" + randomName,
	}
	instance.Init(params)

	// Setup mock to return 503 Service Unavailable
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(
		"POST",
		instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices",
		httpmock.NewStringResponder(503, `{
		"message": "Service temporarily unavailable due to maintenance",
		"errorId": "maintenance-error-202"
		}`))

	resource.UnitTest(t, resource.TestCase{
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
				ExpectError: regexp.MustCompile("Service Unavailable"),
			},
		},
	})
}

func TestErrorHandling_UnexpectedStatusCode(t *testing.T) {
	instance := internal.NewTestInstance()
	randomName := random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		ServiceName:  "Unexpected_Error_Service_" + randomName,
	}
	instance.Init(params)

	// Setup mock to return 500 Internal Server Error
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(
		"POST",
		instance.GetBaseURL()+"/api/v2/missionControl/eventBrokerServices",
		httpmock.NewStringResponder(500, `Internal server error occurred`))

	resource.UnitTest(t, resource.TestCase{
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
				ExpectError: regexp.MustCompile("Internal server error occurred"),
			},
		},
	})
}

func TestErrorHandling_DirectAPICall(t *testing.T) {
	// Test the error handling adaptor with a simulated API response
	t.Run("Direct API call with 401 error", func(t *testing.T) {
		// Simulate an API response structure similar to what the generated client would return
		httpResponse := &http.Response{
			StatusCode: 401,
			Status:     "401 Unauthorized",
		}

		json401 := &missioncontrol.ErrorResponse{
			Message: stringPtr("Invalid API token provided"),
		}

		// Create error handler
		errorHandler := shared.NewMissionControlErrorResponseAdaptor(
			http.StatusAccepted, // Expected status for service creation
			[]byte("Unauthorized"),
			httpResponse,
			nil,     // JSON400
			json401, // JSON401
			nil,     // JSON403
			nil,     // JSON404
			nil,     // JSON503
		)

		// Test error handling
		var diagnostics diag.Diagnostics
		hasError := errorHandler.HandleError(&diagnostics)

		if !hasError {
			t.Errorf("Expected error to be handled, but got hasError = false")
		}

		if !diagnostics.HasError() {
			t.Errorf("Expected diagnostics to have error, but got HasError() = false")
		}

		if len(diagnostics.Errors()) != 1 {
			t.Errorf("Expected 1 error, got %d", len(diagnostics.Errors()))
		}

		errorDiag := diagnostics.Errors()[0]
		if errorDiag.Summary() != "Authentication Failed" {
			t.Errorf("Expected error summary 'Authentication Failed', got '%s'", errorDiag.Summary())
		}

		// Check that the error detail contains helpful information
		detail := errorDiag.Detail()
		expectedStrings := []string{
			"Received HTTP 401 Unauthorized",
			"Verify your API token is correct",
			"SOLACECLOUD_API_TOKEN",
			"provider configuration",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(detail, expected) {
				t.Errorf("Expected error detail to contain '%s', but it didn't. Detail: %s", expected, detail)
			}
		}
	})
}

func stringPtr(s string) *string {
	return &s
}
