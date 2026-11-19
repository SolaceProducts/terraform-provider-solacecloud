package provider_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"terraform-provider-solacecloud/internal/provider"
	"terraform-provider-solacecloud/internal/provider/service"
	"testing"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// importTo matches the resource type in an import block's to argument, such as to = solacecloud_service.x.
var importTo = regexp.MustCompile(`(?m)^\s*to\s*=\s*([a-z_]+)\.`)

func TestDocsProviderArgumentsUnderMatchingHeading(t *testing.T) {
	resp := &fwprovider.SchemaResponse{}
	provider.New("test")().Schema(context.Background(), fwprovider.SchemaRequest{}, resp)
	content := readDoc(t, "index.md")

	for name, attr := range resp.Schema.Attributes {
		want := "Optional"
		if attr.IsRequired() {
			want = "Required"
		}
		if got := docSectionOf(content, "- `"+name+"`"); got != want {
			t.Errorf("docs/index.md lists %s under %q; the provider schema makes it %s", name, got, want)
		}
	}
}

func TestDocsServiceConnectionEndpointPortsRequirement(t *testing.T) {
	resp := &resource.SchemaResponse{}
	service.NewServiceResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	block, ok := resp.Schema.Attributes["connection_endpoint"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("solacecloud_service connection_endpoint is not a single nested attribute; update this test")
	}
	ports, ok := block.Attributes["ports"]
	if !ok {
		t.Fatal("solacecloud_service connection_endpoint has no ports attribute; update this test")
	}
	want := "(Optional)"
	if ports.IsRequired() {
		want = "(Required)"
	}
	if !strings.Contains(readDoc(t, "resources/service.md"), "* `ports` - "+want) {
		t.Errorf("docs/resources/service.md does not mark connection_endpoint.ports %s, as the schema does", want)
	}
}

func TestDocsImportBlocksNameRegisteredResources(t *testing.T) {
	registered := map[string]bool{}
	for _, newResource := range provider.New("test")().Resources(context.Background()) {
		resp := &resource.MetadataResponse{}
		newResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "solacecloud"}, resp)
		registered[resp.TypeName] = true
	}

	found := 0
	for _, page := range registryPages(t) {
		content, err := os.ReadFile(page)
		if err != nil {
			t.Fatalf("reading %s: %v", page, err)
		}
		for _, m := range importTo.FindAllStringSubmatch(string(content), -1) {
			found++
			if !registered[m[1]] {
				t.Errorf("%s imports into %s, which the provider does not register", page, m[1])
			}
		}
	}
	if found == 0 {
		t.Fatal("found no import blocks in the docs; the docs layout or the pattern changed")
	}
}

func TestDocsUseRealDomainsAndPaths(t *testing.T) {
	// The managed domain comes from the API's generated hostnames; the path segment from endpoint_manager.go's import split.
	wrong := map[string]string{
		"messaging.solace.com": "messaging.solace.cloud",
		"connectioneEndpoints": "connectionEndpoints",
	}
	for _, page := range registryPages(t) {
		content, err := os.ReadFile(page)
		if err != nil {
			t.Fatalf("reading %s: %v", page, err)
		}
		for bad, good := range wrong {
			if strings.Contains(string(content), bad) {
				t.Errorf("%s contains %q; use %q", page, bad, good)
			}
		}
	}
}

func readDoc(t *testing.T, rel string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(docsRoot, rel))
	if err != nil {
		t.Fatalf("reading docs/%s: %v", rel, err)
	}
	return string(content)
}

func docSectionOf(content, prefix string) string {
	section := ""
	inSchema := false
	for _, line := range strings.Split(content, "\n") {
		if h, ok := strings.CutPrefix(line, "## "); ok {
			inSchema = strings.TrimSpace(h) == "Schema"
			section = ""
			continue
		}
		if !inSchema {
			continue
		}
		if h, ok := strings.CutPrefix(line, "### "); ok {
			section = strings.TrimSpace(h)
		}
		if strings.HasPrefix(line, prefix) {
			return section
		}
	}
	return "not listed"
}

func TestDocSectionOfIgnoresBulletsOutsideSchema(t *testing.T) {
	content := "## Authentication\n\n### Environment Variables\n\n- `base_url` - env\n\n## Schema\n\n### Optional\n\n- `base_url` - schema\n"
	if got := docSectionOf(content, "- `base_url`"); got != "Optional" {
		t.Errorf("got %q, want Optional", got)
	}
	if got := docSectionOf("## Authentication\n\n### Environment Variables\n\n- `x` - env\n", "- `x`"); got != "not listed" {
		t.Errorf("got %q, want not listed", got)
	}
}
