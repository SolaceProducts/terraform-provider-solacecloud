package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"terraform-provider-solacecloud/internal"
	"terraform-provider-solacecloud/internal/model"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/internal/provider/connectionendpoint"
	"terraform-provider-solacecloud/missioncontrol"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/jarcoal/httpmock"
)

const (
	pwTestServiceId = "svc-password-rotation"
	pwTestBaseUrl   = "https://api.solace.test"
	servicesPath    = "/api/v2/missionControl/eventBrokerServices/"
	pwServiceUrl    = pwTestBaseUrl + servicesPath + pwTestServiceId
	pwResetUrl      = pwServiceUrl + "/resetBrokerPasswords"
	pwOperationUrl  = pwServiceUrl + "/operations/op-1"
	pwOperationBody = `{"data":{"id":"op-1","type":"operation","operationType":"resetBrokerPasswords","resourceId":"` +
		pwTestServiceId + `","resourceType":"service","status":"PENDING","error":null}}`
)

func pwTestResource(t *testing.T) *ServiceResource {
	t.Helper()
	client, err := missioncontrol.NewClientWithResponses(pwTestBaseUrl)
	if err != nil {
		t.Fatalf("failed to create mission control client: %v", err)
	}
	return &ServiceResource{
		APIClient:          apiclient.NewRetryableClient(client, 1, 0),
		APIPollingInterval: 0,
	}
}

func pwRegisterOperation(status, errorBlock string) {
	httpmock.RegisterResponder("GET", pwOperationUrl, internal.JsonResponder(200,
		`{"data":{"id":"op-1","type":"operation","operationType":"resetBrokerPasswords","resourceId":"`+
			pwTestServiceId+`","resourceType":"service","status":"`+status+`","error":`+errorBlock+`}}`))
}

func rotationObject(t *testing.T, trigger string, users ...string) types.Object {
	t.Helper()
	elems := make([]attr.Value, 0, len(users))
	for _, u := range users {
		elems = append(elems, types.StringValue(u))
	}
	obj, diags := types.ObjectValue(brokerPasswordRotationAttrTypes(), map[string]attr.Value{
		"users":   types.SetValueMust(types.StringType, elems),
		"trigger": types.StringValue(trigger),
	})
	if diags.HasError() {
		t.Fatalf("could not build rotation object: %v", diags)
	}
	return obj
}

func nullRotation() types.Object { return types.ObjectNull(brokerPasswordRotationAttrTypes()) }

func rotationStates(state, plan types.Object) (*ServiceResourceModel, *ServiceResourceModel) {
	return &ServiceResourceModel{Id: types.StringValue(pwTestServiceId), BrokerPasswordRotation: state},
		&ServiceResourceModel{BrokerPasswordRotation: plan}
}

func TestRotateBrokerPasswordsSendsNothingWhenNotRotating(t *testing.T) {
	cases := []struct {
		name        string
		state, plan func(t *testing.T) types.Object
	}{
		{"no block", func(t *testing.T) types.Object { return nullRotation() }, func(t *testing.T) types.Object { return nullRotation() }},
		{"same trigger", func(t *testing.T) types.Object { return rotationObject(t, "v1", brokerUserViewer) },
			func(t *testing.T) types.Object { return rotationObject(t, "v1", brokerUserViewer) }},
		{"users edited alone", func(t *testing.T) types.Object { return rotationObject(t, "v1", brokerUserViewer) },
			func(t *testing.T) types.Object { return rotationObject(t, "v1", brokerUserViewer, brokerUserEditor) }},
		{"block removed", func(t *testing.T) types.Object { return rotationObject(t, "v1", brokerUserViewer) },
			func(t *testing.T) types.Object { return nullRotation() }},
		{"block added to existing service", func(t *testing.T) types.Object { return nullRotation() },
			func(t *testing.T) types.Object { return rotationObject(t, "v1", brokerUserViewer) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			state, plan := rotationStates(tc.state(t), tc.plan(t))
			_, diags := pwTestResource(t).rotateBrokerPasswords(context.Background(), state, plan)

			if diags.HasError() {
				t.Fatalf(unexpectedDiags, diags)
			}
			if count := httpmock.GetTotalCallCount(); count != 0 {
				t.Errorf("expected no API calls, got %d", count)
			}
		})
	}
}

func TestRotateBrokerPasswordsSendsUsersAndWaits(t *testing.T) {
	cases := []struct {
		name  string
		users []string
		want  []string
	}{
		{"one user", []string{brokerUserViewer}, []string{brokerUserViewer}},
		{"several users", []string{brokerUserViewer, brokerUserManager, brokerUserEditor},
			[]string{brokerUserEditor, brokerUserManager, brokerUserViewer}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			var sent struct {
				Users []struct {
					Username string `json:"username"`
				} `json:"users"`
			}
			httpmock.RegisterResponder(methodPost, pwResetUrl, func(req *http.Request) (*http.Response, error) {
				raw, _ := io.ReadAll(req.Body)
				if err := json.Unmarshal(raw, &sent); err != nil {
					t.Fatalf("request body is not JSON: %v", err)
				}
				return internal.JsonResponder(202, pwOperationBody)(req)
			})
			pwRegisterOperation(statusSucceeded, noOperationError)

			state, plan := rotationStates(rotationObject(t, "v1", brokerUserViewer), rotationObject(t, "v2", tc.users...))
			_, diags := pwTestResource(t).rotateBrokerPasswords(context.Background(), state, plan)

			if diags.HasError() {
				t.Fatalf(unexpectedDiags, diags)
			}
			// The order of users in the request is not significant, so compare as sets. A
			// duplicate or a missing user still fails.
			got := map[string]int{}
			for _, u := range sent.Users {
				got[u.Username]++
			}
			want := map[string]int{}
			for _, u := range tc.want {
				want[u]++
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("expected users %v, sent %v", tc.want, sent.Users)
			}
			if calls := httpmock.GetCallCountInfo()["POST "+pwResetUrl]; calls != 1 {
				t.Errorf("expected exactly one reset request, got %d", calls)
			}
			if polls := httpmock.GetCallCountInfo()["GET "+pwOperationUrl]; polls < 1 {
				t.Errorf("expected the operation to be polled, got %d polls", polls)
			}
		})
	}
}

func TestRotateBrokerPasswordsSurfacesApiErrors(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		body       string
		wantText   []string
	}{
		{"unknown or absent user", 400,
			`{"message":"the 'mc-manager' user does not exist on this event broker service","errorId":"abc"}`,
			[]string{"does not exist"}},
		// maas-core BrokerPasswordResetV2Manager rejects a shared service with this 400.
		{"shared event broker service", 400,
			`{"message":"You cannot reset Broker Manager passwords on event broker service x because it is a shared event broker service.","errorId":"abc",` +
				`"validationDetails":{"serviceId":["must not be a shared event broker service"]}}`,
			[]string{"because it is a shared event broker service", "serviceId: must not be a shared event broker service"}},
		{"unauthorized", 401, `{"message":"Your permissions have changed.","errorId":"abc"}`, []string{"Authentication Failed"}},
		{"forbidden", 403, `{"message":"You do not have permissions to perform this operation.","errorId":"abc"}`,
			[]string{"You do not have permissions"}},
		{"service not found", 404, `{"message":"Could not find event broker service with id x","errorId":"abc"}`,
			[]string{"Could not find"}},
		{"another operation in progress", 409,
			`{"message":"Job 4blh8jqu78r is still in progress.","errorId":"abc"}`,
			[]string{"Resource Conflict", "Job 4blh8jqu78r is still in progress."}},
		{"gateway timeout", 503,
			`{"error":"Gateway Timeout","message":"Timeout waiting 28000 ms for...."}`,
			[]string{"may have started", "check the event broker service", "Timeout waiting 28000 ms"}},
		{"conflict without a body message", 409, `{}`, []string{"Resource Conflict", "HTTP 409"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			httpmock.RegisterResponder(methodPost, pwResetUrl, internal.JsonResponder(tc.statusCode, tc.body))

			state, plan := rotationStates(rotationObject(t, "v1", brokerUserViewer), rotationObject(t, "v2", brokerUserManager))
			_, diags := pwTestResource(t).rotateBrokerPasswords(context.Background(), state, plan)

			if !diags.HasError() {
				t.Fatal("expected an error diagnostic")
			}
			for _, text := range tc.wantText {
				if !diagnosticsContain(diags.Errors(), text) && !diagnosticsContainFold(diags.Errors(), text) {
					t.Errorf("expected a diagnostic mentioning %q, got %v", text, diags.Errors())
				}
			}
			// The reset must never be replayed, whatever the status.
			if calls := httpmock.GetCallCountInfo()["POST "+pwResetUrl]; calls != 1 {
				t.Errorf("expected exactly one reset request, got %d", calls)
			}
			if polls := httpmock.GetCallCountInfo()["GET "+pwOperationUrl]; polls != 0 {
				t.Errorf("a failed request must not be polled, got %d polls", polls)
			}
		})
	}
}

func TestRotateBrokerPasswordsReportsFailedOperation(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(methodPost, pwResetUrl, internal.JsonResponder(202, pwOperationBody))
	pwRegisterOperation("FAILED", `{"message":"Could not reset the password on the broker."}`)

	state, plan := rotationStates(rotationObject(t, "v1", brokerUserViewer), rotationObject(t, "v2", brokerUserViewer))
	_, diags := pwTestResource(t).rotateBrokerPasswords(context.Background(), state, plan)

	if !diags.HasError() {
		t.Fatal("expected an error diagnostic for a failed operation")
	}
}

func TestRotateBrokerPasswordsRejectsResponseWithoutOperation(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(methodPost, pwResetUrl, internal.JsonResponder(202, `{}`))

	state, plan := rotationStates(rotationObject(t, "v1", brokerUserViewer), rotationObject(t, "v2", brokerUserViewer))
	_, diags := pwTestResource(t).rotateBrokerPasswords(context.Background(), state, plan)

	if !diags.HasError() {
		t.Fatal("expected an error when the 202 carries no operation id")
	}
}

// diagnosticsContainFold is diagnosticsContain ignoring case, for wording that starts a sentence.
func diagnosticsContainFold(errs []diag.Diagnostic, text string) bool {
	for _, e := range errs {
		if strings.Contains(strings.ToLower(e.Summary()+" "+e.Detail()), strings.ToLower(text)) {
			return true
		}
	}
	return false
}

// Create calls updateInternal with the planned block already in state, so it must send nothing;
// a trigger change must send exactly one reset.
func TestUpdateInternalRotatesOnlyOnTriggerChange(t *testing.T) {
	cases := []struct {
		name      string
		state     func(t *testing.T) types.Object
		plan      func(t *testing.T) types.Object
		wantReset int
	}{
		{"create: state already carries the planned block",
			func(t *testing.T) types.Object { return rotationObject(t, "v1", brokerUserViewer) },
			func(t *testing.T) types.Object { return rotationObject(t, "v1", brokerUserViewer) }, 0},
		{"update: trigger changed",
			func(t *testing.T) types.Object { return rotationObject(t, "v1", brokerUserViewer) },
			func(t *testing.T) types.Object { return rotationObject(t, "v2", brokerUserViewer) }, 1},
		{"update: no block configured",
			func(t *testing.T) types.Object { return nullRotation() },
			func(t *testing.T) types.Object { return nullRotation() }, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			httpmock.RegisterResponder(methodPost, pwResetUrl, internal.JsonResponder(202, pwOperationBody))
			pwRegisterOperation(statusSucceeded, noOperationError)

			state, plan := rotationStates(tc.state(t), tc.plan(t))
			_, diags := pwTestResource(t).updateInternal(context.Background(), state, plan)

			if diags.HasError() {
				t.Fatalf(unexpectedDiags, diags)
			}
			if calls := httpmock.GetCallCountInfo()["POST "+pwResetUrl]; calls != tc.wantReset {
				t.Errorf("expected %d reset requests, got %d", tc.wantReset, calls)
			}
		})
	}
}

// A step that fails after a rotation would rotate again on the next apply, so the endpoint
// update must run first and stop the rotation when it fails.
func TestUpdateInternalDoesNotRotateWhenEndpointUpdateFails(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	endpointUrl := pwServiceUrl + "/connectionEndpoints/ep-1"
	failure := internal.JsonResponder(500, `{"message":"boom"}`)
	httpmock.RegisterResponder("PATCH", endpointUrl, failure)
	httpmock.RegisterResponder("PUT", endpointUrl, failure)
	httpmock.RegisterResponder(methodPost, pwResetUrl, internal.JsonResponder(202, pwOperationBody))
	pwRegisterOperation(statusSucceeded, noOperationError)

	endpoint := func(description string) types.Object {
		obj, diags := model.NestedConnectionEndpointModel{
			Id:             types.StringValue("ep-1"),
			ServiceId:      types.StringValue(pwTestServiceId),
			Name:           types.StringValue("default"),
			Description:    types.StringValue(description),
			AccessType:     types.StringValue("public"),
			K8SServiceType: types.StringNull(),
			K8SServiceId:   types.StringNull(),
			Ports:          types.ObjectNull(model.EndpointProtocolSchema().GetType().(types.ObjectType).AttrTypes),
		}.ToObjectValue()
		if diags.HasError() {
			t.Fatalf("could not build endpoint: %v", diags)
		}
		return obj
	}

	state, plan := rotationStates(rotationObject(t, "v1", brokerUserViewer), rotationObject(t, "v2", brokerUserViewer))
	state.ConnectionEndpoint = endpoint("old")
	plan.ConnectionEndpoint = endpoint("new")

	r := pwTestResource(t)
	r.endpointManager = connectionendpoint.NewEndpointManager(r.APIClient, 0)
	_, diags := r.updateInternal(context.Background(), state, plan)

	if !diags.HasError() {
		t.Fatal("expected the endpoint update failure to be reported")
	}
	if calls := httpmock.GetCallCountInfo()["POST "+pwResetUrl]; calls != 0 {
		t.Errorf("expected no reset request after a failed endpoint update, got %d", calls)
	}
}

// A transport failure must fail the apply with a message that warns the reset may have reached
// Solace Cloud, and must not be replayed.
func TestRotateBrokerPasswordsReportsTransportError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(methodPost, pwResetUrl, httpmock.NewErrorResponder(io.ErrUnexpectedEOF))

	state, plan := rotationStates(rotationObject(t, "v1", brokerUserViewer), rotationObject(t, "v2", brokerUserViewer))
	_, diags := pwTestResource(t).rotateBrokerPasswords(context.Background(), state, plan)

	if !diags.HasError() {
		t.Fatal("expected an error diagnostic for a transport failure")
	}
	if !diagnosticsContain(diags.Errors(), "check whether the passwords changed") {
		t.Errorf("expected the error to warn that the request may have been received, got %v", diags.Errors())
	}
	if calls := httpmock.GetCallCountInfo()["POST "+pwResetUrl]; calls != 1 {
		t.Errorf("expected exactly one reset request, got %d", calls)
	}
}

// The schema must reject input the API would refuse, at plan time and before any request.
func TestBrokerPasswordRotationSchemaValidation(t *testing.T) {
	ctx := context.Background()
	nested := brokerPasswordRotationSchema().(schema.SingleNestedAttribute)
	usersAttr := nested.Attributes["users"].(schema.SetAttribute)
	triggerAttr := nested.Attributes["trigger"].(schema.StringAttribute)

	validateUsers := func(users ...string) bool {
		elems := make([]attr.Value, 0, len(users))
		for _, u := range users {
			elems = append(elems, types.StringValue(u))
		}
		value := types.SetValueMust(types.StringType, elems)
		failed := false
		for _, v := range usersAttr.Validators {
			resp := &validator.SetResponse{}
			v.ValidateSet(ctx, validator.SetRequest{Path: path.Root("users"), ConfigValue: value}, resp)
			failed = failed || resp.Diagnostics.HasError()
		}
		return failed
	}
	validateTrigger := func(trigger string) bool {
		failed := false
		for _, v := range triggerAttr.Validators {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, validator.StringRequest{Path: path.Root("trigger"), ConfigValue: types.StringValue(trigger)}, resp)
			failed = failed || resp.Diagnostics.HasError()
		}
		return failed
	}

	cases := []struct {
		name       string
		failed     bool
		wantFailed bool
	}{
		{"all three valid users", validateUsers(brokerUserEditor, brokerUserManager, brokerUserViewer), false},
		{"one valid user", validateUsers(brokerUserViewer), false},
		{"empty users", validateUsers(), true},
		{"unknown username", validateUsers("admin"), true},
		{"valid and unknown username", validateUsers(brokerUserViewer, "vpnAdmin"), true},
		{"non-empty trigger", validateTrigger("2026-11-02"), false},
		{"empty trigger", validateTrigger(""), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.failed != tc.wantFailed {
				t.Errorf("expected validation failure=%t, got %t", tc.wantFailed, tc.failed)
			}
		})
	}
}

// A failed rotation must leave the old trigger and passwords in state. Calls Update directly:
// the acceptance harness doesn't carry a failed apply's state forward.
func TestUpdateKeepsOldStateWhenRotationFails(t *testing.T) {
	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{ServiceClass: "ENTERPRISE_1K_STANDALONE", ServiceName: "rotation-failure"}
	instance.Init(params)
	if !instance.IsMocked() {
		t.Skip("uses mocked API responses; unset SOLACE_BASE_URL to run it")
	}
	defer httpmock.DeactivateAndReset()
	const serviceId = "6q1p55o6ovr" // the id the default mocks serve
	// The re-read still succeeds, so an Update that saved state after the failure would be caught.
	instance.SetupPasswordResetMocks(params, 403, `{"message":"You do not have permissions to perform this operation.","errorId":"abc"}`)

	client, err := missioncontrol.NewClientWithResponses(instance.GetBaseURL())
	if err != nil {
		t.Fatalf("failed to create mission control client: %v", err)
	}
	r := &ServiceResource{APIClient: apiclient.NewRetryableClient(client, 1, 0)}
	r.endpointManager = connectionendpoint.NewEndpointManager(r.APIClient, 0)

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	serviceSchema := schemaResp.Schema
	objectType := serviceSchema.Type().TerraformType(ctx).(tftypes.Object)

	build := func(trigger string) tftypes.Value {
		return serviceObjectValue(objectType, serviceAttributes(map[string]tftypes.Value{
			"id":                       tftypes.NewValue(tftypes.String, serviceId),
			"broker_password_rotation": rotationValue(objectType, strPtr(trigger), brokerUserViewer),
			"message_vpn":              messageVpnValue(objectType),
		}))
	}
	state := tfsdk.State{Raw: build("v1"), Schema: serviceSchema}
	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Raw: build("v2"), Schema: serviceSchema},
		State: state,
	}
	// The framework starts the response state as a copy of the prior state.
	resp := &resource.UpdateResponse{State: tfsdk.State{Raw: state.Raw, Schema: serviceSchema}}

	r.Update(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected the refused rotation to fail the update")
	}
	if calls := httpmock.GetCallCountInfo()["POST "+instance.GetBaseURL()+servicesPath+serviceId+"/resetBrokerPasswords"]; calls != 1 {
		t.Fatalf("expected exactly one reset request, got %d", calls)
	}

	var trigger types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("broker_password_rotation").AtName("trigger"), &trigger)...)
	var password types.String
	viewerPassword := path.Root("message_vpn").AtName(credentialAttributeByUser[brokerUserViewer]).AtName("password")
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, viewerPassword, &password)...)

	if trigger.ValueString() != "v1" {
		t.Errorf("a failed rotation must keep the old trigger so the next apply retries it, state has %q", trigger.ValueString())
	}
	if password.ValueString() != "old-"+credentialAttributeByUser[brokerUserViewer] {
		t.Errorf("a failed rotation must keep the old viewer password in state, which is now %q", password.ValueString())
	}
}

// After a successful rotation the new trigger must be stored even if the re-read fails, or the
// next apply rotates again.
func TestUpdateRecordsRotationWhenReadBackFails(t *testing.T) {
	instance := internal.NewTestInstance()
	params := internal.ConfigurableParams{ServiceClass: "ENTERPRISE_1K_STANDALONE", ServiceName: "rotation-readback-failure"}
	instance.Init(params)
	if !instance.IsMocked() {
		t.Skip("uses mocked API responses; unset SOLACE_BASE_URL to run it")
	}
	defer httpmock.DeactivateAndReset()
	const serviceId = "6q1p55o6ovr" // the id the default mocks serve
	serviceUrl := instance.GetBaseURL() + servicesPath + serviceId
	resetUrl := serviceUrl + "/resetBrokerPasswords"

	// The reset is accepted and its operation succeeds, but from then on the service cannot be
	// read back.
	httpmock.RegisterResponder("GET", serviceUrl+"/operations/op-1", internal.JsonResponder(200,
		`{"data":{"id":"op-1","type":"operation","operationType":"resetBrokerPasswords","resourceId":"`+
			serviceId+`","resourceType":"service","status":"SUCCEEDED","error":null}}`))
	httpmock.RegisterResponder(methodPost, resetUrl, func(r *http.Request) (*http.Response, error) {
		// httpmock falls back to the URL without its query string, and an exact responder there
		// wins over any pattern, so override the bare service URL.
		httpmock.RegisterResponder("GET", serviceUrl,
			internal.JsonResponder(503, `{"message":"Service temporarily unavailable"}`))
		return internal.JsonResponder(202, `{"data":{"id":"op-1","type":"operation","operationType":"resetBrokerPasswords","resourceId":"`+
			serviceId+`","resourceType":"service","status":"PENDING","error":null}}`)(r)
	})

	client, err := missioncontrol.NewClientWithResponses(instance.GetBaseURL())
	if err != nil {
		t.Fatalf("failed to create mission control client: %v", err)
	}
	r := &ServiceResource{APIClient: apiclient.NewRetryableClient(client, 1, 0)}
	r.endpointManager = connectionendpoint.NewEndpointManager(r.APIClient, 0)

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	serviceSchema := schemaResp.Schema
	objectType := serviceSchema.Type().TerraformType(ctx).(tftypes.Object)

	build := func(trigger string) tftypes.Value {
		return serviceObjectValue(objectType, serviceAttributes(map[string]tftypes.Value{
			"id":                       tftypes.NewValue(tftypes.String, serviceId),
			"broker_password_rotation": rotationValue(objectType, strPtr(trigger), brokerUserViewer),
			"message_vpn":              messageVpnValue(objectType),
		}))
	}
	state := tfsdk.State{Raw: build("v1"), Schema: serviceSchema}
	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Raw: build("v2"), Schema: serviceSchema},
		State: state,
	}
	// The framework starts the response state as a copy of the prior state.
	resp := &resource.UpdateResponse{State: tfsdk.State{Raw: state.Raw, Schema: serviceSchema}}

	r.Update(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected the failed read-back to fail the update")
	}
	if calls := httpmock.GetCallCountInfo()["POST "+resetUrl]; calls != 1 {
		t.Fatalf("expected exactly one reset request, got %d", calls)
	}
	if !diagnosticsContainFold(resp.Diagnostics.Warnings(), "rotated but could not be read back") {
		t.Errorf("expected a warning that the rotation could not be read back, got %v", resp.Diagnostics)
	}

	var trigger types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("broker_password_rotation").AtName("trigger"), &trigger)...)
	var password types.String
	viewerPassword := path.Root("message_vpn").AtName(credentialAttributeByUser[brokerUserViewer]).AtName("password")
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, viewerPassword, &password)...)

	if trigger.ValueString() != "v2" {
		t.Errorf("a completed rotation must record the new trigger so the next apply does not rotate again, state has %q", trigger.ValueString())
	}
	if password.ValueString() != "old-"+credentialAttributeByUser[brokerUserViewer] {
		t.Errorf("the passwords could not be read back, so state must keep the old viewer password, which is now %q", password.ValueString())
	}
}
