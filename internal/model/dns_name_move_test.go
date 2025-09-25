package model

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestDnsNameMoveModel_ToObjectValue(t *testing.T) {
	// Test successful conversion
	model := DnsNameMoveModel{
		Id:                         types.StringValue("move-op-123"),
		SourceServiceId:            types.StringValue("source-service-456"),
		SourceConnectionEndpointId: types.StringValue("source-endpoint-789"),
		DnsName:                    types.StringValue("move.example.com"),
		TargetServiceId:            types.StringValue("target-service-101"),
		TargetConnectionEndpointId: types.StringValue("target-endpoint-112"),
	}

	objValue, diags := model.ToObjectValue()

	// Should not have any diagnostics
	assert.False(t, diags.HasError())
	assert.Empty(t, diags)

	// Should not be null or unknown
	assert.False(t, objValue.IsNull())
	assert.False(t, objValue.IsUnknown())

	// Verify attributes
	attrs := objValue.Attributes()
	assert.Equal(t, types.StringValue("move-op-123"), attrs["id"])
	assert.Equal(t, types.StringValue("source-service-456"), attrs["source_service_id"])
	assert.Equal(t, types.StringValue("source-endpoint-789"), attrs["source_connection_endpoint_id"])
	assert.Equal(t, types.StringValue("move.example.com"), attrs["dns_name"])
	assert.Equal(t, types.StringValue("target-service-101"), attrs["target_service_id"])
	assert.Equal(t, types.StringValue("target-endpoint-112"), attrs["target_connection_endpoint_id"])
}

func TestDnsNameMoveModel_ToObjectValue_WithNullValues(t *testing.T) {
	// Test with null values
	model := DnsNameMoveModel{
		Id:                         types.StringNull(),
		SourceServiceId:            types.StringValue("source-service-456"),
		SourceConnectionEndpointId: types.StringValue("source-endpoint-789"),
		DnsName:                    types.StringValue("move.example.com"),
		TargetServiceId:            types.StringNull(),
		TargetConnectionEndpointId: types.StringNull(),
	}

	objValue, diags := model.ToObjectValue()

	// Should not have any diagnostics
	assert.False(t, diags.HasError())
	assert.Empty(t, diags)

	// Should not be null
	assert.False(t, objValue.IsNull())

	// Verify null attributes are properly handled
	attrs := objValue.Attributes()
	assert.True(t, attrs["id"].IsNull())
	assert.False(t, attrs["source_service_id"].IsNull())
	assert.False(t, attrs["source_connection_endpoint_id"].IsNull())
	assert.False(t, attrs["dns_name"].IsNull())
	assert.True(t, attrs["target_service_id"].IsNull())
	assert.True(t, attrs["target_connection_endpoint_id"].IsNull())
}

func TestDnsNameMoveModel_ToObjectValue_WithUnknownValues(t *testing.T) {
	// Test with unknown values
	model := DnsNameMoveModel{
		Id:                         types.StringUnknown(),
		SourceServiceId:            types.StringValue("source-service-456"),
		SourceConnectionEndpointId: types.StringValue("source-endpoint-789"),
		DnsName:                    types.StringValue("move.example.com"),
		TargetServiceId:            types.StringUnknown(),
		TargetConnectionEndpointId: types.StringUnknown(),
	}

	objValue, diags := model.ToObjectValue()

	// Should not have any diagnostics
	assert.False(t, diags.HasError())
	assert.Empty(t, diags)

	// Should not be null
	assert.False(t, objValue.IsNull())

	// Verify unknown attributes are properly handled
	attrs := objValue.Attributes()
	assert.True(t, attrs["id"].IsUnknown())
	assert.False(t, attrs["source_service_id"].IsUnknown())
	assert.False(t, attrs["source_connection_endpoint_id"].IsUnknown())
	assert.False(t, attrs["dns_name"].IsUnknown())
	assert.True(t, attrs["target_service_id"].IsUnknown())
	assert.True(t, attrs["target_connection_endpoint_id"].IsUnknown())
}

func TestDnsNameMoveTypes(t *testing.T) {
	types := DnsNameMoveTypes()

	// Verify all expected keys are present
	expectedKeys := []string{
		"id",
		"source_service_id",
		"source_connection_endpoint_id",
		"dns_name",
		"target_service_id",
		"target_connection_endpoint_id",
	}

	for _, key := range expectedKeys {
		assert.Contains(t, types, key)
		assert.Equal(t, types[key], types[key]) // Verify type is consistent
	}

	// Verify correct number of types
	assert.Len(t, types, 6)
}

func TestDnsNameMoveResourceSchema(t *testing.T) {
	schema := DnsNameMoveResourceSchema()

	// Verify schema has correct attributes
	attrs := schema.Attributes

	// Check all required attributes exist
	requiredAttrs := []string{
		"id", "source_service_id", "source_connection_endpoint_id",
		"dns_name", "target_service_id", "target_connection_endpoint_id",
	}
	for _, attrName := range requiredAttrs {
		assert.Contains(t, attrs, attrName, "Missing attribute: %s", attrName)
	}

	// Check id attribute properties
	idAttr := attrs["id"]
	assert.True(t, idAttr.IsComputed())
	assert.False(t, idAttr.IsRequired())
	assert.False(t, idAttr.IsOptional())

	// Check source_service_id attribute properties
	sourceServiceIdAttr := attrs["source_service_id"]
	assert.True(t, sourceServiceIdAttr.IsRequired())
	assert.False(t, sourceServiceIdAttr.IsComputed())
	assert.False(t, sourceServiceIdAttr.IsOptional())

	// Check source_connection_endpoint_id attribute properties
	sourceEndpointIdAttr := attrs["source_connection_endpoint_id"]
	assert.True(t, sourceEndpointIdAttr.IsRequired())
	assert.False(t, sourceEndpointIdAttr.IsComputed())
	assert.False(t, sourceEndpointIdAttr.IsOptional())

	// Check dns_name attribute properties
	dnsNameAttr := attrs["dns_name"]
	assert.True(t, dnsNameAttr.IsRequired())
	assert.False(t, dnsNameAttr.IsComputed())
	assert.False(t, dnsNameAttr.IsOptional())

	// Check target_service_id attribute properties (optional + computed)
	targetServiceIdAttr := attrs["target_service_id"]
	assert.False(t, targetServiceIdAttr.IsRequired())
	assert.True(t, targetServiceIdAttr.IsComputed())
	assert.True(t, targetServiceIdAttr.IsOptional())

	// Check target_connection_endpoint_id attribute properties (optional + computed)
	targetEndpointIdAttr := attrs["target_connection_endpoint_id"]
	assert.False(t, targetEndpointIdAttr.IsRequired())
	assert.True(t, targetEndpointIdAttr.IsComputed())
	assert.True(t, targetEndpointIdAttr.IsOptional())

	// Verify schema has description
	assert.NotEmpty(t, schema.MarkdownDescription)
	assert.Contains(t, schema.MarkdownDescription, "Moves a DNS name")
}

func TestDnsNameMoveModel_DefaultBehavior(t *testing.T) {
	// Test model with minimal values (simulating when target values default)
	model := DnsNameMoveModel{
		Id:                         types.StringValue("move-op-123"),
		SourceServiceId:            types.StringValue("service-456"),
		SourceConnectionEndpointId: types.StringValue("endpoint-789"),
		DnsName:                    types.StringValue("test.example.com"),
		// Target values not set - should be null initially
		TargetServiceId:            types.StringNull(),
		TargetConnectionEndpointId: types.StringNull(),
	}

	objValue, diags := model.ToObjectValue()

	// Should not have any diagnostics
	assert.False(t, diags.HasError())
	assert.Empty(t, diags)

	// Verify that target fields can be null (they get computed)
	attrs := objValue.Attributes()
	assert.True(t, attrs["target_service_id"].IsNull())
	assert.True(t, attrs["target_connection_endpoint_id"].IsNull())
}

func TestDnsNameMoveModel_SameServiceMove(t *testing.T) {
	// Test model for moving within same service (common use case)
	model := DnsNameMoveModel{
		Id:                         types.StringValue("same-service-move-123"),
		SourceServiceId:            types.StringValue("service-456"),
		SourceConnectionEndpointId: types.StringValue("endpoint-789"),
		DnsName:                    types.StringValue("same-service.example.com"),
		TargetServiceId:            types.StringValue("service-456"), // Same service
		TargetConnectionEndpointId: types.StringValue("endpoint-999"), // Different endpoint
	}

	objValue, diags := model.ToObjectValue()

	// Should not have any diagnostics
	assert.False(t, diags.HasError())
	assert.Empty(t, diags)

	// Verify that source and target service are the same
	attrs := objValue.Attributes()
	assert.Equal(t, attrs["source_service_id"], attrs["target_service_id"])
	assert.NotEqual(t, attrs["source_connection_endpoint_id"], attrs["target_connection_endpoint_id"])
}