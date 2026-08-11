package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"terraform-provider-solacecloud/internal/provider/service"
	"terraform-provider-solacecloud/missioncontrol"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	specPath = "../../../missioncontrol/MissionControl-specs.json"
	docsPath = "../../../docs/resources/service.md"
)

func serviceClassIdAttribute(t *testing.T) schema.StringAttribute {
	t.Helper()

	schemaResp := &resource.SchemaResponse{}
	service.NewServiceResource().Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	attributes := recursiveSearchForAttributes(schemaResp, []string{"service_class_id"})
	found, exists := attributes["service_class_id"]
	if !exists {
		t.Fatal("service_class_id does not exist in the schema")
	}

	stringAttr, ok := found.(schema.StringAttribute)
	if !ok {
		t.Fatal("service_class_id is not a StringAttribute")
	}
	return stringAttr
}

func validateServiceClassId(t *testing.T, value string) error {
	t.Helper()

	req := validator.StringRequest{ConfigValue: types.StringValue(value)}
	resp := &validator.StringResponse{}
	for _, v := range serviceClassIdAttribute(t).Validators {
		v.ValidateString(context.Background(), req, resp)
	}

	if !resp.Diagnostics.HasError() {
		return nil
	}

	details := make([]string, 0, len(resp.Diagnostics.Errors()))
	for _, diagnostic := range resp.Diagnostics.Errors() {
		details = append(details, diagnostic.Detail())
	}
	return errors.New(strings.Join(details, "; "))
}

func specServiceClasses(t *testing.T) []string {
	t.Helper()

	content, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("could not read the Mission Control spec: %s", err)
	}

	var spec struct {
		Components struct {
			Schemas struct {
				ServiceClassId struct {
					Enum []string `json:"enum"`
				} `json:"ServiceClassId"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(content, &spec); err != nil {
		t.Fatalf("could not parse the Mission Control spec: %s", err)
	}

	classes := spec.Components.Schemas.ServiceClassId.Enum
	if len(classes) == 0 {
		t.Fatal("no service classes found in the Mission Control spec")
	}

	return sorted(classes)
}

func documentedServiceClasses(t *testing.T) []string {
	t.Helper()

	content, err := os.ReadFile(docsPath)
	if err != nil {
		t.Fatalf("could not read the service documentation: %s", err)
	}

	pattern := regexp.MustCompile("(?m)^\\s+\\* `((?:DEVELOPER|ENTERPRISE_)[A-Z0-9_]*)`$")
	matches := pattern.FindAllStringSubmatch(string(content), -1)
	if len(matches) == 0 {
		t.Fatal("no service classes found in docs/resources/service.md")
	}

	classes := make([]string, 0, len(matches))
	for _, match := range matches {
		classes = append(classes, match[1])
	}

	return sorted(classes)
}

func validatedServiceClasses(t *testing.T, candidates []string) []string {
	t.Helper()

	accepted := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if validateServiceClassId(t, candidate) == nil {
			accepted = append(accepted, candidate)
		}
	}

	return sorted(accepted)
}

func sorted(values []string) []string {
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return copied
}

func TestSchemaValidatorMatchesSpecServiceClasses(t *testing.T) {
	spec := specServiceClasses(t)
	validated := validatedServiceClasses(t, spec)

	if strings.Join(spec, ",") != strings.Join(validated, ",") {
		t.Errorf("the Mission Control spec and the schema validator disagree.\nspec:      %v\nvalidator: %v", spec, validated)
	}
}

func TestDocumentationMatchesSpecServiceClasses(t *testing.T) {
	spec := specServiceClasses(t)
	documented := documentedServiceClasses(t)

	if strings.Join(spec, ",") != strings.Join(documented, ",") {
		t.Errorf("the Mission Control spec and the documentation disagree.\nspec: %v\ndocs: %v", spec, documented)
	}
}

func TestSchemaValidatorAcceptsEverySpecServiceClass(t *testing.T) {
	for _, serviceClassId := range specServiceClasses(t) {
		t.Run(serviceClassId, func(t *testing.T) {
			if err := validateServiceClassId(t, serviceClassId); err != nil {
				t.Errorf("service class %s was rejected but the spec supports it: %s", serviceClassId, err)
			}
		})
	}
}

func TestSpecDefines200KServiceClasses(t *testing.T) {
	spec := specServiceClasses(t)

	for _, serviceClassId := range []string{"ENTERPRISE_200K_STANDALONE", "ENTERPRISE_200K_HIGHAVAILABILITY"} {
		t.Run(serviceClassId, func(t *testing.T) {
			if !slices.Contains(spec, serviceClassId) {
				t.Errorf("the Mission Control spec does not define %s", serviceClassId)
			}
		})
	}
}

func TestServiceClassIdRejectsInvalidClasses(t *testing.T) {
	invalidClasses := []string{
		"ENTERPRISE_200K",
		"ENTERPRISE_200000_STANDALONE",
		"ENTERPRISE_200K_SA",
		"enterprise_200k_standalone",
		"ENTERPRISE_500K_STANDALONE",
		"",
	}

	for _, serviceClassId := range invalidClasses {
		t.Run(serviceClassId, func(t *testing.T) {
			if validateServiceClassId(t, serviceClassId) == nil {
				t.Errorf("service class %q was accepted but should be rejected", serviceClassId)
			}
		})
	}
}

func TestServiceClassIdDefaultsToDeveloper(t *testing.T) {
	defaultValue := serviceClassIdAttribute(t).Default
	if defaultValue == nil {
		t.Fatal("service_class_id has no default")
	}

	resp := &defaults.StringResponse{}
	defaultValue.DefaultString(context.Background(), defaults.StringRequest{}, resp)

	if resp.PlanValue.ValueString() != "DEVELOPER" {
		t.Errorf("expected default DEVELOPER, got %s", resp.PlanValue.ValueString())
	}
}

func TestGeneratedClientDefines200KServiceClasses(t *testing.T) {
	testCases := map[string]struct {
		actual   string
		expected string
	}{
		"ServiceClassId standalone": {
			actual:   string(missioncontrol.ServiceClassIdENTERPRISE200KSTANDALONE),
			expected: "ENTERPRISE_200K_STANDALONE",
		},
		"ServiceClassId high availability": {
			actual:   string(missioncontrol.ServiceClassIdENTERPRISE200KHIGHAVAILABILITY),
			expected: "ENTERPRISE_200K_HIGHAVAILABILITY",
		},
		"GetServiceClassParamsId standalone": {
			actual:   string(missioncontrol.GetServiceClassParamsIdENTERPRISE200KSTANDALONE),
			expected: "ENTERPRISE_200K_STANDALONE",
		},
		"GetServiceClassParamsId high availability": {
			actual:   string(missioncontrol.GetServiceClassParamsIdENTERPRISE200KHIGHAVAILABILITY),
			expected: "ENTERPRISE_200K_HIGHAVAILABILITY",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			if testCase.actual != testCase.expected {
				t.Errorf("expected %s, got %s", testCase.expected, testCase.actual)
			}
		})
	}
}
