# Getting Started with Solace Cloud Provider

This guide will help you get started with the Solace Cloud Terraform Provider, allowing you to manage your Solace Cloud resources using Terraform.

## Prerequisites

Before you begin, ensure you have:

1. [Terraform](https://www.terraform.io/downloads.html) installed (version 0.13.0 or later)
2. A Solace Cloud account with the Mission Control Manager or Administrator role. For more information, see [Managing Users, Groups, Roles and Permissions](https://docs.solace.com/Cloud/cloud-user-management.htm).
3. A Solace Cloud API token with the `services:post` permissions

## Obtaining a Solace Cloud API Token

To use the Solace Cloud Terraform Provider, you'll need an API token:

1. Log in to your [Solace Cloud account](https://console.solace.cloud/)
2. Click User & Accounts (lower-left corner) and select Token Management.
3. Enter a name for your API token in the Token Name field.
4. Select the checkboxes for the required permissions, in this case, at least Create Services (`services:post`).
5. Click Create Token.
6. On the API Token dialog, click Copy (you won't see the token again after closing the dialog).

## Provider Configuration

Create a new directory for your Terraform configuration and create a file named `main.tf` with the following content:

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

variable "solace_api_token" {
  description = "Solace Cloud API token"
  type        = string
  sensitive   = true
}
```

Create a `terraform.tfvars` file to store your API token (make sure to add this file to your `.gitignore` to avoid committing sensitive information):

```hcl
solace_api_token = "your-api-token-here"
```

## Creating Your First Service

Add the following to your `main.tf` file to create an event broker service:

```hcl
# Fetch the Default environment
data "solacecloud_environment" "environment" {
  name = "Default"
}

# Create a service in the Default environment
resource "solacecloud_service" "broker_service" {
  name             = "my-first-service"
  datacenter_id    = "aws-us-east-1a"  # Replace with an available datacenter
  service_class_id = "DEVELOPER"
  environment_id   = data.solacecloud_environment.environment.id
}

# Output the service details
output "service_id" {
  value = solacecloud_service.broker_service.id
}

output "service_management_url" {
  value = "https://${solacecloud_service.broker_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service.connection_endpoints[0].ports.management_tls.port}"
}
```

## Initialize and Apply

Initialize the Terraform working directory:

```bash
terraform init
```

Apply the configuration to create the resources:

```bash
terraform apply
```

When prompted, confirm the action by typing `yes`.

## Accessing Your Service

After the service is created, you can access it using Terraform state:

```bash
terraform state show solacecloud_service.broker_service
```

You can also access your service through the Cloud Console. For more information, see, see [Logging In to the Cloud Console](https://docs.solace.com/Cloud/cloud-login-urls.htm).

## Modifying Your Service

To modify your service, update the configuration in your `main.tf` file and run `terraform apply` again. For example, to increase the message spool size:

```hcl
resource "solacecloud_service" "broker_service" {
  name             = "my-first-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "DEVELOPER"
  environment_id   = data.solacecloud_environment.environment.id
  max_spool_usage  = 20  # Increase spool size to 20GB
}
```

## Destroying Resources

When you're done, you can destroy the resources:

```bash
terraform destroy
```

When prompted, confirm the action by typing `yes`.

## Next Steps

Now that you've created your first Solace Cloud service with Terraform, you can:

1. Explore the [Solace Cloud Provider documentation](../index.md) for more details on available resources and data sources.
2. Learn how to [configure your service](./service_configuration.md) with advanced options.
3. Set up [multiple services](./multiple_services.md) across different environments.
4. Integrate with the [Solace Broker Provider](./broker_integration.md) to configure messaging resources within your service.

## Troubleshooting

If you encounter any issues, refer to the [Troubleshooting Guide](../troubleshooting.md) for common problems and solutions.
