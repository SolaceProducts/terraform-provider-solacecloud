package provider_test

import (
	"fmt"
	"os"
	"strings"
	"terraform-provider-solacecloud/internal"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/labstack/gommon/random"
)

// Runs against a real API only when SOLACE_TEST_CERTIFICATE_AUTHORITIES=true.
func TestAccCertificateAuthorityLifecycle(t *testing.T) {
	instance := internal.NewTestInstance()
	serviceName := "E2E_CA_Lifecycle_" + random.String(8)
	params := internal.ConfigurableParams{
		ServiceClass: e2eServiceClassId(),
		ServiceName:  serviceName,
		ServiceId:    "ca" + strings.ToLower(random.String(10, random.Alphanumeric)),
	}
	instance.Init(params)
	if instance.IsMocked() {
		instance.SetupCertificateAuthorityMocks(params.ServiceId)
	} else if os.Getenv("SOLACE_TEST_CERTIFICATE_AUTHORITIES") != "true" {
		t.Skip("set SOLACE_TEST_CERTIFICATE_AUTHORITIES=true to run the certificate authority lifecycle against a real API")
	}

	const client = "solacecloud_client_certificate_authority.e2e"
	const domain = "solacecloud_domain_certificate_authority.e2e"
	certA, certB := internal.CertificatePem("e2e-ca-a"), internal.CertificatePem("e2e-ca-b")
	config := func(cert, ocsp string) string {
		return e2eServiceHcl(instance, serviceName, "") + fmt.Sprintf(`
resource "solacecloud_client_certificate_authority" "e2e" {
  service_id   = solacecloud_service.%[1]s.id
  name         = "e2e-client"
  cert_content = %[2]q
%[3]s}

resource "solacecloud_domain_certificate_authority" "e2e" {
  service_id   = solacecloud_service.%[1]s.id
  name         = "e2e-domain"
  cert_content = %[2]q
}
`, serviceName, cert, ocsp)
	}
	withOverride := `
  revocation_check_enabled = true
  ocsp_override_url        = "http://ocsp.example.com"
  ocsp_timeout             = 20
`
	withoutOverride := `
  revocation_check_enabled = true
  ocsp_timeout             = 30
`

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(certA, withOverride),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(client, "ocsp_override_url", "http://ocsp.example.com"),
					resource.TestCheckResourceAttr(client, "ocsp_timeout", "20"),
					resource.TestCheckResourceAttrSet(domain, "id"),
				),
			},
			{
				Config: config(certB, withoutOverride),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(client, "ocsp_override_url"),
					resource.TestCheckResourceAttr(client, "ocsp_timeout", "30"),
				),
			},
			{Config: config(certB, withoutOverride), PlanOnly: true},
			{ResourceName: client, ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"cert_content"}},
			{ResourceName: domain, ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"cert_content"}},
		},
	})
}
