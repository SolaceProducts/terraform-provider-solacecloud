package certificateauthority_test

import (
	"encoding/pem"
	"fmt"
	"regexp"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestDomainCertificateAuthorityLifecycle(t *testing.T) {
	instance := mockedInstance(t)
	serviceId := "domain-ca-lifecycle-service"
	mock := instance.SetupCertificateAuthorityMocks(serviceId)
	const name = "domain-a"
	const resourceName = "solacecloud_domain_certificate_authority." + name
	certA, certB := internal.CertificatePem("domain-a"), internal.CertificatePem("domain-a-renewed")
	config := func(cert string) string { return baseHcl(instance) + domainCaHcl(serviceId, name, cert) }

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(certA),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", serviceId+"/"+name),
					checkExists(mock, internal.DomainCertificateAuthorities, name, true),
				),
			},
			{Config: config(certA), PlanOnly: true},
			{
				Config: config(certB),
				Check:  checkLastWrite(mock, "PUT", name, map[string]any{"certContent": certB}),
			},
			{
				PreConfig: func() { mock.Remove(internal.DomainCertificateAuthorities, name) },
				Config:    config(certB),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate)},
				},
				Check: checkExists(mock, internal.DomainCertificateAuthorities, name, true),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"cert_content"},
			},
			{
				Config: baseHcl(instance) + fmt.Sprintf(`
resource "solacecloud_domain_certificate_authority" %q {
  service_id   = %q
  name         = "domain-b"
  cert_content = %q
}
`, name, serviceId, certB),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate)},
				},
				Check: resource.ComposeTestCheckFunc(
					checkExists(mock, internal.DomainCertificateAuthorities, name, false),
					checkExists(mock, internal.DomainCertificateAuthorities, "domain-b", true),
				),
			},
		},
		CheckDestroy: resource.ComposeTestCheckFunc(
			checkExists(mock, internal.DomainCertificateAuthorities, name, false),
			checkExists(mock, internal.DomainCertificateAuthorities, "domain-b", false),
		),
	})
}

func TestDomainCertificateAuthorityRejected(t *testing.T) {
	instance := mockedInstance(t)
	serviceId := "domain-ca-rejected-service"
	mock := instance.SetupCertificateAuthorityMocks(serviceId)
	mock.Put(internal.DomainCertificateAuthorities, "taken", `{"type": "domainCertificateAuthority"}`)
	cert := internal.CertificatePem("domain-rejected")
	csr := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: []byte("not a real request")}))
	const resourceName = "solacecloud_domain_certificate_authority.half-made"
	config := func(name, cert string) string { return baseHcl(instance) + domainCaHcl(serviceId, name, cert) }

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config("not-pem", "not a certificate"),
				ExpectError: regexp.MustCompile(`must\s+be\s+a\s+PEM-encoded\s+X\.509\s+certificate`),
			},
			{
				Config:      config("taken", cert),
				ExpectError: regexp.MustCompile(`terraform\s+import\s+<resource\s+address>\s+domain-ca-rejected-service/taken`),
			},
			{
				Config:      config("with-csr", cert+csr),
				ExpectError: regexp.MustCompile(`contains\s+a\s+"CERTIFICATE\s+REQUEST"\s+PEM\s+block`),
			},
			{
				PreConfig:   func() { mock.FailNextJob("The event broker service could not apply the certificate authority") },
				Config:      config("half-made", cert),
				ExpectError: regexp.MustCompile(`could\s+not\s+apply\s+the\s+certificate\s+authority`),
			},
			{
				Config: config("half-made", cert),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate)},
				},
				Check: checkExists(mock, internal.DomainCertificateAuthorities, "half-made", true),
			},
			{
				Config:      config("half-made", "not a certificate"),
				ExpectError: regexp.MustCompile(`must\s+be\s+a\s+PEM-encoded\s+X\.509\s+certificate`),
			},
		},
		CheckDestroy: checkExists(mock, internal.DomainCertificateAuthorities, "half-made", false),
	})
}
