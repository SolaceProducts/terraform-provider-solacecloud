---
page_title: "Provider: Solace Cloud"
description: |-
  The Solace Cloud provider manages Solace Cloud event broker services and their connection endpoints.
---

# Solace Cloud Terraform Provider (Beta)

**Note:** This provider is currently in beta and under active development. Functionality may change and issues may be encountered.

The Solace Cloud Provider for Terraform enables you to manage Solace Cloud resources through Terraform. This provider allows you to create, configure, and manage event broker services programmatically using Terraform's infrastructure as code approach.

## Example Usage

```hcl
terraform {
  required_providers {
    solacecloud = {
      source = "registry.terraform.io/solaceproducts/solacecloud"
    }
  }
}

# Configure the Solace Cloud Provider
provider "solacecloud" {
  base_url             = "https://production-api.solace.cloud/"
  api_token            = var.solace_api_token # or use SOLACECLOUD_API_TOKEN env variable
  api_polling_interval = 30
}

# Fetch the Default environment
data "solacecloud_environment" "environment" {
  name = "Default"
}

# Create a service in the Default environment
resource "solacecloud_service" "broker_service" {
  name             = "my-broker-service"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
  environment_id   = data.solacecloud_environment.environment.id
}
```

## Authentication

The Solace Cloud Terraform Provider offers the following methods for authentication:

- Static credentials in the provider configuration
- Environment variables

### Static Credentials

```hcl
provider "solacecloud" {
  base_url  = "https://api.solace.cloud/"
  api_token = "your-api-token"
}
```

### Environment Variables

```bash
export SOLACECLOUD_API_TOKEN="your-api-token"
```

```hcl
provider "solacecloud" {
  base_url = "https://api.solace.cloud/"
}
```

## Schema

The provider configuration accepts the following arguments.

### Optional

- `base_url` (String) - Base URL for REST API Endpoints. The regional location of your account's Home Cloud determines the base URL you use. The default is `https://production-api.solace.cloud`. For more information, see [Home Cloud](https://docs.solace.com/Cloud/Security/security-home-cloud.htm).
- `api_token` (String, Sensitive) - Token for authenticating with the Solace Cloud API. Can be set as environment variable `SOLACECLOUD_API_TOKEN`.
- `api_polling_interval` (Number) - Polling interval in seconds for API calls that need to wait until a process changes status. For example, wait until a SC service is marked as COMPLETED. Default value is 30 seconds.
