# Shared event broker services are not supported.
variable "service_id" {
  description = "The identifier of the event broker service"
  type        = string
}

# Clients presenting a certificate issued by this authority can authenticate.
resource "solacecloud_client_certificate_authority" "corp" {
  service_id   = var.service_id
  name         = "corp-root"
  cert_content = file("${path.module}/corp-root.pem")

  revocation_check_enabled = true
  ocsp_override_url        = "http://ocsp.example.com"
  ocsp_timeout             = 10
}

# The event broker service trusts this authority when it connects to other servers.
resource "solacecloud_domain_certificate_authority" "corp" {
  service_id   = var.service_id
  name         = "corp-root"
  cert_content = file("${path.module}/corp-root.pem")
}
