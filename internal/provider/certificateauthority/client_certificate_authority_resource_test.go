package certificateauthority_test

import (
	"encoding/json"
	"encoding/pem"
	"fmt"
	"regexp"
	"strings"
	"terraform-provider-solacecloud/internal"
	"terraform-provider-solacecloud/internal/provider"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"solacecloud": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func mockedInstance(t *testing.T) *internal.TestInstance {
	instance := internal.NewTestInstance()
	instance.Init(internal.ConfigurableParams{})
	if !instance.IsMocked() {
		t.Skip("certificate authority tests run against mocks only")
	}
	return instance
}

func baseHcl(instance *internal.TestInstance) string {
	return fmt.Sprintf(`
provider "solacecloud" {
  base_url             = %q
  api_polling_interval = 1
  api_token            = "mocked_api_token"
}
`, instance.GetBaseURL())
}

func clientCaHcl(serviceId, name, cert, extra string) string {
	return fmt.Sprintf(`
resource "solacecloud_client_certificate_authority" %q {
  service_id   = %q
  name         = %q
  cert_content = %q
%s
}
`, name, serviceId, name, cert, extra)
}

func domainCaHcl(serviceId, name, cert string) string {
	return fmt.Sprintf(`
resource "solacecloud_domain_certificate_authority" %q {
  service_id   = %q
  name         = %q
  cert_content = %q
}
`, name, serviceId, name, cert)
}

func checkLastWrite(mock *internal.CertificateAuthorityMock, method, name string, fields map[string]any) resource.TestCheckFunc {
	return func(*terraform.State) error {
		writes := mock.Writes()
		if len(writes) == 0 {
			return fmt.Errorf("expected a %s, the mock saw no write", method)
		}
		last := writes[len(writes)-1]
		if last.Method != method || last.Name != name {
			return fmt.Errorf("expected the last write to be %s %s, got %s %s", method, name, last.Method, last.Name)
		}
		if len(last.Body) != len(fields) {
			return fmt.Errorf("expected the last %s to send only %v, it sent %v", method, fields, last.Body)
		}
		for field, value := range fields {
			want, _ := json.Marshal(value)
			if string(last.Body[field]) != string(want) {
				return fmt.Errorf("expected the last %s to send %s = %s, it sent %s", method, field, want, last.Body[field])
			}
		}
		return nil
	}
}

func checkExists(mock *internal.CertificateAuthorityMock, kind, name string, exists bool) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if mock.Exists(kind, name) != exists {
			return fmt.Errorf("expected %s %s exists = %v", kind, name, exists)
		}
		return nil
	}
}

func TestClientCertificateAuthorityLifecycle(t *testing.T) {
	instance := mockedInstance(t)
	serviceId := "ca-lifecycle-service"
	mock := instance.SetupCertificateAuthorityMocks(serviceId)
	const name = "ca-a"
	const resourceName = "solacecloud_client_certificate_authority." + name
	certA, certB := internal.CertificatePem("ca-a"), internal.CertificatePem("ca-a-renewed")
	config := func(cert, extra string) string { return baseHcl(instance) + clientCaHcl(serviceId, name, cert, extra) }

	withOcsp := `
  revocation_check_enabled = true
  ocsp_override_url        = "http://ocsp.example.com"
  ocsp_timeout             = 20`
	withoutOverride := `
  revocation_check_enabled = true
  ocsp_timeout             = 20`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(certA, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", serviceId+"/"+name),
					resource.TestCheckResourceAttr(resourceName, "revocation_check_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "ocsp_timeout", "5"),
					resource.TestCheckResourceAttr(resourceName, "ocsp_non_responder_cert_enabled", "false"),
					resource.TestCheckNoResourceAttr(resourceName, "ocsp_override_url"),
					checkExists(mock, internal.ClientCertificateAuthorities, name, true),
				),
			},
			{Config: config(certA, ""), PlanOnly: true},
			{
				Config: config(certB, ""),
				Check:  checkLastWrite(mock, "PATCH", name, map[string]any{"certContent": certB}),
			},
			{
				Config: config(certB, withOcsp),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "revocation_check_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "ocsp_override_url", "http://ocsp.example.com"),
					resource.TestCheckResourceAttr(resourceName, "ocsp_timeout", "20"),
					checkLastWrite(mock, "PATCH", name, map[string]any{
						"revocationCheckEnabled": true, "ocspOverrideUrl": "http://ocsp.example.com", "ocspTimeout": 20,
					}),
				),
			},
			{
				Config: config(certB, withoutOverride),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "ocsp_override_url"),
					checkLastWrite(mock, "PATCH", name, map[string]any{"ocspOverrideUrl": ""}),
				),
			},
			{Config: config(certB, withoutOverride), PlanOnly: true},
			{
				PreConfig: func() { mock.Put(internal.ClientCertificateAuthorities, name, `{"ocspOverrideUrl": ""}`) },
				Config:    config(certB, withoutOverride),
				PlanOnly:  true,
			},
			{
				PreConfig: func() { mock.Put(internal.ClientCertificateAuthorities, name, `{"ocspTimeout": 30}`) },
				Config:    config(certB, withoutOverride),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "ocsp_timeout", "20"),
					checkLastWrite(mock, "PATCH", name, map[string]any{"ocspTimeout": 20}),
				),
			},
			{
				PreConfig: func() { mock.Remove(internal.ClientCertificateAuthorities, name) },
				Config:    config(certB, withoutOverride),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate)},
				},
				Check: checkExists(mock, internal.ClientCertificateAuthorities, name, true),
			},
			{
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: serviceId + "/",
				ExpectError:   regexp.MustCompile(`Invalid import ID`),
			},
			{
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: "/" + name,
				ExpectError:   regexp.MustCompile(`Invalid import ID`),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"cert_content"},
			},
		},
		CheckDestroy: checkExists(mock, internal.ClientCertificateAuthorities, name, false),
	})
}

func TestClientCertificateAuthorityImportThenApplyResendsCertificate(t *testing.T) {
	instance := mockedInstance(t)
	serviceId := "ca-import-service"
	mock := instance.SetupCertificateAuthorityMocks(serviceId)
	const name = "imported"
	const resourceName = "solacecloud_client_certificate_authority." + name
	mock.Put(internal.ClientCertificateAuthorities, name, `{"revocationCheckEnabled": true, "ocspTimeout": 20, "ocspNonResponderCertEnabled": false}`)
	cert := internal.CertificatePem(name)
	config := baseHcl(instance) + clientCaHcl(serviceId, name, cert, `
  revocation_check_enabled = true
  ocsp_timeout             = 20`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             config,
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStateId:      serviceId + "/" + name,
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					attributes := states[0].Attributes
					if attributes["ocsp_timeout"] != "20" || attributes["revocation_check_enabled"] != "true" {
						return fmt.Errorf("expected the OCSP settings to be read on import, got %v", attributes)
					}
					if _, present := attributes["cert_content"]; present {
						return fmt.Errorf("expected cert_content to be unknown after import, got %q", attributes["cert_content"])
					}
					return nil
				},
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: checkLastWrite(mock, "PATCH", name, map[string]any{"certContent": cert}),
			},
			{Config: config, PlanOnly: true},
		},
	})
}

func TestClientCertificateAuthorityFailedCreateIsTrackedAndReplaced(t *testing.T) {
	instance := mockedInstance(t)
	serviceId := "ca-failed-create-service"
	mock := instance.SetupCertificateAuthorityMocks(serviceId)
	const name = "half-made"
	const resourceName = "solacecloud_client_certificate_authority." + name
	config := baseHcl(instance) + clientCaHcl(serviceId, name, internal.CertificatePem(name), "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig:   func() { mock.FailNextJob("The event broker service could not apply the certificate authority") },
				Config:      config,
				ExpectError: regexp.MustCompile(`could\s+not\s+apply\s+the\s+certificate\s+authority`),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate)},
				},
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						writes := mock.Writes()
						if n := len(writes); n < 3 || writes[n-2].Method != "DELETE" || writes[n-1].Method != "POST" {
							return fmt.Errorf("expected the tainted certificate authority to be deleted and created again, writes: %v", writes)
						}
						return nil
					},
					checkExists(mock, internal.ClientCertificateAuthorities, name, true),
				),
			},
		},
	})
}

func TestClientCertificateAuthorityRejected(t *testing.T) {
	instance := mockedInstance(t)
	serviceId := "ca-rejected-service"
	mock := instance.SetupCertificateAuthorityMocks(serviceId)
	mock.Put(internal.ClientCertificateAuthorities, "taken", `{"revocationCheckEnabled": false, "ocspTimeout": 5, "ocspNonResponderCertEnabled": false}`)
	cert := internal.CertificatePem("rejected")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      baseHcl(instance) + clientCaHcl(serviceId, "not-pem", "not a certificate", ""),
				ExpectError: regexp.MustCompile(`must\s+be\s+a\s+PEM-encoded\s+X\.509\s+certificate`),
			},
			{
				Config:      baseHcl(instance) + clientCaHcl(serviceId, "two-certs", cert+cert, ""),
				ExpectError: regexp.MustCompile(`must\s+contain\s+exactly\s+one\s+certificate`),
			},
			{
				Config:      baseHcl(instance) + clientCaHcl(serviceId, "taken", cert, ""),
				ExpectError: regexp.MustCompile(`terraform\s+import\s+<resource\s+address>\s+ca-rejected-service/taken(.|\n)*A\s+certificate\s+authority\s+named\s+taken\s+already\s+exists`),
			},
			{
				Config:      baseHcl(instance) + clientCaHcl(serviceId, "with-key", cert+string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("not a real key")})), ""),
				ExpectError: regexp.MustCompile(`contains\s+a\s+private\s+key`),
			},
			{
				Config:      baseHcl(instance) + clientCaHcl(serviceId, "with-damaged-key", cert+"-----BEGIN EC PRIVATE KEY-----\nnot base64!\n-----END EC PRIVATE KEY-----\n", ""),
				ExpectError: regexp.MustCompile(`contains\s+a\s+private\s+key`),
			},
			{
				Config:      baseHcl(instance) + clientCaHcl(serviceId, strings.Repeat("a", 65), cert, ""),
				ExpectError: regexp.MustCompile(`string\s+length\s+must\s+be\s+between\s+1\s+and\s+64`),
			},
			{
				Config:      baseHcl(instance) + clientCaHcl(serviceId, "empty-url", cert, `  ocsp_override_url = ""`),
				ExpectError: regexp.MustCompile(`must\s+include\s+a\s+scheme\s+other\s+than\s+https`),
			},
			{
				Config:      baseHcl(instance) + clientCaHcl(serviceId, "https-url", cert, `  ocsp_override_url = "HTTPS://ocsp.example.com"`),
				ExpectError: regexp.MustCompile(`HTTPS\s+is\s+not\s+supported`),
			},
			{
				Config:      baseHcl(instance) + clientCaHcl(serviceId, "no-scheme", cert, `  ocsp_override_url = "ocsp.example.com"`),
				ExpectError: regexp.MustCompile(`must\s+include\s+a\s+scheme\s+other\s+than\s+https`),
			},
		},
	})
}

func TestCertificateAuthoritiesOnOneServiceApplyWithoutConflict(t *testing.T) {
	instance := mockedInstance(t)
	serviceId := "ca-parallel-service"
	mock := instance.SetupCertificateAuthorityMocks(serviceId)
	config := baseHcl(instance)
	checks := []resource.TestCheckFunc{}
	destroyed := []resource.TestCheckFunc{}
	for i := 1; i <= 5; i++ {
		name := fmt.Sprintf("client-%d", i)
		config += clientCaHcl(serviceId, name, internal.CertificatePem(name), "")
		checks = append(checks, checkExists(mock, internal.ClientCertificateAuthorities, name, true))
		destroyed = append(destroyed, checkExists(mock, internal.ClientCertificateAuthorities, name, false))
	}
	config += domainCaHcl(serviceId, "domain-1", internal.CertificatePem("domain-1"))
	checks = append(checks, checkExists(mock, internal.DomainCertificateAuthorities, "domain-1", true))
	destroyed = append(destroyed, checkExists(mock, internal.DomainCertificateAuthorities, "domain-1", false))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config, Check: resource.ComposeTestCheckFunc(checks...)},
		},
		CheckDestroy: resource.ComposeTestCheckFunc(destroyed...),
	})
}
