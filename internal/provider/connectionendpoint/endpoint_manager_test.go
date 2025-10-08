package connectionendpoint_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/internal/provider/connectionendpoint"
	"terraform-provider-solacecloud/missioncontrol"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// MockAPIClient implements the CRUDClientWithResponses interface
type MockAPIClient struct {
	updateResponse *missioncontrol.UpdateConnectionEndpointResponse
	updateError    error
	opResponse     *missioncontrol.GetServiceOperationResponse
	opError        error
	opCallCount    int
}

// Implement all required methods from CRUDClientWithResponses interface
func (m *MockAPIClient) CreateServiceWithResponse(ctx context.Context, body missioncontrol.CreateServiceJSONRequestBody, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.CreateServiceResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) GetServiceWithResponse(ctx context.Context, id string, params *missioncontrol.GetServiceParams, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.GetServiceResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) DeleteServiceWithResponse(ctx context.Context, id string, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.DeleteServiceResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) UpdateServiceWithBodyWithResponse(ctx context.Context, id string, contentType string, body io.Reader, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.UpdateServiceResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) UpdateMessageSpoolWithBodyWithResponse(ctx context.Context, serviceId string, contentType string, body io.Reader, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.UpdateMessageSpoolResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) GetServiceOperationWithResponse(ctx context.Context, serviceId string, operationId string, params *missioncontrol.GetServiceOperationParams, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.GetServiceOperationResponse, error) {
	m.opCallCount++
	return m.opResponse, m.opError
}

func (m *MockAPIClient) CreateConnectionEndpointWithResponse(ctx context.Context, serviceId string, body missioncontrol.CreateConnectionEndpointJSONRequestBody, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.CreateConnectionEndpointResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) GetConnectionEndpointsWithResponse(ctx context.Context, serviceId string, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.GetConnectionEndpointsResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) GetConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.GetConnectionEndpointResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) UpdateConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, body missioncontrol.UpdateConnectionEndpointJSONRequestBody, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.UpdateConnectionEndpointResponse, error) {
	return m.updateResponse, m.updateError
}

func (m *MockAPIClient) DeleteConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.DeleteConnectionEndpointResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) GetConnectionEndpointDnsNamesWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.GetConnectionEndpointDnsNamesResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) CreateConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, body missioncontrol.CreateConnectionEndpointDnsNameJSONRequestBody, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.CreateConnectionEndpointDnsNameResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) DeleteConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, dnsName string, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.DeleteConnectionEndpointDnsNameResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAPIClient) MoveConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, dnsName string, body missioncontrol.MoveConnectionEndpointDnsNameJSONRequestBody, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.MoveConnectionEndpointDnsNameResponse, error) {
	return nil, errors.New("not implemented")
}

// Helper functions
func operationStatusPtr(status missioncontrol.OperationStatus) *missioncontrol.OperationStatus {
	return &status
}

func TestUpdateEndpoint_Success(t *testing.T) {
	// Setup mock client
	operationId := "test-operation-id"
	mockClient := &MockAPIClient{
		updateResponse: &missioncontrol.UpdateConnectionEndpointResponse{
			HTTPResponse: &http.Response{StatusCode: 202},
			Body:         []byte(`{"data": {"id": "test-operation-id"}}`),
			JSON202: &missioncontrol.OperationResponse{
				Data: missioncontrol.Operation{
					Id: &operationId,
				},
			},
		},
		updateError: nil,
		opResponse: &missioncontrol.GetServiceOperationResponse{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &missioncontrol.OperationResponse{
				Data: missioncontrol.Operation{
					Status: operationStatusPtr(missioncontrol.OperationStatusSUCCEEDED),
				},
			},
		},
		opError: nil,
	}

	// Create endpoint manager with wrapped mock client
	wrappedClient := apiclient.NewRetryableClient(mockClient, 1, 0)
	manager := connectionendpoint.NewEndpointManager(wrappedClient, 0)

	// Setup endpoint config
	port := int32(8080)
	description := "Test description"
	config := connectionendpoint.EndpointConfig{
		Name:        "Test Endpoint",
		Description: &description,
		AccessType:  missioncontrol.ConnectionEndpointAccessTypePUBLIC,
		Ports: []missioncontrol.ServiceConnectionEndpointPort{
			{
				Protocol: missioncontrol.ServiceSmfPlainTextListenPort,
				Port:     &port,
			},
		},
	}

	// Execute update
	diagnostics := &diag.Diagnostics{}
	manager.UpdateEndpoint(
		context.Background(),
		"test-service-id",
		"test-endpoint-id",
		config,
		diagnostics,
	)

	// Verify no errors
	if diagnostics.HasError() {
		t.Errorf("Expected no errors, but got: %v", diagnostics.Errors())
	}

	// Verify operation polling happened
	if mockClient.opCallCount < 1 {
		t.Errorf("Expected at least 1 operation status check, but got %d", mockClient.opCallCount)
	}
}
