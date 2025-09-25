package connectionendpoint_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/require"
	"terraform-provider-solacecloud/internal/provider/connectionendpoint"
)

func TestConnectionEndpointSchema(t *testing.T) {
	schemaAttributes := connectionendpoint.ConnectionEndpointSchema()

	expectedAttributes := []string{
		"service_id",
		"name",
		"description",
		"access_type",
		"ports",
	}

	// Ensure all expected attributes are present in the schema
	for _, attr := range expectedAttributes {
		t.Run(attr, func(t *testing.T) {
			_, exists := schemaAttributes[attr]
			require.True(t, exists, "Attribute %s should exist in the schema", attr)
		})
	}

	// Check specific attribute details, e.g., for `service_id`
	t.Run("service_id attribute", func(t *testing.T) {
		serviceIDAttr, exists := schemaAttributes["service_id"]
		require.True(t, exists, "Expected 'service_id' attribute to exist")

		stringAttr, ok := serviceIDAttr.(schema.StringAttribute)
		require.True(t, ok, "'service_id' should be a StringAttribute")
		require.True(t, stringAttr.IsRequired(), "'service_id' attribute should be required")
	})

	// Add similar tests for other attributes as needed...
}
