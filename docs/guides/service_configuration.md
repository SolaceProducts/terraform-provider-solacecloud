# Service Configuration Guide

This guide provides detailed information on configuring event broker services using the Terraform provider.

## Service Classes

The Solace Cloud Provider supports various service classes, each offering different capabilities and performance characteristics. When creating a service, you specify the service class using the `service_class_id` attribute. For more information, see [Service Class Options for Event Broker Services](https://docs.solace.com/Cloud/service-class-limits.htm).

### Available Service Classes

For more information, see [Service Class Options for Event Broker Services](https://docs.solace.com/Cloud/service-class-limits.htm).

| Service Class ID | Description |
|-----------------|-------------|
| `DEVELOPER` | Free tier for development and testing |
| `ENTERPRISE_250_STANDALONE` | Enterprise 250 connections (standalone) |
| `ENTERPRISE_1K_STANDALONE` | Enterprise 1,000 connections (standalone) |
| `ENTERPRISE_5K_STANDALONE` | Enterprise 5,000 connections (standalone) |
| `ENTERPRISE_10K_STANDALONE` | Enterprise 10,000 connections (standalone) |
| `ENTERPRISE_50K_STANDALONE` | Enterprise 50,000 connections (standalone) |
| `ENTERPRISE_100K_STANDALONE` | Enterprise 100,000 connections (standalone) |
| `ENTERPRISE_250_HIGHAVAILABILITY` | Enterprise 250 connections (HA) |
| `ENTERPRISE_1K_HIGHAVAILABILITY` | Enterprise 1,000 connections (HA) |
| `ENTERPRISE_5K_HIGHAVAILABILITY` | Enterprise 5,000 connections (HA) |
| `ENTERPRISE_10K_HIGHAVAILABILITY` | Enterprise 10,000 connections (HA) |
| `ENTERPRISE_50K_HIGHAVAILABILITY` | Enterprise 50,000 connections (HA) |
| `ENTERPRISE_100K_HIGHAVAILABILITY` | Enterprise 100,000 connections (HA) |

Example:

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

You can specify a particular version of the Solace PubSub+ event broker:

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

```hcl
output "smf_host" {
  value = solacecloud_service.broker_service.connection_endpoints[0].hostnames[0]
}

output "smf_port" {
  value = solacecloud_service.broker_service.connection_endpoints[0].ports.smf.port
}

output "web_messaging_url" {
  value = "https://${solacecloud_service.broker_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service.connection_endpoints[0].ports.web_tls.port}"
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

### Accessing DMR Cluster Information

```hcl
output "dmr_cluster_name" {
  value = solacecloud_service.broker_service.dmr_cluster.name
}

output "dmr_remote_address" {
  value = solacecloud_service.broker_service.dmr_cluster.remote_address
}
```

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
  cluster_name        = "prod-dmr-cluster"
}

# Output important service information
output "service_id" {
  value = solacecloud_service.ha_service.id
}

output "management_url" {
  value = "https://${solacecloud_service.ha_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.ha_service.connection_endpoints[0].ports.management_tls.port}"
}
```
