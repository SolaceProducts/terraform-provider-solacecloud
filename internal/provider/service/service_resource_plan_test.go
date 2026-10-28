package service

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// serviceObjectValue builds a full solacecloud_service object value with every attribute
// null except the overrides, so tests can drive ModifyPlan with realistic config/plan/state.
func serviceObjectValue(objectType tftypes.Object, overrides map[string]tftypes.Value) tftypes.Value {
	values := map[string]tftypes.Value{}
	for name, attributeType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	for name, value := range overrides {
		values[name] = value
	}
	return tftypes.NewValue(objectType, values)
}

// serviceAttributes returns the always-required attributes plus any overrides.
func serviceAttributes(overrides ...map[string]tftypes.Value) map[string]tftypes.Value {
	merged := map[string]tftypes.Value{
		"name":          tftypes.NewValue(tftypes.String, "dmr-plan-test"),
		"datacenter_id": tftypes.NewValue(tftypes.String, "gke-gcp-us-central1-a"),
	}
	for _, override := range overrides {
		for name, value := range override {
			merged[name] = value
		}
	}
	return merged
}

const (
	orphanClusterName   = "orphan-cluster"
	existingClusterName = "existing-cluster"
)

func withClusterName(name string) map[string]tftypes.Value {
	return map[string]tftypes.Value{"cluster_name": tftypes.NewValue(tftypes.String, name)}
}

func withDmrEnabled(value interface{}) map[string]tftypes.Value {
	return map[string]tftypes.Value{"dmr_enabled": tftypes.NewValue(tftypes.Bool, value)}
}

// TestModifyPlanRejectsClusterNameWithoutDmrOptIn covers the cross-attribute rule that a cluster
// name is only valid together with the DMR opt-in, both when creating and when updating.
func TestModifyPlanRejectsClusterNameWithoutDmrOptIn(t *testing.T) {
	ctx := context.Background()
	r := &ServiceResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	serviceSchema := schemaResp.Schema
	objectType := serviceSchema.Type().TerraformType(ctx).(tftypes.Object)

	nullState := tftypes.NewValue(objectType, nil)
	// An existing service created without DMR: no cluster name, dmr_enabled false.
	dmrOffState := serviceObjectValue(objectType, serviceAttributes(withDmrEnabled(false)))
	// An existing service that has DMR, as a configuration written before dmr_enabled existed
	// would have created it: a cluster name and no dmr_enabled.
	dmrOnState := serviceObjectValue(objectType, serviceAttributes(withDmrEnabled(true), withClusterName(existingClusterName)))

	cases := []struct {
		name        string
		config      map[string]tftypes.Value
		plan        map[string]tftypes.Value
		state       tftypes.Value
		destroy     bool
		expectError bool
	}{
		// Regression: a configuration written before dmr_enabled existed names the cluster and
		// says nothing about dmr_enabled. Against a service that has DMR it must keep applying
		// after the provider upgrade; the plan carries the service's dmr_enabled = true forward.
		{name: "update: pre-upgrade config naming the cluster of a DMR-on service is accepted",
			config: serviceAttributes(withClusterName(existingClusterName)),
			plan:   serviceAttributes(withClusterName(existingClusterName), withDmrEnabled(true)),
			state:  dmrOnState},
		{name: "update: disabling DMR while keeping cluster_name is rejected",
			config: serviceAttributes(withClusterName(existingClusterName), withDmrEnabled(false)),
			state:  dmrOnState, expectError: true},
		{name: "update: unrefreshed state leaves cluster_name to the API",
			config: serviceAttributes(withClusterName(existingClusterName)),
			plan:   serviceAttributes(withClusterName(existingClusterName), withDmrEnabled(tftypes.UnknownValue)),
			state:  serviceObjectValue(objectType, serviceAttributes(withClusterName(existingClusterName)))},
		{name: "create: cluster_name without dmr_enabled is rejected",
			config: serviceAttributes(withClusterName(orphanClusterName)), state: nullState, expectError: true},
		{name: "create: cluster_name with dmr_enabled false is rejected",
			config: serviceAttributes(withClusterName(orphanClusterName), withDmrEnabled(false)), state: nullState, expectError: true},
		{name: "create: cluster_name with dmr_enabled true is accepted",
			config: serviceAttributes(withClusterName(orphanClusterName), withDmrEnabled(true)), state: nullState},
		{name: "create: cluster_name with dmr_enabled unknown is left to the API",
			config: serviceAttributes(withClusterName(orphanClusterName), withDmrEnabled(tftypes.UnknownValue)), state: nullState},
		{name: "create: no cluster_name and no dmr_enabled is accepted",
			config: serviceAttributes(), state: nullState},
		// Regression: adding only cluster_name to an existing service without DMR used to pass
		// the plan and then fail the apply with an inconsistent-result error.
		{name: "update: adding cluster_name to a service without DMR is rejected",
			config: serviceAttributes(withClusterName(orphanClusterName)),
			plan:   serviceAttributes(withClusterName(orphanClusterName), withDmrEnabled(false)),
			state:  dmrOffState, expectError: true},
		{name: "update: adding cluster_name while enabling DMR is accepted",
			config: serviceAttributes(withClusterName(orphanClusterName), withDmrEnabled(true)), state: dmrOffState},
		{name: "destroy is ignored",
			config: serviceAttributes(withClusterName(orphanClusterName)), state: dmrOffState, destroy: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configValue := serviceObjectValue(objectType, tc.config)
			// The plan is the configuration unless the case models what the framework would
			// have filled in for computed attributes (the prior state's dmr_enabled).
			planValue := configValue
			if tc.plan != nil {
				planValue = serviceObjectValue(objectType, tc.plan)
			}
			if tc.destroy {
				planValue = tftypes.NewValue(objectType, nil)
			}
			req := resource.ModifyPlanRequest{
				Config: tfsdk.Config{Raw: configValue, Schema: serviceSchema},
				Plan:   tfsdk.Plan{Raw: planValue, Schema: serviceSchema},
				State:  tfsdk.State{Raw: tc.state, Schema: serviceSchema},
			}
			resp := &resource.ModifyPlanResponse{Plan: req.Plan}

			r.ModifyPlan(ctx, req, resp)

			if resp.Diagnostics.HasError() != tc.expectError {
				t.Fatalf("expected error=%t, got diagnostics: %v", tc.expectError, resp.Diagnostics)
			}
			if tc.expectError {
				errs := resp.Diagnostics.Errors()
				if len(errs) != 1 || errs[0].Summary() != errClusterNameRequiresDmrOptIn ||
					!strings.Contains(errs[0].Detail(), "cluster_name \"") {
					t.Errorf("unexpected error diagnostics: %v", errs)
				}
			}
		})
	}
}

// TestModifyPlanMarksClusterUnknownWhenDmrChanges verifies that toggling dmr_enabled on an
// existing service re-plans the computed cluster attributes. Without this the plan would carry
// the previous cluster forward and the apply would report an inconsistent result.
func TestModifyPlanMarksClusterUnknownWhenDmrChanges(t *testing.T) {
	ctx := context.Background()
	r := &ServiceResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	serviceSchema := schemaResp.Schema
	objectType := serviceSchema.Type().TerraformType(ctx).(tftypes.Object)

	dmrOffState := serviceObjectValue(objectType, serviceAttributes(withDmrEnabled(false)))
	dmrOnState := serviceObjectValue(objectType, serviceAttributes(
		withDmrEnabled(true), withClusterName(existingClusterName)))

	cases := []struct {
		name                     string
		state                    tftypes.Value
		config                   map[string]tftypes.Value
		expectClusterNameUnknown bool
	}{
		{name: "enabling DMR without a name recomputes the cluster name",
			state: dmrOffState, config: serviceAttributes(withDmrEnabled(true)),
			expectClusterNameUnknown: true},
		{name: "enabling DMR with a name keeps the configured name",
			state: dmrOffState, config: serviceAttributes(withDmrEnabled(true), withClusterName("chosen-name")),
			expectClusterNameUnknown: false},
		{name: "disabling DMR recomputes the cluster name",
			state: dmrOnState, config: serviceAttributes(withDmrEnabled(false)),
			expectClusterNameUnknown: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configValue := serviceObjectValue(objectType, tc.config)
			req := resource.ModifyPlanRequest{
				Config: tfsdk.Config{Raw: configValue, Schema: serviceSchema},
				Plan:   tfsdk.Plan{Raw: configValue, Schema: serviceSchema},
				State:  tfsdk.State{Raw: tc.state, Schema: serviceSchema},
			}
			resp := &resource.ModifyPlanResponse{Plan: req.Plan}

			r.ModifyPlan(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}

			var planClusterName types.String
			var planDmrCluster types.Object
			resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("cluster_name"), &planClusterName)...)
			resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("dmr_cluster"), &planDmrCluster)...)
			if resp.Diagnostics.HasError() {
				t.Fatalf("could not read back the plan: %v", resp.Diagnostics)
			}

			if planClusterName.IsUnknown() != tc.expectClusterNameUnknown {
				t.Errorf("expected cluster_name unknown=%t, got %s",
					tc.expectClusterNameUnknown, planClusterName)
			}
			// The cluster details are always recomputed: the cluster is created or deleted.
			if !planDmrCluster.IsUnknown() {
				t.Errorf("expected dmr_cluster to be unknown, got %s", planDmrCluster)
			}
		})
	}
}

// TestModifyPlanLeavesUnchangedDmrAlone verifies the common case: when dmr_enabled does not
// change, the plan is left exactly as the framework proposed it.
func TestModifyPlanLeavesUnchangedDmrAlone(t *testing.T) {
	ctx := context.Background()
	r := &ServiceResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	serviceSchema := schemaResp.Schema
	objectType := serviceSchema.Type().TerraformType(ctx).(tftypes.Object)

	state := serviceObjectValue(objectType, serviceAttributes(
		withDmrEnabled(true), withClusterName(existingClusterName)))
	configValue := serviceObjectValue(objectType, serviceAttributes(
		withDmrEnabled(true), withClusterName(existingClusterName)))

	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Raw: configValue, Schema: serviceSchema},
		Plan:   tfsdk.Plan{Raw: configValue, Schema: serviceSchema},
		State:  tfsdk.State{Raw: state, Schema: serviceSchema},
	}
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}

	r.ModifyPlan(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !resp.Plan.Raw.Equal(configValue) {
		t.Errorf("expected the plan to be unchanged, got %s", resp.Plan.Raw)
	}
}
