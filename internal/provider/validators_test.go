package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestImmutableStringPlanModifier_Description(t *testing.T) {
	modifier := ImmutableStringPlanModifier{}
	ctx := context.Background()

	expected := "prevents changes to attribute after resource creation"
	actual := modifier.Description(ctx)

	if actual != expected {
		t.Errorf("Expected description %q, got %q", expected, actual)
	}
}

func TestImmutableStringPlanModifier_MarkdownDescription(t *testing.T) {
	modifier := ImmutableStringPlanModifier{}
	ctx := context.Background()

	expected := "**prevents** changes to attribute after resource creation"
	actual := modifier.MarkdownDescription(ctx)

	if actual != expected {
		t.Errorf("Expected markdown description %q, got %q", expected, actual)
	}
}

func TestImmutableStringPlanModifier_PlanModifyString(t *testing.T) {
	tests := []struct {
		name             string
		stateValue       types.String
		planValue        types.String
		expectError      bool
		expectPlanChange bool
	}{
		{
			name:             "no change - same value",
			stateValue:       types.StringValue("test-value"),
			planValue:        types.StringValue("test-value"),
			expectError:      false,
			expectPlanChange: false,
		},
		{
			name:             "change attempted - should error",
			stateValue:       types.StringValue("original-value"),
			planValue:        types.StringValue("new-value"),
			expectError:      true,
			expectPlanChange: true,
		},
		{
			name:             "null state - no error (creation)",
			stateValue:       types.StringNull(),
			planValue:        types.StringValue("new-value"),
			expectError:      false,
			expectPlanChange: false,
		},
		{
			name:             "unknown state - no error",
			stateValue:       types.StringUnknown(),
			planValue:        types.StringValue("new-value"),
			expectError:      false,
			expectPlanChange: false,
		},
		{
			name:             "null plan - no error",
			stateValue:       types.StringValue("test-value"),
			planValue:        types.StringNull(),
			expectError:      false,
			expectPlanChange: false,
		},
		{
			name:             "unknown plan - no error",
			stateValue:       types.StringValue("test-value"),
			planValue:        types.StringUnknown(),
			expectError:      false,
			expectPlanChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifier := ImmutableStringPlanModifier{}
			ctx := context.Background()

			req := planmodifier.StringRequest{
				Path:       path.Root("test_attribute"),
				StateValue: tt.stateValue,
				PlanValue:  tt.planValue,
			}

			resp := &planmodifier.StringResponse{
				PlanValue: tt.planValue,
			}

			modifier.PlanModifyString(ctx, req, resp)

			hasError := resp.Diagnostics.HasError()
			if hasError != tt.expectError {
				t.Errorf("Expected error: %v, got error: %v", tt.expectError, hasError)
				if hasError {
					for _, diag := range resp.Diagnostics.Errors() {
						t.Logf("Error: %s - %s", diag.Summary(), diag.Detail())
					}
				}
			}

			if tt.expectPlanChange {
				// When there's an error, the plan should be reverted to state value
				if hasError && !resp.PlanValue.Equal(tt.stateValue) {
					t.Errorf("Expected plan value to be reverted to state value %v, got %v",
						tt.stateValue, resp.PlanValue)
				}
			}
		})
	}
}

func TestNewImmutableStringPlanModifier(t *testing.T) {
	modifier := NewImmutableStringPlanModifier()

	if modifier == nil {
		t.Error("Expected non-nil plan modifier")
	}

	// Verify it's the correct type
	_, ok := modifier.(ImmutableStringPlanModifier)
	if !ok {
		t.Errorf("Expected ImmutableStringPlanModifier type, got %T", modifier)
	}
}

func TestValidateImmutableNotChanged(t *testing.T) {
	tests := []struct {
		name        string
		stateValue  types.String
		planValue   types.String
		expectError bool
	}{
		{
			name:        "no change",
			stateValue:  types.StringValue("same"),
			planValue:   types.StringValue("same"),
			expectError: false,
		},
		{
			name:        "value changed",
			stateValue:  types.StringValue("old"),
			planValue:   types.StringValue("new"),
			expectError: true,
		},
		{
			name:        "empty to value",
			stateValue:  types.StringValue(""),
			planValue:   types.StringValue("new"),
			expectError: false,
		},
		{
			name:        "value to empty",
			stateValue:  types.StringValue("old"),
			planValue:   types.StringValue(""),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			req := planmodifier.StringRequest{
				Path:       path.Root("test"),
				StateValue: tt.stateValue,
				PlanValue:  tt.planValue,
			}

			resp := &planmodifier.StringResponse{
				PlanValue: tt.planValue,
			}

			validateImmutableNotChanged(ctx, req, resp)

			hasError := resp.Diagnostics.HasError()
			if hasError != tt.expectError {
				t.Errorf("Expected error: %v, got error: %v", tt.expectError, hasError)
			}

			if tt.expectError {
				// Plan should be reverted to state value when there's an error
				if !resp.PlanValue.Equal(tt.stateValue) {
					t.Errorf("Expected plan value to be reverted to state value %v, got %v",
						tt.stateValue, resp.PlanValue)
				}
			}
		})
	}
}
