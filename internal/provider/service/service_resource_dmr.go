package service

import (
	"context"
	"fmt"
	"net/http"
	"terraform-provider-solacecloud/internal/provider/operationutils"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/internal/util"
	"terraform-provider-solacecloud/missioncontrol"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// dmrOperationTimeout bounds the wait for an enable or disable DMR operation. Both reconfigure
// the broker's DMR cluster and its message VPN, so they take appreciably longer than a service
// metadata change.
const dmrOperationTimeout = 15 * time.Minute

// updateDmrEnablement enables or disables Dynamic Message Routing on an existing service when
// dmr_enabled changes. Solace Cloud performs each as an asynchronous operation, so the request
// is followed by polling until the operation finishes.
//
// Nothing is sent when the planned value is unknown or matches the current state, which is the
// common case: dmr_enabled is a computed attribute that ordinarily carries the service's own
// DMR state forward unchanged.
func (r *ServiceResource) updateDmrEnablement(ctx context.Context, state *ServiceResourceModel, plan *ServiceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if !util.IsKnown(plan.DmrEnabled) || !util.IsKnown(state.DmrEnabled) {
		return diags
	}
	if plan.DmrEnabled.ValueBool() == state.DmrEnabled.ValueBool() {
		return diags
	}

	serviceId := state.Id.ValueString()
	if plan.DmrEnabled.ValueBool() {
		r.enableDmr(ctx, serviceId, plan, &diags)
	} else {
		r.disableDmr(ctx, serviceId, &diags)
	}
	return diags
}

// enableDmr asks Solace Cloud to create and enable a cloud-managed DMR cluster. A cluster name
// is sent only when the practitioner set one; otherwise Solace Cloud generates it.
func (r *ServiceResource) enableDmr(ctx context.Context, serviceId string, plan *ServiceResourceModel, diags *diag.Diagnostics) {
	requestBody := missioncontrol.EnableDmrClusterJSONRequestBody{}
	if util.IsKnown(plan.ClusterName) && plan.ClusterName.ValueString() != "" {
		requestBody.ClusterName = plan.ClusterName.ValueStringPointer()
	}

	tflog.Info(ctx, fmt.Sprintf("Enabling DMR on service %s", serviceId))

	apiResp, err := r.APIClient.EnableDmrClusterWithResponse(ctx, serviceId, requestBody)
	if err != nil {
		diags.AddError(
			"Error calling Solace Cloud API",
			"Could not enable DMR on the event broker service, unexpected error: "+err.Error(),
		)
		return
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
	if errorHandler.HandleError(diags) {
		return
	}

	r.waitForDmrOperation(ctx, serviceId, apiResp.JSON202, diags)
}

// disableDmr asks Solace Cloud to delete the service's DMR cluster. Solace Cloud refuses this
// while the service belongs to an event mesh, which surfaces as a bad-request diagnostic.
func (r *ServiceResource) disableDmr(ctx context.Context, serviceId string, diags *diag.Diagnostics) {
	tflog.Info(ctx, fmt.Sprintf("Disabling DMR on service %s", serviceId))

	apiResp, err := r.APIClient.DeleteDmrClusterWithResponse(ctx, serviceId)
	if err != nil {
		diags.AddError(
			"Error calling Solace Cloud API",
			"Could not disable DMR on the event broker service, unexpected error: "+err.Error(),
		)
		return
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
	if errorHandler.HandleError(diags) {
		return
	}

	r.waitForDmrOperation(ctx, serviceId, apiResp.JSON202, diags)
}

// waitForDmrOperation polls the operation returned by an enable or disable request until it
// finishes, so the value written to state reflects the service's settled DMR configuration.
func (r *ServiceResource) waitForDmrOperation(ctx context.Context, serviceId string, operation *missioncontrol.OperationResponse, diags *diag.Diagnostics) {
	if operation == nil || operation.Data.Id == nil {
		diags.AddError(
			"Unexpected Solace Cloud API response",
			"The DMR request was accepted without an operation identifier, so its progress cannot be tracked.",
		)
		return
	}

	operationutils.WaitForOperationToComplete(&operationutils.OperationParams{
		Ctx:             ctx,
		ServiceId:       serviceId,
		OperationId:     *operation.Data.Id,
		R:               r.APIClient,
		PollingInterval: r.APIPollingInterval,
		Timeout:         dmrOperationTimeout,
		Diagnostics:     diags,
	})
}

// warnWhenServicePackageIgnoredDmr explains, at creation, that the service came up with a DMR
// state other than the one configured and that Terraform is about to change it. Service packages
// that always enable DMR ignore dmr_enabled when creating the service, so a configuration with
// dmr_enabled = false ends up with a cluster that the create-time update then deletes.
func warnWhenServicePackageIgnoredDmr(plan *ServiceResourceModel, observed types.Bool, diags *diag.Diagnostics) {
	if !dmrDiffersFromPlan(plan, observed) {
		return
	}
	if plan.DmrEnabled.ValueBool() {
		diags.AddAttributeWarning(
			path.Root("dmr_enabled"),
			"Service was created without DMR despite dmr_enabled = true",
			"Solace Cloud created the event broker service without a Dynamic Message Routing (DMR) cluster. "+
				"Terraform is now enabling DMR so the service matches the configuration.",
		)
		return
	}
	diags.AddAttributeWarning(
		path.Root("dmr_enabled"),
		"Service package created the service with DMR enabled",
		"dmr_enabled = false is set, but this service package always creates services with a Dynamic Message Routing (DMR) cluster. "+
			"Terraform is now disabling DMR and deleting that cluster so the service matches the configuration. "+
			"If you want to keep DMR on this service, remove dmr_enabled = false from the configuration.",
	)
}

// verifyDmrMatchesPlan reports a DMR change that was requested but is not reflected by the
// service. State always records what the service reports, so this is the only signal that the
// configuration and the service still disagree after an apply.
func verifyDmrMatchesPlan(plan *ServiceResourceModel, observed types.Bool, diags *diag.Diagnostics) {
	if !dmrDiffersFromPlan(plan, observed) {
		return
	}
	diags.AddAttributeError(
		path.Root("dmr_enabled"),
		"DMR configuration did not take effect",
		fmt.Sprintf("dmr_enabled = %t was applied, but the event broker service reports DMR %s. "+
			"The service is recorded as it is; run terraform plan to see the remaining change and apply again, "+
			"or check the service in Solace Cloud if the change keeps failing.",
			plan.DmrEnabled.ValueBool(), dmrStateWord(observed.ValueBool())),
	)
}

func dmrDiffersFromPlan(plan *ServiceResourceModel, observed types.Bool) bool {
	return util.IsKnown(plan.DmrEnabled) && util.IsKnown(observed) &&
		plan.DmrEnabled.ValueBool() != observed.ValueBool()
}

func dmrStateWord(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
