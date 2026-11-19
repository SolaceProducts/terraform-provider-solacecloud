---
page_title: "solacecloud_client_certificate_authority Resource - terraform-provider-solacecloud"
subcategory: ""
---

# Resource: solacecloud_client_certificate_authority

This resource manages a client certificate authority of an event broker service. Clients that present a certificate issued by this certificate authority can authenticate with the event broker service. The resource also controls whether the event broker service checks the revocation status of those certificates with Online Certificate Status Protocol (OCSP).

A certificate authority is identified by the event broker service ID and its name.

**Prerequisites:**
- An existing event broker service. Shared event broker services are not supported.

## Example Usage

```hcl
resource "solacecloud_client_certificate_authority" "corp" {
  service_id   = solacecloud_service.broker_service.id
  name         = "corp-root"
  cert_content = file("${path.module}/corp-root.pem")

  revocation_check_enabled = true
  ocsp_override_url        = "http://ocsp.example.com"
  ocsp_timeout             = 10
}
```

## Argument Reference

### Required Arguments

* `service_id` — (Required) The identifier of the event broker service. Changing it replaces the resource.
* `name` — (Required) The name of the certificate authority, 1 to 64 characters. Use only letters, digits, periods, underscores, and hyphens. The name must be unique among the client certificate authorities of the event broker service. Changing it replaces the resource.
* `cert_content` — (Required) The X.509 certificate of the certificate authority, encoded in privacy enhanced mail (PEM) format. Provide exactly one certificate. The provider rejects a value that contains any other PEM block, such as a private key, at plan time.

### Optional Arguments

* `revocation_check_enabled` — (Optional) Indicates whether the event broker service uses OCSP to check the revocation status of client certificates issued by this certificate authority. Defaults to `false`.
* `ocsp_override_url` — (Optional) The URL of the OCSP responder to use instead of the one named in the client certificate. Include the scheme, for example `http://ocsp.example.com`. HTTPS is not supported. Remove the argument to use the URL in the client certificate again.
* `ocsp_timeout` — (Optional) The number of seconds to wait for an OCSP response, 1 to 86,400 (24 hours). Defaults to `5`.
* `ocsp_non_responder_cert_enabled` — (Optional) Indicates whether the event broker service accepts an OCSP response signed by a certificate other than the certificate of the OCSP responder. Defaults to `false`.

## Attribute Reference

* `id` — The identifier of the certificate authority, `<service_id>/<name>`.

## Update Behavior

Changes to `cert_content` and the OCSP arguments are applied in place. The provider sends only the arguments that changed and waits for the update to finish. Changing `service_id` or `name` replaces the resource.

The API never returns the certificate. Terraform keeps the value from your configuration, so a certificate replaced outside Terraform is not detected. Changes to the OCSP arguments made outside Terraform show up as drift on the next plan.

## Concurrency

An event broker service runs one job at a time. Within one provider instance, writes to the certificate authorities of one service run one after another, so a configuration with many certificate authorities on one service applies without conflicts. Each certificate authority takes the time of one job, so many of them take correspondingly longer.

The provider also runs writes from `solacecloud_ldap_management_access`, `solacecloud_connection_endpoint`, and `solacecloud_connection_endpoint_dns_name` to the same event broker service in this order.

This ordering does not cover other jobs on the same event broker service: changes to `solacecloud_service`, `solacecloud_connection_endpoint_dns_name_move`, changes made in the Cloud Console, or a second provider instance (a provider alias). If the same `terraform apply` also includes one of those changes, the apply can fail with "Job ... is still in progress". Add `depends_on` so the changes run one after another, or apply again.

## Delete Behavior

Deleting the resource removes the certificate authority from the event broker service and waits for the deletion to finish. If the certificate authority was already removed outside Terraform, the deletion succeeds.

## Import

You can import a client certificate authority using `<service_id>/<name>`.

**Using Terraform 1.5 or later import blocks:**

```hcl
import {
  to = solacecloud_client_certificate_authority.corp
  id = "myserviceid/corp-root"
}

resource "solacecloud_client_certificate_authority" "corp" {
  service_id   = "myserviceid"
  name         = "corp-root"
  cert_content = file("${path.module}/corp-root.pem")
}
```

**Using the command line (for one-off imports):**

```bash
terraform import solacecloud_client_certificate_authority.corp myserviceid/corp-root
```

The import reads the name and the OCSP settings, but the API does not return the certificate. The first `terraform apply` after an import sends `cert_content` from your configuration to the event broker service.

For the same reason, Terraform does not detect a certificate changed outside Terraform, for example, in the Cloud Console. Terraform sends the certificate only when `cert_content` in your configuration changes.

### Legacy Certificate Authorities

A certificate authority created before client and domain certificate authorities were separated has no type. It appears in both the client and the domain lists, and the API rejects every update to it. You can import it, but the first `terraform apply` after the import fails, because that apply sends the certificate as an update. Deleting it through either resource removes it from both lists.

To manage a legacy certificate authority with Terraform, delete it in the Cloud Console, then create it again with this resource, or with `solacecloud_domain_certificate_authority` if the event broker service uses it as a domain certificate authority.

## Troubleshooting

### "... has no client or domain type and cannot be updated"

The certificate authority has no type. For more information, see [Legacy Certificate Authorities](#legacy-certificate-authorities).

### "A certificate authority named ... already exists"

The event broker service already has a client certificate authority with that name that Terraform does not manage, for example, one made in the Cloud Console. Import it instead of creating it.

This error also happens if the Terraform process was stopped during a create, after the create request was accepted. Import the certificate authority, then apply again.
