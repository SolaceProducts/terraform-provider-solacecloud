package service

import (
	"context"
	"fmt"
	"net/http"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/internal/util"
	"terraform-provider-solacecloud/missioncontrol"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const mateLinkOperationTimeout = 15 * time.Minute

// Sends only when the planned value differs from state.
func (r *ServiceResource) updateMateLinkEncryption(ctx context.Context, state *ServiceResourceModel, plan *ServiceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if !util.IsKnown(plan.MateLinkEncryption) || state.MateLinkEncryption.IsUnknown() {
		return diags
	}
	if state.MateLinkEncryption.Equal(plan.MateLinkEncryption) {
		return diags
	}

	serviceId := state.Id.ValueString()
	enabled := plan.MateLinkEncryption.ValueBool()
	tflog.Info(ctx, fmt.Sprintf("Setting mate-link encryption to %t on service %s", enabled, serviceId))

	apiResp, err := r.APIClient.UpdateMateLinkEncryptionWithResponse(ctx, serviceId,
		missioncontrol.UpdateMateLinkEncryptionJSONRequestBody{Enabled: enabled})
	if err != nil {
		diags.AddError(
			"Error calling Solace Cloud API",
			"Could not update mate-link encryption on the event broker service, unexpected error: "+err.Error(),
		)
		return diags
	}

	var apiDiags diag.Diagnostics
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
	// Refusals are reported on mate_link_encryption, so a failed apply names the argument that caused it.
	if errorHandler.HandleError(&apiDiags) {
		for _, d := range apiDiags.Errors() {
			diags.AddAttributeError(path.Root("mate_link_encryption"), d.Summary(),
				"Changing mate_link_encryption failed: "+d.Detail())
		}
		return diags
	}

	r.waitForServiceOperation(ctx, serviceId, apiResp.JSON202, mateLinkOperationTimeout, "mate-link encryption", &diags)
	return diags
}

func verifyMateLinkMatchesPlan(plan *ServiceResourceModel, observed types.Bool, diags *diag.Diagnostics) {
	verifyBoolMatchesPlan("mate_link_encryption", "Mate-link encryption configuration did not take effect", "mate-link encryption",
		plan.MateLinkEncryption, observed, diags)
}
