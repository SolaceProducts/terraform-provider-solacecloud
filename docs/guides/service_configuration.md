---
page_title: "Configuring Event Broker Services"
subcategory: "Services"
---

# Configuring Event Broker Services

This guide provides detailed information on configuring event broker services using the Terraform provider.

## Service Classes

The Solace Cloud Terraform Provider supports various service classes, each offering different capabilities and performance characteristics. When creating a service, you specify the service class using the `service_class_id` attribute. For more information, see [Service Class Options for Event Broker Services](https://docs.solace.com/Cloud/service-class-limits.htm).

### Available Service Classes

The following table lists the values you can use for `service_class_id`.

| Service Class ID | Description |
|-----------------|-------------|
| `DEVELOPER` | Free tier for development and testing |
| `ENTERPRISE_250_STANDALONE` | Enterprise 250 connections (standalone) |
| `ENTERPRISE_1K_STANDALONE` | Enterprise 1,000 connections (standalone) |
| `ENTERPRISE_5K_STANDALONE` | Enterprise 5,000 connections (standalone) |
| `ENTERPRISE_10K_STANDALONE` | Enterprise 10,000 connections (standalone) |
| `ENTERPRISE_50K_STANDALONE` | Enterprise 50,000 connections (standalone) |
| `ENTERPRISE_100K_STANDALONE` | Enterprise 100,000 connections (standalone) |
| `ENTERPRISE_200K_STANDALONE` | Enterprise 200,000 connections (standalone) |
| `ENTERPRISE_250_HIGHAVAILABILITY` | Enterprise 250 connections (HA) |
| `ENTERPRISE_1K_HIGHAVAILABILITY` | Enterprise 1,000 connections (HA) |
| `ENTERPRISE_5K_HIGHAVAILABILITY` | Enterprise 5,000 connections (HA) |
| `ENTERPRISE_10K_HIGHAVAILABILITY` | Enterprise 10,000 connections (HA) |
| `ENTERPRISE_50K_HIGHAVAILABILITY` | Enterprise 50,000 connections (HA) |
| `ENTERPRISE_100K_HIGHAVAILABILITY` | Enterprise 100,000 connections (HA) |
| `ENTERPRISE_200K_HIGHAVAILABILITY` | Enterprise 200,000 connections (HA) |

The following example creates a high-availability event broker service:

```hcl
resource "solacecloud_service" "broker_service" {
  name             = "production-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_10K_HIGHAVAILABILITY"
}
```

## Datacenters

The `datacenter_id` attribute specifies where your service will be deployed. Solace Cloud offers various datacenters across different cloud providers and regions. For more information, see [Choosing the Right Cloud Region](https://docs.solace.com/Cloud/ght_regions.htm).

Common datacenter IDs include:

- AWS: `aws-us-east-1a`, `aws-us-west-2a`, `aws-eu-central-1a`, etc.
- Azure: `azure-eastus`, `azure-westeurope`, etc.
- GCP: `gcp-us-central1`, `gcp-europe-west1`, etc.

[Contact Solace](https://docs.solace.com/get-support.htm) support or check the Cloud Console for a complete list of available datacenters.

## Message VPN Configuration

A Message VPN (Virtual Private Network) is a virtual messaging domain that allows for multi-tenancy in Solace event brokers. When you create a service, a default Message VPN is created, but you can customize its name:

```hcl
resource "solacecloud_service" "broker_service" {
  name             = "my-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
  message_vpn_name = "my-custom-vpn"
}
```

**Note:** You cannot change the VPN name after creating the event broker service.

## Message Spool Configuration

The message spool is used for guaranteed messaging. You can configure the size of the message spool using the `max_spool_usage` attribute:

```hcl
resource "solacecloud_service" "broker_service" {
  name             = "my-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
  max_spool_usage  = 20  # 20GB of spool
}
```

The `max_spool_usage` value is specified in gigabytes (GB) and must be between 10 and 6000. For more information, see [Configuring Message Spool Sizes](https://docs.solace.com/Cloud/Configure-Message-Spools.htm).

## Service Security

You can protect an event broker service from deletion and encrypt its mate link.

### Locking a Service

You can prevent accidental deletion of a service by setting the `locked` attribute to `true`:

```hcl
resource "solacecloud_service" "broker_service" {
  name             = "critical-production-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_10K_HIGHAVAILABILITY"
  locked           = true
}
```

To delete a locked service, you must first update it to set `locked = false`. For more information, see [Using Deletion Protection](https://docs.solace.com/Cloud/ght_service_deletion_protection.htm).

### Mate Link Encryption

For high availability (HA) services, you can enable encryption for the mate link (the connection between the primary and backup brokers) using the mate_link_encryption attribute. For more information, see [HA-Link Security](https://docs.solace.com/Cloud/ha_concept.htm#ha-link-security).

```hcl
resource "solacecloud_service" "broker_service" {
  name                = "secure-ha-service"
  datacenter_id       = "aws-us-east-1a"
  service_class_id    = "ENTERPRISE_1K_HIGHAVAILABILITY"
  mate_link_encryption = true
}
```

You can also enable or disable mate-link encryption on a service that already exists by changing `mate_link_encryption` and applying. The service is not replaced; the mate link is reconfigured in place, and the provider waits for that to finish. Note that:

* Enabling mate-link encryption also enables Config-Sync encryption if it is not already enabled. Disabling mate-link encryption leaves Config-Sync encryption unchanged.
* The change is refused on a service that is not high-availability, and the provider reports the error. Omit `mate_link_encryption` for standalone services: setting it to `true` when creating one creates the service, then fails the apply and leaves the service tainted, so the next apply replaces it.
* If you remove `mate_link_encryption` from the configuration, the service keeps its current setting and Terraform plans no change.

## Environment Configuration

Environments in Solace Cloud allow you to organize and manage services. You can specify an environment when creating a service:

```hcl
data "solacecloud_environment" "prod_env" {
  name = "Production"
}

resource "solacecloud_service" "broker_service" {
  name             = "prod-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
  environment_id   = data.solacecloud_environment.prod_env.id
}
```

## Advanced Configuration

Most configurations can omit the options in this section, because each has a default.

### Custom Router Name

In some scenarios, such as replacing a part of a DMR Cluster or disaster recovery setup, you may need to specify a custom router name:

```hcl
resource "solacecloud_service" "broker_service" {
  name               = "replacement-service"
  datacenter_id      = "aws-us-east-1a"
  service_class_id   = "ENTERPRISE_1K_STANDALONE"
  custom_router_name = "original-router-prefix"
}
```

This is an advanced feature and should be left undefined for most use cases.

### Event Broker Version

You can specify specific event broker versions:

```hcl
resource "solacecloud_service" "broker_service" {
  name                = "specific-version-service"
  datacenter_id       = "aws-us-east-1a"
  service_class_id    = "ENTERPRISE_1K_STANDALONE"
  event_broker_version = "10.0.1.7-3"
}
```

If not specified, a default version is provided. For more information, see [Selecting the Event Broker Release and Version](https://docs.solace.com/Cloud/create-service.htm#selecting-the-event-broker-release-and-version) :

## Working with Service Attributes

After a service is created, you can access various attributes for use in other resources or outputs:

### Accessing Connection Endpoints

To access connection endpoint information, use the `solacecloud_connection_endpoints` data source and `solacecloud_connection_endpoint_dns_names` data source:

```hcl
data "solacecloud_connection_endpoints" "endpoints" {
  service_id = solacecloud_service.broker_service.id
}

data "solacecloud_connection_endpoint_dns_names" "dns" {
  service_id             = solacecloud_service.broker_service.id
  connection_endpoint_id = data.solacecloud_connection_endpoints.endpoints.endpoints[0].id
}

output "smf_host" {
  value = data.solacecloud_connection_endpoint_dns_names.dns.dns_names[0].dns_name
}

output "smf_port" {
  value = data.solacecloud_connection_endpoints.endpoints.endpoints[0].ports.smf.port
}

output "web_messaging_url" {
  value = "https://${data.solacecloud_connection_endpoint_dns_names.dns.dns_names[0].dns_name}:${data.solacecloud_connection_endpoints.endpoints.endpoints[0].ports.web_tls.port}"
}
```

### Accessing Credentials

```hcl
output "management_username" {
  value = solacecloud_service.broker_service.message_vpn.manager_management_credential.username
}

output "management_password" {
  value     = solacecloud_service.broker_service.message_vpn.manager_management_credential.password
  sensitive = true
}

output "messaging_username" {
  value = solacecloud_service.broker_service.message_vpn.messaging_client_credential.username
}

output "messaging_password" {
  value     = solacecloud_service.broker_service.message_vpn.messaging_client_credential.password
  sensitive = true
}
```

### Rotating Event Broker Service Passwords

Use the `broker_password_rotation` block to rotate the passwords for the event broker service's management users. The new passwords are generated for you and are not returned in the rotation response, so after the rotation finishes the provider reads the service again and stores them in the credential attributes in the same apply. You do not need a separate `terraform refresh`.

```hcl
resource "solacecloud_service" "broker_service" {
  name             = "my-broker-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"

  broker_password_rotation = {
    users   = ["mc-manager", "mc-viewer"]
    trigger = "2026-11-02"
  }
}

# Updated in the apply that rotates the passwords.
output "viewer_password" {
  value     = solacecloud_service.broker_service.message_vpn.viewer_management_credential.password
  sensitive = true
}
```

The following table shows the credential attribute that each value in `users` maps to.

| User | Credential attribute |
|------|----------------------|
| `mc-editor` | `message_vpn.editor_management_credential` |
| `mc-manager` | `message_vpn.manager_management_credential` |
| `mc-viewer` | `message_vpn.viewer_management_credential` |

Only the passwords of the listed users change. Usernames and the other credentials, such as the messaging client credential, are not affected.

#### When a rotation happens

Passwords are rotated when `trigger` changes on an existing event broker service that already has the block. Its value means nothing; only a change matters, so a date or a ticket number works well. To rotate again, change it and apply.

The following do **not** rotate anything:

* Creating an event broker service with the block. New event broker services already have fresh passwords, and the first `trigger` is only recorded.
* Adding the block to an event broker service that already exists. The first `trigger` is only recorded, and a later change to it rotates. `terraform plan` shows a "Passwords not rotated" warning, unless the whole block depends on a value that isn't known until apply. Either way nothing rotates, so change `trigger` once more if you wanted to rotate when adding the block.
* Changing `users` without changing `trigger`.
* Removing the block.

When `trigger` changes, `terraform plan` shows the passwords of the listed users as `(known after apply)`.

The `mc-manager` user exists only on event brokers that have a manager credential, which older event broker versions lack. If you name it for such an event broker, `terraform plan` rejects it when you add the block or change `trigger`, and the error names the user. It isn't checked at plan when you create the event broker service with the block, because the event broker service doesn't exist yet, or when `users` isn't known until apply.

#### If a rotation fails

The provider sends the rotation request once and never retries it. Retrying could rotate the passwords a second time, or collide with a rotation that has already started. In most failures below, state keeps the previous `trigger` and passwords, so the next apply requests the rotation again. The exception is a rotation that succeeds when the event broker service can't be read back afterwards: see the last case.

* **Resource Conflict.** Only one operation can run on a service at a time, and a rotation request that arrives while one is running is refused. The error shows the API's message, which names the job that is still running. Wait for it to finish, then apply again.
* **Password rotation may have started.** The API timed out before answering. The rotation may already have started or finished. Check the event broker service in the Cloud Console before you apply again; applying requests a new rotation, and the passwords change again.
* **The operation failed or timed out.** The request was accepted, but the operation then reported a failure, or did not finish within 15 minutes. Some of the listed passwords may already have changed, even though the apply failed. State keeps the previous `trigger` and passwords until a later refresh reads the service, so run `terraform refresh` (or `terraform plan`) to see the current credentials, and update anything that holds a copy. Applying again requests a new rotation, and the passwords change again.
* **Forbidden.** Your API token needs the `mission_control:access` or `services:post` permission, and access to the specific service.
* **Passwords were rotated but could not be read back.** The rotation finished, but reading the event broker service afterwards failed. The apply fails with a warning saying so. The new `trigger` is recorded, so the next apply does not rotate again, but state still holds the old passwords, and they no longer work. Run `terraform refresh` (or `terraform apply`) once the event broker service can be read to store the new passwords, then update anything that holds a copy.

#### Credentials in state

The new passwords are stored in the Terraform state like the existing ones. They are marked sensitive, which hides them from plan and apply output, but they are not encrypted in the state file. Protect the state with your backend's access controls and encryption. Anything that holds a copy of the old password, such as a secrets manager or an application configuration, must be updated after a rotation.

### Accessing DMR Cluster Information

These attributes are only populated for a service created with `dmr_enabled = true`. For a service without DMR, `dmr_cluster` is null. See [Dynamic Message Routing (DMR)](#dynamic-message-routing-dmr).

```hcl
output "dmr_cluster_name" {
  value = solacecloud_service.broker_service.dmr_cluster.name
}

output "dmr_remote_address" {
  value = solacecloud_service.broker_service.dmr_cluster.remote_address
}
```

## Dynamic Message Routing (DMR)

Solace Cloud creates new event broker services without a DMR cluster unless you opt in. Set `dmr_enabled = true` to create the service with a cloud-managed DMR cluster, and optionally name the cluster with `cluster_name`.

```hcl
resource "solacecloud_service" "mesh_member" {
  name             = "mesh-member"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
  dmr_enabled      = true
  cluster_name     = "prod-dmr-cluster" # Optional. Solace Cloud generates a name when omitted.
}
```

Keep the following in mind:

* The `cluster_name` argument requires `dmr_enabled = true`. Setting `cluster_name` without opting in is rejected when you run `terraform plan`.
* A service without DMR has no `cluster_name`, and its `dmr_cluster` attribute is null.
* Existing configurations that set neither argument keep applying without a plan diff. The provider reads `dmr_enabled` from the service, so an existing DMR-enabled service reports `true`.
* Service packages that always enable DMR (legacy packages) ignore `dmr_enabled` when creating the service. If you set `dmr_enabled = false` on such a package, the provider disables DMR and deletes the cluster right after the service is created so it matches your configuration, and reports a warning saying so. Remove `dmr_enabled = false` if you want to keep the DMR cluster.

### Enabling and Disabling DMR

Changing `dmr_enabled` on a service that already exists enables or disables DMR in place. The service is not replaced and keeps its identifier, endpoints and credentials.

```hcl
resource "solacecloud_service" "mesh_member" {
  name             = "mesh-member"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
  dmr_enabled      = true # Was false. Applying this creates the DMR cluster.
}
```

* Enabling DMR creates a cloud-managed DMR cluster and enables DMR on the Message VPN. Solace Cloud generates and stores the cluster credential. You can name the cluster in the same change by adding `cluster_name`; otherwise a name is generated.
* Disabling DMR deletes the DMR cluster and disables DMR on the Message VPN. Solace Cloud refuses the request while the service belongs to an event mesh, and the refusal is reported as a Terraform error. Remove the service from the event mesh first.
* Each direction runs as an asynchronous Solace Cloud operation that reconfigures the event broker, so the apply waits for it to finish.
* An existing cluster cannot be renamed. Changing `cluster_name` on a service that already has a DMR cluster is rejected.

## Complete Example

Here's a complete example that demonstrates various configuration options:

```hcl
# Fetch the Production environment
data "solacecloud_environment" "prod_env" {
  name = "Production"
}

# Create a high-availability service
resource "solacecloud_service" "ha_service" {
  name                = "prod-messaging-service"
  datacenter_id       = "aws-us-east-1a"
  service_class_id    = "ENTERPRISE_10K_HIGHAVAILABILITY"
  environment_id      = data.solacecloud_environment.prod_env.id
  message_vpn_name    = "prod-vpn"
  max_spool_usage     = 100
  mate_link_encryption = true
  locked              = true
  dmr_enabled         = true
  cluster_name        = "prod-dmr-cluster"
}

# Output important service information
output "service_id" {
  value = solacecloud_service.ha_service.id
}

data "solacecloud_connection_endpoints" "ha_endpoints" {
  service_id = solacecloud_service.ha_service.id
}

data "solacecloud_connection_endpoint_dns_names" "ha_dns" {
  service_id             = solacecloud_service.ha_service.id
  connection_endpoint_id = data.solacecloud_connection_endpoints.ha_endpoints.endpoints[0].id
}

output "management_url" {
  value = "https://${data.solacecloud_connection_endpoint_dns_names.ha_dns.dns_names[0].dns_name}:${data.solacecloud_connection_endpoints.ha_endpoints.endpoints[0].ports.management_tls.port}"
}
```
