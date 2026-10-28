package service

import (
	"context"
	"strings"
	"terraform-provider-solacecloud/internal"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/missioncontrol"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jarcoal/httpmock"
)

// TestReadDataInternalDerivesDmrEnabledFromClusterPresence verifies that dmr_enabled is
// read from the service itself: true when the broker reports a DMR cluster, false when it
// reports none (a service created without DMR).
func TestReadDataInternalDerivesDmrEnabledFromClusterPresence(t *testing.T) {
	cases := []struct {
		name   string
		dmrOff bool
		want   bool
	}{
		{name: "service with a DMR cluster reads dmr_enabled true", dmrOff: false, want: true},
		{name: "service without a DMR cluster reads dmr_enabled false", dmrOff: true, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			params := internal.ConfigurableParams{
				ServiceId:    "svc-dmr-read",
				ServiceName:  "dmr-read-service",
				ServiceClass: "ENTERPRISE_1K_STANDALONE",
				DmrOff:       tc.dmrOff,
			}
			baseUrl := "https://api.solace.test"
			httpmock.RegisterResponder(
				"GET",
				`=~^https://api\.solace\.test/api/v2/missionControl/eventBrokerServices/svc-dmr-read`,
				internal.JsonResponder(200, internal.CreateGetServiceResponse(params)))

			mcClient, err := missioncontrol.NewClientWithResponses(baseUrl)
			if err != nil {
				t.Fatalf("failed to create mission control client: %v", err)
			}
			r := &ServiceResource{
				APIClient:          apiclient.NewRetryableClient(mcClient, 1, 0),
				APIPollingInterval: 1,
			}

			state := &ServiceResourceModel{Id: types.StringValue(params.ServiceId)}
			diags := r.readDataInternal(context.Background(), state, nil)
			if diags.HasError() {
				t.Fatalf("readDataInternal returned diagnostics errors: %v", diags)
			}

			if !state.DmrEnabled.Equal(types.BoolValue(tc.want)) {
				t.Errorf("expected dmr_enabled %t, got %s", tc.want, state.DmrEnabled)
			}
			if tc.dmrOff != state.DmrClusterInfo.IsNull() {
				t.Errorf("expected dmr_cluster null=%t, got %s", tc.dmrOff, state.DmrClusterInfo)
			}
		})
	}
}

// TestWarnWhenServicePackageIgnoredDmr covers the create-time notice that Terraform is about to
// change the DMR state a service package gave the new service.
func TestWarnWhenServicePackageIgnoredDmr(t *testing.T) {
	cases := []struct {
		name        string
		plan        types.Bool
		observed    types.Bool
		wantWarning bool
		wantText    string
	}{
		{name: "matching false", plan: types.BoolValue(false), observed: types.BoolValue(false)},
		{name: "matching true", plan: types.BoolValue(true), observed: types.BoolValue(true)},
		{name: "unset dmr_enabled never warns", plan: types.BoolUnknown(), observed: types.BoolValue(true)},
		{name: "legacy package ignored dmr_enabled = false", plan: types.BoolValue(false), observed: types.BoolValue(true),
			wantWarning: true, wantText: "disabling DMR and deleting that cluster"},
		{name: "service came up without DMR despite true", plan: types.BoolValue(true), observed: types.BoolValue(false),
			wantWarning: true, wantText: "enabling DMR"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			warnWhenServicePackageIgnoredDmr(&ServiceResourceModel{DmrEnabled: tc.plan}, tc.observed, &diags)

			if diags.HasError() {
				t.Fatalf("expected no errors, got %v", diags)
			}
			if (diags.WarningsCount() > 0) != tc.wantWarning {
				t.Fatalf("expected warning=%t, got %v", tc.wantWarning, diags)
			}
			if tc.wantWarning && !strings.Contains(diags.Warnings()[0].Detail(), tc.wantText) {
				t.Errorf("expected the warning to mention %q, got %q", tc.wantText, diags.Warnings()[0].Detail())
			}
		})
	}
}

// TestVerifyDmrMatchesPlan covers the post-apply check that the requested DMR state took effect.
func TestVerifyDmrMatchesPlan(t *testing.T) {
	cases := []struct {
		name      string
		plan      types.Bool
		observed  types.Bool
		wantError bool
	}{
		{name: "matching", plan: types.BoolValue(true), observed: types.BoolValue(true)},
		{name: "unset dmr_enabled never errors", plan: types.BoolUnknown(), observed: types.BoolValue(false)},
		{name: "requested true but service reports disabled", plan: types.BoolValue(true), observed: types.BoolValue(false), wantError: true},
		{name: "requested false but service reports enabled", plan: types.BoolValue(false), observed: types.BoolValue(true), wantError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			verifyDmrMatchesPlan(&ServiceResourceModel{DmrEnabled: tc.plan}, tc.observed, &diags)

			if diags.HasError() != tc.wantError {
				t.Fatalf("expected error=%t, got %v", tc.wantError, diags)
			}
			if tc.wantError && !strings.Contains(diags.Errors()[0].Detail(), "did not take effect") &&
				!strings.Contains(diags.Errors()[0].Summary(), "did not take effect") {
				t.Errorf("expected the error to say the change did not take effect, got %v", diags.Errors()[0])
			}
		})
	}
}
