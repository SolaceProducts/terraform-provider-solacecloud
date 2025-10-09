package connectionendpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"terraform-provider-solacecloud/internal/model"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/internal/provider/operationutils"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/missioncontrol"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Error message constants
const (
	errCallingAPITitle               = "Error calling Solace Cloud API"
	errReadingConnectionEndpoints    = "Error Reading Connection Endpoints"
	errReadingConnectionEndpointsFmt = "Could not read connection endpoints for service %s: %s"
)

// EndpointManager provides shared connection endpoint management functionality
type EndpointManager struct {
	APIClient          *apiclient.RetryableClientWithResponses
	APIPollingInterval int
}

// EndpointConfig represents the configuration for a connection endpoint
type EndpointConfig struct {
	Name        string
	Description *string
	AccessType  missioncontrol.ConnectionEndpointAccessType
	Ports       []missioncontrol.ServiceConnectionEndpointPort
}

// EndpointData represents the data returned from reading an endpoint
type EndpointData struct {
	Id             string
	Name           string
	Description    *string
	AccessType     missioncontrol.GetConnectionEndpointAccessType
	K8SServiceId   *string
	K8SServiceType *missioncontrol.GetConnectionEndpointK8sServiceType
	Ports          []missioncontrol.ServiceConnectionEndpointPort
}

// AsyncOperationParams holds parameters for async operation handling
type AsyncOperationParams struct {
	ServiceId     string
	OperationName string
	ResponseBody  []byte
	HTTPResponse  *http.Response
	Operation     *missioncontrol.OperationResponse
	JSON400       *missioncontrol.ErrorResponse
	JSON401       *missioncontrol.ErrorResponse
	JSON403       *missioncontrol.ErrorResponse
	JSON404       *missioncontrol.ErrorResponse
	JSON503       *missioncontrol.ErrorResponse
}

// NewEndpointManager creates a new EndpointManager instance
func NewEndpointManager(client *apiclient.RetryableClientWithResponses, pollingInterval int) *EndpointManager {
	return &EndpointManager{
		APIClient:          client,
		APIPollingInterval: pollingInterval,
	}
}

// handleAsyncOperation handles common async operation logic for create/update/delete operations
// Returns true if there was an error, false if the operation completed successfully
func (em *EndpointManager) handleAsyncOperation(ctx context.Context, params *AsyncOperationParams, diagnostics *diag.Diagnostics) bool {
	// Handle errors
	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		http.StatusAccepted,
		params.ResponseBody,
		params.HTTPResponse,
		params.JSON400,
		params.JSON401,
		params.JSON403,
		params.JSON404,
		params.JSON503,
	)

	tflog.Debug(ctx, fmt.Sprintf("%s response: %s", params.OperationName, params.ResponseBody))

	if errorHandler.HandleError(diagnostics) {
		return true
	}

	// log operation object as json
	if opJson, err := json.Marshal(params.Operation); err == nil {
		tflog.Debug(ctx, fmt.Sprintf("%s operation object: %s", params.OperationName, opJson))
	}

	operationutils.WaitForOperationToComplete(&operationutils.OperationParams{
		Ctx:             ctx,
		ServiceId:       params.ServiceId,
		OperationId:     *params.Operation.Data.Id,
		R:               em.APIClient,
		PollingInterval: em.APIPollingInterval,
		Timeout:         time.Minute * 5,
		Diagnostics:     diagnostics,
	})

	tflog.Trace(ctx, fmt.Sprintf("Connection Endpoint %s Http Response body: %s", params.OperationName, params.ResponseBody))
	return false
}

// getConnectionEndpointsList fetches all connection endpoints for a service
func (em *EndpointManager) getConnectionEndpointsList(ctx context.Context, serviceId string, diagnostics *diag.Diagnostics) []missioncontrol.GetConnectionEndpoint {
	apiResp, err := em.APIClient.GetConnectionEndpointsWithResponse(ctx, serviceId)
	if err != nil {
		diagnostics.AddError(
			errReadingConnectionEndpoints,
			fmt.Sprintf(errReadingConnectionEndpointsFmt, serviceId, err),
		)
		return nil
	}

	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		http.StatusOK,
		apiResp.Body,
		apiResp.HTTPResponse,
		nil,
		apiResp.JSON401,
		apiResp.JSON403,
		apiResp.JSON404,
		apiResp.JSON503,
	)

	if errorHandler.HandleError(diagnostics) {
		return nil
	}

	return apiResp.JSON200.Data
}

// UpdateEndpoint updates an existing connection endpoint
func (em *EndpointManager) UpdateEndpoint(ctx context.Context, serviceId string, connectionEndpointId string, config EndpointConfig, diagnostics *diag.Diagnostics) {
	// Prepare the update request body
	updateRequest := missioncontrol.UpdateConnectionEndpointJSONRequestBody{
		Name: config.Name,
	}

	if config.Description != nil {
		updateRequest.Description = config.Description
	}

	if len(config.Ports) > 0 {
		updateRequest.Ports = config.Ports
	}

	// Make the update API call
	apiClientUpdateResp, err := em.APIClient.UpdateConnectionEndpointWithResponse(ctx, serviceId, connectionEndpointId, updateRequest)
	if err != nil {
		diagnostics.AddError(
			errCallingAPITitle,
			"Could not update connection endpoint, unexpected error: "+err.Error(),
		)
		return
	}

	em.handleAsyncOperation(ctx, &AsyncOperationParams{
		ServiceId:     serviceId,
		OperationName: "UpdateEndpoint",
		ResponseBody:  apiClientUpdateResp.Body,
		HTTPResponse:  apiClientUpdateResp.HTTPResponse,
		Operation:     apiClientUpdateResp.JSON202,
		JSON400:       apiClientUpdateResp.JSON400,
		JSON401:       apiClientUpdateResp.JSON401,
		JSON403:       apiClientUpdateResp.JSON403,
		JSON404:       apiClientUpdateResp.JSON404,
		JSON503:       apiClientUpdateResp.JSON503,
	}, diagnostics)
}

// ConvertObjectValueToPorts converts the ObjectValue ports structure back to the API format
// using proper protocol name mapping from Terraform attributes to API protocol names
func (em *EndpointManager) ConvertObjectValueToPorts(ctx context.Context, portsObj types.Object) ([]missioncontrol.ServiceConnectionEndpointPort, diag.Diagnostics) {
	var diags diag.Diagnostics
	var ports []missioncontrol.ServiceConnectionEndpointPort

	// Extract attributes from the object
	attrs := portsObj.Attributes()
	for attrName, attrValue := range attrs {
		// Map the Terraform attribute name to the API protocol name
		apiProtocolName := model.GetAPIProtocolName(attrName)

		zero := int32(0)
		if attrValue.IsNull() || attrValue.IsUnknown() {
			ports = append(ports, missioncontrol.ServiceConnectionEndpointPort{
				Protocol: missioncontrol.ServiceConnectionEndpointPortProtocol(apiProtocolName),
				Port:     &zero,
			})
		}

		// Extract the port from the nested object
		if obj, ok := attrValue.(types.Object); ok {
			portAttrs := obj.Attributes()
			if portValue, exists := portAttrs["port"]; exists && !portValue.IsNull() {
				if portInt, ok := portValue.(types.Int64); ok {
					port := int32(portInt.ValueInt64())
					ports = append(ports, missioncontrol.ServiceConnectionEndpointPort{
						Protocol: missioncontrol.ServiceConnectionEndpointPortProtocol(apiProtocolName),
						Port:     &port,
					})
				}
			}
		}
	}

	return ports, diags
}

func (em *EndpointManager) SplitCombinedId(combinedId string, diagnostics *diag.Diagnostics) (string, string) {
	idParts := strings.Split(combinedId, "/connectionEndpoints/")
	if len(idParts) != 2 {
		diagnostics.AddError(
			"Invalid ID received",
			"ID must be in the format {serviceId}/connectionEndpoints/{connectionEndpointId}, was given: "+combinedId,
		)
		return "", ""
	}

	serviceId := idParts[0]
	connectionEndpointId := idParts[1]
	return serviceId, connectionEndpointId
}

// GetEndpointById retrieves a connection endpoint by service and endpoint IDs
func (em *EndpointManager) GetEndpointById(ctx context.Context, id string, ceId string, diagnostics *diag.Diagnostics) *missioncontrol.GetConnectionEndpoint {
	return em.ReadEndpoint(ctx, id, ceId, diagnostics)
}

// FindEndpointByName finds an endpoint by its name and service id by getting all endpoints on a service and filtering them
func (em *EndpointManager) FindEndpointByName(ctx context.Context, serviceId string, name string, diagnostics *diag.Diagnostics) *missioncontrol.GetConnectionEndpoint {
	endpoints := em.getConnectionEndpointsList(ctx, serviceId, diagnostics)
	if endpoints == nil {
		return nil
	}

	// Look for an existing endpoint with the matching name
	for _, endpoint := range endpoints {
		if endpoint.Name == name {
			return &endpoint
		}
	}

	return nil
}

// ListEndpoints fetches all connection endpoints for a service and logs debug information
func (em *EndpointManager) ListEndpoints(ctx context.Context, serviceId string, diagnostics *diag.Diagnostics) []missioncontrol.GetConnectionEndpoint {
	endpoints := em.getConnectionEndpointsList(ctx, serviceId, diagnostics)
	if endpoints == nil {
		return nil
	}

	// Log the endpoints for debugging
	if endpointsJson, err := json.Marshal(endpoints); err == nil {
		tflog.Debug(ctx, fmt.Sprintf("Connection Endpoints API Response: %s", string(endpointsJson)))
	}

	return endpoints
}

// CreateEndpoint creates a new connection endpoint and returns its ID
func (em *EndpointManager) CreateEndpoint(ctx context.Context, serviceId string, config EndpointConfig, diagnostics *diag.Diagnostics) string {
	// Prepare the create request body
	createRequest := missioncontrol.CreateConnectionEndpointJSONRequestBody{
		Name:       config.Name,
		AccessType: missioncontrol.CreateConnectionEndpointAccessType(config.AccessType),
	}

	if config.Description != nil {
		createRequest.Description = config.Description
	}

	if len(config.Ports) > 0 {
		createRequest.Ports = config.Ports
	}

	// Make the create API call
	apiClientCreateResp, err := em.APIClient.CreateConnectionEndpointWithResponse(ctx, serviceId, createRequest)
	if err != nil {
		diagnostics.AddError(
			errCallingAPITitle,
			"Could not create connection endpoint, unexpected error: "+err.Error(),
		)
		return ""
	}

	if em.handleAsyncOperation(ctx, &AsyncOperationParams{
		ServiceId:     serviceId,
		OperationName: "CreateEndpoint",
		ResponseBody:  apiClientCreateResp.Body,
		HTTPResponse:  apiClientCreateResp.HTTPResponse,
		Operation:     apiClientCreateResp.JSON202,
		JSON400:       apiClientCreateResp.JSON400,
		JSON401:       apiClientCreateResp.JSON401,
		JSON403:       apiClientCreateResp.JSON403,
		JSON404:       apiClientCreateResp.JSON404,
		JSON503:       apiClientCreateResp.JSON503,
	}, diagnostics) {
		return ""
	}

	// Extract and return the connection endpoint ID from the operation response
	if apiClientCreateResp.JSON202.Data.ResourceId != nil {
		_, connectionEndpointId := em.SplitCombinedId(*apiClientCreateResp.JSON202.Data.ResourceId, diagnostics)
		return connectionEndpointId
	}

	diagnostics.AddError(
		"Error Creating Connection Endpoint",
		"The connection endpoint was created but no ID was returned in the response",
	)

	return ""
}

// DeleteEndpoint deletes an existing connection endpoint
func (em *EndpointManager) DeleteEndpoint(ctx context.Context, serviceId string, connectionEndpointId string, diagnostics *diag.Diagnostics) {
	// Make the delete API call
	apiClientDeleteResp, err := em.APIClient.DeleteConnectionEndpointWithResponse(ctx, serviceId, connectionEndpointId)
	if err != nil {
		diagnostics.AddError(
			errCallingAPITitle,
			"Could not delete connection endpoint, unexpected error: "+err.Error(),
		)
		return
	}

	em.handleAsyncOperation(ctx, &AsyncOperationParams{
		ServiceId:     serviceId,
		OperationName: "DeleteEndpoint",
		ResponseBody:  apiClientDeleteResp.Body,
		HTTPResponse:  apiClientDeleteResp.HTTPResponse,
		Operation:     apiClientDeleteResp.JSON202,
		JSON400:       apiClientDeleteResp.JSON400,
		JSON401:       apiClientDeleteResp.JSON401,
		JSON403:       apiClientDeleteResp.JSON403,
		JSON404:       apiClientDeleteResp.JSON404,
		JSON503:       apiClientDeleteResp.JSON503,
	}, diagnostics)
}

func (em *EndpointManager) ReadEndpoint(ctx context.Context, serviceId string, connectionEndpointId string, diagnostics *diag.Diagnostics) *missioncontrol.GetConnectionEndpoint {
	apiClientGetResp, err := em.APIClient.GetConnectionEndpointWithResponse(ctx, serviceId, connectionEndpointId)
	if err != nil {
		diagnostics.AddError(errReadingConnectionEndpoints, fmt.Sprintf(errReadingConnectionEndpointsFmt, serviceId, err))
		return nil
	}

	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		http.StatusOK,
		apiClientGetResp.Body,
		apiClientGetResp.HTTPResponse,
		apiClientGetResp.JSON400,
		apiClientGetResp.JSON401,
		apiClientGetResp.JSON403,
		apiClientGetResp.JSON404,
		apiClientGetResp.JSON503,
	)
	if errorHandler.HandleError(diagnostics) {
		return nil
	}

	tflog.Debug(ctx, fmt.Sprintf("ReadEndpoint response: %s", apiClientGetResp.Body))

	return &apiClientGetResp.JSON200.Data
}
