package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"terraform-provider-solacecloud/internal"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/missioncontrol"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jarcoal/httpmock"
)

const (
	methodPost       = "POST"
	methodDelete     = "DELETE"
	unexpectedDiags  = "unexpected diagnostics: %v"
	statusSucceeded  = "SUCCEEDED"
	noOperationError = "null"
	dmrTestServiceId = "svc-dmr-update"
	dmrTestBaseUrl   = "https://api.solace.test"
	dmrClusterUrl    = dmrTestBaseUrl + "/api/v2/missionControl/eventBrokerServices/" + dmrTestServiceId + "/broker/cluster"
	dmrOperationUrl  = dmrTestBaseUrl + "/api/v2/missionControl/eventBrokerServices/" + dmrTestServiceId + "/operations/op-1"
)

// dmrOperationAccepted is the 202 body both DMR requests return.
const dmrOperationAccepted = `{"data":{"id":"op-1","type":"operation","operationType":"enableDmrCluster","resourceId":"` +
	dmrTestServiceId + `","resourceType":"service","status":"PENDING","error":null}}`

func dmrTestResource(t *testing.T) *ServiceResource {
	t.Helper()
	client, err := missioncontrol.NewClientWithResponses(dmrTestBaseUrl)
	if err != nil {
		t.Fatalf("failed to create mission control client: %v", err)
	}
	return &ServiceResource{
		APIClient:          apiclient.NewRetryableClient(client, 1, 0),
		APIPollingInterval: 0,
	}
}

// registerOperation registers the operation the provider polls after a DMR request.
func registerOperation(status string, errorBlock string) {
	httpmock.RegisterResponder("GET", dmrOperationUrl, internal.JsonResponder(200,
		`{"data":{"id":"op-1","type":"operation","operationType":"enableDmrCluster","resourceId":"`+
			dmrTestServiceId+`","resourceType":"service","status":"`+status+`","error":`+errorBlock+`}}`))
}

// TestUpdateDmrEnablementSendsNothingWhenUnchanged covers the common case: dmr_enabled carries
// the service's own state forward, so no DMR request may be sent.
func TestUpdateDmrEnablementSendsNothingWhenUnchanged(t *testing.T) {
	cases := []struct {
		name  string
		state types.Bool
		plan  types.Bool
	}{
		{name: "same value true", state: types.BoolValue(true), plan: types.BoolValue(true)},
		{name: "same value false", state: types.BoolValue(false), plan: types.BoolValue(false)},
		{name: "unknown plan", state: types.BoolValue(false), plan: types.BoolUnknown()},
		{name: "null plan", state: types.BoolValue(false), plan: types.BoolNull()},
		{name: "unknown state", state: types.BoolUnknown(), plan: types.BoolValue(true)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			r := dmrTestResource(t)
			state := &ServiceResourceModel{Id: types.StringValue(dmrTestServiceId), DmrEnabled: tc.state}
			plan := &ServiceResourceModel{DmrEnabled: tc.plan}

			diags := r.updateDmrEnablement(context.Background(), state, plan)

			if diags.HasError() {
				t.Fatalf(unexpectedDiags, diags)
			}
			if count := httpmock.GetTotalCallCount(); count != 0 {
				t.Errorf("expected no API calls, got %d", count)
			}
		})
	}
}

// TestUpdateDmrEnablementEnablesDmr verifies that turning DMR on posts to the cluster endpoint,
// sends a cluster name only when one is configured, and waits for the operation.
func TestUpdateDmrEnablementEnablesDmr(t *testing.T) {
	cases := []struct {
		name            string
		clusterName     types.String
		wantClusterName *string
	}{
		{name: "without a cluster name", clusterName: types.StringUnknown(), wantClusterName: nil},
		{name: "with an empty cluster name", clusterName: types.StringValue(""), wantClusterName: nil},
		{name: "with a cluster name", clusterName: types.StringValue("chosen"), wantClusterName: strPtrLocal("chosen")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			var sentBody map[string]interface{}
			httpmock.RegisterResponder(methodPost, dmrClusterUrl, func(req *http.Request) (*http.Response, error) {
				raw, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("could not read the request body: %v", err)
				}
				if err := json.Unmarshal(raw, &sentBody); err != nil {
					t.Fatalf("request body is not JSON: %v", err)
				}
				return internal.JsonResponder(202, dmrOperationAccepted)(req)
			})
			registerOperation(statusSucceeded, noOperationError)

			r := dmrTestResource(t)
			state := &ServiceResourceModel{Id: types.StringValue(dmrTestServiceId), DmrEnabled: types.BoolValue(false)}
			plan := &ServiceResourceModel{DmrEnabled: types.BoolValue(true), ClusterName: tc.clusterName}

			diags := r.updateDmrEnablement(context.Background(), state, plan)

			if diags.HasError() {
				t.Fatalf(unexpectedDiags, diags)
			}
			if sentBody == nil {
				t.Fatal("the enable DMR request was not sent")
			}

			value, present := sentBody["clusterName"]
			if tc.wantClusterName == nil {
				if present {
					t.Errorf("expected clusterName to be omitted, got %v", value)
				}
			} else if value != *tc.wantClusterName {
				t.Errorf("expected clusterName %q, got %v", *tc.wantClusterName, value)
			}
			if calls := httpmock.GetCallCountInfo()["GET "+dmrOperationUrl]; calls == 0 {
				t.Error("expected the operation to be polled")
			}
		})
	}
}

// TestUpdateDmrEnablementDisablesDmr verifies that turning DMR off deletes the cluster and waits
// for the operation.
func TestUpdateDmrEnablementDisablesDmr(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(methodDelete, dmrClusterUrl, internal.JsonResponder(202, dmrOperationAccepted))
	registerOperation(statusSucceeded, noOperationError)

	r := dmrTestResource(t)
	state := &ServiceResourceModel{Id: types.StringValue(dmrTestServiceId), DmrEnabled: types.BoolValue(true)}
	plan := &ServiceResourceModel{DmrEnabled: types.BoolValue(false)}

	diags := r.updateDmrEnablement(context.Background(), state, plan)

	if diags.HasError() {
		t.Fatalf(unexpectedDiags, diags)
	}
	if calls := httpmock.GetCallCountInfo()["DELETE "+dmrClusterUrl]; calls != 1 {
		t.Errorf("expected one delete cluster call, got %d", calls)
	}
	if calls := httpmock.GetCallCountInfo()["GET "+dmrOperationUrl]; calls == 0 {
		t.Error("expected the operation to be polled")
	}
}

// TestUpdateDmrEnablementSurfacesApiErrors verifies that a refusal from Solace Cloud becomes an
// error diagnostic naming the reason, rather than a silent no-op.
func TestUpdateDmrEnablementSurfacesApiErrors(t *testing.T) {
	const meshRefusal = "DMR cannot be disabled on a service that is part of an event mesh."
	const alreadyEnabled = "DMR is already enabled on this event broker service"

	cases := []struct {
		name       string
		method     string
		body       string
		statusCode int
		state      bool
		plan       bool
		wantText   string
	}{
		{name: "enable refused", method: methodPost, statusCode: 400, state: false, plan: true,
			body:     `{"message":"` + alreadyEnabled + `","errorId":"abc"}`,
			wantText: alreadyEnabled},
		{name: "disable refused", method: methodDelete, statusCode: 400, state: true, plan: false,
			body:     `{"message":"` + meshRefusal + `","errorId":"abc"}`,
			wantText: "event mesh"},
		{name: "enable not found", method: methodPost, statusCode: 404, state: false, plan: true,
			body:     `{"message":"Could not find event broker service with id svc-dmr-update","errorId":"abc"}`,
			wantText: "Could not find"},
		{name: "disable conflicts with another operation", method: methodDelete, statusCode: 409, state: true, plan: false,
			body:     `{"message":"Another operation is already in progress.","errorId":"abc"}`,
			wantText: "409"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponder(tc.method, dmrClusterUrl, internal.JsonResponder(tc.statusCode, tc.body))

			r := dmrTestResource(t)
			state := &ServiceResourceModel{Id: types.StringValue(dmrTestServiceId), DmrEnabled: types.BoolValue(tc.state)}
			plan := &ServiceResourceModel{DmrEnabled: types.BoolValue(tc.plan)}

			diags := r.updateDmrEnablement(context.Background(), state, plan)

			if !diags.HasError() {
				t.Fatal("expected an error diagnostic")
			}
			if !diagnosticsContain(diags.Errors(), tc.wantText) {
				t.Errorf("expected a diagnostic mentioning %q, got %v", tc.wantText, diags.Errors())
			}
		})
	}
}

// TestUpdateDmrEnablementReportsFailedOperation verifies that an operation that ends in FAILED is
// reported, so a broker reconfiguration that did not take effect is never recorded as success.
func TestUpdateDmrEnablementReportsFailedOperation(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(methodPost, dmrClusterUrl, internal.JsonResponder(202, dmrOperationAccepted))
	registerOperation("FAILED", `{"message":"the broker rejected the DMR cluster"}`)

	r := dmrTestResource(t)
	state := &ServiceResourceModel{Id: types.StringValue(dmrTestServiceId), DmrEnabled: types.BoolValue(false)}
	plan := &ServiceResourceModel{DmrEnabled: types.BoolValue(true)}

	diags := r.updateDmrEnablement(context.Background(), state, plan)

	if !diags.HasError() {
		t.Fatal("expected an error diagnostic for the failed operation")
	}
	if !diagnosticsContain(diags.Errors(), "the broker rejected the DMR cluster") {
		t.Errorf("expected the operation's own message, got %v", diags.Errors())
	}
}

// TestUpdateDmrEnablementRejectsResponseWithoutOperation verifies that an accepted request with
// no operation identifier is reported rather than silently treated as complete.
func TestUpdateDmrEnablementRejectsResponseWithoutOperation(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(methodPost, dmrClusterUrl,
		internal.JsonResponder(202, `{"data":{"type":"operation","resourceType":"service"}}`))

	r := dmrTestResource(t)
	state := &ServiceResourceModel{Id: types.StringValue(dmrTestServiceId), DmrEnabled: types.BoolValue(false)}
	plan := &ServiceResourceModel{DmrEnabled: types.BoolValue(true)}

	diags := r.updateDmrEnablement(context.Background(), state, plan)

	if !diags.HasError() {
		t.Fatal("expected an error diagnostic when no operation identifier is returned")
	}
	if !diagnosticsContain(diags.Errors(), "operation identifier") {
		t.Errorf("expected a diagnostic about the missing operation identifier, got %v", diags.Errors())
	}
}

func strPtrLocal(s string) *string { return &s }

func diagnosticsContain(errs []diag.Diagnostic, text string) bool {
	for _, e := range errs {
		if strings.Contains(e.Summary(), text) || strings.Contains(e.Detail(), text) {
			return true
		}
	}
	return false
}
