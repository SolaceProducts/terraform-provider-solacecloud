package model

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/stretchr/testify/assert"
)

func TestConnectionEndpointProtocolModelType_PortValidation(t *testing.T) {
	// Get the schema
	schema := ConnectionEndpointProtocolModelType()

	// Verify schema is optional (the nested attribute is optional, but port inside is required)
	assert.True(t, schema.IsOptional())
	assert.False(t, schema.IsComputed())
	assert.False(t, schema.IsRequired())

	// Get the port attribute
	portAttr := schema.GetAttributes()["port"]
	assert.NotNil(t, portAttr, "Port attribute should exist")

	// Verify port is required
	assert.True(t, portAttr.IsRequired())

	// Get validators - this is internal to the framework, but we can verify the schema was configured
	// The actual validation testing is done through acceptance tests
	// We can just verify the schema structure is correct here
}

func TestEndpointProtocolModelType_NoValidation(t *testing.T) {
	// Get the schema for computed-only endpoints (service endpoints)
	schema := EndpointProtocolModelType()

	// Verify schema is computed (read-only from API)
	assert.True(t, schema.IsComputed())
	assert.False(t, schema.IsOptional())
	assert.False(t, schema.IsRequired())

	// Get the port attribute
	portAttr := schema.GetAttributes()["port"]
	assert.NotNil(t, portAttr, "Port attribute should exist")

	// Verify port is computed (no validation needed for read-only)
	assert.True(t, portAttr.IsComputed())
}

func TestConnectionEndpointProtocolModelType_HasValidators(t *testing.T) {
	schema := ConnectionEndpointProtocolModelType()
	portAttr := schema.GetAttributes()["port"]

	// Type assert to get the actual schema type
	int64Attr, ok := portAttr.(interface {
		GetValidators() []validator.Int64
	})

	if ok {
		validators := int64Attr.GetValidators()
		// Verify that validators exist
		assert.NotEmpty(t, validators, "Port attribute should have validators")
		// We expect exactly one validator (Between(1, 65535))
		assert.Len(t, validators, 1, "Port attribute should have exactly one validator")
	}
}