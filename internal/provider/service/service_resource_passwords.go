package service

import (
	"context"
	"fmt"
	"net/http"
	"terraform-provider-solacecloud/internal/provider/operationutils"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/missioncontrol"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const passwordRotationOperationTimeout = 15 * time.Minute

// The reset response has no passwords, so the caller reads them back. Returns true once the
// reset finished, so the caller can record the trigger even if that read fails.
func (r *ServiceResource) rotateBrokerPasswords(ctx context.Context, state *ServiceResourceModel, plan *ServiceResourceModel) (bool, diag.Diagnostics) {
	users, diags := rotatedUsers(ctx, state.BrokerPasswordRotation, plan.BrokerPasswordRotation)
	if diags.HasError() || len(users) == 0 {
		return false, diags
	}

	serviceId := state.Id.ValueString()
	requestBody := missioncontrol.ResetBrokerPasswordsJSONRequestBody{
		Users: make([]missioncontrol.BrokerManagerUser, 0, len(users)),
	}
	for _, user := range users {
		requestBody.Users = append(requestBody.Users, missioncontrol.BrokerManagerUser{Username: user})
	}

	tflog.Info(ctx, fmt.Sprintf("Rotating passwords of %v on service %s", users, serviceId))

	// Not retried (see ResetBrokerPasswordsWithResponse), so every failure here is reported.
	apiResp, err := r.APIClient.ResetBrokerPasswordsWithResponse(ctx, serviceId, requestBody)
	if err != nil {
		diags.AddAttributeError(
			path.Root("broker_password_rotation"),
			"Error calling Solace Cloud API",
			"Could not rotate the event broker passwords, unexpected error: "+err.Error()+". "+
				"The request may have reached the API, so check whether the passwords changed "+
				"before applying again.",
		)
		return false, diags
	}

	if addRotationTimeoutDiagnostics(apiResp, &diags) {
		return false, diags
	}

	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		http.StatusAccepted,
		apiResp.Body,
		apiResp.HTTPResponse,
		apiResp.JSON400,
		apiResp.JSON401,
		apiResp.JSON403,
		apiResp.JSON404,
		apiResp.JSON503,
	)
	if errorHandler.HandleError(&diags) {
		return false, diags
	}

	r.waitForPasswordRotationOperation(ctx, serviceId, apiResp.JSON202, &diags)
	return !diags.HasError(), diags
}

// The shared handler words a 503 as an outage to retry, but here it can arrive after the reset
// began, so it must not be retried. A 409 is left to the shared handler.
func addRotationTimeoutDiagnostics(apiResp *missioncontrol.ResetBrokerPasswordsResponse, diags *diag.Diagnostics) bool {
	if apiResp.StatusCode() == http.StatusServiceUnavailable {
		detail := "The API timed out before answering. The password rotation may already have started or " +
			"finished, and the provider does not retry it because a repeat would rotate the passwords again. " +
			"Check the event broker service in the Cloud Console, then apply again; the rotation trigger in state " +
			"has not changed, so Terraform will request the rotation again."
		if msg := errorMessage(apiResp.JSON503); msg != "" {
			detail += " The API said: " + msg
		}
		diags.AddAttributeError(path.Root("broker_password_rotation"), "Password rotation may have started", detail)
		return true
	}
	return false
}

func errorMessage(e *missioncontrol.ErrorResponse) string {
	if e == nil || e.Message == nil {
		return ""
	}
	return *e.Message
}

// Waits so the read-back that follows sees the new passwords.
func (r *ServiceResource) waitForPasswordRotationOperation(ctx context.Context, serviceId string, operation *missioncontrol.OperationResponse, diags *diag.Diagnostics) {
	if operation == nil || operation.Data.Id == nil {
		diags.AddError(
			"Unexpected Solace Cloud API response",
			"The password rotation request was accepted without an operation identifier, so its progress cannot be tracked.",
		)
		return
	}

	operationutils.WaitForOperationToComplete(&operationutils.OperationParams{
		Ctx:             ctx,
		ServiceId:       serviceId,
		OperationId:     *operation.Data.Id,
		R:               r.APIClient,
		PollingInterval: r.APIPollingInterval,
		Timeout:         passwordRotationOperationTimeout,
		Diagnostics:     diags,
	})
}
