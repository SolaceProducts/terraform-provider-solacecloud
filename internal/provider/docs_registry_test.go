package provider_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"terraform-provider-solacecloud/internal/provider"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

const docsRoot = "../../docs"

// registryDocDirs are the docs/ subdirectories the Terraform Registry publishes; it skips nested directories in them.
var registryDocDirs = []string{"guides", "resources", "data-sources"}

// relativeMarkdownLink matches the target of a relative link to a markdown page, such as (service_lifecycle.md#upgrades).
var relativeMarkdownLink = regexp.MustCompile(`\]\(([^):#]+\.md)(#[^)]*)?\)`)

// hclBlock matches a fenced HCL code block.
var hclBlock = regexp.MustCompile("(?s)```(?:hcl|terraform)\n(.*?)```")

// connectionEndpointBlock matches a connection_endpoint object argument, whatever its alignment.
var connectionEndpointBlock = regexp.MustCompile(`(?m)^\s*connection_endpoint\s*=\s*\{`)

func TestDocsRegistryLayout(t *testing.T) {
	for _, dir := range registryDocDirs {
		entries, err := os.ReadDir(filepath.Join(docsRoot, dir))
		if err != nil {
			t.Fatalf("reading docs/%s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				t.Errorf("docs/%s/%s is a directory; the registry does not publish nested pages, use subcategory frontmatter", dir, e.Name())
			} else if filepath.Ext(e.Name()) != ".md" {
				t.Errorf("docs/%s/%s is not a .md page", dir, e.Name())
			}
		}
	}

	rootEntries, err := os.ReadDir(docsRoot)
	if err != nil {
		t.Fatalf("reading docs: %v", err)
	}
	for _, e := range rootEntries {
		if e.IsDir() && !slices.Contains(registryDocDirs, e.Name()) {
			t.Errorf("docs/%s/ is not a directory the registry publishes (%v); move its pages to docs/guides", e.Name(), registryDocDirs)
		} else if !e.IsDir() && e.Name() != "index.md" {
			t.Errorf("docs/%s is not published by the registry; move it to docs/guides", e.Name())
		}
	}
}

func TestDocsRegistryFrontmatter(t *testing.T) {
	for _, page := range registryPages(t) {
		frontmatter, ok := frontmatterOf(t, page)
		if !ok {
			t.Errorf("%s has no frontmatter", page)
			continue
		}
		if !strings.Contains(frontmatter, "page_title:") {
			t.Errorf("%s frontmatter has no page_title", page)
		}
	}
}

func TestDocsEveryResourceAndDataSourceHasAPage(t *testing.T) {
	p := provider.New("test")()
	for _, newResource := range p.Resources(context.Background()) {
		resp := &resource.MetadataResponse{}
		newResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "solacecloud"}, resp)
		assertPageExists(t, "resources", resp.TypeName)
	}
	for _, newDataSource := range p.DataSources(context.Background()) {
		resp := &datasource.MetadataResponse{}
		newDataSource().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "solacecloud"}, resp)
		assertPageExists(t, "data-sources", resp.TypeName)
	}
}

func TestDocsServiceExampleSetsConnectionEndpoint(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(docsRoot, "resources", "service.md"))
	if err != nil {
		t.Fatalf("reading service.md: %v", err)
	}
	for _, m := range hclBlock.FindAllStringSubmatch(string(content), -1) {
		if strings.Contains(m[1], `resource "solacecloud_service"`) && connectionEndpointBlock.MatchString(m[1]) {
			return
		}
	}
	t.Error("docs/resources/service.md has no solacecloud_service example that sets connection_endpoint")
}

func TestDocsRelativeLinksResolve(t *testing.T) {
	found := 0
	defer func() {
		if found == 0 && !t.Failed() {
			t.Error("no relative markdown links found; the link pattern is probably broken")
		}
	}()
	for _, page := range registryPages(t) {
		content, err := os.ReadFile(page)
		if err != nil {
			t.Fatalf("reading %s: %v", page, err)
		}
		for _, m := range relativeMarkdownLink.FindAllStringSubmatch(string(content), -1) {
			found++
			target := filepath.Join(filepath.Dir(page), m[1])
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s links to %s, which does not exist", page, m[1])
			} else if !isPublishedPage(target) {
				t.Errorf("%s links to %s, which the registry does not publish", page, m[1])
			}
		}
	}
}

// isPublishedPage reports whether path is docs/index.md or a page under a published docs directory.
func isPublishedPage(path string) bool {
	rel, err := filepath.Rel(docsRoot, path)
	if err != nil {
		return false
	}
	if rel == "index.md" {
		return true
	}
	dir, _, nested := strings.Cut(filepath.ToSlash(rel), "/")
	return nested && slices.Contains(registryDocDirs, dir)
}

// registryPages returns docs/index.md and every page in the published docs directories, nested ones included.
func registryPages(t *testing.T) []string {
	t.Helper()
	pages := []string{filepath.Join(docsRoot, "index.md")}
	for _, dir := range registryDocDirs {
		err := filepath.WalkDir(filepath.Join(docsRoot, dir), func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && filepath.Ext(path) == ".md" {
				pages = append(pages, path)
			}
			return err
		})
		if err != nil {
			t.Fatalf("walking docs/%s: %v", dir, err)
		}
	}
	return pages
}

// frontmatterOf returns the YAML frontmatter between the leading --- lines of page.
func frontmatterOf(t *testing.T, page string) (string, bool) {
	t.Helper()
	content, err := os.ReadFile(page)
	if err != nil {
		t.Fatalf("reading %s: %v", page, err)
	}
	rest, ok := strings.CutPrefix(string(content), "---\n")
	if !ok {
		return "", false
	}
	frontmatter, _, ok := strings.Cut(rest, "\n---\n")
	return frontmatter, ok
}

func assertPageExists(t *testing.T, dir, typeName string) {
	t.Helper()
	name := strings.TrimPrefix(typeName, "solacecloud_") + ".md"
	if _, err := os.Stat(filepath.Join(docsRoot, dir, name)); err != nil {
		t.Errorf("%s has no page docs/%s/%s", typeName, dir, name)
	}
}
