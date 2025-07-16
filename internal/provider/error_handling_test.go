package provider

import (
	"net/http"
	"strings"
	"terraform-provider-solacecloud/missioncontrol"
	"terraform-provider-solacecloud/internal/shared"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestErrorResponseAdaptor_Status(t *testing.T) {
	tests := []struct {
		name         string
		httpResponse *http.Response
		expected     string
	}{
		{
			name: "with valid http response",
			httpResponse: &http.Response{
				Status:     "200 OK",
				StatusCode: 200,
			},
			expected: "200 OK",
		},
		{
			name:         "with nil http response",
			httpResponse: nil,
			expected:     "", // http.StatusText(0) returns empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adaptor := shared.ErrorResponseAdaptor{
				HTTPResponse: tt.httpResponse,
			}
			result := adaptor.Status()
			if result != tt.expected {
				t.Errorf("Status() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorResponseAdaptor_StatusCode(t *testing.T) {
	tests := []struct {
		name         string
		httpResponse *http.Response
		expected     int
	}{
		{
			name: "with valid http response",
			httpResponse: &http.Response{
				StatusCode: 404,
			},
			expected: 404,
		},
		{
			name:         "with nil http response",
			httpResponse: nil,
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adaptor := shared.ErrorResponseAdaptor{
				HTTPResponse: tt.httpResponse,
			}
			result := adaptor.StatusCode()
			if result != tt.expected {
				t.Errorf("StatusCode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewMissionControlErrorResponseAdaptor(t *testing.T) {
	expectedStatusCode := 200
	body := []byte("test body")
	httpResponse := &http.Response{StatusCode: 404}
	json400 := &missioncontrol.ErrorResponse{}
	json401 := &missioncontrol.ErrorResponse{}
	json403 := &missioncontrol.ErrorResponse{}
	json404 := &missioncontrol.ErrorResponse{}
	json503 := &missioncontrol.ErrorResponse{}

	adaptor := shared.NewMissionControlErrorResponseAdaptor(
		expectedStatusCode,
		body,
		httpResponse,
		json400,
		json401,
		json403,
		json404,
		json503,
	)

	if adaptor.ExpectedStatusCode != expectedStatusCode {
		t.Errorf("ExpectedStatusCode = %v, want %v", adaptor.ExpectedStatusCode, expectedStatusCode)
	}
	if string(adaptor.Body) != string(body) {
		t.Errorf("Body = %v, want %v", adaptor.Body, body)
	}
	if adaptor.HTTPResponse != httpResponse {
		t.Errorf("HTTPResponse = %v, want %v", adaptor.HTTPResponse, httpResponse)
	}

	// Check that the JSON400 is a MissionControlErrorResponse wrapping the original error response
	mcErr400, ok := adaptor.JSON400.(*shared.MissionControlErrorResponse)
	if !ok {
		t.Errorf("JSON400 is not a *shared.MissionControlErrorResponse")
	} else if mcErr400.ErrorResponse != json400 {
		t.Errorf("JSON400.ErrorResponse = %v, want %v", mcErr400.ErrorResponse, json400)
	}

	// Check that the JSON401 is a MissionControlErrorResponse wrapping the original error response
	mcErr401, ok := adaptor.JSON401.(*shared.MissionControlErrorResponse)
	if !ok {
		t.Errorf("JSON401 is not a *shared.MissionControlErrorResponse")
	} else if mcErr401.ErrorResponse != json401 {
		t.Errorf("JSON401.ErrorResponse = %v, want %v", mcErr401.ErrorResponse, json401)
	}

	// Check that the JSON403 is a MissionControlErrorResponse wrapping the original error response
	mcErr403, ok := adaptor.JSON403.(*shared.MissionControlErrorResponse)
	if !ok {
		t.Errorf("JSON403 is not a *shared.MissionControlErrorResponse")
	} else if mcErr403.ErrorResponse != json403 {
		t.Errorf("JSON403.ErrorResponse = %v, want %v", mcErr403.ErrorResponse, json403)
	}

	// Check that the JSON404 is a MissionControlErrorResponse wrapping the original error response
	mcErr404, ok := adaptor.JSON404.(*shared.MissionControlErrorResponse)
	if !ok {
		t.Errorf("JSON404 is not a *shared.MissionControlErrorResponse")
	} else if mcErr404.ErrorResponse != json404 {
		t.Errorf("JSON404.ErrorResponse = %v, want %v", mcErr404.ErrorResponse, json404)
	}

	// Check that the JSON503 is a MissionControlErrorResponse wrapping the original error response
	mcErr503, ok := adaptor.JSON503.(*shared.MissionControlErrorResponse)
	if !ok {
		t.Errorf("JSON503 is not a *shared.MissionControlErrorResponse")
	} else if mcErr503.ErrorResponse != json503 {
		t.Errorf("JSON503.ErrorResponse = %v, want %v", mcErr503.ErrorResponse, json503)
	}
}

func TestErrorResponseAdaptor_HandleError_NoError(t *testing.T) {
	adaptor := &shared.ErrorResponseAdaptor{
		ExpectedStatusCode: 200,
		HTTPResponse: &http.Response{
			StatusCode: 200,
		},
	}

	var diagnostics diag.Diagnostics
	hasError := adaptor.HandleError(&diagnostics)

	if hasError {
		t.Errorf("HandleError() = %v, want false", hasError)
	}
	if diagnostics.HasError() {
		t.Errorf("diagnostics.HasError() = %v, want false", diagnostics.HasError())
	}
}

func TestErrorResponseAdaptor_HandleError_Unauthorized(t *testing.T) {
	adaptor := &shared.ErrorResponseAdaptor{
		ExpectedStatusCode: 200,
		HTTPResponse: &http.Response{
			StatusCode: 401,
		},
	}

	var diagnostics diag.Diagnostics
	hasError := adaptor.HandleError(&diagnostics)

	if !hasError {
		t.Errorf("HandleError() = %v, expected true", hasError)
	}
	if !diagnostics.HasError() {
		t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
	}
	if len(diagnostics.Errors()) != 1 {
		t.Errorf("len(diagnostics.Errors()) = %v, expected 1", len(diagnostics.Errors()))
	}

	errorDiag := diagnostics.Errors()[0]
	if errorDiag.Summary() != "Authentication Failed" {
		t.Errorf("errorDiag.Summary() = %v, expected 'Authentication Failed'", errorDiag.Summary())
	}
	if !strings.Contains(errorDiag.Detail(), "Received HTTP 401 Unauthorized") {
		t.Errorf("errorDiag.Detail() should contain 'Received HTTP 401 Unauthorized', got: %v", errorDiag.Detail())
	}
	if !strings.Contains(errorDiag.Detail(), "Verify your API token is correct") {
		t.Errorf("errorDiag.Detail() should contain 'Verify your API token is correct', got: %v", errorDiag.Detail())
	}
	if !strings.Contains(errorDiag.Detail(), "SOLACECLOUD_API_TOKEN") {
		t.Errorf("errorDiag.Detail() should contain 'SOLACECLOUD_API_TOKEN', got: %v", errorDiag.Detail())
	}
}

func TestErrorResponseAdaptor_HandleError_BadRequest(t *testing.T) {
	tests := []struct {
		name           string
		json400        *missioncontrol.ErrorResponse
		expectedMsg    string
		expectedDetail string
	}{
		{
			name: "with error message",
			json400: &missioncontrol.ErrorResponse{
				Message: stringPtr("Invalid request parameters"),
			},
			expectedMsg:    "Bad Request",
			expectedDetail: "Invalid request parameters",
		},
		{
			name:        "without error message",
			json400:     nil,
			expectedMsg: "Bad Request",
			expectedDetail: "Received HTTP 400 Bad Request. " +
				"This usually indicates a malformed request or missing required parameters. Check your request body and parameters.",
		},
		{
			name: "with nil message",
			json400: &missioncontrol.ErrorResponse{
				Message: nil,
			},
			expectedMsg: "Bad Request",
			expectedDetail: "Received HTTP 400 Bad Request. " +
				"This usually indicates a malformed request or missing required parameters. Check your request body and parameters.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var json400Provider shared.ErrorResponseProvider
			if tt.json400 != nil {
				json400Provider = &shared.MissionControlErrorResponse{ErrorResponse: tt.json400}
			}

			adaptor := &shared.ErrorResponseAdaptor{
				ExpectedStatusCode: 200,
				HTTPResponse: &http.Response{
					StatusCode: 400,
				},
				JSON400: json400Provider,
			}

			var diagnostics diag.Diagnostics
			hasError := adaptor.HandleError(&diagnostics)

			if !hasError {
				t.Errorf("HandleError() = %v, expected true", hasError)
			}
			if !diagnostics.HasError() {
				t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
			}
			if len(diagnostics.Errors()) != 1 {
				t.Errorf("len(diagnostics.Errors()) = %v, expected 1", len(diagnostics.Errors()))
			}

			errorDiag := diagnostics.Errors()[0]
			if errorDiag.Summary() != tt.expectedMsg {
				t.Errorf("errorDiag.Summary() = %v, expected %v", errorDiag.Summary(), tt.expectedMsg)
			}
			if errorDiag.Detail() != tt.expectedDetail {
				t.Errorf("errorDiag.Detail() = %v, expected %v", errorDiag.Detail(), tt.expectedDetail)
			}
		})
	}
}

func TestErrorResponseAdaptor_HandleError_Forbidden(t *testing.T) {
	tests := []struct {
		name           string
		json403        *missioncontrol.ErrorResponse
		expectedMsg    string
		expectedDetail string
	}{
		{
			name: "with error message",
			json403: &missioncontrol.ErrorResponse{
				Message: stringPtr("Access denied to resource"),
			},
			expectedMsg:    "Forbidden",
			expectedDetail: "Access denied to resource",
		},
		{
			name:        "without error message",
			json403:     nil,
			expectedMsg: "Forbidden",
		},
		{
			name: "with nil message",
			json403: &missioncontrol.ErrorResponse{
				Message: nil,
			},
			expectedMsg: "Forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var json403Provider shared.ErrorResponseProvider
			if tt.json403 != nil {
				json403Provider = &shared.MissionControlErrorResponse{ErrorResponse: tt.json403}
			}

			adaptor := &shared.ErrorResponseAdaptor{
				ExpectedStatusCode: 200,
				HTTPResponse: &http.Response{
					StatusCode: 403,
				},
				JSON403: json403Provider,
			}

			var diagnostics diag.Diagnostics
			hasError := adaptor.HandleError(&diagnostics)

			if !hasError {
				t.Errorf("HandleError() = %v, expected true", hasError)
			}
			if !diagnostics.HasError() {
				t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
			}
			if len(diagnostics.Errors()) != 1 {
				t.Errorf("len(diagnostics.Errors()) = %v, expected 1", len(diagnostics.Errors()))
			}

			errorDiag := diagnostics.Errors()[0]
			if errorDiag.Summary() != tt.expectedMsg {
				t.Errorf("errorDiag.Summary() = %v, expected %v", errorDiag.Summary(), tt.expectedMsg)
			}
		})
	}
}

func TestErrorResponseAdaptor_HandleError_NotFound(t *testing.T) {
	tests := []struct {
		name           string
		json404        *missioncontrol.ErrorResponse
		expectedMsg    string
		expectedDetail string
	}{
		{
			name: "with error message",
			json404: &missioncontrol.ErrorResponse{
				Message: stringPtr("Resource not found"),
			},
			expectedMsg:    "Not Found",          // Now correctly uses JSON404.Message
			expectedDetail: "Resource not found", // Now correctly uses JSON404.Message
		},
		{
			name:        "without error message",
			json404:     nil,
			expectedMsg: "Not Found",
		},
		{
			name: "with nil message",
			json404: &missioncontrol.ErrorResponse{
				Message: nil,
			},
			expectedMsg: "Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var json404Provider shared.ErrorResponseProvider
			if tt.json404 != nil {
				json404Provider = &shared.MissionControlErrorResponse{ErrorResponse: tt.json404}
			}

			adaptor := &shared.ErrorResponseAdaptor{
				ExpectedStatusCode: 200,
				HTTPResponse: &http.Response{
					StatusCode: 404,
				},
				JSON404: json404Provider,
			}

			var diagnostics diag.Diagnostics
			hasError := adaptor.HandleError(&diagnostics)

			if !hasError {
				t.Errorf("HandleError() = %v, expected true", hasError)
			}
			if !diagnostics.HasError() {
				t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
			}
			if len(diagnostics.Errors()) != 1 {
				t.Errorf("len(diagnostics.Errors()) = %v, expected 1", len(diagnostics.Errors()))
			}

			errorDiag := diagnostics.Errors()[0]
			if errorDiag.Summary() != tt.expectedMsg {
				t.Errorf("errorDiag.Summary() = %v, expected %v", errorDiag.Summary(), tt.expectedMsg)
			}
		})
	}
}

func TestErrorResponseAdaptor_HandleError_ServiceUnavailable(t *testing.T) {
	tests := []struct {
		name           string
		json503        *missioncontrol.ErrorResponse
		expectedMsg    string
		expectedDetail string
	}{
		{
			name: "with error message",
			json503: &missioncontrol.ErrorResponse{
				Message: stringPtr("Service temporarily unavailable"),
			},
			expectedMsg:    "Service Unavailable",             // Now correctly uses JSON503.Message
			expectedDetail: "Service temporarily unavailable", // Now correctly uses JSON503.Message
		},
		{
			name:        "without error message",
			json503:     nil,
			expectedMsg: "Service Unavailable",
		},
		{
			name: "with nil message",
			json503: &missioncontrol.ErrorResponse{
				Message: nil,
			},
			expectedMsg: "Service Unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var json503Provider shared.ErrorResponseProvider
			if tt.json503 != nil {
				json503Provider = &shared.MissionControlErrorResponse{ErrorResponse: tt.json503}
			}

			adaptor := &shared.ErrorResponseAdaptor{
				ExpectedStatusCode: 200,
				HTTPResponse: &http.Response{
					StatusCode: 503,
				},
				JSON503: json503Provider,
			}

			var diagnostics diag.Diagnostics
			hasError := adaptor.HandleError(&diagnostics)

			if !hasError {
				t.Errorf("HandleError() = %v, expected true", hasError)
			}
			if !diagnostics.HasError() {
				t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
			}
			if len(diagnostics.Errors()) != 1 {
				t.Errorf("len(diagnostics.Errors()) = %v, expected 1", len(diagnostics.Errors()))
			}

			errorDiag := diagnostics.Errors()[0]
			if errorDiag.Summary() != tt.expectedMsg {
				t.Errorf("errorDiag.Summary() = %v, expected %v", errorDiag.Summary(), tt.expectedMsg)
			}
		})
	}
}

func TestErrorResponseAdaptor_HandleError_DefaultCase(t *testing.T) {
	body := []byte("Custom error response body")
	adaptor := &shared.ErrorResponseAdaptor{
		ExpectedStatusCode: 200,
		Body:               body,
		HTTPResponse: &http.Response{
			StatusCode: 500,
		},
	}

	var diagnostics diag.Diagnostics
	hasError := adaptor.HandleError(&diagnostics)

	if !hasError {
		t.Errorf("HandleError() = %v, expected true", hasError)
	}
	if !diagnostics.HasError() {
		t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
	}
	if len(diagnostics.Errors()) != 1 {
		t.Errorf("len(diagnostics.Errors()) = %v, expected 1", len(diagnostics.Errors()))
	}

	errorDiag := diagnostics.Errors()[0]
	if errorDiag.Summary() != string(body) {
		t.Errorf("errorDiag.Summary() = %v, expected %v", errorDiag.Summary(), string(body))
	}
}

func TestErrorResponseAdaptor_HandleError_MultipleStatusCodes(t *testing.T) {
	testCases := []struct {
		name            string
		statusCode      int
		expectedSummary string
	}{
		{"Bad Request", 400, "Bad Request"},
		{"Unauthorized", 401, "Authentication Failed"},
		{"Forbidden", 403, "Forbidden"},
		{"Not Found", 404, "Not Found"},
		{"Internal Server Error", 500, "Custom error body"},
		{"Service Unavailable", 503, "Service Unavailable"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adaptor := &shared.ErrorResponseAdaptor{
				ExpectedStatusCode: 200,
				Body:               []byte("Custom error body"),
				HTTPResponse: &http.Response{
					StatusCode: tc.statusCode,
				},
			}

			var diagnostics diag.Diagnostics
			hasError := adaptor.HandleError(&diagnostics)

			if !hasError {
				t.Errorf("HandleError() = %v, expected true", hasError)
			}
			if !diagnostics.HasError() {
				t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
			}
			if len(diagnostics.Errors()) != 1 {
				t.Errorf("len(diagnostics.Errors()) = %v, expected 1", len(diagnostics.Errors()))
			}

			errorDiag := diagnostics.Errors()[0]
			if errorDiag.Summary() != tc.expectedSummary {
				t.Errorf("errorDiag.Summary() = %v, expected %v", errorDiag.Summary(), tc.expectedSummary)
			}
		})
	}
}

// Test helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Integration test that simulates real API error scenarios
func TestErrorResponseAdaptor_IntegrationScenarios(t *testing.T) {
	t.Run("API token expired during service creation", func(t *testing.T) {
		adaptor := shared.NewMissionControlErrorResponseAdaptor(
			202, // Expected for service creation
			nil,
			&http.Response{
				StatusCode: 401,
				Status:     "401 Unauthorized",
			},
			nil,
			nil,
			nil,
			nil,
			nil,
		)

		var diagnostics diag.Diagnostics
		hasError := adaptor.HandleError(&diagnostics)

		if !hasError {
			t.Errorf("HandleError() = %v, expected true", hasError)
		}
		if !diagnostics.HasError() {
			t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
		}

		errorDiag := diagnostics.Errors()[0]
		if errorDiag.Summary() != "Authentication Failed" {
			t.Errorf("errorDiag.Summary() = %v, expected 'Authentication Failed'", errorDiag.Summary())
		}
		if !strings.Contains(errorDiag.Detail(), "provider configuration") {
			t.Errorf("errorDiag.Detail() should contain 'provider configuration', got: %v", errorDiag.Detail())
		}
		if !strings.Contains(errorDiag.Detail(), "SOLACECLOUD_API_TOKEN") {
			t.Errorf("errorDiag.Detail() should contain 'SOLACECLOUD_API_TOKEN', got: %v", errorDiag.Detail())
		}
	})

	t.Run("Service creation with invalid parameters", func(t *testing.T) {
		adaptor := shared.NewMissionControlErrorResponseAdaptor(
			202,
			nil,
			&http.Response{
				StatusCode: 400,
			},
			&missioncontrol.ErrorResponse{
				Message: stringPtr("Invalid service class specified"),
			},
			nil,
			nil,
			nil,
			nil,
		)

		var diagnostics diag.Diagnostics
		hasError := adaptor.HandleError(&diagnostics)

		if !hasError {
			t.Errorf("HandleError() = %v, expected true", hasError)
		}
		if !diagnostics.HasError() {
			t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
		}

		errorDiag := diagnostics.Errors()[0]
		if errorDiag.Summary() != "Bad Request" {
			t.Errorf("errorDiag.Summary() = %v, expected 'Bad Request'", errorDiag.Summary())
		}
		if errorDiag.Detail() != "Invalid service class specified" {
			t.Errorf("errorDiag.Detail() = %v, expected 'Invalid service class specified'", errorDiag.Detail())
		}
	})

	t.Run("Service not found during read operation", func(t *testing.T) {
		adaptor := shared.NewMissionControlErrorResponseAdaptor(
			200,
			nil,
			&http.Response{
				StatusCode: 404,
			},
			nil,
			nil,
			nil,
			&missioncontrol.ErrorResponse{
				Message: stringPtr("Could not find event broker service with id abc123"),
			},
			nil,
		)

		var diagnostics diag.Diagnostics
		hasError := adaptor.HandleError(&diagnostics)

		if !hasError {
			t.Errorf("HandleError() = %v, expected true", hasError)
		}
		if !diagnostics.HasError() {
			t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
		}

		errorDiag := diagnostics.Errors()[0]
		if errorDiag.Summary() != "Not Found" {
			t.Errorf("errorDiag.Summary() = %v, expected 'Not Found'", errorDiag.Summary())
		}
		if errorDiag.Detail() != "Could not find event broker service with id abc123" {
			t.Errorf("errorDiag.Detail() = %v, expected 'Could not find event broker service with id abc123'", errorDiag.Detail())
		}
	})

	t.Run("Service deletion forbidden due to lock", func(t *testing.T) {
		adaptor := shared.NewMissionControlErrorResponseAdaptor(
			202,
			nil,
			&http.Response{
				StatusCode: 403,
			},
			nil,
			nil,
			&missioncontrol.ErrorResponse{
				Message: stringPtr("This service couldn't be deleted because its locked"),
			},
			nil,
			nil,
		)

		var diagnostics diag.Diagnostics
		hasError := adaptor.HandleError(&diagnostics)

		if !hasError {
			t.Errorf("HandleError() = %v, expected true", hasError)
		}
		if !diagnostics.HasError() {
			t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
		}

		errorDiag := diagnostics.Errors()[0]
		if errorDiag.Detail() != "This service couldn't be deleted because its locked" {
			t.Errorf("errorDiag.Detail() = %v, expected 'This service couldn't be deleted because its locked'", errorDiag.Detail())
		}
		if errorDiag.Summary() != "Forbidden" {
			t.Errorf("errorDiag.Summary() = %v, expected 'Forbidden'", errorDiag.Summary())
		}
	})

	t.Run("Service unavailable during peak hours", func(t *testing.T) {
		adaptor := shared.NewMissionControlErrorResponseAdaptor(
			200,
			nil,
			&http.Response{
				StatusCode: 503,
			},
			nil,
			nil,
			nil,
			nil,
			&missioncontrol.ErrorResponse{
				Message: stringPtr("Service temporarily unavailable due to maintenance"),
			},
		)

		var diagnostics diag.Diagnostics
		hasError := adaptor.HandleError(&diagnostics)

		if !hasError {
			t.Errorf("HandleError() = %v, expected true", hasError)
		}
		if !diagnostics.HasError() {
			t.Errorf("diagnostics.HasError() = %v, expected true", diagnostics.HasError())
		}

		errorDiag := diagnostics.Errors()[0]
		if errorDiag.Detail() != "Service temporarily unavailable due to maintenance" {
			t.Errorf("errorDiag.Detail() = %v, expected 'Service temporarily unavailable due to maintenance'", errorDiag.Detail())
		}
		if errorDiag.Summary() != "Service Unavailable" {
			t.Errorf("errorDiag.Summary() = %v, expected 'Service Unavailable'", errorDiag.Summary())
		}
	})
}

// Test that verifies the error handling code works correctly (bugs have been fixed)
func TestErrorResponseAdaptor_CorrectBehavior(t *testing.T) {
	t.Run("404 handler correctly uses JSON404 message", func(t *testing.T) {
		adaptor := shared.NewMissionControlErrorResponseAdaptor(
			200,
			nil,
			&http.Response{
				StatusCode: 404,
			},
			nil,
			nil,
			&missioncontrol.ErrorResponse{
				Message: stringPtr("This is from JSON403"),
			},
			&missioncontrol.ErrorResponse{
				Message: stringPtr("This should be used and is"),
			},
			nil,
		)

		var diagnostics diag.Diagnostics
		adaptor.HandleError(&diagnostics)

		// Now correctly uses JSON404.Message
		errorDiag := diagnostics.Errors()[0]
		if errorDiag.Summary() != "Not Found" {
			t.Errorf("errorDiag.Summary() = %v, expected 'Not Found'", errorDiag.Summary())
		}
		if errorDiag.Detail() != "This should be used and is" {
			t.Errorf("errorDiag.Summary() = %v, expected 'This should be used and is'", errorDiag.Summary())
		}
	})

	t.Run("503 handler correctly uses JSON503 message", func(t *testing.T) {
		adaptor := shared.NewMissionControlErrorResponseAdaptor(
			200,
			nil,
			&http.Response{
				StatusCode: 503,
			},
			nil,
			nil,
			&missioncontrol.ErrorResponse{
				Message: stringPtr("This is from JSON403"),
			},
			nil,
			&missioncontrol.ErrorResponse{
				Message: stringPtr("This should be used and is"),
			},
		)

		var diagnostics diag.Diagnostics
		adaptor.HandleError(&diagnostics)

		// Now correctly uses JSON503.Message
		errorDiag := diagnostics.Errors()[0]
		if errorDiag.Summary() != "Service Unavailable" {
			t.Errorf("errorDiag.Summary() = %v, expected 'Service Unavailable'", errorDiag.Summary())
		}
		if errorDiag.Detail() != "This should be used and is" {
			t.Errorf("errorDiag.Summary() = %v, expected 'This should be used and is'", errorDiag.Summary())
		}
	})
}
