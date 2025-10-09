# Importing Existing Services

This guide explains how to import existing event broker services into your Terraform configuration, allowing you to manage them using infrastructure as code.

## Overview

When you have existing event broker services that were created manually or through other means, you can import them into your Terraform state to bring them under Terraform management. This is useful for:

- Managing existing services with Terraform
- Ensuring consistency across your infrastructure
- Implementing infrastructure as code for all your services

## Prerequisites

Before importing services, ensure you have:

1. Terraform installed (version 0.13.0 or later)
2. The Solace Cloud Terraform Provider configured
3. The service ID of the existing service you want to import

## Finding Your Service ID

To import a service, you need its service ID. You can find this in the Solace Cloud Console:

1. Log in to your [Solace Cloud account](https://console.solace.cloud/). For more information, see [Logging In to the Cloud Console](https://docs.solace.com/Cloud/cloud-login-urls.htm).
2. Navigate to the Services page.
3. Click on the service you want to import.
4. The service ID is displayed in the service details page or in the URL (for example, `https://console.solace.cloud/services/{service-id}`)

## Importing Services with Terraform Import Blocks

Terraform 1.5.0 and later supports the `import` block, which provides a more declarative way to import resources. This is the recommended approach for importing event broker services.

### Step 1: Create a Terraform Configuration with Import Blocks

Create a Terraform configuration file (`main.tf`) with import blocks for the services you want to import:

```hcl
terraform {
  required_providers {
    solacecloud = {
      source = "registry.terraform.io/solaceproducts/solacecloud"
    }
  }
}

provider "solacecloud" {
  base_url = "https://api.solace.cloud/"
  # API token can be provided via environment variable SOLACECLOUD_API_TOKEN
}

# Import block for the service
import {
  id = "your-service-id"
  to = solacecloud_service.imported_service
}

resource "solacecloud_service" "imported_service" {
  name             = "your-service-name"
  datacenter_id    = "gke-gcp-us-central1-a"
  service_class_id = "ENTERPRISE_250_HIGHAVAILABILITY"
}
```

### Step 2: Apply the Configuration

Apply it to import the services into your Terraform state:

```bash
terraform apply
```

After a successful import, the services will be managed by Terraform, and you can make changes to them using your Terraform configuration.

## Complete Import Example with Generated Configuration

Here's a complete example of importing existing services using the import block:

### 1. Create the initial configuration file (`import.tf`):

```hcl
terraform {
  required_providers {
    solacecloud = {
      source = "registry.terraform.io/solaceproducts/solacecloud"
    }
  }
}

provider "solacecloud" {
  base_url = "https://api.solace.cloud/"
  # API token can be provided via environment variable SOLACECLOUD_API_TOKEN
}

# Import blocks for the services
import {
  id = "your-service-id"
  to = solacecloud_service.service1
}

import {
  id = "your-service-id2"
  to = solacecloud_service.service2
}

resource "solacecloud_service" "service1" {
  name             = "your-service-name"
  datacenter_id    = "gke-gcp-us-central1-a"
  service_class_id = "ENTERPRISE_250_HIGHAVAILABILITY"
}

resource "solacecloud_service" "service2" {
  name             = "your-service-name2"
  datacenter_id    = "gke-gcp-us-central1-a"
  service_class_id = "ENTERPRISE_250_HIGHAVAILABILITY"
}
```

### 2. Apply the configuration:

```bash
terraform apply
```

## Best Practices for Importing Services

1. **Document Service IDs**: Keep a record of service IDs and their corresponding Terraform resource names.

2. **Start with a Clean State**: Before importing, ensure your Terraform state doesn't already have resources with the same names.

3. **Version Control**: Keep your Terraform configurations in version control to track changes.

4. **Test in Non-Production**: Test the import process in a non-production environment first.

5. **Backup State Files**: Always backup your Terraform state files before making significant changes.

6. **Use Workspaces**: Consider using Terraform workspaces to separate different environments.

7. **Validate After Import**: After importing, validate that the service works as expected and that Terraform can manage it properly.

## Handling Service Upgrades

When a service is upgraded or upscaled, you have two options:

1. **Remove the version attribute**: If you remove the `event_broker_version` from your Terraform configuration, it won't be managed by Terraform at all and will never be changed by Terraform.

2. **Update the version in the configuration**: If you want to keep the version synchronized across environments, update the `event_broker_version` in your Terraform configuration to match what it should be.

## Handling New Configuration Options

When new configuration options are added to the Solace Cloud API:

- **Optional attributes**: If the new attributes are optional, your existing Terraform configuration will continue to work without changes.
- **Required attributes**: If new required attributes are added, you may need to update your Terraform configuration to include these attributes.

## Troubleshooting Import Issues

### Error: Resource Not Found

**Error Message:**
```
Error: Resource Import Failed
Could not find service with ID 'service-id'
```

**Solution:**

1. Verify that the service ID is correct. For more information, (see [Finding Your Service ID](#finding-your-service-id) section)
2. Ensure your API token has the correct permissions. In this case, you require `services:post`. For more information, see [Managing API Tokens](https://docs.solace.com/Cloud/ght_api_tokens.htm).
3. Check if the service still exists in the Cloud Console. For more information, see [Viewing Event Broker Services](https://docs.solace.com/Cloud/cloud-configure-messaging-services.htm).

### Error: Resource Already Exists

**Error Message:**
```
Error: Resource already managed by Terraform
```

**Solution:**
1. Check if the resource is already in your Terraform state
2. Use `terraform state list` to see all resources in your state
3. Use a different resource name for the import

### Error: Invalid Resource Type

**Error Message:**
```
Error: Invalid resource type
```

**Solution:**
1. Ensure you're using the correct resource type (`solacecloud_service`)
2. Check your provider configuration and version
