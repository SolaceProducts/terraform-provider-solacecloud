package service

import (
	"context"
	"fmt"
	"terraform-provider-solacecloud/internal/model"
	"terraform-provider-solacecloud/internal/util"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const errClusterNameRequiresDmrOptIn = "cluster_name requires dmr_enabled = true"

// ModifyPlan enforces the DMR contract across attributes and keeps the plan honest when DMR
// is enabled or disabled on an existing service.
//
// A cluster_name is only meaningful for a service that has Dynamic Message Routing, so
// configuring one without dmr_enabled = true is rejected. Solace Cloud ignores a cluster name
// for a service without DMR, so accepting it would produce a service whose state no longer
// matches the configuration after apply.
//
// When dmr_enabled changes, the cluster attributes are marked unknown. Solace Cloud creates the
// cluster (generating a name when none is given) or deletes it, so their values are only known
// once the operation has run.
func (r *ServiceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to check when the resource is being destroyed.
	if req.Plan.Raw.IsNull() {
		return
	}

	var configClusterName types.String
	var configDmrEnabled types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("cluster_name"), &configClusterName)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("dmr_enabled"), &configDmrEnabled)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dmrEnabled := effectiveDmrEnabled(ctx, req, configDmrEnabled, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	validateClusterNameNeedsDmr(configClusterName, dmrEnabled, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	markClusterUnknownOnDmrChange(ctx, req, resp)
}

// effectiveDmrEnabled resolves what dmr_enabled means for this plan when the configuration does
// not set it. On create the service package default applies, which is DMR off. On update an
// unset dmr_enabled keeps the service's current DMR state, which the plan already carries; this
// is what lets a configuration written before dmr_enabled existed, naming the cluster of a
// service that has DMR, keep applying after the provider upgrade.
func effectiveDmrEnabled(ctx context.Context, req resource.ModifyPlanRequest, configDmrEnabled types.Bool, resp *resource.ModifyPlanResponse) types.Bool {
	if !configDmrEnabled.IsNull() {
		return configDmrEnabled
	}
	if req.State.Raw.IsNull() {
		return types.BoolValue(false)
	}
	var planDmrEnabled types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("dmr_enabled"), &planDmrEnabled)...)
	return planDmrEnabled
}

// validateClusterNameNeedsDmr rejects a configured cluster name on a service that does not have
// DMR, whether the service is being created or already exists. It reads cluster_name from the
// configuration rather than the plan, so removing it from a service that already has a cluster
// is not treated as a contradiction.
func validateClusterNameNeedsDmr(configClusterName types.String, dmrEnabled types.Bool, resp *resource.ModifyPlanResponse) {
	if !util.IsKnown(configClusterName) || configClusterName.ValueString() == "" {
		return
	}

	// A value that is only known at apply time (for example, from another resource, or an old
	// state file that has not been refreshed) is left for the API to validate.
	if !util.IsKnown(dmrEnabled) || dmrEnabled.ValueBool() {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("cluster_name"),
		errClusterNameRequiresDmrOptIn,
		fmt.Sprintf("cluster_name %q is set, but the service does not opt in to Dynamic Message Routing (DMR). "+
			"Solace Cloud ignores a cluster name for a service without DMR. "+
			"Set dmr_enabled = true to give the service a DMR cluster named %q, or remove cluster_name.",
			configClusterName.ValueString(), configClusterName.ValueString()),
	)
}

// markClusterUnknownOnDmrChange marks cluster_name and dmr_cluster unknown when dmr_enabled is
// changing on an existing service. Both are computed, so without this the plan would carry the
// previous values forward and the apply would report a result inconsistent with the plan.
func markClusterUnknownOnDmrChange(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Only an existing service can change; on create everything is already unknown.
	if req.State.Raw.IsNull() {
		return
	}

	var stateDmrEnabled, planDmrEnabled types.Bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("dmr_enabled"), &stateDmrEnabled)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("dmr_enabled"), &planDmrEnabled)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !util.IsKnown(stateDmrEnabled) || !util.IsKnown(planDmrEnabled) ||
		stateDmrEnabled.ValueBool() == planDmrEnabled.ValueBool() {
		return
	}

	var configClusterName types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("cluster_name"), &configClusterName)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Keep an explicitly configured cluster name in the plan; Solace Cloud creates the cluster
	// with that name. Otherwise the name is generated or removed, so it is not known yet.
	if !util.IsKnown(configClusterName) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("cluster_name"), types.StringUnknown())...)
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("dmr_cluster"), types.ObjectUnknown(model.DmrClusterInfoObjectType().AttrTypes))...)
}
