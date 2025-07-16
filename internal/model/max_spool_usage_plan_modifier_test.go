package model_test

import (
	"context"
	"terraform-provider-solacecloud/internal/model"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

func TestMaxSpoolUsagePlanModifier_Description(t *testing.T) {
	modifier := model.MaxSpoolUsagePlanModifier{}
	ctx := context.Background()

	expected := "Updates max_msg_spool_usage when max_spool_usage changes"
	actual := modifier.Description(ctx)

	if actual != expected {
		t.Errorf("Expected description %q, got %q", expected, actual)
	}
}

func TestMaxSpoolUsagePlanModifier_MarkdownDescription(t *testing.T) {
	modifier := model.MaxSpoolUsagePlanModifier{}
	ctx := context.Background()

	expected := "Updates max_msg_spool_usage when max_spool_usage changes"
	actual := modifier.MarkdownDescription(ctx)

	if actual != expected {
		t.Errorf("Expected markdown description %q, got %q", expected, actual)
	}
}

func TestMaxSpoolUsagePlanModifier_PlanModifyInt64_UnknownConfigValue(t *testing.T) {
	modifier := model.MaxSpoolUsagePlanModifier{}
	ctx := context.Background()

	// Test that the modifier does nothing when config value is unknown
	req := planmodifier.Int64Request{}
	resp := &planmodifier.Int64Response{}

	// Since we can't easily mock the complex request structure,
	// we'll test the basic functionality that can be tested
	modifier.PlanModifyInt64(ctx, req, resp)

	// The modifier should handle the case gracefully without panicking
	if resp.Diagnostics.HasError() {
		t.Errorf("Expected no errors, but got: %v", resp.Diagnostics)
	}
}

func TestNewMaxSpoolUsagePlanModifier(t *testing.T) {
	modifier := model.NewMaxSpoolUsagePlanModifier()

	if modifier == nil {
		t.Error("Expected NewMaxSpoolUsagePlanModifier to return a non-nil modifier")
	}

	// Verify it returns the correct type
	if _, ok := modifier.(model.MaxSpoolUsagePlanModifier); !ok {
		t.Error("Expected NewMaxSpoolUsagePlanModifier to return MaxSpoolUsagePlanModifier type")
	}
}

func TestMaxSpoolUsagePlanModifier_Interface(t *testing.T) {
	modifier := model.MaxSpoolUsagePlanModifier{}

	// Verify that the modifier implements the planmodifier.Int64 interface
	var _ planmodifier.Int64 = modifier

	// Test that Description and MarkdownDescription don't panic
	ctx := context.Background()
	desc := modifier.Description(ctx)
	markdownDesc := modifier.MarkdownDescription(ctx)

	if desc == "" {
		t.Error("Description should not be empty")
	}

	if markdownDesc == "" {
		t.Error("MarkdownDescription should not be empty")
	}
}

func TestCalculateMsgVpnSpoolUsageFromMaxSpoolUsage(t *testing.T) {
	tests := []struct {
		name           string
		maxSpoolUsage  int64
		expectedResult int64
	}{
		{
			name:           "zero max spool usage",
			maxSpoolUsage:  0,
			expectedResult: 0,
		},
		{
			name:           "positive max spool usage",
			maxSpoolUsage:  1024,
			expectedResult: 1024000,
		},
		{
			name:           "large max spool usage",
			maxSpoolUsage:  1073741824, // 1GB
			expectedResult: 1073741824000,
		},
		{
			name:           "negative max spool usage",
			maxSpoolUsage:  -1,
			expectedResult: -1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := model.CalculateMsgVpnSpoolUsageFromMaxSpoolUsage(tt.maxSpoolUsage)
			if result != tt.expectedResult {
				t.Errorf("CalculateMsgVpnSpoolUsageFromMaxSpoolUsage(%d) = %d, want %d",
					tt.maxSpoolUsage, result, tt.expectedResult)
			}
		})
	}
}
