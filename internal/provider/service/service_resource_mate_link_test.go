package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jarcoal/httpmock"
)

const (
	methodPatch            = "PATCH"
	mateLinkTestServiceId  = "svc-mate-link-update"
	mateLinkUrl            = dmrTestBaseUrl + "/api/v2/missionControl/eventBrokerServices/" + mateLinkTestServiceId + "/mateLinkEncryption"
	mateLinkOperationUrl   = dmrTestBaseUrl + "/api/v2/missionControl/eventBrokerServices/" + mateLinkTestServiceId + "/operations/op-mlk"
	mateLinkAcceptedBody   = `{"data":{"id":"op-mlk","type":"operation","operationType":"updateMateLinkEncryption","resourceId":"` + mateLinkTestServiceId + `","resourceType":"service","status":"PENDING","error":null}}`
	mateLinkNotHaRefusal   = "Mate-link encryption can only be changed on a high-availability event broker service."
	mateLinkNotFoundReason = "Could not find event broker service with id svc-mate-link-update."
	mateLinkConflictReason = "Another operation is already in progress."
	// Written out rather than taken from the code under test.
	mateLinkChangeFailed = "Changing mate_link_encryption failed: "
)

func registerMateLinkOperation(status, errorBlock string) {
	httpmock.RegisterResponder("GET", mateLinkOperationUrl, internal.JsonResponder(200,
		`{"data":{"id":"op-mlk","type":"operation","operationType":"updateMateLinkEncryption","resourceId":"`+
			mateLinkTestServiceId+`","resourceType":"service","status":"`+status+`","error":`+errorBlock+`}}`))
}

func mateLinkModels(state, plan types.Bool) (*ServiceResourceModel, *ServiceResourceModel) {
	return &ServiceResourceModel{Id: types.StringValue(mateLinkTestServiceId), MateLinkEncryption: state},
		&ServiceResourceModel{MateLinkEncryption: plan}
}

func TestUpdateMateLinkEncryptionSendsNothingWhenUnchanged(t *testing.T) {
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

			state, plan := mateLinkModels(tc.state, tc.plan)
			diags := dmrTestResource(t).updateMateLinkEncryption(context.Background(), state, plan)

			if diags.HasError() {
				t.Fatalf(unexpectedDiags, diags)
			}
			if count := httpmock.GetTotalCallCount(); count != 0 {
				t.Errorf("expected no API calls, got %d", count)
			}
		})
	}
}

func TestUpdateMateLinkEncryptionSendsPlannedValue(t *testing.T) {
	cases := []struct {
		name  string
		state types.Bool
		plan  bool
	}{
		{name: "enable", state: types.BoolValue(false), plan: true},
		{name: "disable", state: types.BoolValue(true), plan: false},
		{name: "enable when the service reports no value", state: types.BoolNull(), plan: true},
		{name: "disable when the service reports no value", state: types.BoolNull(), plan: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			var sentBody map[string]interface{}
			var contentType string
			httpmock.RegisterResponder(methodPatch, mateLinkUrl, func(req *http.Request) (*http.Response, error) {
				contentType = req.Header.Get("Content-Type")
				raw, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("could not read the request body: %v", err)
				}
				if err := json.Unmarshal(raw, &sentBody); err != nil {
					t.Fatalf("request body is not JSON: %v", err)
				}
				return internal.JsonResponder(202, mateLinkAcceptedBody)(req)
			})
			registerMateLinkOperation(statusSucceeded, noOperationError)

			state, plan := mateLinkModels(tc.state, types.BoolValue(tc.plan))
			diags := dmrTestResource(t).updateMateLinkEncryption(context.Background(), state, plan)

			if diags.HasError() {
				t.Fatalf(unexpectedDiags, diags)
			}
			if calls := httpmock.GetCallCountInfo()[methodPatch+" "+mateLinkUrl]; calls != 1 {
				t.Fatalf("expected one mate-link request, got %d", calls)
			}
			if want := map[string]interface{}{"enabled": tc.plan}; !reflect.DeepEqual(sentBody, want) {
				t.Errorf("expected request body %v, got %v", want, sentBody)
			}
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %q", contentType)
			}
			if httpmock.GetCallCountInfo()["GET "+mateLinkOperationUrl] == 0 {
				t.Error("expected the operation to be polled")
			}
		})
	}
}

func TestUpdateMateLinkEncryptionSurfacesApiErrors(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		body       string
		wantText   []string
		notText    []string
	}{
		{name: "not a high-availability service", statusCode: 400,
			body:     `{"message":"` + mateLinkNotHaRefusal + `","errorId":"abc"}`,
			wantText: []string{mateLinkNotHaRefusal}},
		{name: "service not found", statusCode: 404,
			body:     `{"message":"` + mateLinkNotFoundReason + `","errorId":"abc"}`,
			wantText: []string{mateLinkNotFoundReason}},
		{name: "another operation in progress", statusCode: 409,
			body:     `{"message":"` + mateLinkConflictReason + `","errorId":"abc"}`,
			wantText: []string{mateLinkConflictReason},
			notText:  []string{"same name", "apply again"}},
		{name: "bad gateway", statusCode: 502,
			body:     `{"message":"Marketplace integration error.","errorId":"abc"}`,
			wantText: []string{"Marketplace integration error."}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponder(methodPatch, mateLinkUrl, internal.JsonResponder(tc.statusCode, tc.body))

			state, plan := mateLinkModels(types.BoolValue(false), types.BoolValue(true))
			diags := dmrTestResource(t).updateMateLinkEncryption(context.Background(), state, plan)

			if !diags.HasError() {
				t.Fatal("expected an error diagnostic")
			}
			for _, want := range tc.wantText {
				if !diagnosticsContain(diags.Errors(), want) {
					t.Errorf("expected a diagnostic mentioning %q, got %v", want, diags.Errors())
				}
			}
			for _, not := range tc.notText {
				if diagnosticsContain(diags.Errors(), not) {
					t.Errorf("expected no diagnostic mentioning %q, got %v", not, diags.Errors())
				}
			}
			for _, e := range diags.Errors() {
				withPath, ok := e.(diag.DiagnosticWithPath)
				if !ok || !withPath.Path().Equal(path.Root("mate_link_encryption")) {
					t.Errorf("expected the error on mate_link_encryption, got %v", e)
				}
				if !strings.HasPrefix(e.Detail(), mateLinkChangeFailed) {
					t.Errorf("expected the detail to start with %q, got %q", mateLinkChangeFailed, e.Detail())
				}
			}
			if calls := httpmock.GetCallCountInfo()["GET "+mateLinkOperationUrl]; calls != 0 {
				t.Errorf("expected no operation polling after a refused request, got %d", calls)
			}
		})
	}
}

func TestUpdateMateLinkEncryptionReportsTransportError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(methodPatch, mateLinkUrl, httpmock.NewErrorResponder(errors.New("connection reset by peer")))

	state, plan := mateLinkModels(types.BoolValue(false), types.BoolValue(true))
	diags := dmrTestResource(t).updateMateLinkEncryption(context.Background(), state, plan)

	if !diags.HasError() {
		t.Fatal("expected an error diagnostic for the failed request")
	}
	if !diagnosticsContain(diags.Errors(), "Could not update mate-link encryption") ||
		!diagnosticsContain(diags.Errors(), "connection reset by peer") {
		t.Errorf("expected a diagnostic naming the request and its cause, got %v", diags.Errors())
	}
	if calls := httpmock.GetCallCountInfo()["GET "+mateLinkOperationUrl]; calls != 0 {
		t.Errorf("expected no operation polling after a failed request, got %d", calls)
	}
}

func TestUpdateMateLinkEncryptionReportsFailedOperation(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(methodPatch, mateLinkUrl, internal.JsonResponder(202, mateLinkAcceptedBody))
	registerMateLinkOperation("FAILED", `{"message":"the broker rejected the mate-link change"}`)

	state, plan := mateLinkModels(types.BoolValue(true), types.BoolValue(false))
	diags := dmrTestResource(t).updateMateLinkEncryption(context.Background(), state, plan)

	if !diags.HasError() {
		t.Fatal("expected an error diagnostic for the failed operation")
	}
	if !diagnosticsContain(diags.Errors(), "the broker rejected the mate-link change") {
		t.Errorf("expected the operation's own message, got %v", diags.Errors())
	}
}

func TestUpdateMateLinkEncryptionRejectsResponseWithoutOperation(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(methodPatch, mateLinkUrl,
		internal.JsonResponder(202, `{"data":{"type":"operation","resourceType":"service"}}`))

	state, plan := mateLinkModels(types.BoolValue(false), types.BoolValue(true))
	diags := dmrTestResource(t).updateMateLinkEncryption(context.Background(), state, plan)

	if !diags.HasError() {
		t.Fatal("expected an error diagnostic when no operation identifier is returned")
	}
	assertSingleError(t, diags, "Unexpected Solace Cloud API response",
		"The mate-link encryption request was accepted without an operation identifier, so its progress cannot be tracked.")
}

func TestVerifyMateLinkMatchesPlan(t *testing.T) {
	cases := []struct {
		name      string
		plan      types.Bool
		observed  types.Bool
		wantError bool
		wantStart string
	}{
		{name: "matching true", plan: types.BoolValue(true), observed: types.BoolValue(true)},
		{name: "matching false", plan: types.BoolValue(false), observed: types.BoolValue(false)},
		{name: "unset mate_link_encryption never errors", plan: types.BoolUnknown(), observed: types.BoolValue(false)},
		{name: "service reports no value", plan: types.BoolValue(true), observed: types.BoolNull()},
		{name: "requested true but service reports disabled", plan: types.BoolValue(true), observed: types.BoolValue(false), wantError: true,
			wantStart: "mate_link_encryption = true was applied, but the event broker service reports mate-link encryption disabled. "},
		{name: "requested false but service reports enabled", plan: types.BoolValue(false), observed: types.BoolValue(true), wantError: true,
			wantStart: "mate_link_encryption = false was applied, but the event broker service reports mate-link encryption enabled. "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			verifyMateLinkMatchesPlan(&ServiceResourceModel{MateLinkEncryption: tc.plan}, tc.observed, &diags)

			if diags.HasError() != tc.wantError {
				t.Fatalf("expected error=%t, got %v", tc.wantError, diags)
			}
			if !tc.wantError {
				return
			}
			assertSingleError(t, diags, "Mate-link encryption configuration did not take effect", tc.wantStart+notTakenEffectAdvice)
			withPath, ok := diags.Errors()[0].(diag.DiagnosticWithPath)
			if !ok || !withPath.Path().Equal(path.Root("mate_link_encryption")) {
				t.Errorf("expected the error on mate_link_encryption, got %v", diags.Errors()[0])
			}
		})
	}
}
