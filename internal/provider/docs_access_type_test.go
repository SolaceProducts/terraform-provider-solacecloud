package provider_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"terraform-provider-solacecloud/internal/provider/connectionendpoint"
	"terraform-provider-solacecloud/internal/provider/service"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// quotedUpperValue matches an upper-case value in backticks or double quotes, such as `PUBLIC` or "PRIVATE".
var quotedUpperValue = regexp.MustCompile("[`\"]([A-Z][A-Z_]+)[`\"]")

// TestDocsAccessTypeValuesMatchSchema checks that documented access_type values pass both schemas' validators.
func TestDocsAccessTypeValuesMatchSchema(t *testing.T) {
	validators := map[string][]validator.String{
		"solacecloud_connection_endpoint":         accessTypeValidators(t, connectionendpoint.NewConnectionEndpointResource(), ""),
		"solacecloud_service.connection_endpoint": accessTypeValidators(t, service.NewServiceResource(), "connection_endpoint"),
	}

	// guard against a vacuous pass: the validators must reject an obviously invalid value
	for name, vs := range validators {
		if accepts(vs, "NOT_AN_ACCESS_TYPE") {
			t.Fatalf("%s access_type validators accept NOT_AN_ACCESS_TYPE; the test cannot detect a wrong documented value", name)
		}
	}

	found := 0
	for _, root := range []string{"../../docs", "../../examples"} {
		for _, doc := range documentedAccessTypes(t, root) {
			found++
			for name, vs := range validators {
				if !accepts(vs, doc.value) {
					t.Errorf("%s documents access_type value %q, which %s rejects", doc.where, doc.value, name)
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("found no documented access_type values; the docs layout or the pattern changed")
	}
}

type documentedValue struct {
	where string
	value string
}

// documentedAccessTypes returns the quoted upper-case values on lines under root that mention access_type.
func documentedAccessTypes(t *testing.T, root string) []documentedValue {
	t.Helper()
	var values []documentedValue
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".terraform" {
				return filepath.SkipDir
			}
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(content), "\n") {
			if !strings.Contains(line, "access_type") {
				continue
			}
			for _, m := range quotedUpperValue.FindAllStringSubmatch(line, -1) {
				values = append(values, documentedValue{where: fmt.Sprintf("%s:%d", path, i+1), value: m[1]})
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return values
}

// accessTypeValidators returns the access_type validators, nested under block when it is set.
func accessTypeValidators(t *testing.T, r resource.Resource, block string) []validator.String {
	t.Helper()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	attributes := resp.Schema.Attributes
	if block != "" {
		nested, ok := attributes[block].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s is not a single nested attribute", block)
		}
		attributes = nested.Attributes
	}
	accessType, ok := attributes["access_type"].(schema.StringAttribute)
	if !ok {
		t.Fatal("access_type is not a string attribute")
	}
	return accessType.Validators
}

func accepts(validators []validator.String, value string) bool {
	for _, v := range validators {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue(value)}, resp)
		if resp.Diagnostics.HasError() {
			return false
		}
	}
	return true
}
