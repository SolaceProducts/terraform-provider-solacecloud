# Broker Integration Guide

This guide explains how to integrate the Solace Cloud Terraform Provider with the Solace Broker Terraform Provider to manage both your Solace Cloud services and their internal messaging resources.

## Overview

The Solace Cloud Provider allows you to create and manage Solace Cloud services, while the Solace Broker Provider allows you to configure the messaging resources within those services, such as queues, topics, and client profiles.

By using these providers together, you can create a complete infrastructure as code solution for your Solace messaging environment.

## Prerequisites

Before you begin, ensure you have:

1. Terraform installed (version 0.13.0 or later)
2. A Solace Cloud account with Mission Control Manager or Administrator role, and API token with the `services:post` permission. For more information, see [Managing Users, Groups, Roles, and Permissions](https://docs.solace.com/Cloud/cloud-user-management.htm) and [Managing API Tokens](https://docs.solace.com/Cloud/ght_api_tokens.htm).
3. The Solace Cloud Terraform Provider configured
4. The Solace Broker Terraform Provider installed

## Provider Configuration

First, configure both providers in your Terraform configuration:

```hcl
terraform {
  required_providers {
    solacecloud = {
      source = "registry.terraform.io/solaceproducts/solacecloud"
    }
    solacebroker = {
      source  = "SolaceProducts/solacebroker"
      version = "1.1.0"
    }
  }
}

# Configure the Solace Cloud Provider
provider "solacecloud" {
  base_url  = "https://api.solace.cloud/"
  api_token = var.solace_api_token
}

provider "solacebroker" {
  alias    = "broker1"
  url      = "https://${solacecloud_service.broker_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service.connection_endpoints[0].ports.management_tls.port}"
  username = solacecloud_service.broker_service.message_vpn.manager_management_credential.username
  password = solacecloud_service.broker_service.message_vpn.manager_management_credential.password
}
```

## Managing Multiple Services

If you need to manage multiple event broker services, you can use provider aliases:

:

```hcl
resource "solacecloud_service" "broker_service" {
  name             = "my-service"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

resource "solacecloud_service" "broker_service2" {
  name             = "my-service2"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

# Configure Solace Broker Provider for the first service
provider "solacebroker" {
  alias    = "broker1"
  url      = "https://${solacecloud_service.broker_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service.connection_endpoints[0].ports.management_tls.port}"
  username = solacecloud_service.broker_service.message_vpn.manager_management_credential.username
  password = solacecloud_service.broker_service.message_vpn.manager_management_credential.password
}

# Configure Solace Broker Provider for the second service
provider "solacebroker" {
  alias    = "broker2"
  url      = "https://${solacecloud_service.broker_service2.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service2.connection_endpoints[0].ports.management_tls.port}"
  username = solacecloud_service.broker_service2.message_vpn.manager_management_credential.username
  password = solacecloud_service.broker_service2.message_vpn.manager_management_credential.password
}
```

## Creating Messaging Resources

Once you have configured both providers, you can create messaging resources within your event broker service:

### Queues

```hcl
resource "solacebroker_msg_vpn_queue" "queue11" {
  provider = solacebroker.broker1
  queue_name = "my-service-queue1"
  msg_vpn_name  = "msgvpn-${solacecloud_service.broker_service.id}"
  ingress_enabled = true
  egress_enabled = true
  max_msg_size = 54321
  partition_count = 4
}

resource "solacebroker_msg_vpn_queue" "queue12" {
  provider = solacebroker.broker1
  queue_name = "my-service-queue2"
  msg_vpn_name  = "msgvpn-${solacecloud_service.broker_service.id}"
  ingress_enabled = true
  egress_enabled = true
  max_msg_size = 54321
  partition_count = 4
}

resource "solacebroker_msg_vpn_queue" "queue21" {
  provider = solacebroker.broker2
  queue_name = "my-service2-queue1"
  msg_vpn_name  = "msgvpn-${solacecloud_service.broker_service2.id}"
  ingress_enabled = true
  egress_enabled = true
  max_msg_size = 54321
  partition_count = 4
}
```

## Complete Example

Here is a complete example that demonstrates creating an event broker service and configuring various messaging resources:

```hcl
terraform {
  required_providers {
    solacecloud = {
      source = "registry.terraform.io/solaceproducts/solacecloud"
    }
    solacebroker = {
      source  = "SolaceProducts/solacebroker"
      version = "1.1.0"
    }
  }
}

# Configure the Solace Cloud Provider
provider "solacecloud" {
  base_url  = "https://api.solace.cloud/"
  api_token = var.solace_api_token
}

# Create a Solace Cloud service
resource "solacecloud_service" "broker_service" {
  name             = "my-service"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

resource "solacecloud_service" "broker_service2" {
  name             = "my-service2"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

# Configure the Solace Broker Provider
provider "solacebroker" {
  alias    = "broker1"
  url      = "https://${solacecloud_service.broker_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service.connection_endpoints[0].ports.management_tls.port}"
  username = solacecloud_service.broker_service.message_vpn.manager_management_credential.username
  password = solacecloud_service.broker_service.message_vpn.manager_management_credential.password
}

provider "solacebroker" {
  alias    = "broker2"
  url      = "https://${solacecloud_service.broker_service2.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service2.connection_endpoints[0].ports.management_tls.port}"
  username = solacecloud_service.broker_service2.message_vpn.manager_management_credential.username
  password = solacecloud_service.broker_service2.message_vpn.manager_management_credential.password
}

# Create queues
resource "solacebroker_msg_vpn_queue" "queue11" {
  provider = solacebroker.broker1
  queue_name = "my-service-queue1"
  msg_vpn_name  = "msgvpn-${solacecloud_service.broker_service.id}"
  ingress_enabled = true
  egress_enabled = true
  max_msg_size = 54321
  partition_count = 4
}

resource "solacebroker_msg_vpn_queue" "queue12" {
  provider = solacebroker.broker1
  queue_name = "my-service-queue2"
  msg_vpn_name  = "msgvpn-${solacecloud_service.broker_service.id}"
  ingress_enabled = true
  egress_enabled = true
  max_msg_size = 54321
  partition_count = 4
}

resource "solacebroker_msg_vpn_queue" "queue21" {
  provider = solacebroker.broker2
  queue_name = "my-service2-queue1"
  msg_vpn_name  = "msgvpn-${solacecloud_service.broker_service2.id}"
  ingress_enabled = true
  egress_enabled = true
  max_msg_size = 54321
  partition_count = 4
}

# Output connection information
output "broker_username" {
  value = solacecloud_service.broker_service.message_vpn.manager_management_credential.username
}

output "broker_password" {
  value = solacecloud_service.broker_service.message_vpn.manager_management_credential.password
  sensitive = true
}

output "broker_semp_url" {
  value = "https://${solacecloud_service.broker_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service.connection_endpoints[0].ports.management_tls.port}"
}
```

## Best Practices

1. **Use Separate State Files**: Consider using separate Terraform state files for event broker services and broker configurations to allow independent management.

2. **Manage Credentials Securely**: Store credentials in secure locations like Terraform Cloud, HashiCorp Vault, or environment variables.

3. **Use Variables for Common Values**: Define variables for common values like Message VPN names to avoid hardcoding.

4. **Implement Proper Dependencies**: Ensure proper dependencies between resources to avoid race conditions during creation.

5. **Use Modules**: Create reusable Terraform modules for common patterns like service creation with standard configurations.

6. **Version Control**: Keep your Terraform configurations in version control to track changes and collaborate with team members.

## Troubleshooting

If you encounter issues with the integration between the Solace Cloud Provider and the Solace Broker Provider, consider the following:

1. **Connection Issues**: Ensure the service is fully provisioned before attempting to configure broker resources.

2. **Authentication Issues**: Verify that the credentials used for the Solace Broker Provider are correct.

3. **Message VPN Name**: Confirm that you're using the correct Message VPN name in your broker resources.


4. **API Permissions**: Ensure that the API token used for the Solace Cloud Provider has sufficient permissions. For more information, (see [Prerequisites](#prerequisites) section).

5. **Provider Versions**: Check that you are using compatible versions of both providers.

For more detailed troubleshooting information, refer to the [Troubleshooting Guide](../troubleshooting.md).
