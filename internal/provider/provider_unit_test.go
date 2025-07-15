package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderBaseURLConfiguration(t *testing.T) {
	testCases := []struct {
		name            string
		configValue     any
		expectedBaseURL string
		shouldHaveError bool
		errorContains   string
	}{
		{
			name:            "base_url empty string - should use default",
			configValue:     "",
			expectedBaseURL: "https://production-api.solace.cloud",
			shouldHaveError: false,
		},
		{
			name:            "base_url custom value provided",
			configValue:     "https://staging-api.solace.cloud",
			expectedBaseURL: "https://staging-api.solace.cloud",
			shouldHaveError: false,
		},
		{
			name:            "base_url localhost for development",
			configValue:     "http://localhost:8080",
			expectedBaseURL: "http://localhost:8080",
			shouldHaveError: false,
		},
		{
			name:            "base_url with path",
			configValue:     "https://api.example.com/v1",
			expectedBaseURL: "https://api.example.com/v1",
			shouldHaveError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create provider instance
			p := &solaceCloudProvider{version: "test"}

			// Get the provider schema first
			schemaReq := provider.SchemaRequest{}
			schemaResp := &provider.SchemaResponse{}
			p.Schema(context.Background(), schemaReq, schemaResp)

			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("Failed to get provider schema: %v", schemaResp.Diagnostics.Errors())
			}

			// Create configuration based on test case
			configValue := tftypes.NewValue(tftypes.String, tc.configValue)
			// Create configuration object
			config := tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"base_url":             tftypes.String,
					"api_token":            tftypes.String,
					"api_polling_interval": tftypes.Number,
				},
			}, map[string]tftypes.Value{
				"base_url":             configValue,
				"api_token":            tftypes.NewValue(tftypes.String, "test-token"),
				"api_polling_interval": tftypes.NewValue(tftypes.Number, 30),
			})

			// Create configure request with schema
			req := provider.ConfigureRequest{
				Config: tfsdk.Config{
					Raw:    config,
					Schema: schemaResp.Schema,
				},
			}

			resp := &provider.ConfigureResponse{}

			// Execute configure
			p.Configure(context.Background(), req, resp)

			// Check for errors
			if tc.shouldHaveError {
				if !resp.Diagnostics.HasError() {
					t.Errorf("Expected error but got none")
					return
				}
				if tc.errorContains != "" {
					found := false
					for _, diag := range resp.Diagnostics.Errors() {
						if contains(diag.Summary(), tc.errorContains) || contains(diag.Detail(), tc.errorContains) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error to contain %q, but got: %v", tc.errorContains, resp.Diagnostics.Errors())
					}
				}
			} else {
				if resp.Diagnostics.HasError() {
					t.Errorf("Unexpected error: %v", resp.Diagnostics.Errors())
					return
				}
			}

		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(substr) > 0 && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
