---
page_title: "solacecloud_connection_endpoint_dns_name Resource - terraform-provider-solacecloud"
subcategory: ""
---

# Resource: solacecloud_connection_endpoint_dns_name

This resource allows you to create and manage custom DNS names (hostnames) for Solace Cloud connection endpoints.

## About Custom Hostnames

Each connection endpoint has a unique, randomly generated hostname, such as `mr1egxydp8fguv.messaging.solace.cloud`. You can configure custom hostnames to provide personalized names in the `messaging.solace.cloud` domain or use your own custom domains.

Custom hostnames are useful for:
- **Migration** - Maintain consistent hostnames when migrating from other messaging platforms.
- **Client integration** - Use short, memorable names that align with your naming conventions.
- **Application partitioning** - Assign distinct hostnames to different applications or environments.

You can access the event broker service using any configured custom hostname, as well as the initial generated hostname.

Before you use this resource, check the following:

- You have an existing event broker service (`solacecloud_service`).
- You have configured a connection endpoint (`solacecloud_connection_endpoint`).
- If you use a custom domain outside `messaging.solace.cloud`, you own and control that domain.


## Example Usage

The following example shows how to configure the resource.

### Basic DNS Name

```hcl
resource "solacecloud_connection_endpoint_dns_name" "custom_hostname" {
  service_id             = solacecloud_service.broker_service.id
  connection_endpoint_id = solacecloud_connection_endpoint.endpoint.id
  dns_name               = "broker.messaging.solace.cloud"
}
```

## Argument Reference

The following arguments are supported.

### Required Arguments

* `service_id` - (Required) The unique identifier of the event broker service. Cannot be changed after creation.
* `connection_endpoint_id` - (Required) The unique identifier of the connection endpoint. Cannot be changed after creation.
* `dns_name` - (Required) The fully qualified domain name (FQDN) to use as the custom hostname. Cannot be changed after creation.

### DNS Name Validation

The `dns_name` argument must meet the following requirements:

* Use a valid fully qualified domain name (FQDN).
* Limit each label (segment between dots) to 1-63 characters.
* Include only alphanumeric characters and hyphens in labels.
* Cannot start or end a label with hyphens.
* Keep the total length to 230 characters or fewer.
* Include at least one dot (e.g., `broker.example.com` is valid, but `broker` is not).

The following examples are valid:
* `broker.example.com`
* `my-broker.prod.company.com`
* `event-mesh-1.us-east.example.org`

The following examples are not valid:
* `broker` (missing domain)
* `-broker.example.com` (label starts with hyphen)
* `broker-.example.com` (label ends with hyphen)
* `broker..example.com` (empty label)

## Attribute Reference

* `id` - The unique identifier for the DNS name.
* `domain_type` - Indicates who owns the domain. Valid values:
  * `CustomerManaged` - You own and control the domain (e.g., `broker.example.com`).
  * `SolaceManaged` - Solace owns the domain (e.g., `broker.messaging.solace.cloud`).

## DNS Name Limitations

* Each connection endpoint supports a maximum of 5 DNS names.
* DNS names are immutable - any changes require resource replacement (destroy and recreate).
* The default Solace-provided DNS name cannot be deleted.

## Update Behavior

DNS names cannot be updated after creation. Any changes to `service_id`, `connection_endpoint_id`, or `dns_name` replace the resource (destroy and recreate).

## Delete Behavior

Deleting a DNS name resource removes the custom hostname from the connection endpoint. This is an asynchronous operation that may take several minutes to complete. The default Solace-provided DNS name cannot be deleted.

## Import

DNS names can be imported using a combination of the service ID, connection endpoint ID, and DNS name separated by slashes.

The format is: `${service_id}/${connection_endpoint_id}/${dns_name}`

### Example

**Using Terraform 1.5+ import blocks :**

```hcl
import {
  to = solacecloud_connection_endpoint_dns_name.custom_hostname
  id = "myserviceid/myendpointid/broker.example.com"
}

resource "solacecloud_connection_endpoint_dns_name" "custom_hostname" {
  service_id             = "myserviceid"
  connection_endpoint_id = "myendpointid"
  dns_name               = "broker.example.com"
}
```

**Using the command line (for one-off imports):**

```bash
terraform import solacecloud_connection_endpoint_dns_name.custom_hostname myserviceid/myendpointid/broker.example.com
```

