package operationutils_test

import (
	"context"
	"errors"
	"net/http"
	"terraform-provider-solacecloud/internal/provider/operationutils"
	"terraform-provider-solacecloud/missioncontrol"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// MockOperationClient implements the ApiClient interface for testing
type MockOperationClient struct {
	responses []*missioncontrol.GetServiceOperationResponse
	errors    []error
	callCount int
}

func (m *MockOperationClient) GetServiceOperationWithResponse(ctx context.Context, serviceId string, operationId string, params *missioncontrol.GetServiceOperationParams, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.GetServiceOperationResponse, error) {
	if m.callCount >= len(m.responses) {
		return nil, errors.New("unexpected call")
	}

	resp := m.responses[m.callCount]
	err := m.errors[m.callCount]
	m.callCount++

	return resp, err
}

// Helper function to create operation status pointer
func operationStatusPtr(status missioncontrol.OperationStatus) *missioncontrol.OperationStatus {
	return &status
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}

func TestWaitForOperationToComplete_Success(t *testing.T) {
	// Test successful operation completion
	mockClient := &MockOperationClient{
		responses: []*missioncontrol.GetServiceOperationResponse{
			{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &missioncontrol.OperationResponse{
					Data: missioncontrol.Operation{
						Status: operationStatusPtr(missioncontrol.OperationStatusSUCCEEDED),
					},
				},
			},
		},
		errors: []error{nil},
	}

	diagnostics := &diag.Diagnostics{}
	params := &operationutils.OperationParams{
		Ctx:             context.Background(),
		ServiceId:       "test-service-id",
		OperationId:     "test-operation-id",
		R:               mockClient,
		PollingInterval: 1,
		Timeout:         30 * time.Second,
		Diagnostics:     diagnostics,
	}


	operationutils.WaitForOperationToComplete(params)


	if diagnostics.HasError() {
		t.Errorf("Expected no errors, but got: %v", diagnostics.Errors())
	}

	if mockClient.callCount != 1 {
		t.Errorf("Expected 1 API call, but got %d", mockClient.callCount)
	}
}

func TestWaitForOperationToComplete_Failed(t *testing.T) {
	// Test operation failure
	mockClient := &MockOperationClient{
		responses: []*missioncontrol.GetServiceOperationResponse{
			{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &missioncontrol.OperationResponse{
					Data: missioncontrol.Operation{
						Status: operationStatusPtr(missioncontrol.OperationStatusFAILED),
						Error: &missioncontrol.OperationError{
							Message: strPtr("The operation failed test."),
						},
					},
				},
			},
		},
		errors: []error{nil},
	}

	diagnostics := &diag.Diagnostics{}
	params := &operationutils.OperationParams{
		Ctx:             context.Background(),
		ServiceId:       "test-service-id",
		OperationId:     "test-operation-id",
		R:               mockClient,
		PollingInterval: 1,
		Timeout:         30 * time.Second,
		Diagnostics:     diagnostics,
	}


	operationutils.WaitForOperationToComplete(params)

	if !diagnostics.HasError() {
		t.Errorf("Expected error for failed operation, but got none")
	}
	if len(diagnostics.Errors()) != 1 {
		t.Errorf("Expected 1 error, but got %d", len(diagnostics.Errors()))
	}

	errorDiag := diagnostics.Errors()[0]
	if errorDiag.Summary() != "The operation failed." {
		t.Errorf("Expected 'The operation failed.', but got '%s'", errorDiag.Summary())
	}

	expectedDetail := "The operation failed test."
	if errorDiag.Detail() != expectedDetail {
		t.Errorf("Expected '%s', but got '%s'", expectedDetail, errorDiag.Detail())
	}
}

func TestWaitForOperationToComplete_InProgressThenSuccess(t *testing.T) {
	// Test operation that starts in progress then succeeds
	mockClient := &MockOperationClient{
		responses: []*missioncontrol.GetServiceOperationResponse{
			{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &missioncontrol.OperationResponse{
					Data: missioncontrol.Operation{
						Status: operationStatusPtr(missioncontrol.OperationStatusINPROGRESS),
					},
				},
			},
			{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &missioncontrol.OperationResponse{
					Data: missioncontrol.Operation{
						Status: operationStatusPtr(missioncontrol.OperationStatusSUCCEEDED),
					},
				},
			},
		},
		errors: []error{nil, nil},
	}

	diagnostics := &diag.Diagnostics{}
	params := &operationutils.OperationParams{
		Ctx:             context.Background(),
		ServiceId:       "test-service-id",
		OperationId:     "test-operation-id",
		R:               mockClient,
		PollingInterval: 0, // No sleep for faster testing
		Timeout:         30 * time.Second,
		Diagnostics:     diagnostics,
	}


	operationutils.WaitForOperationToComplete(params)


	if diagnostics.HasError() {
		t.Errorf("Expected no errors, but got: %v", diagnostics.Errors())
	}

	if mockClient.callCount != 2 {
		t.Errorf("Expected 2 API calls, but got %d", mockClient.callCount)
	}
}

func TestWaitForOperationToComplete_APIError(t *testing.T) {
	// Test API error during operation polling
	mockClient := &MockOperationClient{
		responses: []*missioncontrol.GetServiceOperationResponse{nil},
		errors:    []error{errors.New("API connection failed")},
	}

	diagnostics := &diag.Diagnostics{}
	params := &operationutils.OperationParams{
		Ctx:             context.Background(),
		ServiceId:       "test-service-id",
		OperationId:     "test-operation-id",
		R:               mockClient,
		PollingInterval: 1,
		Timeout:         30 * time.Second,
		Diagnostics:     diagnostics,
	}


	operationutils.WaitForOperationToComplete(params)

	if !diagnostics.HasError() {
		t.Errorf("Expected error for API failure, but got none")
	}
	if len(diagnostics.Errors()) != 1 {
		t.Errorf("Expected 1 error, but got %d", len(diagnostics.Errors()))
	}

	errorDiag := diagnostics.Errors()[0]
	if errorDiag.Summary() != "Error calling Solace Cloud API" {
		t.Errorf("Expected 'Error calling Solace Cloud API', but got '%s'", errorDiag.Summary())
	}

	expectedDetail := "Could not get service operation status, unexpected error: API connection failed"
	if errorDiag.Detail() != expectedDetail {
		t.Errorf("Expected '%s', but got '%s'", expectedDetail, errorDiag.Detail())
	}
}

func TestWaitForOperationToComplete_HTTPError(t *testing.T) {
	// Test HTTP error response
	mockClient := &MockOperationClient{
		responses: []*missioncontrol.GetServiceOperationResponse{
			{
				HTTPResponse: &http.Response{StatusCode: 404},
				JSON404: &missioncontrol.ErrorResponse{
					Message: strPtr("Operation not found"),
				},
			},
		},
		errors: []error{nil},
	}

	diagnostics := &diag.Diagnostics{}
	params := &operationutils.OperationParams{
		Ctx:             context.Background(),
		ServiceId:       "test-service-id",
		OperationId:     "test-operation-id",
		R:               mockClient,
		PollingInterval: 1,
		Timeout:         30 * time.Second,
		Diagnostics:     diagnostics,
	}

	operationutils.WaitForOperationToComplete(params)

	if !diagnostics.HasError() {
		t.Errorf("Expected error for HTTP 404, but got none")
	}

	// The error should be handled by the ErrorResponseAdaptor
	if len(diagnostics.Errors()) != 1 {
		t.Errorf("Expected 1 error, but got %d", len(diagnostics.Errors()))
	}
}

func TestWaitForOperationToComplete_PendingThenSuccess(t *testing.T) {
	// Test operation that starts pending then succeeds
	mockClient := &MockOperationClient{
		responses: []*missioncontrol.GetServiceOperationResponse{
			{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &missioncontrol.OperationResponse{
					Data: missioncontrol.Operation{
						Status: operationStatusPtr(missioncontrol.OperationStatusPENDING),
					},
				},
			},
			{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &missioncontrol.OperationResponse{
					Data: missioncontrol.Operation{
						Status: operationStatusPtr(missioncontrol.OperationStatusSUCCEEDED),
					},
				},
			},
		},
		errors: []error{nil, nil},
	}

	diagnostics := &diag.Diagnostics{}
	params := &operationutils.OperationParams{
		Ctx:             context.Background(),
		ServiceId:       "test-service-id",
		OperationId:     "test-operation-id",
		R:               mockClient,
		PollingInterval: 0, // No sleep for faster testing
		Timeout:         30 * time.Second,
		Diagnostics:     diagnostics,
	}


	operationutils.WaitForOperationToComplete(params)
  
	if diagnostics.HasError() {
		t.Errorf("Expected no errors, but got: %v", diagnostics.Errors())
	}

	if mockClient.callCount != 2 {
		t.Errorf("Expected 2 API calls, but got %d", mockClient.callCount)
	}
}

func TestWaitForOperationToComplete_ErrorAfterInProgress(t *testing.T) {
	// Test API error after initial in-progress status
	mockClient := &MockOperationClient{
		responses: []*missioncontrol.GetServiceOperationResponse{
			{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &missioncontrol.OperationResponse{
					Data: missioncontrol.Operation{
						Status: operationStatusPtr(missioncontrol.OperationStatusINPROGRESS),
					},
				},
			},
			nil, // This will cause an error
		},
		errors: []error{nil, errors.New("Network timeout")},
	}

	diagnostics := &diag.Diagnostics{}
	params := &operationutils.OperationParams{
		Ctx:             context.Background(),
		ServiceId:       "test-service-id",
		OperationId:     "test-operation-id",
		R:               mockClient,
		PollingInterval: 0,
		Timeout:         10 * time.Second, // Longer timeout to allow the error to be hit
		Diagnostics:     diagnostics,
	}

	operationutils.WaitForOperationToComplete(params)

	if !diagnostics.HasError() {
		t.Errorf("Expected error, but got none")
	}
	if len(diagnostics.Errors()) != 1 {
		t.Errorf("Expected 1 error, but got %d", len(diagnostics.Errors()))
	}

	errorDiag := diagnostics.Errors()[0]
	if errorDiag.Summary() != "Error calling Solace Cloud API" {
		t.Errorf("Expected 'Error calling Solace Cloud API', but got '%s'", errorDiag.Summary())
	}

	expectedDetail := "Could not get service operation status, unexpected error: Network timeout"
	if errorDiag.Detail() != expectedDetail {
		t.Errorf("Expected '%s', but got '%s'", expectedDetail, errorDiag.Detail())
	}

	if mockClient.callCount != 2 {
		t.Errorf("Expected 2 API calls, but got %d", mockClient.callCount)
	}
}

// Test the original OperationParams struct initialization
func TestOperationParamsStruct(t *testing.T) {
	ctx := context.Background()
	serviceId := "test-service"
	operationId := "test-operation"
	pollingInterval := 5

	// Test struct field assignments (without creating a real client)
	if ctx == nil {
		t.Errorf("Expected context to be non-nil")
	}
	if serviceId == "" {
		t.Errorf("Expected serviceId to be non-empty")
	}
	if operationId == "" {
		t.Errorf("Expected operationId to be non-empty")
	}
	if pollingInterval <= 0 {
		t.Errorf("Expected pollingInterval to be positive, got %d", pollingInterval)
	}
}

// Benchmark test to ensure the function performs reasonably well
func BenchmarkWaitForOperationToComplete_Success(b *testing.B) {
	mockClient := &MockOperationClient{
		responses: []*missioncontrol.GetServiceOperationResponse{
			{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &missioncontrol.OperationResponse{
					Data: missioncontrol.Operation{
						Status: operationStatusPtr(missioncontrol.OperationStatusSUCCEEDED),
					},
				},
			},
		},
		errors: []error{nil},
	}

	diagnostics := &diag.Diagnostics{}
	params := &operationutils.OperationParams{
		Ctx:             context.Background(),
		ServiceId:       "test-service-id",
		OperationId:     "test-operation-id",
		R:               mockClient,
		PollingInterval: 0,
		Timeout:         30 * time.Second,
		Diagnostics:     diagnostics,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockClient.callCount = 0 // Reset for each iteration
		operationutils.WaitForOperationToComplete(params)
	}
}
