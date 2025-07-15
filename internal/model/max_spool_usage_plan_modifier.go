package model

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MaxSpoolUsagePlanModifier is a plan modifier that updates the max_msg_spool_usage
// in the message_vpn when the max_spool_usage parameter in the service resource changes.
type MaxSpoolUsagePlanModifier struct{}

func (m MaxSpoolUsagePlanModifier) Description(ctx context.Context) string {
	return "Updates max_msg_spool_usage when max_spool_usage changes"
}

func (m MaxSpoolUsagePlanModifier) MarkdownDescription(ctx context.Context) string {
	return "Updates max_msg_spool_usage when max_spool_usage changes"
}

func (m MaxSpoolUsagePlanModifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() {
		return
	}

	// Do nothing if there is no state value.
	if req.State.Raw.IsNull() {
		return
	}

	// Get the max_spool_usage value from the parent service resource
	var maxSpoolUsage types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("max_spool_usage"), &maxSpoolUsage)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the planned max_spool_usage value from the parent service resource
	var plannedMaxSpoolUsage types.Int64
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("max_spool_usage"), &plannedMaxSpoolUsage)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If max_spool_usage has changed, update the planned value
	if !maxSpoolUsage.Equal(plannedMaxSpoolUsage) && !plannedMaxSpoolUsage.IsNull() && !plannedMaxSpoolUsage.IsUnknown() {
		resp.PlanValue = types.Int64Value(CalculateMsgVpnSpoolUsageFromMaxSpoolUsage(plannedMaxSpoolUsage.ValueInt64()))
	}
}

func CalculateMsgVpnSpoolUsageFromMaxSpoolUsage(maxSpoolUsage int64) int64 {
	return maxSpoolUsage * 1000
}

// NewMaxSpoolUsagePlanModifier creates a new MaxSpoolUsagePlanModifier
func NewMaxSpoolUsagePlanModifier() planmodifier.Int64 {
	return MaxSpoolUsagePlanModifier{}
}
