# Data Source: solacecloud_environment

This data source provides information about a Solace Cloud environment. Environments in Solace Cloud are used to organize and manage services, allowing for better control over resource allocation and access. For more information, see [Creating and Managing Environments](https://docs.solace.com/Cloud/environments.htm).

## Example Usage

```hcl
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

## Argument Reference

* `name` - (Required) The name of the environment to fetch.

## Attribute Reference

* `id` - The unique identifier for this environment.
* `type` - The type of object for informational purposes (typically "environment").
