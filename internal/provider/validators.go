package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// ImmutableStringPlanModifier is a plan modifier that prevents changes to a string attribute
type ImmutableStringPlanModifier struct{}

// Description returns a plain text description of the plan modifier
func (m ImmutableStringPlanModifier) Description(ctx context.Context) string {
	return "prevents changes to attribute after resource creation"
}

// MarkdownDescription returns a markdown description of the plan modifier
func (m ImmutableStringPlanModifier) MarkdownDescription(ctx context.Context) string {
	return "**prevents** changes to attribute after resource creation"
}

// PlanModifyString prevents changes to the attribute after resource creation
func (m ImmutableStringPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	validateImmutableNotChanged(ctx, req, resp)
}

// NewImmutableStringPlanModifier returns a plan modifier that prevents changes to an attribute
// after resource creation. It will return an error if a change is attempted.
func NewImmutableStringPlanModifier() planmodifier.String {
	return ImmutableStringPlanModifier{}
}

func validateImmutableNotChanged(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// when the value is not in state, skip
	if req.StateValue.ValueString() == "" {
		return
	}

	// if plan value is undefined it can't change
	if resp.PlanValue.ValueString() == "" {
		return
	}

	if req.PlanValue.ValueString() != req.StateValue.ValueString() {
		// Add error diagnostic
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Immutable Attribute Change",
			fmt.Sprintf("You cannot change this attribute after resource creation. "+
				"State value: %q, Planned value: %q",
				req.StateValue.ValueString(), req.PlanValue.ValueString()),
		)

		// Force the plan to use the state value
		resp.PlanValue = req.StateValue
	}
}

// ImmutableStringPlanModifier is a plan modifier that prevents changes to a string attribute
type CustomRouterNamePlanModifier struct{}

// Description returns a plain text description of the plan modifier
func (m CustomRouterNamePlanModifier) Description(ctx context.Context) string {
	return "prevents changes to attribute after resource creation"
}

// MarkdownDescription returns a markdown description of the plan modifier
func (m CustomRouterNamePlanModifier) MarkdownDescription(ctx context.Context) string {
	return "**prevents** changes to attribute after resource creation"
}

// PlanModifyString prevents changes to the attribute after resource creation
func (m CustomRouterNamePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Do not replace on resource creation.
	if req.State.Raw.IsNull() {
		return
	}

	// Do not replace on resource destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	// Do not replace if the plan and state values are equal.
	if req.PlanValue.Equal(req.StateValue) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Immutable Attribute Change",
		fmt.Sprintf("You cannot change this attribute after resource creation. "+
			"State value: %q, Planned value: %q",
			req.StateValue.ValueString(), req.PlanValue.ValueString()),
	)
}

func NewCustomRouterNamePlanModifier() planmodifier.String {
	return CustomRouterNamePlanModifier{}
}
