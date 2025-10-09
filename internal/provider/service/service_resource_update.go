package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"net/http"
	"terraform-provider-solacecloud/internal/model"
	"terraform-provider-solacecloud/internal/provider/connectionendpoint"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/internal/util"
	"terraform-provider-solacecloud/missioncontrol"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan ServiceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.updateInternal(ctx, &state, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	r.readDataInternal(ctx, &state, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServiceResource) updateInternal(ctx context.Context, state *ServiceResourceModel, plan *ServiceResourceModel) diag.Diagnostics {
	// create new empty diagnostics
	var diags diag.Diagnostics

	// check if
	nameChanged := !plan.Name.IsUnknown() && !plan.Name.IsNull() && state.Name != plan.Name
	ownedByChanged := !plan.OwnedBy.IsUnknown() && !plan.OwnedBy.IsNull() && state.OwnedBy != plan.OwnedBy
	lockedChanged := !plan.Locked.IsUnknown() && !plan.Locked.IsNull() && state.Locked != plan.Locked
	storageSizeChanged := !plan.MaxSpoolUsage.IsUnknown() && !plan.MaxSpoolUsage.IsNull() && state.MaxSpoolUsage != plan.MaxSpoolUsage

	if nameChanged || ownedByChanged || lockedChanged {
		updateRequest := missioncontrol.UpdateServiceRequest{}

		if nameChanged {
			updateRequest.Name = plan.Name.ValueStringPointer()
		}
		if ownedByChanged {
			updateRequest.OwnedBy = plan.OwnedBy.ValueStringPointer()
		}
		if lockedChanged {
			updateRequest.Locked = plan.Locked.ValueBoolPointer()
		}

		patchDiags := r.patchService(ctx, state, &updateRequest)
		diags.Append(patchDiags...)
		if diags.HasError() {
			return diags
		}
	}

	if storageSizeChanged {
		updateDiags := r.updateStorageSize(ctx, *state, *plan)
		diags.Append(*updateDiags...)
		if diags.HasError() {
			return diags
		}
	}
	diags = r.updateDefaultConnectionEndpoint(ctx, state, plan)
	return diags
}

func (r *ServiceResource) updateDefaultConnectionEndpoint(ctx context.Context, state *ServiceResourceModel, plan *ServiceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// on delete case. Where connection endpoint is removed from plan. Do nothing
	if util.IsKnown(state.ConnectionEndpoint) && !util.IsKnown(plan.ConnectionEndpoint) {
		return diags
	}

	if util.IsKnown(state.ConnectionEndpoint) {
		var connectionEndpointResourceModel model.NestedConnectionEndpointModel
		state.ConnectionEndpoint.As(ctx, &connectionEndpointResourceModel, basetypes.ObjectAsOptions{})
		var connectionEndpointPlan model.NestedConnectionEndpointModel
		plan.ConnectionEndpoint.As(ctx, &connectionEndpointPlan, basetypes.ObjectAsOptions{})
		// log ports for debugging
		tflog.Info(ctx, fmt.Sprintf("Current ports: %+v", connectionEndpointResourceModel.Ports.Attributes()))
		tflog.Info(ctx, fmt.Sprintf("Planned ports: %+v", connectionEndpointPlan.Ports.Attributes()))

		endpointChanged := connectionEndpointResourceModel.Name != connectionEndpointPlan.Name ||
			connectionEndpointResourceModel.Description != connectionEndpointPlan.Description ||
			!connectionEndpointResourceModel.Ports.Equal(connectionEndpointPlan.Ports)
		if !endpointChanged {
			return diags
		}

		req := r.buildConnectionEndpointRequest(ctx, plan)

		patchRequest := connectionendpoint.EndpointConfig{
			Name:        connectionEndpointPlan.Name.ValueString(),
			Description: connectionEndpointPlan.Description.ValueStringPointer(),
		}
		if req.Ports != nil {
			tflog.Debug(ctx, fmt.Sprintf("Sending %d ports in update request", len(req.Ports)))
			for i, port := range req.Ports {
				tflog.Debug(ctx, fmt.Sprintf("Port[%d]: protocol=%s, port=%d", i, port.Protocol, *port.Port))
			}
			patchRequest.Ports = req.Ports
		}

		r.endpointManager.UpdateEndpoint(ctx, state.Id.ValueString(), connectionEndpointResourceModel.Id.ValueString(), patchRequest, &diags)
	}

	return diags

}

func (r *ServiceResource) patchService(ctx context.Context, state *ServiceResourceModel, updateRequest *missioncontrol.UpdateServiceRequest) diag.Diagnostics {
	var diags diag.Diagnostics
	serviceId := state.Id.ValueString()
	updateRequestBody, err := json.Marshal(updateRequest)
	if err != nil {
		diags.AddError(
			"Error marshaling update request",
			"Could not marshal update request to JSON: "+err.Error(),
		)
		return diags
	}

	apiClientUpdateResp, err := r.APIClient.UpdateServiceWithBodyWithResponse(ctx, serviceId, "application/json", bytes.NewReader(updateRequestBody))
	if err != nil {
		diags.AddError(
			"Error calling Solace Cloud API",
			"Could not update service, unexpected error: "+err.Error(),
		)
		return diags
	}

	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		http.StatusOK,
		apiClientUpdateResp.Body,
		apiClientUpdateResp.HTTPResponse,
		apiClientUpdateResp.JSON400,
		apiClientUpdateResp.JSON401,
		apiClientUpdateResp.JSON403,
		apiClientUpdateResp.JSON404,
		apiClientUpdateResp.JSON503,
	)

	if errorHandler.HandleError(&diags) {
		return diags
	}

	return diags
}

func (r *ServiceResource) updateStorageSize(ctx context.Context, state ServiceResourceModel, plan ServiceResourceModel) *diag.Diagnostics {
	diagnostics := diag.Diagnostics{}
	updateSpoolRequest := missioncontrol.UpdateMessageSpoolJSONRequestBody{
		MessageSpoolSizeInGB: int32(plan.MaxSpoolUsage.ValueInt64()),
	}

	spoolRequestBody, err := json.Marshal(updateSpoolRequest)
	if err != nil {
		diagnostics.AddError(
			"Error marshaling spool update request",
			"Could not marshal spool update request to JSON: "+err.Error(),
		)
		return &diagnostics
	}

	apiClientUpdateSpoolResp, err := r.APIClient.UpdateMessageSpoolWithBodyWithResponse(ctx, state.Id.ValueString(), "application/json", bytes.NewReader(spoolRequestBody))
	if err != nil {
		diagnostics.AddError(
			"Error calling Solace Cloud API",
			"Could not update message spool, unexpected error: "+err.Error(),
		)
		return &diagnostics
	}

	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		http.StatusAccepted,
		apiClientUpdateSpoolResp.Body,
		apiClientUpdateSpoolResp.HTTPResponse,
		apiClientUpdateSpoolResp.JSON400,
		apiClientUpdateSpoolResp.JSON401,
		apiClientUpdateSpoolResp.JSON403,
		apiClientUpdateSpoolResp.JSON404,
		apiClientUpdateSpoolResp.JSON503,
	)

	if errorHandler.HandleError(&diagnostics) {
		return &diagnostics
	}
	operationId := apiClientUpdateSpoolResp.JSON202.Data.Id

	timeout := time.NewTimer(5 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case <-timeout.C:
			diagnostics.AddError(
				"Service operation timeout",
				"Message spool update operation timed out after 5 minutes",
			)
			return &diagnostics
		default:
			apiClientOperationResp, err := r.APIClient.GetServiceOperationWithResponse(ctx, state.Id.ValueString(), *operationId, nil)
			if err != nil {
				diagnostics.AddError(
					"Error calling Solace Cloud API",
					"Could not get service operation status, unexpected error: "+err.Error(),
				)
				return &diagnostics
			}

			errHandler := shared.NewMissionControlErrorResponseAdaptor(
				http.StatusOK,
				apiClientOperationResp.Body,
				apiClientOperationResp.HTTPResponse,
				nil,
				apiClientOperationResp.JSON401,
				apiClientOperationResp.JSON403,
				apiClientOperationResp.JSON404,
				apiClientOperationResp.JSON503,
			)

			if errHandler.HandleError(&diagnostics) {
				return &diagnostics
			}

			operationStatus := *apiClientOperationResp.JSON200.Data.Status
			if operationStatus == missioncontrol.OperationStatusFAILED {
				diagnostics.AddError(
					"Service operation failed",
					fmt.Sprintf("Message spool update operation failed with status: %s", operationStatus),
				)
				return &diagnostics
			}

			if operationStatus == missioncontrol.OperationStatusSUCCEEDED {
				tflog.Info(ctx, "Message spool update operation completed successfully")
				return &diagnostics
			}

			tflog.Info(ctx, fmt.Sprintf("Waiting for message spool update operation to complete, current status: %s", operationStatus))
			time.Sleep(time.Duration(r.APIPollingInterval) * time.Second)
		}
	}
}
