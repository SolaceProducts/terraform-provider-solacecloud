// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"terraform-provider-solacecloud/internal/model"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/internal/provider/connectionendpoint"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/internal/util"
	"terraform-provider-solacecloud/missioncontrol"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Error message constants
const (
	errReadingDefaultConnectionEndpoint = "Error Reading Default Connection Endpoint"
)

// Ensure provider defined types fully satisfy framework interfaces.
func (r *ServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerConfig := req.ProviderData.(shared.ProviderConfig)

	//r.APIClient = providerConfig.APIClient
	r.APIClient = apiclient.NewRetryableClient(providerConfig.APIClient, 3, 10)
	r.APIPollingInterval = providerConfig.APIPollingInterval
	r.endpointManager = connectionendpoint.NewEndpointManager(r.APIClient, r.APIPollingInterval)
}

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServiceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	///////////////////////////////////////////////
	// Send SCService Create Request
	///////////////////////////////////////////////
	varServiceBody := missioncontrol.CreateServiceRequest{
		Name:         data.Name.ValueString(),
		DatacenterId: data.DatacenterId.ValueString(),
	}

	// Add default connection endpoint if configured for service create (POST request)
	if defaultEndpoint := r.buildConnectionEndpointRequest(ctx, &data); defaultEndpoint != nil {
		varServiceBody.ServiceConnectionEndpoints = &[]missioncontrol.ConnectionEndpoint{*defaultEndpoint}
	}
	if util.IsKnown(data.MateLinkEncryption) {
		varServiceBody.RedundancyGroupSslEnabled = data.MateLinkEncryption.ValueBoolPointer()
	}
	if util.IsKnown(data.MessageVpnName) {
		varServiceBody.MsgVpnName = data.MessageVpnName.ValueStringPointer()
	}
	if util.IsKnown(data.MaxSpoolUsage) {
		maxSpoolUsage := int32(data.MaxSpoolUsage.ValueInt64())
		varServiceBody.MaxSpoolUsage = &maxSpoolUsage
	}
	if util.IsKnown(data.EventBrokerVersion) {
		varServiceBody.EventBrokerVersion = data.EventBrokerVersion.ValueStringPointer()
	}
	if util.IsKnown(data.ClusterName) {
		varServiceBody.ClusterName = data.ClusterName.ValueStringPointer()
	}
	if util.IsKnown(data.CustomRouterName) {
		varServiceBody.CustomRouterName = data.CustomRouterName.ValueStringPointer()
	}
	if util.IsKnown(data.ServiceClassId) {
		varServiceBody.ServiceClassId = missioncontrol.ServiceClassId(data.ServiceClassId.ValueString())
	}
	if util.IsKnown(data.EnvironmentId) {
		varServiceBody.EnvironmentId = data.EnvironmentId.ValueStringPointer()
	}
	if util.IsKnown(data.Locked) {
		varServiceBody.Locked = data.Locked.ValueBoolPointer()
	}

	apiClientCreateResp, err := r.APIClient.CreateServiceWithResponse(ctx, varServiceBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error calling Solace Cloud API",
			"Could not create/get service, unexpected error: "+err.Error(),
		)
		return
	}

	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		http.StatusAccepted,
		apiClientCreateResp.Body,
		apiClientCreateResp.HTTPResponse,
		apiClientCreateResp.JSON400,
		apiClientCreateResp.JSON401,
		apiClientCreateResp.JSON403,
		nil, // JSON404 not available for CreateServiceResponse
		apiClientCreateResp.JSON503,
	)
	if errorHandler.HandleError(&resp.Diagnostics) {
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("Service CREATED Http Response body: %s", apiClientCreateResp.Body))

	var resData = apiClientCreateResp.JSON202.Data
	var serviceResourceID = *resData.ResourceId

	// Save SC Resource Values into the Terraform state.
	data.Id = types.StringValue(serviceResourceID)

	tflog.Info(ctx, fmt.Sprintf("Service Resource ID: %s", serviceResourceID))

	//////////////////////////////////////////////////////
	// Wait & Check if SCService is still being created
	//////////////////////////////////////////////////////

	//Send Empty Params, we only need basic Info
	createServParam := missioncontrol.GetServiceParams{}

	for {

		apiClientStatusResp, err := r.APIClient.GetServiceWithResponse(ctx, serviceResourceID, &createServParam)
		if err != nil {
			return
		}

		if apiClientStatusResp.StatusCode() != http.StatusOK {
			if apiClientStatusResp.StatusCode() == http.StatusUnauthorized {
				resp.Diagnostics.AddError(
					"Authentication Failed During Service Status Check",
					"Received HTTP 401 Unauthorized while checking service creation status. "+
						"This may indicate that your API token has expired or been revoked during the service creation process. "+
						"Verify your authentication configuration and try again.",
				)
				return
			}
			resp.Diagnostics.AddError(
				"Failed to get service while waiting for service creation to complete.",
				fmt.Sprintf("Expected HTTP 200 but received %d while waiting for service to complete", apiClientStatusResp.StatusCode()),
			)
			return
		}

		var SCServiceStatus = *apiClientStatusResp.JSON200.Data.CreationState

		tflog.Trace(ctx, fmt.Sprintf("Service STATUS Http Response body: %s", apiClientStatusResp.Body))

		if SCServiceStatus == missioncontrol.FAILED {
			resp.Diagnostics.AddError(
				"Resource Creation FAILED",
				fmt.Sprintf("Received creationState as: %s from the GetService API Request", SCServiceStatus),
			)
			return
		}
		if SCServiceStatus == missioncontrol.COMPLETED {
			tflog.Info(ctx, fmt.Sprintf("Service Status reported as %s, finished Waiting", missioncontrol.OperationStatusSUCCEEDED))
			break
		}

		tflog.Info(ctx, fmt.Sprintf("Waiting for Service Status: %s to Complete", SCServiceStatus))
		time.Sleep(time.Duration(r.APIPollingInterval) * time.Second)
	}

	var assumedState ServiceResourceModel
	assumedState.Id = types.StringValue(serviceResourceID)
	diags := r.readDataInternal(ctx, &assumedState, &data)
	resp.Diagnostics.Append(*diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	///////////////////////////////////////////////
	// After SCService creation has been COMPLETED
	// Get Connection Properties from the Service
	///////////////////////////////////////////////

	updateDiags := r.updateInternal(ctx, &assumedState, &data)
	resp.Diagnostics.Append(updateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Read the updated service to data after potential updates and endpoint creation
	diags = r.readDataInternal(ctx, &assumedState, &data)
	resp.Diagnostics.Append(*diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// update state with data
	resp.Diagnostics.Append(resp.State.Set(ctx, assumedState)...)
	tflog.Trace(ctx, "##### Created Solace Cloud Resources #####")
}

// Reads the data and sets the data on the "data" object. if DefaultEndpointIsDefined, then reads and sets that also.
func (r *ServiceResource) readDataInternal(ctx context.Context, currentState *ServiceResourceModel, initialPlan *ServiceResourceModel) *diag.Diagnostics {
	serviceResourceID := currentState.Id.ValueString()
	varExpand := []missioncontrol.GetServiceParamsExpand{missioncontrol.GetServiceParamsExpandBroker}
	varExpand = append(varExpand, missioncontrol.GetServiceParamsExpandServiceConnectionEndpoints)
	varExpand = append(varExpand, missioncontrol.GetServiceParamsExpandMessageSpoolDetails)
	getCredServParam := missioncontrol.GetServiceParams{Expand: &varExpand}
	diagnostics := diag.Diagnostics{}

	apiClientGetCredResp, err := r.APIClient.GetServiceWithResponse(ctx, serviceResourceID, &getCredServParam)
	if err != nil {
		diagnostics.AddError("Error Reading Service", fmt.Sprintf("Could not read service %s: %s", serviceResourceID, err))
		return &diagnostics
	}

	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		http.StatusOK,
		apiClientGetCredResp.Body,
		apiClientGetCredResp.HTTPResponse,
		nil,
		apiClientGetCredResp.JSON401,
		apiClientGetCredResp.JSON403,
		apiClientGetCredResp.JSON404,
		apiClientGetCredResp.JSON503,
	)

	if errorHandler.HandleError(&diagnostics) {
		return &diagnostics
	}

	tflog.Trace(ctx, fmt.Sprintf("SC Service Broker Details Http Response body: %s", apiClientGetCredResp.Body))

	var diags diag.Diagnostics
	var respData = apiClientGetCredResp.JSON200.Data
	tflog.Info(ctx, fmt.Sprintf("Response Data: %+v", respData))

	var respBroker = respData.Broker

	var respMsgVPN = (*respBroker.MsgVpns)[0]

	// Check if MissionControlManagerLoginCredential is available
	var managerManagementCredential basetypes.ObjectValue
	if respMsgVPN.MissionControlManagerLoginCredential == nil {
		tflog.Warn(ctx, "MissionControlManagerLoginCredential is not supported in this broker version. Setting null credentials.")
		managerManagementCredential = types.ObjectNull(model.BasicAuthCredentialObjectType().AttrTypes)
	} else {
		var diags diag.Diagnostics
		managerManagementCredential, diags = model.BasicAuthCredentialModel{
			Username: types.StringValue(*respMsgVPN.MissionControlManagerLoginCredential.Username),
			Password: types.StringValue(*respMsgVPN.MissionControlManagerLoginCredential.Password),
		}.ToObjectValue()
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return &diagnostics
		}
	}

	editorManagementCredential, diags := model.BasicAuthCredentialModel{
		Username: types.StringValue(*respMsgVPN.ManagementAdminLoginCredential.Username),
		Password: types.StringValue(*respMsgVPN.ManagementAdminLoginCredential.Password),
	}.ToObjectValue()
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return &diagnostics
	}

	viewerManagementCredential, diags := model.BasicAuthCredentialModel{
		Username: types.StringValue(*respData.Broker.ManagementReadOnlyLoginCredential.Username),
		Password: types.StringValue(*respData.Broker.ManagementReadOnlyLoginCredential.Password),
	}.ToObjectValue()
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return &diagnostics
	}

	messagingClientCredential, diags := model.BasicAuthCredentialModel{
		Username: types.StringValue(*respMsgVPN.ServiceLoginCredential.Username),
		Password: types.StringValue(*respMsgVPN.ServiceLoginCredential.Password),
	}.ToObjectValue()
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return &diagnostics
	}

	supportedDmrAuthenticationModes, diags := types.ListValueFrom(ctx, types.StringType, respBroker.Cluster.SupportedAuthenticationMode)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return &diagnostics
	}

	dmrClusterInfo, diags := model.DmrClusterInfoModel{
		Name:                         types.StringValue(*respBroker.Cluster.Name),
		Password:                     types.StringValue(*respBroker.Cluster.Password),
		RemoteAddress:                types.StringValue(*respBroker.Cluster.RemoteAddress),
		PrimaryRouterName:            types.StringValue(*respBroker.Cluster.PrimaryRouterName),
		SupportedAuthenticationModes: supportedDmrAuthenticationModes,
	}.ToObjectValue()
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return &diagnostics
	}

	messageVpn := model.MessageVpnModel{
		Name:                            types.StringValue(*respMsgVPN.MsgVpnName),
		AuthenticationBasicEnabled:      types.BoolValue(*respMsgVPN.AuthenticationBasicEnabled),
		AuthenticationBasicType:         types.StringValue(string(*respMsgVPN.AuthenticationBasicType)),
		AuthenticationClientCertEnabled: types.BoolValue(*respMsgVPN.AuthenticationClientCertEnabled),
		AuthenticationClientCertValidateDateEnabled: types.BoolValue(*respMsgVPN.AuthenticationClientCertValidateDateEnabled),
		MaxConnectionCount:                          types.Int64Value(int64(*respMsgVPN.MaxConnectionCount)),
		MaxEgressFlowCount:                          types.Int64Value(int64(*respMsgVPN.MaxEgressFlowCount)),
		MaxEndpointCount:                            types.Int64Value(int64(*respMsgVPN.MaxEndpointCount)),
		MaxIngressFlowCount:                         types.Int64Value(int64(*respMsgVPN.MaxIngressFlowCount)),
		MaxMsgSpoolUsage:                            types.Int64Value(int64(*respMsgVPN.MaxMsgSpoolUsage)),
		MaxSubscriptionCount:                        types.Int64Value(int64(*respMsgVPN.MaxSubscriptionCount)),
		MaxTransactedSessionCount:                   types.Int64Value(int64(*respMsgVPN.MaxTransactedSessionCount)),
		MaxTransactionCount:                         types.Int64Value(int64(*respMsgVPN.MaxTransactionCount)),
		TruststoreUri:                               types.StringPointerValue(respMsgVPN.TruststoreUri),
		ManagerManagementCredential:                 managerManagementCredential,
		EditorManagementCredential:                  editorManagementCredential,
		ViewerManagementCredential:                  viewerManagementCredential,
		MessagingClientCredential:                   messagingClientCredential,
	}
	currentState.MessageVpn, diags = messageVpn.ToObjectValue()
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return &diagnostics
	}

	currentState.DatacenterId = types.StringValue(*respData.DatacenterId)
	currentState.ServiceClassId = types.StringValue(string(*respData.ServiceClassId))
	currentState.Name = types.StringValue(*respData.Name)
	currentState.EventBrokerVersion = types.StringValue(respData.EventBrokerServiceVersion)
	currentState.MessageVpnName = types.StringPointerValue(respMsgVPN.MsgVpnName)
	currentState.MaxSpoolUsage = types.Int64Value(int64(*respBroker.MaxSpoolUsage))
	currentState.Locked = types.BoolPointerValue(respData.Locked)
	currentState.EnvironmentId = types.StringPointerValue(respData.EnvironmentId)
	currentState.MateLinkEncryption = types.BoolPointerValue(respBroker.RedundancyGroupSslEnabled)
	currentState.ClusterName = types.StringPointerValue(respBroker.Cluster.Name)
	currentState.OwnedBy = types.StringValue(*respData.OwnedBy)
	currentState.DmrClusterInfo = dmrClusterInfo
	name := determineCustomRouterName(respBroker.Cluster.PrimaryRouterName)
	if name != "" {
		currentState.CustomRouterName = types.StringValue(name)
	}
	readDefaultDiags := r.readDefaultConnectionEndpoint(ctx, currentState, initialPlan)
	diagnostics.Append(*readDefaultDiags...)
	if diagnostics.HasError() {
		return &diagnostics
	}
	tflog.Info(ctx, fmt.Sprintf("ResourceId: %s - ResourceVPNName: %s - ResourceServiceClass: %s - ResourceDatacenterId: %s - ",
		serviceResourceID,
		*respMsgVPN.MsgVpnName,
		*apiClientGetCredResp.JSON200.Data.ServiceClassId,
		*apiClientGetCredResp.JSON200.Data.DatacenterId))
	return &diagnostics
}

// updates the ConnectionEndpoint state.
func (r *ServiceResource) readDefaultConnectionEndpoint(ctx context.Context,
	currentState *ServiceResourceModel,
	initialPlan *ServiceResourceModel,
) *diag.Diagnostics {
	var resp diag.Diagnostics

	createOrUpdateOperation := initialPlan != nil

	// if it is not readonly, and the initial plan does not have a default connection endpoint, skip
	if createOrUpdateOperation && !util.IsKnown(initialPlan.ConnectionEndpoint) {
		currentState.ConnectionEndpoint = types.ObjectNull(model.NestedConnectionEndpointTypes())
		return &diag.Diagnostics{}
	}

	// if it is readonly, and the current state does not have a default connection endpoint, skip
	if !createOrUpdateOperation && !util.IsKnown(currentState.ConnectionEndpoint) {
		currentState.ConnectionEndpoint = types.ObjectNull(model.NestedConnectionEndpointTypes())
		return &diag.Diagnostics{}
	}

	id, diagnostics, done := r.findEndpointId(ctx, currentState, initialPlan, createOrUpdateOperation, resp)
	if done {
		return diagnostics
	}

	endpoint := r.endpointManager.GetEndpointById(ctx, currentState.Id.ValueString(), id, &resp)
	if resp.HasError() {
		return &resp
	}
	if endpoint == nil {
		resp.AddError(errReadingDefaultConnectionEndpoint, fmt.Sprintf("Could not read default connection endpoint %s", id))
		return &resp
	}
	// convert to default endpoint
	portsObject, diags := model.ToObjectValue(endpoint.Ports)
	if diags.HasError() {
		return &diags
	}
	defaultConnectionEndpointModel := model.NestedConnectionEndpointModel{
		Id:             types.StringPointerValue(endpoint.Id),
		ServiceId:      types.StringPointerValue(currentState.Id.ValueStringPointer()),
		Name:           types.StringValue(endpoint.Name),
		Description:    types.StringPointerValue(endpoint.Description),
		AccessType:     types.StringValue(string(endpoint.AccessType)),
		K8SServiceType: types.StringValue(string(*endpoint.K8sServiceType)),
		K8SServiceId:   types.StringPointerValue(endpoint.K8sServiceId),
		Ports:          portsObject,
	}

	currentState.ConnectionEndpoint, diags = defaultConnectionEndpointModel.ToObjectValue()
	resp.Append(diags...)
	if resp.HasError() {
		return &resp
	}

	return &diag.Diagnostics{}
}

func (r *ServiceResource) findEndpointId(ctx context.Context, currentState *ServiceResourceModel, initialPlan *ServiceResourceModel, createOrUpdateOperation bool, resp diag.Diagnostics) (string, *diag.Diagnostics, bool) {
	var id string
	if createOrUpdateOperation {
		// needs to fetch connection endpoints
		name := initialPlan.ConnectionEndpoint.Attributes()["name"].(types.String).ValueString()
		endpoint := r.endpointManager.FindEndpointByName(ctx, currentState.Id.ValueString(), name, &resp)
		if resp.HasError() {
			return "", &resp, true
		}
		if endpoint == nil {
			resp.AddError(errReadingDefaultConnectionEndpoint, fmt.Sprintf("Could not find default connection endpoint with name %s", name))
			return "", &resp, true
		}
		id = *endpoint.Id
	} else {
		id = currentState.ConnectionEndpoint.Attributes()["id"].(types.String).ValueString()
		if id == "" {
			resp.AddError(errReadingDefaultConnectionEndpoint, "Default connection endpoint id is not set in state.")
			return "", &resp, true
		}
	}
	return id, nil, false
}

func determineCustomRouterName(primaryRouterName *string) string {
	if strings.Contains(*primaryRouterName, "primarycn") {
		return strings.Split(*primaryRouterName, "primarycn")[0]
	}

	return ""
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(resp.State.Get(ctx, &data)...)

	diags := r.readDataInternal(ctx, &data, nil)
	if diags.HasError() {
		// Check if the error is because the service doesn't exist
		for _, diagnostic := range *diags {
			// Check for "Not Found" summary or message containing "Could not find"
			if diagnostic.Summary() == "Not Found" || strings.Contains(diagnostic.Detail(), "Could not find") {
				// Service no longer exists, remove it from state
				// Don't return the could not find service error
				resp.State.RemoveResource(ctx)
				return
			} else {
				resp.Diagnostics.Append(diagnostic)
			}
		}
		return
	}
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log

	var serviceId = data.Id.ValueString()
	tflog.Info(ctx, fmt.Sprintf("About to delete SC ResourceID ID: %s", serviceId))

	apiClientResp, err := r.APIClient.DeleteServiceWithResponse(ctx, serviceId)
	if err != nil {
		resp.Diagnostics.AddError("An internal error has occurred", err.Error())
		return
	}

	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		http.StatusAccepted,
		apiClientResp.Body,
		apiClientResp.HTTPResponse,
		apiClientResp.JSON400,
		apiClientResp.JSON401,
		apiClientResp.JSON403,
		apiClientResp.JSON404,
		apiClientResp.JSON503,
	)

	if errorHandler.HandleError(&resp.Diagnostics) {
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("SC Service DELETE Http Response body: %s", apiClientResp.Body))

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Converts the default connection endpoint configuration from the Terraform model
// to the Mission Control API model for creating the service.
func (r *ServiceResource) buildConnectionEndpointRequest(ctx context.Context, s *ServiceResourceModel) *missioncontrol.ConnectionEndpoint {
	// Check if default connection endpoint is configured
	if s.ConnectionEndpoint.IsNull() || s.ConnectionEndpoint.IsUnknown() {
		return nil
	}

	// Extract attributes from the default connection endpoint object
	attrs := s.ConnectionEndpoint.Attributes()

	// Extract name (required)
	nameAttr := attrs["name"]
	if nameAttr.IsNull() || nameAttr.IsUnknown() {
		return nil
	}
	name := nameAttr.(types.String).ValueString()

	// Extract access type (required)
	accessTypeAttr := attrs["access_type"]
	if accessTypeAttr.IsNull() || accessTypeAttr.IsUnknown() {
		return nil
	}
	accessType := missioncontrol.ConnectionEndpointAccessType(accessTypeAttr.(types.String).ValueString())

	// Build the connection endpoint
	endpoint := &missioncontrol.ConnectionEndpoint{
		Name:       name,
		AccessType: accessType,
	}

	// Extract description (optional)
	if descAttr, exists := attrs["description"]; exists && !descAttr.IsNull() && !descAttr.IsUnknown() {
		desc := descAttr.(types.String).ValueString()
		endpoint.Description = &desc
	}

	// Extract ports (optional)
	if portsAttr, exists := attrs["ports"]; exists && !portsAttr.IsNull() && !portsAttr.IsUnknown() {
		portsObj := portsAttr.(types.Object)
		if ports, diags := r.endpointManager.ConvertObjectValueToPorts(ctx, portsObj); !diags.HasError() {
			endpoint.Ports = ports
		}
	}

	return endpoint
}
