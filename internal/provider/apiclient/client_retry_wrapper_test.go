package apiclient

import (
	"context"
	"errors"
	"io"
	"strings"
	mc "terraform-provider-solacecloud/missioncontrol"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockApiError implements the ApiError interface for testing
type MockApiError struct {
	statusCode int
}

func (m MockApiError) Error() string {
	return "mock API error"
}

func (m MockApiError) StatusCode() int {
	return m.statusCode
}

// MockCRUDClient is a mock implementation of CRUDClientWithResponses
type MockCRUDClient struct {
	mock.Mock
}

func (m *MockCRUDClient) CreateServiceWithResponse(ctx context.Context, body mc.CreateServiceJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.CreateServiceResponse, error) {
	args := m.Called(ctx, body, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.CreateServiceResponse), args.Error(1)
}

func (m *MockCRUDClient) GetServiceWithResponse(ctx context.Context, id string, params *mc.GetServiceParams, reqEditors ...mc.RequestEditorFn) (*mc.GetServiceResponse, error) {
	args := m.Called(ctx, id, params, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.GetServiceResponse), args.Error(1)
}

func (m *MockCRUDClient) DeleteServiceWithResponse(ctx context.Context, id string, reqEditors ...mc.RequestEditorFn) (*mc.DeleteServiceResponse, error) {
	args := m.Called(ctx, id, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.DeleteServiceResponse), args.Error(1)
}

func (m *MockCRUDClient) UpdateServiceWithBodyWithResponse(ctx context.Context, id string, contentType string, body io.Reader, reqEditors ...mc.RequestEditorFn) (*mc.UpdateServiceResponse, error) {
	args := m.Called(ctx, id, contentType, body, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.UpdateServiceResponse), args.Error(1)
}

func (m *MockCRUDClient) UpdateMessageSpoolWithBodyWithResponse(ctx context.Context, serviceId string, contentType string, body io.Reader, reqEditors ...mc.RequestEditorFn) (*mc.UpdateMessageSpoolResponse, error) {
	args := m.Called(ctx, serviceId, contentType, body, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.UpdateMessageSpoolResponse), args.Error(1)
}

func (m *MockCRUDClient) GetServiceOperationWithResponse(ctx context.Context, serviceId string, operationId string, params *mc.GetServiceOperationParams, reqEditors ...mc.RequestEditorFn) (*mc.GetServiceOperationResponse, error) {
	args := m.Called(ctx, serviceId, operationId, params, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.GetServiceOperationResponse), args.Error(1)
}

func (m *MockCRUDClient) CreateConnectionEndpointWithResponse(ctx context.Context, serviceId string, body mc.CreateConnectionEndpointJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.CreateConnectionEndpointResponse, error) {
	args := m.Called(ctx, serviceId, body, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.CreateConnectionEndpointResponse), args.Error(1)
}

func (m *MockCRUDClient) GetConnectionEndpointsWithResponse(ctx context.Context, serviceId string, reqEditors ...mc.RequestEditorFn) (*mc.GetConnectionEndpointsResponse, error) {
	args := m.Called(ctx, serviceId, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.GetConnectionEndpointsResponse), args.Error(1)
}

func (m *MockCRUDClient) GetConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...mc.RequestEditorFn) (*mc.GetConnectionEndpointResponse, error) {
	args := m.Called(ctx, serviceId, connectionEndpointId, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.GetConnectionEndpointResponse), args.Error(1)
}

func (m *MockCRUDClient) UpdateConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, body mc.UpdateConnectionEndpointJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.UpdateConnectionEndpointResponse, error) {
	args := m.Called(ctx, serviceId, connectionEndpointId, body, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.UpdateConnectionEndpointResponse), args.Error(1)
}

func (m *MockCRUDClient) DeleteConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...mc.RequestEditorFn) (*mc.DeleteConnectionEndpointResponse, error) {
	args := m.Called(ctx, serviceId, connectionEndpointId, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.DeleteConnectionEndpointResponse), args.Error(1)
}

// DNS Name operations
func (m *MockCRUDClient) GetConnectionEndpointDnsNamesWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...mc.RequestEditorFn) (*mc.GetConnectionEndpointDnsNamesResponse, error) {
	args := m.Called(ctx, serviceId, connectionEndpointId, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.GetConnectionEndpointDnsNamesResponse), args.Error(1)
}

func (m *MockCRUDClient) CreateConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, body mc.CreateConnectionEndpointDnsNameJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.CreateConnectionEndpointDnsNameResponse, error) {
	args := m.Called(ctx, serviceId, connectionEndpointId, body, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.CreateConnectionEndpointDnsNameResponse), args.Error(1)
}

func (m *MockCRUDClient) DeleteConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, dnsName string, reqEditors ...mc.RequestEditorFn) (*mc.DeleteConnectionEndpointDnsNameResponse, error) {
	args := m.Called(ctx, serviceId, connectionEndpointId, dnsName, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.DeleteConnectionEndpointDnsNameResponse), args.Error(1)
}

func (m *MockCRUDClient) MoveConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, dnsName string, body mc.MoveConnectionEndpointDnsNameJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.MoveConnectionEndpointDnsNameResponse, error) {
	args := m.Called(ctx, serviceId, connectionEndpointId, dnsName, body, reqEditors)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mc.MoveConnectionEndpointDnsNameResponse), args.Error(1)
}

func TestNewRetryableClient(t *testing.T) {
	mockApi := &MockCRUDClient{}
	maxRetries := 3
	waitSeconds := 1

	client := NewRetryableClient(mockApi, maxRetries, waitSeconds)

	assert.NotNil(t, client)
	assert.Equal(t, mockApi, client.api)
	assert.Equal(t, maxRetries, client.maxRetries)
	assert.Equal(t, waitSeconds, client.waitSeconds)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "500 error should be retryable",
			err:      MockApiError{statusCode: 500},
			expected: true,
		},
		{
			name:     "502 error should be retryable",
			err:      MockApiError{statusCode: 502},
			expected: true,
		},
		{
			name:     "503 error should be retryable",
			err:      MockApiError{statusCode: 503},
			expected: true,
		},
		{
			name:     "599 error should be retryable",
			err:      MockApiError{statusCode: 599},
			expected: true,
		},
		{
			name:     "400 error should not be retryable",
			err:      MockApiError{statusCode: 400},
			expected: false,
		},
		{
			name:     "401 error should not be retryable",
			err:      MockApiError{statusCode: 401},
			expected: false,
		},
		{
			name:     "404 error should not be retryable",
			err:      MockApiError{statusCode: 404},
			expected: false,
		},
		{
			name:     "499 error should not be retryable",
			err:      MockApiError{statusCode: 499},
			expected: false,
		},
		{
			name:     "600 error should not be retryable",
			err:      MockApiError{statusCode: 600},
			expected: false,
		},
		{
			name:     "non-ApiError should not be retryable",
			err:      errors.New("generic error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetry_Success(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	result, err := retry(ctx, func() (string, error) {
		callCount++
		return "success", nil
	}, 3, 0)

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, callCount) // Should succeed on first attempt
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	result, err := retry(ctx, func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", MockApiError{statusCode: 500}
		}
		return "success", nil
	}, 3, 0) // waitSec = 0 for faster test

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount) // Should succeed on third attempt
}

func TestRetry_ExhaustRetries(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	expectedErr := MockApiError{statusCode: 500}

	result, err := retry(ctx, func() (string, error) {
		callCount++
		return "", expectedErr
	}, 3, 0)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, "", result)
	assert.Equal(t, 3, callCount) // Should attempt all retries
}

func TestRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	expectedErr := MockApiError{statusCode: 400}

	result, err := retry(ctx, func() (string, error) {
		callCount++
		return "", expectedErr
	}, 3, 0)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, "", result)
	assert.Equal(t, 1, callCount) // Should not retry for 4xx errors
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	// Cancel context after first call
	result, err := retry(ctx, func() (string, error) {
		callCount++
		if callCount == 1 {
			cancel()
		}
		return "", MockApiError{statusCode: 500}
	}, 3, 1) // waitSec = 1 to allow context cancellation to take effect

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, "", result)
	assert.Equal(t, 1, callCount) // Should stop after context cancellation
}

func TestRetryableClient_CreateServiceWithResponse_Success(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	body := mc.CreateServiceJSONRequestBody{}
	expectedResponse := &mc.CreateServiceResponse{}

	mockApi.On("CreateServiceWithResponse", ctx, body, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil)

	result, err := client.CreateServiceWithResponse(ctx, body)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_CreateServiceWithResponse_RetryableError(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 2, 0)

	ctx := context.Background()
	body := mc.CreateServiceJSONRequestBody{}
	expectedResponse := &mc.CreateServiceResponse{}
	retryableErr := MockApiError{statusCode: 500}

	// First call fails, second succeeds
	mockApi.On("CreateServiceWithResponse", ctx, body, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(nil, retryableErr).Once()
	mockApi.On("CreateServiceWithResponse", ctx, body, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil).Once()

	result, err := client.CreateServiceWithResponse(ctx, body)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_GetServiceWithResponse_Success(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	id := "test-id"
	params := &mc.GetServiceParams{}
	expectedResponse := &mc.GetServiceResponse{}

	mockApi.On("GetServiceWithResponse", ctx, id, params, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil)

	result, err := client.GetServiceWithResponse(ctx, id, params)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_DeleteServiceWithResponse_Success(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	id := "test-id"
	expectedResponse := &mc.DeleteServiceResponse{}

	mockApi.On("DeleteServiceWithResponse", ctx, id, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil)

	result, err := client.DeleteServiceWithResponse(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_UpdateServiceWithBodyWithResponse_Success(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	id := "test-id"
	contentType := "application/json"
	body := strings.NewReader("{}")
	expectedResponse := &mc.UpdateServiceResponse{}

	mockApi.On("UpdateServiceWithBodyWithResponse", ctx, id, contentType, body, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil)

	result, err := client.UpdateServiceWithBodyWithResponse(ctx, id, contentType, body)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_UpdateMessageSpoolWithBodyWithResponse_Success(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	serviceId := "test-service-id"
	contentType := "application/json"
	body := strings.NewReader("{}")
	expectedResponse := &mc.UpdateMessageSpoolResponse{}

	mockApi.On("UpdateMessageSpoolWithBodyWithResponse", ctx, serviceId, contentType, body, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil)

	result, err := client.UpdateMessageSpoolWithBodyWithResponse(ctx, serviceId, contentType, body)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_GetServiceOperationWithResponse_Success(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	serviceId := "test-service-id"
	operationId := "test-operation-id"
	expectedResponse := &mc.GetServiceOperationResponse{}

	mockApi.On("GetServiceOperationWithResponse", ctx, serviceId, operationId, (*mc.GetServiceOperationParams)(nil), mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil)

	result, err := client.GetServiceOperationWithResponse(ctx, serviceId, operationId, nil)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

// Connection Endpoint Tests

func TestRetryableClient_CreateConnectionEndpointWithResponse_Success(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	serviceId := "test-service-id"
	body := mc.CreateConnectionEndpointJSONRequestBody{}
	expectedResponse := &mc.CreateConnectionEndpointResponse{}

	mockApi.On("CreateConnectionEndpointWithResponse", ctx, serviceId, body, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil)

	result, err := client.CreateConnectionEndpointWithResponse(ctx, serviceId, body)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_CreateConnectionEndpointWithResponse_RetryableError(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 2, 0)

	ctx := context.Background()
	serviceId := "test-service-id"
	body := mc.CreateConnectionEndpointJSONRequestBody{}
	expectedResponse := &mc.CreateConnectionEndpointResponse{}
	retryableErr := MockApiError{statusCode: 503}

	// First call fails, second succeeds
	mockApi.On("CreateConnectionEndpointWithResponse", ctx, serviceId, body, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(nil, retryableErr).Once()
	mockApi.On("CreateConnectionEndpointWithResponse", ctx, serviceId, body, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil).Once()

	result, err := client.CreateConnectionEndpointWithResponse(ctx, serviceId, body)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_GetConnectionEndpointsWithResponse_Success(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	serviceId := "test-service-id"
	expectedResponse := &mc.GetConnectionEndpointsResponse{}

	mockApi.On("GetConnectionEndpointsWithResponse", ctx, serviceId, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil)

	result, err := client.GetConnectionEndpointsWithResponse(ctx, serviceId)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_GetConnectionEndpointWithResponse_Success(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	serviceId := "test-service-id"
	connectionEndpointId := "test-endpoint-id"
	expectedResponse := &mc.GetConnectionEndpointResponse{}

	mockApi.On("GetConnectionEndpointWithResponse", ctx, serviceId, connectionEndpointId, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil)

	result, err := client.GetConnectionEndpointWithResponse(ctx, serviceId, connectionEndpointId)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_UpdateConnectionEndpointWithResponse_Success(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	serviceId := "test-service-id"
	connectionEndpointId := "test-endpoint-id"
	body := mc.UpdateConnectionEndpointJSONRequestBody{}
	expectedResponse := &mc.UpdateConnectionEndpointResponse{}

	mockApi.On("UpdateConnectionEndpointWithResponse", ctx, serviceId, connectionEndpointId, body, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil)

	result, err := client.UpdateConnectionEndpointWithResponse(ctx, serviceId, connectionEndpointId, body)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_DeleteConnectionEndpointWithResponse_Success(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	serviceId := "test-service-id"
	connectionEndpointId := "test-endpoint-id"
	expectedResponse := &mc.DeleteConnectionEndpointResponse{}

	mockApi.On("DeleteConnectionEndpointWithResponse", ctx, serviceId, connectionEndpointId, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedResponse, nil)

	result, err := client.DeleteConnectionEndpointWithResponse(ctx, serviceId, connectionEndpointId)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockApi.AssertExpectations(t)
}

func TestRetryableClient_NonRetryableError_DoesNotRetry(t *testing.T) {
	mockApi := &MockCRUDClient{}
	client := NewRetryableClient(mockApi, 3, 0)

	ctx := context.Background()
	serviceId := "test-service-id"
	nonRetryableErr := MockApiError{statusCode: 404}

	mockApi.On("GetConnectionEndpointsWithResponse", ctx, serviceId, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(nil, nonRetryableErr).Once()

	result, err := client.GetConnectionEndpointsWithResponse(ctx, serviceId)

	assert.Error(t, err)
	assert.Equal(t, nonRetryableErr, err)
	assert.Nil(t, result)
	mockApi.AssertExpectations(t) // Should only be called once
}

// Test timing behavior (without waiting)
func TestRetry_TimingBehavior(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	start := time.Now()

	_, err := retry(ctx, func() (string, error) {
		callCount++
		return "", MockApiError{statusCode: 500}
	}, 2, 0) // waitSec = 0 for fast test

	duration := time.Since(start)

	assert.Error(t, err)
	assert.Equal(t, 2, callCount)
	// Should complete quickly since waitSec = 0
	assert.Less(t, duration, 100*time.Millisecond)
}

// Integration test combining multiple scenarios
func TestRetryableClient_IntegrationScenarios(t *testing.T) {
	t.Run("Service operation succeeds after connection endpoint fails", func(t *testing.T) {
		mockApi := &MockCRUDClient{}
		client := NewRetryableClient(mockApi, 2, 0)

		ctx := context.Background()
		serviceId := "test-service-id"

		// Connection endpoint fails with 404 (non-retryable)
		nonRetryableErr := MockApiError{statusCode: 404}
		mockApi.On("GetConnectionEndpointsWithResponse", ctx, serviceId, mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(nil, nonRetryableErr).Once()

		// Service operation succeeds
		expectedServiceResponse := &mc.GetServiceResponse{}
		mockApi.On("GetServiceWithResponse", ctx, serviceId, (*mc.GetServiceParams)(nil), mock.AnythingOfType("[]missioncontrol.RequestEditorFn")).Return(expectedServiceResponse, nil).Once()

		// Test connection endpoint failure
		endpointResult, endpointErr := client.GetConnectionEndpointsWithResponse(ctx, serviceId)
		assert.Error(t, endpointErr)
		assert.Equal(t, nonRetryableErr, endpointErr)
		assert.Nil(t, endpointResult)

		// Test service success
		serviceResult, serviceErr := client.GetServiceWithResponse(ctx, serviceId, nil)
		assert.NoError(t, serviceErr)
		assert.Equal(t, expectedServiceResponse, serviceResult)

		mockApi.AssertExpectations(t)
	})
}

// Test DNS name operations
func TestRetryableClient_GetConnectionEndpointDnsNamesWithResponse_Success(t *testing.T) {
	mockClient := &MockCRUDClient{}
	retryableClient := NewRetryableClient(mockClient, 3, 1)
	ctx := context.Background()

	expectedResponse := &mc.GetConnectionEndpointDnsNamesResponse{}
	mockClient.On("GetConnectionEndpointDnsNamesWithResponse", ctx, "service123", "endpoint456", mock.Anything).Return(expectedResponse, nil)

	response, err := retryableClient.GetConnectionEndpointDnsNamesWithResponse(ctx, "service123", "endpoint456")

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
	mockClient.AssertExpectations(t)
}

func TestRetryableClient_CreateConnectionEndpointDnsNameWithResponse_Success(t *testing.T) {
	mockClient := &MockCRUDClient{}
	retryableClient := NewRetryableClient(mockClient, 3, 1)
	ctx := context.Background()

	body := mc.CreateConnectionEndpointDnsNameJSONRequestBody{
		DnsName: "api.example.com",
	}
	expectedResponse := &mc.CreateConnectionEndpointDnsNameResponse{}
	mockClient.On("CreateConnectionEndpointDnsNameWithResponse", ctx, "service123", "endpoint456", body, mock.Anything).Return(expectedResponse, nil)

	response, err := retryableClient.CreateConnectionEndpointDnsNameWithResponse(ctx, "service123", "endpoint456", body)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
	mockClient.AssertExpectations(t)
}

func TestRetryableClient_DeleteConnectionEndpointDnsNameWithResponse_Success(t *testing.T) {
	mockClient := &MockCRUDClient{}
	retryableClient := NewRetryableClient(mockClient, 3, 1)
	ctx := context.Background()

	expectedResponse := &mc.DeleteConnectionEndpointDnsNameResponse{}
	mockClient.On("DeleteConnectionEndpointDnsNameWithResponse", ctx, "service123", "endpoint456", "api.example.com", mock.Anything).Return(expectedResponse, nil)

	response, err := retryableClient.DeleteConnectionEndpointDnsNameWithResponse(ctx, "service123", "endpoint456", "api.example.com")

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
	mockClient.AssertExpectations(t)
}

func TestRetryableClient_MoveConnectionEndpointDnsNameWithResponse_Success(t *testing.T) {
	mockClient := &MockCRUDClient{}
	retryableClient := NewRetryableClient(mockClient, 3, 1)
	ctx := context.Background()

	body := mc.MoveConnectionEndpointDnsNameJSONRequestBody{
		TargetServiceId:            StringPtr("target-service"),
		TargetConnectionEndpointId: StringPtr("target-endpoint"),
	}
	expectedResponse := &mc.MoveConnectionEndpointDnsNameResponse{}
	mockClient.On("MoveConnectionEndpointDnsNameWithResponse", ctx, "service123", "endpoint456", "api.example.com", body, mock.Anything).Return(expectedResponse, nil)

	response, err := retryableClient.MoveConnectionEndpointDnsNameWithResponse(ctx, "service123", "endpoint456", "api.example.com", body)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
	mockClient.AssertExpectations(t)
}

func TestRetryableClient_GetConnectionEndpointDnsNamesWithResponse_RetryableError(t *testing.T) {
	mockClient := &MockCRUDClient{}
	retryableClient := NewRetryableClient(mockClient, 2, 1)
	ctx := context.Background()

	// First call fails with retryable error
	retryableErr := MockApiError{statusCode: 503}
	mockClient.On("GetConnectionEndpointDnsNamesWithResponse", ctx, "service123", "endpoint456", mock.Anything).Return(nil, retryableErr).Once()

	// Second call succeeds
	expectedResponse := &mc.GetConnectionEndpointDnsNamesResponse{}
	mockClient.On("GetConnectionEndpointDnsNamesWithResponse", ctx, "service123", "endpoint456", mock.Anything).Return(expectedResponse, nil).Once()

	response, err := retryableClient.GetConnectionEndpointDnsNamesWithResponse(ctx, "service123", "endpoint456")

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
	mockClient.AssertExpectations(t)
}

func TestRetryableClient_CreateConnectionEndpointDnsNameWithResponse_RetryableError(t *testing.T) {
	mockClient := &MockCRUDClient{}
	retryableClient := NewRetryableClient(mockClient, 2, 1)
	ctx := context.Background()

	body := mc.CreateConnectionEndpointDnsNameJSONRequestBody{
		DnsName: "api.example.com",
	}

	// First call fails with retryable error
	retryableErr := MockApiError{statusCode: 502}
	mockClient.On("CreateConnectionEndpointDnsNameWithResponse", ctx, "service123", "endpoint456", body, mock.Anything).Return(nil, retryableErr).Once()

	// Second call succeeds
	expectedResponse := &mc.CreateConnectionEndpointDnsNameResponse{}
	mockClient.On("CreateConnectionEndpointDnsNameWithResponse", ctx, "service123", "endpoint456", body, mock.Anything).Return(expectedResponse, nil).Once()

	response, err := retryableClient.CreateConnectionEndpointDnsNameWithResponse(ctx, "service123", "endpoint456", body)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
	mockClient.AssertExpectations(t)
}

// Helper function for string pointers
func StringPtr(s string) *string {
	return &s
}