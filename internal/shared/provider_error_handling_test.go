package shared

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// MockErrorResponseProvider is a mock implementation of ErrorResponseProvider for testing
type MockErrorResponseProvider struct {
	message           string
	errorId           string
	validationDetails string
}

func (m *MockErrorResponseProvider) GetMessage() string {
	return m.message
}

func (m *MockErrorResponseProvider) GetErrorId() string {
	return m.errorId
}

func (m *MockErrorResponseProvider) GetFirstValidationDetail() string {
	return m.validationDetails
}

func TestErrorResponseAdaptor_addBadRequestDiagnostics(t *testing.T) {
	tests := []struct {
		name                    string
		json400                 ErrorResponseProvider
		expectedErrorCount      int
		expectedErrorSummaries  []string
		expectedErrorDetails    []string
	}{
		{
			name: "with validation details only",
			json400: &MockErrorResponseProvider{
				validationDetails: "field1: invalid value",
			},
			expectedErrorCount:     1,
			expectedErrorSummaries: []string{"Bad Request"},
			expectedErrorDetails:   []string{"field1: invalid value"},
		},
		{
			name: "with message only",
			json400: &MockErrorResponseProvider{
				message: "The request body is malformed",
			},
			expectedErrorCount:     1,
			expectedErrorSummaries: []string{"Bad Request"},
			expectedErrorDetails:   []string{"The request body is malformed"},
		},
		{
			name: "with both validation details and message",
			json400: &MockErrorResponseProvider{
				message:           "The request body is malformed",
				validationDetails: "field1: invalid value",
			},
			expectedErrorCount:     2,
			expectedErrorSummaries: []string{"Bad Request", "Bad Request"},
			expectedErrorDetails: []string{
				"field1: invalid value",
				"The request body is malformed",
			},
		},
		{
			name:                   "with no message or validation details",
			json400:                &MockErrorResponseProvider{},
			expectedErrorCount:     1,
			expectedErrorSummaries: []string{"Bad Request"},
			expectedErrorDetails: []string{
				"Received HTTP 400 Bad Request. This usually indicates a malformed request or missing required parameters. Check your request body and parameters.",
			},
		},
		{
			name:                   "with nil JSON400",
			json400:                nil,
			expectedErrorCount:     1,
			expectedErrorSummaries: []string{"Bad Request"},
			expectedErrorDetails: []string{
				"Received HTTP 400 Bad Request. This usually indicates a malformed request or missing required parameters. Check your request body and parameters.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create error handler
			handler := &ErrorResponseAdaptor{
				ExpectedStatusCode: http.StatusOK,
				Body:               []byte("test body"),
				HTTPResponse: &http.Response{
					StatusCode: http.StatusBadRequest,
				},
				JSON400: tt.json400,
			}

			// Create diagnostics
			var diagnostics diag.Diagnostics

			// Call the method under test
			handler.addBadRequestDiagnostics(&diagnostics)

			// Verify error count
			if len(diagnostics) != tt.expectedErrorCount {
				t.Errorf("expected %d errors, got %d", tt.expectedErrorCount, len(diagnostics))
			}

			// Verify each error
			for i, expectedSummary := range tt.expectedErrorSummaries {
				if i >= len(diagnostics) {
					t.Errorf("expected error at index %d but diagnostics only has %d errors", i, len(diagnostics))
					continue
				}

				actualSummary := diagnostics[i].Summary()
				if actualSummary != expectedSummary {
					t.Errorf("error %d: expected summary %q, got %q", i, expectedSummary, actualSummary)
				}

				actualDetail := diagnostics[i].Detail()
				if actualDetail != tt.expectedErrorDetails[i] {
					t.Errorf("error %d: expected detail %q, got %q", i, tt.expectedErrorDetails[i], actualDetail)
				}
			}
		})
	}
}

func TestErrorResponseAdaptor_HandleError(t *testing.T) {
	tests := []struct {
		name               string
		statusCode         int
		expectedStatusCode int
		expectError        bool
	}{
		{
			name:               "no error when status matches expected",
			statusCode:         http.StatusOK,
			expectedStatusCode: http.StatusOK,
			expectError:        false,
		},
		{
			name:               "handles 401 unauthorized",
			statusCode:         http.StatusUnauthorized,
			expectedStatusCode: http.StatusOK,
			expectError:        true,
		},
		{
			name:               "handles 400 bad request",
			statusCode:         http.StatusBadRequest,
			expectedStatusCode: http.StatusOK,
			expectError:        true,
		},
		{
			name:               "handles 403 forbidden",
			statusCode:         http.StatusForbidden,
			expectedStatusCode: http.StatusOK,
			expectError:        true,
		},
		{
			name:               "handles 404 not found",
			statusCode:         http.StatusNotFound,
			expectedStatusCode: http.StatusOK,
			expectError:        true,
		},
		{
			name:               "handles 503 service unavailable",
			statusCode:         http.StatusServiceUnavailable,
			expectedStatusCode: http.StatusOK,
			expectError:        true,
		},
		{
			name:               "handles 409 conflict",
			statusCode:         http.StatusConflict,
			expectedStatusCode: http.StatusOK,
			expectError:        true,
		},
		{
			name:               "handles unexpected status codes",
			statusCode:         http.StatusInternalServerError,
			expectedStatusCode: http.StatusOK,
			expectError:        true,
		},
		{
			name:               "expected 201 got 201",
			statusCode:         http.StatusCreated,
			expectedStatusCode: http.StatusCreated,
			expectError:        false,
		},
		{
			name:               "expected 204 got 204",
			statusCode:         http.StatusNoContent,
			expectedStatusCode: http.StatusNoContent,
			expectError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create error handler
			handler := &ErrorResponseAdaptor{
				ExpectedStatusCode: tt.expectedStatusCode,
				Body:               []byte("test body"),
				HTTPResponse: &http.Response{
					StatusCode: tt.statusCode,
				},
			}

			// Create diagnostics
			var diagnostics diag.Diagnostics

			// Call the method under test
			hasError := handler.HandleError(&diagnostics)

			// Verify error occurred flag
			if hasError != tt.expectError {
				t.Errorf("expected hasError=%v, got %v", tt.expectError, hasError)
			}

			// Verify that an error was added to diagnostics if expected
			if tt.expectError && len(diagnostics) == 0 {
				t.Error("expected at least one diagnostic error, got none")
			}

			// Verify no errors were added if not expected
			if !tt.expectError && len(diagnostics) > 0 {
				t.Errorf("expected no diagnostic errors, got %d", len(diagnostics))
			}
		})
	}
}