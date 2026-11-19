package service

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// rotationValue builds a broker_password_rotation value. A nil trigger yields a null block.
func rotationValue(objectType tftypes.Object, trigger *string, users ...string) tftypes.Value {
	rotationType := objectType.AttributeTypes["broker_password_rotation"].(tftypes.Object)
	if trigger == nil {
		return tftypes.NewValue(rotationType, nil)
	}
	elems := make([]tftypes.Value, 0, len(users))
	for _, u := range users {
		elems = append(elems, tftypes.NewValue(tftypes.String, u))
	}
	return tftypes.NewValue(rotationType, map[string]tftypes.Value{
		"users":   tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, elems),
		"trigger": tftypes.NewValue(tftypes.String, *trigger),
	})
}

// messageVpnValue builds message_vpn with a known password on each credential, except the
// credentials named in absent, which are null (as the manager credential is on older brokers).
func messageVpnValue(objectType tftypes.Object, absent ...string) tftypes.Value {
	vpnType := objectType.AttributeTypes["message_vpn"].(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, attrType := range vpnType.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	for _, attrName := range credentialAttributeByUser {
		credType := vpnType.AttributeTypes[attrName].(tftypes.Object)
		values[attrName] = tftypes.NewValue(credType, map[string]tftypes.Value{
			"username": tftypes.NewValue(tftypes.String, attrName),
			"password": tftypes.NewValue(tftypes.String, "old-"+attrName),
		})
	}
	for _, user := range absent {
		attrName := credentialAttributeByUser[user]
		values[attrName] = tftypes.NewValue(vpnType.AttributeTypes[attrName], nil)
	}
	return tftypes.NewValue(vpnType, values)
}

func strPtr(s string) *string { return &s }

func TestModifyPlanBrokerPasswordRotation(t *testing.T) {
	ctx := context.Background()
	r := &ServiceResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	serviceSchema := schemaResp.Schema
	objectType := serviceSchema.Type().TerraformType(ctx).(tftypes.Object)

	build := func(rotation tftypes.Value, absent ...string) tftypes.Value {
		return serviceObjectValue(objectType, serviceAttributes(map[string]tftypes.Value{
			"broker_password_rotation": rotation,
			"message_vpn":              messageVpnValue(objectType, absent...),
		}))
	}

	cases := []struct {
		name          string
		state         tftypes.Value
		plan          tftypes.Value
		expectUnknown []string // users whose password must be unknown
		expectError   string   // summary of the expected error, if any
		expectWarning string   // summary of the expected warning, if any
	}{
		{name: "create rotates nothing",
			state: tftypes.NewValue(objectType, nil),
			plan:  build(rotationValue(objectType, strPtr("v1"), brokerUserViewer))},
		{name: "trigger change rotates the listed user only",
			state:         build(rotationValue(objectType, strPtr("v1"), brokerUserViewer)),
			plan:          build(rotationValue(objectType, strPtr("v2"), brokerUserViewer)),
			expectUnknown: []string{brokerUserViewer}},
		{name: "trigger change rotates every listed user",
			state:         build(rotationValue(objectType, strPtr("v1"), brokerUserViewer)),
			plan:          build(rotationValue(objectType, strPtr("v2"), brokerUserViewer, brokerUserEditor, brokerUserManager)),
			expectUnknown: []string{brokerUserViewer, brokerUserEditor, brokerUserManager}},
		{name: "same trigger rotates nothing",
			state: build(rotationValue(objectType, strPtr("v1"), brokerUserViewer)),
			plan:  build(rotationValue(objectType, strPtr("v1"), brokerUserViewer))},
		{name: "editing users alone rotates nothing",
			state: build(rotationValue(objectType, strPtr("v1"), brokerUserViewer)),
			plan:  build(rotationValue(objectType, strPtr("v1"), brokerUserViewer, brokerUserEditor))},
		{name: "removing the block rotates nothing",
			state: build(rotationValue(objectType, strPtr("v1"), brokerUserViewer)),
			plan:  build(rotationValue(objectType, nil))},
		{name: "adding the block to an existing service records it without rotating",
			state:         build(rotationValue(objectType, nil)),
			plan:          build(rotationValue(objectType, strPtr("v1"), brokerUserViewer)),
			expectWarning: warnPasswordsNotRotated},
		{name: "adding the block with mc-manager and no manager credential is rejected",
			state:         build(rotationValue(objectType, nil), brokerUserManager),
			plan:          build(rotationValue(objectType, strPtr("v1"), brokerUserManager), brokerUserManager),
			expectError:   "Cannot rotate password for " + brokerUserManager,
			expectWarning: warnPasswordsNotRotated},
		{name: "adding the block with mc-manager and a manager credential is accepted",
			state:         build(rotationValue(objectType, nil)),
			plan:          build(rotationValue(objectType, strPtr("v1"), brokerUserManager)),
			expectWarning: warnPasswordsNotRotated},
		{name: "adding a block that is not yet known neither warns nor rotates",
			state: build(rotationValue(objectType, nil)),
			plan: build(tftypes.NewValue(
				objectType.AttributeTypes["broker_password_rotation"], tftypes.UnknownValue))},
		{name: "rotating mc-manager without a manager credential is rejected",
			state:       build(rotationValue(objectType, strPtr("v1"), brokerUserViewer), brokerUserManager),
			plan:        build(rotationValue(objectType, strPtr("v2"), brokerUserManager), brokerUserManager),
			expectError: "Cannot rotate password for " + brokerUserManager},
	}

	allUsers := []string{brokerUserEditor, brokerUserManager, brokerUserViewer}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := resource.ModifyPlanRequest{
				Config: tfsdk.Config{Raw: tc.plan, Schema: serviceSchema},
				Plan:   tfsdk.Plan{Raw: tc.plan, Schema: serviceSchema},
				State:  tfsdk.State{Raw: tc.state, Schema: serviceSchema},
			}
			resp := &resource.ModifyPlanResponse{Plan: req.Plan}

			r.ModifyPlan(ctx, req, resp)

			gotWarning := ""
			for _, d := range resp.Diagnostics.Warnings() {
				gotWarning = d.Summary()
			}
			if gotWarning != tc.expectWarning {
				t.Errorf("expected warning %q, got %q", tc.expectWarning, gotWarning)
			}

			if resp.Diagnostics.HasError() != (tc.expectError != "") {
				t.Fatalf("expectError=%q, diagnostics: %v", tc.expectError, resp.Diagnostics)
			}
			if tc.expectError != "" {
				for _, d := range resp.Diagnostics.Errors() {
					if d.Summary() == tc.expectError {
						return
					}
				}
				t.Fatalf("expected error %q, diagnostics: %v", tc.expectError, resp.Diagnostics)
			}

			unknown := map[string]bool{}
			for _, u := range tc.expectUnknown {
				unknown[u] = true
			}
			for _, user := range allUsers {
				var password types.String
				p := path.Root("message_vpn").AtName(credentialAttributeByUser[user]).AtName("password")
				if diags := resp.Plan.GetAttribute(ctx, p, &password); diags.HasError() {
					// A null credential has no password to read; it can only be expected null.
					if unknown[user] {
						t.Fatalf("could not read %s password: %v", user, diags)
					}
					continue
				}
				if password.IsUnknown() != unknown[user] {
					t.Errorf("%s: expected password unknown=%t, got %s", user, unknown[user], password)
				}
			}
		})
	}
}
