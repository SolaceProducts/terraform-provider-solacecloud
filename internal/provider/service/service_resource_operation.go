package service

import (
	"context"
	"fmt"
	"terraform-provider-solacecloud/internal/provider/operationutils"
	"terraform-provider-solacecloud/internal/util"
	"terraform-provider-solacecloud/missioncontrol"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// waitForServiceOperation polls the operation returned by an accepted request until it
// finishes, so the value written to state reflects the service's settled configuration.
func (r *ServiceResource) waitForServiceOperation(ctx context.Context, serviceId string, operation *missioncontrol.OperationResponse,
	timeout time.Duration, label string, diags *diag.Diagnostics) {
	if operation == nil || operation.Data.Id == nil {
		diags.AddError(
			"Unexpected Solace Cloud API response",
			"The "+label+" request was accepted without an operation identifier, so its progress cannot be tracked.",
		)
		return
	}

	operationutils.WaitForOperationToComplete(&operationutils.OperationParams{
		Ctx:             ctx,
		ServiceId:       serviceId,
		OperationId:     *operation.Data.Id,
		R:               r.APIClient,
		PollingInterval: r.APIPollingInterval,
		Timeout:         timeout,
		Diagnostics:     diags,
	})
}

func verifyBoolMatchesPlan(attribute, summary, feature string, planned, observed types.Bool, diags *diag.Diagnostics) {
	if !boolDiffersFromPlan(planned, observed) {
		return
	}
	diags.AddAttributeError(
		path.Root(attribute),
		summary,
		fmt.Sprintf("%s = %t was applied, but the event broker service reports %s %s. "+
			"The service is recorded as it is; run terraform plan to see the remaining change and apply again, "+
			"or check the service in Solace Cloud if the change keeps failing.",
			attribute, planned.ValueBool(), feature, stateWord(observed.ValueBool())),
	)
}

func boolDiffersFromPlan(planned, observed types.Bool) bool {
	return util.IsKnown(planned) && util.IsKnown(observed) && planned.ValueBool() != observed.ValueBool()
}

func stateWord(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
