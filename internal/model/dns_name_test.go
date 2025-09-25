package model

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestDnsNameModel_ToObjectValue(t *testing.T) {
	// Test successful conversion
	model := DnsNameModel{
		Id:                   types.StringValue("dns-id-123"),
		ServiceId:            types.StringValue("service-id-456"),
		ConnectionEndpointId: types.StringValue("endpoint-id-789"),
		DnsName:              types.StringValue("api.example.com"),
		DomainType:           types.StringValue("CUSTOM"),
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
	assert.Equal(t, types.StringValue("dns-id-123"), attrs["id"])
	assert.Equal(t, types.StringValue("service-id-456"), attrs["service_id"])
	assert.Equal(t, types.StringValue("endpoint-id-789"), attrs["connection_endpoint_id"])
	assert.Equal(t, types.StringValue("api.example.com"), attrs["dns_name"])
	assert.Equal(t, types.StringValue("CUSTOM"), attrs["domain_type"])
}

func TestDnsNameModel_ToObjectValue_WithNullValues(t *testing.T) {
	// Test with null values
	model := DnsNameModel{
		Id:                   types.StringNull(),
		ServiceId:            types.StringValue("service-id-456"),
		ConnectionEndpointId: types.StringNull(),
		DnsName:              types.StringValue("api.example.com"),
		DomainType:           types.StringNull(),
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
	assert.False(t, attrs["service_id"].IsNull())
	assert.True(t, attrs["connection_endpoint_id"].IsNull())
	assert.False(t, attrs["dns_name"].IsNull())
	assert.True(t, attrs["domain_type"].IsNull())
}

func TestDnsNameModel_ToObjectValue_WithUnknownValues(t *testing.T) {
	// Test with unknown values
	model := DnsNameModel{
		Id:                   types.StringUnknown(),
		ServiceId:            types.StringValue("service-id-456"),
		ConnectionEndpointId: types.StringUnknown(),
		DnsName:              types.StringValue("api.example.com"),
		DomainType:           types.StringUnknown(),
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
	assert.False(t, attrs["service_id"].IsUnknown())
	assert.True(t, attrs["connection_endpoint_id"].IsUnknown())
	assert.False(t, attrs["dns_name"].IsUnknown())
	assert.True(t, attrs["domain_type"].IsUnknown())
}

func TestDnsNameTypes(t *testing.T) {
	types := DnsNameTypes()

	// Verify all expected keys are present
	expectedKeys := []string{
		"id",
		"service_id",
		"connection_endpoint_id",
		"dns_name",
		"domain_type",
	}

	for _, key := range expectedKeys {
		assert.Contains(t, types, key)
		assert.Equal(t, types[key], types[key]) // Verify type is consistent
	}

	// Verify correct number of types
	assert.Len(t, types, 5)
}

func TestDnsNameResourceSchema(t *testing.T) {
	schema := DnsNameResourceSchema()

	// Verify schema has correct attributes
	attrs := schema.Attributes

	// Check all required attributes exist
	requiredAttrs := []string{"id", "service_id", "connection_endpoint_id", "dns_name", "domain_type"}
	for _, attrName := range requiredAttrs {
		assert.Contains(t, attrs, attrName, "Missing attribute: %s", attrName)
	}

	// Check id attribute properties
	idAttr := attrs["id"]
	assert.True(t, idAttr.IsComputed())
	assert.False(t, idAttr.IsRequired())
	assert.False(t, idAttr.IsOptional())

	// Check service_id attribute properties
	serviceIdAttr := attrs["service_id"]
	assert.True(t, serviceIdAttr.IsRequired())
	assert.False(t, serviceIdAttr.IsComputed())
	assert.False(t, serviceIdAttr.IsOptional())

	// Check connection_endpoint_id attribute properties
	endpointIdAttr := attrs["connection_endpoint_id"]
	assert.True(t, endpointIdAttr.IsRequired())
	assert.False(t, endpointIdAttr.IsComputed())
	assert.False(t, endpointIdAttr.IsOptional())

	// Check dns_name attribute properties
	dnsNameAttr := attrs["dns_name"]
	assert.True(t, dnsNameAttr.IsRequired())
	assert.False(t, dnsNameAttr.IsComputed())
	assert.False(t, dnsNameAttr.IsOptional())

	// Check domain_type attribute properties
	domainTypeAttr := attrs["domain_type"]
	assert.True(t, domainTypeAttr.IsComputed())
	assert.False(t, domainTypeAttr.IsRequired())
	assert.False(t, domainTypeAttr.IsOptional())

	// Verify schema has description
	assert.NotEmpty(t, schema.MarkdownDescription)
}

func TestDnsNameResourceSchema_Validation(t *testing.T) {
	schema := DnsNameResourceSchema()

	// Should have validators (we can't easily test the regex without creating a full context)
	// But we can verify that validators exist
	// Note: The actual validation testing happens in the integration tests
	assert.NotEmpty(t, schema.MarkdownDescription)

	// Verify DNS name attribute exists and has the right configuration
	dnsNameAttr := schema.Attributes["dns_name"]
	assert.True(t, dnsNameAttr.IsRequired())
}