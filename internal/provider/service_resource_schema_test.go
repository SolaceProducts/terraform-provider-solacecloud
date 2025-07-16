package provider_test

import (
	"context"
	"terraform-provider-solacecloud/internal/provider"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestCorrectFieldsHaveImmutableStringPlanModifier(t *testing.T) {

	// Create a new service resource
	serviceResource := provider.NewServiceResource()

	// Get the schema
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	serviceResource.Schema(context.Background(), schemaReq, schemaResp)

	// Fields that should have ImmutableStringPlanModifier
	fieldsToCheck := []string{
		"cluster_name",
		"datacenter_id",
		"environment_id",
		"message_vpn_name",
		"service_class_id",
		"event_broker_version",
	}

	searchResults := recursiveSearchForAttributes(schemaResp, fieldsToCheck)
	// Check each field
	for fieldName, field := range searchResults {
		t.Run(fieldName, func(t *testing.T) {
			if searchResults == nil {
				t.Errorf("Field %s does not exist in schema", fieldName)
				return
			}

			stringAttr, ok := field.(schema.StringAttribute)
			if !ok {
				t.Errorf("Field %s is not a StringAttribute", fieldName)
				return
			}

			// Check if ImmutableStringPlanModifier is present
			hasImmutableModifier := false
			for _, modifier := range stringAttr.PlanModifiers {
				// This will need to be updated when ImmutableStringPlanModifier is implemented
				if _, ok := modifier.(provider.ImmutableStringPlanModifier); ok {
					hasImmutableModifier = true
					break
				}
			}

			if !hasImmutableModifier {
				t.Errorf("Field %s does not have ImmutableStringPlanModifier", fieldName)
			}
		})
	}
}

func TestCorrectFieldsHaveSensitive(t *testing.T) {

	// Create a new service resource
	serviceResource := provider.NewServiceResource()

	// Get the schema
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	serviceResource.Schema(context.Background(), schemaReq, schemaResp)

	// Fields that should have ImmutableStringPlanModifier
	fieldsToCheck := []string{
		"password",
	}

	attributes := recursiveSearchForAttributes(schemaResp, fieldsToCheck)
	if len(attributes) < 1 {
		t.Errorf("One of the fields does not exist in the schema")
		return
	}

	// Check each field
	for fieldName, attribute := range attributes {
		t.Run(fieldName, func(t *testing.T) {

			stringAttr, ok := attribute.(schema.StringAttribute)
			if !ok {
				t.Errorf("Field %s is not a StringAttribute", fieldName)
				return
			}

			// Check if ImmutableStringPlanModifier is present
			hasSensitive := stringAttr.IsSensitive()

			if !hasSensitive {
				t.Errorf("Field %s does not have Sensitive", fieldName)
			}
		})
	}
}

/*
*
Function that searches for multiple attributes and returns a map of found attributes
*
*/
func recursiveSearchForAttributes(res *resource.SchemaResponse, attributeNames []string) map[string]schema.Attribute {
	if res == nil || res.Schema.Attributes == nil {
		return make(map[string]schema.Attribute)
	}

	foundAttributes := make(map[string]schema.Attribute)

	// Search for each attribute name
	for _, attributeName := range attributeNames {
		// Search in top-level attributes
		if attr, exists := res.Schema.Attributes[attributeName]; exists {
			foundAttributes[attributeName] = attr
			continue
		}

		// Recursively search in nested attributes
		found := false
		for _, attr := range res.Schema.Attributes {
			if foundAttr := searchInAttribute(attr, attributeName); foundAttr != nil {
				foundAttributes[attributeName] = foundAttr
				found = true
				break
			}
		}

		if found {
			continue
		}

		// Search in blocks
		for _, block := range res.Schema.Blocks {
			if foundAttr := searchInBlock(block, attributeName); foundAttr != nil {
				foundAttributes[attributeName] = foundAttr
				break
			}
		}
	}

	return foundAttributes
}

// Helper function to search within an attribute for nested attributes
func searchInAttribute(attr schema.Attribute, attributeName string) schema.Attribute {
	switch a := attr.(type) {
	case schema.SingleNestedAttribute:
		if foundAttr, exists := a.Attributes[attributeName]; exists {
			return foundAttr
		}
		for _, nestedAttr := range a.Attributes {
			if found := searchInAttribute(nestedAttr, attributeName); found != nil {
				return found
			}
		}
	case schema.ListNestedAttribute:
		if foundAttr, exists := a.NestedObject.Attributes[attributeName]; exists {
			return foundAttr
		}
		for _, nestedAttr := range a.NestedObject.Attributes {
			if found := searchInAttribute(nestedAttr, attributeName); found != nil {
				return found
			}
		}
	case schema.SetNestedAttribute:
		if foundAttr, exists := a.NestedObject.Attributes[attributeName]; exists {
			return foundAttr
		}
		for _, nestedAttr := range a.NestedObject.Attributes {
			if found := searchInAttribute(nestedAttr, attributeName); found != nil {
				return found
			}
		}
	case schema.MapNestedAttribute:
		if foundAttr, exists := a.NestedObject.Attributes[attributeName]; exists {
			return foundAttr
		}
		for _, nestedAttr := range a.NestedObject.Attributes {
			if found := searchInAttribute(nestedAttr, attributeName); found != nil {
				return found
			}
		}
	}
	return nil
}

// Helper function to search within a block for nested attributes
func searchInBlock(block schema.Block, attributeName string) schema.Attribute {
	switch b := block.(type) {
	case schema.SingleNestedBlock:
		if foundAttr, exists := b.Attributes[attributeName]; exists {
			return foundAttr
		}
		for _, attr := range b.Attributes {
			if found := searchInAttribute(attr, attributeName); found != nil {
				return found
			}
		}
		for _, nestedBlock := range b.Blocks {
			if found := searchInBlock(nestedBlock, attributeName); found != nil {
				return found
			}
		}
	case schema.ListNestedBlock:
		if foundAttr, exists := b.NestedObject.Attributes[attributeName]; exists {
			return foundAttr
		}
		for _, attr := range b.NestedObject.Attributes {
			if found := searchInAttribute(attr, attributeName); found != nil {
				return found
			}
		}
		for _, nestedBlock := range b.NestedObject.Blocks {
			if found := searchInBlock(nestedBlock, attributeName); found != nil {
				return found
			}
		}
	case schema.SetNestedBlock:
		if foundAttr, exists := b.NestedObject.Attributes[attributeName]; exists {
			return foundAttr
		}
		for _, attr := range b.NestedObject.Attributes {
			if found := searchInAttribute(attr, attributeName); found != nil {
				return found
			}
		}
		for _, nestedBlock := range b.NestedObject.Blocks {
			if found := searchInBlock(nestedBlock, attributeName); found != nil {
				return found
			}
		}
	}
	return nil
}
