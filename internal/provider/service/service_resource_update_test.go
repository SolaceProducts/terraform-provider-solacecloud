package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jarcoal/httpmock"
)

const (
	patchTestServiceId = "svc-patch-attributes"
	patchServiceUrl    = dmrTestBaseUrl + "/api/v2/missionControl/eventBrokerServices/" + patchTestServiceId
)

func patchTestState() *ServiceResourceModel {
	return &ServiceResourceModel{
		Id:      types.StringValue(patchTestServiceId),
		Name:    types.StringValue("service-name"),
		OwnedBy: types.StringValue("owner-id"),
		Locked:  types.BoolValue(false),
	}
}

func TestPatchChangedAttributesSendsNothingWhenUnchanged(t *testing.T) {
	cases := []struct {
		name string
		plan ServiceResourceModel
	}{
		{name: "same values", plan: *patchTestState()},
		{name: "unknown values", plan: ServiceResourceModel{Name: types.StringUnknown(), OwnedBy: types.StringUnknown(), Locked: types.BoolUnknown()}},
		{name: "null values", plan: ServiceResourceModel{Name: types.StringNull(), OwnedBy: types.StringNull(), Locked: types.BoolNull()}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			diags := dmrTestResource(t).patchChangedAttributes(context.Background(), patchTestState(), &tc.plan)

			if diags.HasError() {
				t.Fatalf(unexpectedDiags, diags)
			}
			if count := httpmock.GetTotalCallCount(); count != 0 {
				t.Errorf("expected no API calls, got %d", count)
			}
		})
	}
}

func TestPatchChangedAttributesSendsOnlyChangedFields(t *testing.T) {
	cases := []struct {
		name   string
		change func(*ServiceResourceModel)
		want   map[string]interface{}
	}{
		{name: "name", change: func(p *ServiceResourceModel) { p.Name = types.StringValue("renamed") },
			want: map[string]interface{}{"name": "renamed"}},
		{name: "owned_by", change: func(p *ServiceResourceModel) { p.OwnedBy = types.StringValue("new-owner") },
			want: map[string]interface{}{"ownedBy": "new-owner"}},
		{name: "locked", change: func(p *ServiceResourceModel) { p.Locked = types.BoolValue(true) },
			want: map[string]interface{}{"locked": true}},
		{name: "all three", change: func(p *ServiceResourceModel) {
			p.Name = types.StringValue("renamed")
			p.OwnedBy = types.StringValue("new-owner")
			p.Locked = types.BoolValue(true)
		}, want: map[string]interface{}{"name": "renamed", "ownedBy": "new-owner", "locked": true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			var sentBody map[string]interface{}
			httpmock.RegisterResponder(methodPatch, patchServiceUrl, func(req *http.Request) (*http.Response, error) {
				raw, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("could not read the request body: %v", err)
				}
				if err := json.Unmarshal(raw, &sentBody); err != nil {
					t.Fatalf("request body is not JSON: %v", err)
				}
				return internal.JsonResponder(200, `{"data":{"id":"`+patchTestServiceId+`"}}`)(req)
			})

			plan := patchTestState()
			tc.change(plan)
			diags := dmrTestResource(t).patchChangedAttributes(context.Background(), patchTestState(), plan)

			if diags.HasError() {
				t.Fatalf(unexpectedDiags, diags)
			}
			if calls := httpmock.GetCallCountInfo()[methodPatch+" "+patchServiceUrl]; calls != 1 {
				t.Fatalf("expected one service update, got %d", calls)
			}
			if !reflect.DeepEqual(sentBody, tc.want) {
				t.Errorf("expected request body %v, got %v", tc.want, sentBody)
			}
		})
	}
}

func TestPatchChangedAttributesSurfacesApiErrors(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(methodPatch, patchServiceUrl,
		internal.JsonResponder(400, `{"message":"The service name is already in use.","errorId":"abc"}`))

	plan := patchTestState()
	plan.Name = types.StringValue("taken-name")
	diags := dmrTestResource(t).patchChangedAttributes(context.Background(), patchTestState(), plan)

	if !diags.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	if !diagnosticsContain(diags.Errors(), "The service name is already in use.") {
		t.Errorf("expected a diagnostic naming the refusal, got %v", diags.Errors())
	}
}
