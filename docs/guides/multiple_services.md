# Managing Multiple Services

This guide explains how to manage multiple event broker services using the Terraform provider, including best practices for organization, configuration, and maintenance.

## Overview

As your messaging infrastructure grows, you may need to deploy multiple event broker services across different environments, regions, or for different applications. The Solace Cloud Terraform Provider makes it easy to manage multiple services in a consistent and repeatable way.

## Organizing Multiple Services

There are several approaches to organizing multiple services in your Terraform configuration:

### Approach 1: Single Configuration File

For a small number of services, you can define them all in a single configuration file:

```hcl
resource "solacecloud_service" "dev_service" {
  name             = "dev-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "DEVELOPER"
}

resource "solacecloud_service" "qa_service" {
  name             = "qa-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

resource "solacecloud_service" "prod_service" {
  name             = "prod-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_10K_HIGHAVAILABILITY"
}
```

### Approach 2: Separate Files by Environment

For better organization, you can split your configuration into multiple files based on environment:

**dev.tf**:
```hcl
resource "solacecloud_service" "dev_service" {
  name             = "dev-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "DEVELOPER"
}
```

**qa.tf**:
```hcl
resource "solacecloud_service" "qa_service" {
  name             = "qa-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}
```

**prod.tf**:
```hcl
resource "solacecloud_service" "prod_service" {
  name             = "prod-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_10K_HIGHAVAILABILITY"
}
```

### Approach 3: Using Terraform Workspaces

Terraform workspaces allow you to manage multiple environments with the same configuration files:

```hcl
locals {
  env = terraform.workspace

  service_configs = {
    dev = {
      name             = "dev-service"
      datacenter_id    = "aws-us-east-1a"
      service_class_id = "DEVELOPER"
    },
    qa = {
      name             = "qa-service"
      datacenter_id    = "aws-us-east-1a"
      service_class_id = "ENTERPRISE_1K_STANDALONE"
    },
    prod = {
      name             = "prod-service"
      datacenter_id    = "aws-us-east-1a"
      service_class_id = "ENTERPRISE_10K_HIGHAVAILABILITY"
    }
  }
}

resource "solacecloud_service" "service" {
  name             = local.service_configs[local.env].name
  datacenter_id    = local.service_configs[local.env].datacenter_id
  service_class_id = local.service_configs[local.env].service_class_id
}
```

To use this approach, create and select a workspace for each environment:

```bash
terraform workspace new dev
terraform workspace select dev
terraform apply
```

### Approach 4: Using Terraform Modules

For the most scalable approach, create a reusable module for your Solace Cloud services:

**modules/solace-service/main.tf**:
```hcl
variable "name" {
  description = "The name of the service"
  type        = string
}

variable "datacenter_id" {
  description = "The datacenter ID"
  type        = string
}

variable "service_class_id" {
  description = "The service class ID"
  type        = string
  default     = "DEVELOPER"
}

variable "environment_id" {
  description = "The environment ID"
  type        = string
  default     = null
}

variable "message_vpn_name" {
  description = "The message VPN name"
  type        = string
  default     = null
}

variable "max_spool_usage" {
  description = "The maximum spool usage in GB"
  type        = number
  default     = null
}

resource "solacecloud_service" "service" {
  name             = var.name
  datacenter_id    = var.datacenter_id
  service_class_id = var.service_class_id
  environment_id   = var.environment_id
  message_vpn_name = var.message_vpn_name
  max_spool_usage  = var.max_spool_usage
}

output "id" {
  value = solacecloud_service.service.id
}

output "message_vpn" {
  value = solacecloud_service.service.message_vpn
  sensitive = true
}

output "connection_endpoints" {
  value = solacecloud_service.service.connection_endpoints
}
```

**main.tf**:
```hcl
module "dev_service" {
  source          = "./modules/solace-service"
  name            = "dev-service"
  datacenter_id   = "aws-us-east-1a"
  service_class_id = "DEVELOPER"
}

module "qa_service" {
  source          = "./modules/solace-service"
  name            = "qa-service"
  datacenter_id   = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

module "prod_service" {
  source          = "./modules/solace-service"
  name            = "prod-service"
  datacenter_id   = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_10K_HIGHAVAILABILITY"
  max_spool_usage = 100
}
```

## Multi-Region Deployment

For high availability and disaster recovery, you may want to deploy services across [multiple regions](https://docs.solace.com/Cloud/Deployment-Considerations/deployment-options-customer-regions.htm):

```hcl
resource "solacecloud_service" "primary_service" {
  name             = "primary-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_10K_HIGHAVAILABILITY"
}

resource "solacecloud_service" "dr_service" {
  name             = "backup-service"
  datacenter_id    = "aws-us-west-2a"
  service_class_id = "ENTERPRISE_10K_HIGHAVAILABILITY"
}
```

## Using Multiple Environments

Solace Cloud environments allow you to organize services. You can create services in different environments:

```hcl
data "solacecloud_environment" "dev_env" {
  name = "Development"
}

data "solacecloud_environment" "prod_env" {
  name = "Production"
}

resource "solacecloud_service" "dev_service" {
  name             = "dev-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "DEVELOPER"
  environment_id   = data.solacecloud_environment.dev_env.id
}

resource "solacecloud_service" "prod_service" {
  name             = "prod-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_10K_HIGHAVAILABILITY"
  environment_id   = data.solacecloud_environment.prod_env.id
}
```


## Managing Service Credentials

When managing multiple services, it's important to handle credentials securely. You can output the credentials for each service:

```hcl
output "service_credentials" {
  value = {
    dev_service = {
      management_url     = "https://${solacecloud_service.dev_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.dev_service.connection_endpoints[0].ports.management_tls.port}"
      manager_username   = solacecloud_service.dev_service.message_vpn.editor_management_credential.username
      manager_password   = solacecloud_service.dev_service.message_vpn.editor_management_credential.password
      messaging_username = solacecloud_service.dev_service.message_vpn.messaging_client_credential.username
      messaging_password = solacecloud_service.dev_service.message_vpn.messaging_client_credential.password
    },
    prod_service = {
      management_url     = "https://${solacecloud_service.prod_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.prod_service.connection_endpoints[0].ports.management_tls.port}"
      manager_username   = solacecloud_service.prod_service.message_vpn.editor_management_credential.username
      manager_password   = solacecloud_service.prod_service.message_vpn.editor_management_credential.password
      messaging_username = solacecloud_service.prod_service.message_vpn.messaging_client_credential.username
      messaging_password = solacecloud_service.prod_service.message_vpn.messaging_client_credential.password
    }
  }
  sensitive = true
}
```

## Best Practices

When managing multiple Solace Cloud services with Terraform, consider these best practices:

1. **Use Consistent Naming Conventions**: Adopt a consistent naming convention for your services, such as `<environment>-<application>-<purpose>`.

2. **Parameterize Common Values**: Use variables for common values like datacenter IDs and service classes to avoid repetition.

3. **Separate State Files**: Consider using separate Terraform state files for different environments or applications to minimize the blast radius of changes.

4. **Use Terraform Modules**: Create reusable modules for common service patterns to ensure consistency.

5. **Version Control**: Keep your Terraform configurations in version control and use branches for different environments.

6. **Automate Deployments**: Use CI/CD pipelines to automate the deployment of your Terraform configurations.

7. **Monitor Service Limits**: Be aware of your cloud account limits for the number of services you can create.

8. **Document Service Relationships**: Maintain documentation of how your services are related, especially if they are connected via DMR.

## Complete Example

Here's a complete example that demonstrates managing multiple services across different environments:

```hcl
terraform {
  required_providers {
    solacecloud = {
      source = "registry.terraform.io/solaceproducts/solacecloud"
    }
  }
}

provider "solacecloud" {
  base_url  = "https://api.solace.cloud/"
  api_token = var.solace_api_token
}

# Fetch environments
data "solacecloud_environment" "dev" {
  name = "Development"
}

data "solacecloud_environment" "prod" {
  name = "Production"
}

# Create development services
resource "solacecloud_service" "dev_app1" {
  name             = "dev-app1"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "DEVELOPER"
  environment_id   = data.solacecloud_environment.dev.id
  max_spool_usage  = 10
}

resource "solacecloud_service" "dev_app2" {
  name             = "dev-app2"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "DEVELOPER"
  environment_id   = data.solacecloud_environment.dev.id
  max_spool_usage  = 10
}

# Create production services
resource "solacecloud_service" "prod_app1" {
  name             = "prod-app1"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_HIGHAVAILABILITY"
  environment_id   = data.solacecloud_environment.prod.id
  max_spool_usage  = 50
}

resource "solacecloud_service" "prod_app2" {
  name             = "prod-app2"
  datacenter_id    = "aws-us-west-2a"
  service_class_id = "ENTERPRISE_1K_HIGHAVAILABILITY"
  environment_id   = data.solacecloud_environment.prod.id
  max_spool_usage  = 50
}

# Output service credentials (sensitive)
output "service_credentials" {
  value = {
    dev_app1 = {
      manager_username   = solacecloud_service.dev_app1.message_vpn.manager_management_credential.username
      manager_password   = solacecloud_service.dev_app1.message_vpn.manager_management_credential.password
      messaging_username = solacecloud_service.dev_app1.message_vpn.messaging_client_credential.username
      messaging_password = solacecloud_service.dev_app1.message_vpn.messaging_client_credential.password
    },
    dev_app2 = {
      manager_username   = solacecloud_service.dev_app2.message_vpn.manager_management_credential.username
      manager_password   = solacecloud_service.dev_app2.message_vpn.manager_management_credential.password
      messaging_username = solacecloud_service.dev_app2.message_vpn.messaging_client_credential.username
      messaging_password = solacecloud_service.dev_app2.message_vpn.messaging_client_credential.password
    },
    prod_app1 = {
      manager_username   = solacecloud_service.prod_app1.message_vpn.manager_management_credential.username
      manager_password   = solacecloud_service.prod_app1.message_vpn.manager_management_credential.password
      messaging_username = solacecloud_service.prod_app1.message_vpn.messaging_client_credential.username
      messaging_password = solacecloud_service.prod_app1.message_vpn.messaging_client_credential.password
    },
    prod_app2 = {
      manager_username   = solacecloud_service.prod_app2.message_vpn.manager_management_credential.username
      manager_password   = solacecloud_service.prod_app2.message_vpn.manager_management_credential.password
      messaging_username = solacecloud_service.prod_app2.message_vpn.messaging_client_credential.username
      messaging_password = solacecloud_service.prod_app2.message_vpn.messaging_client_credential.password
    }
  }
  sensitive = true
}
```

This example creates four services across two environments, with different configurations for each service. The outputs provide the URLs and credentials for each service, which can be used to configure the services further or connect to them from applications.
