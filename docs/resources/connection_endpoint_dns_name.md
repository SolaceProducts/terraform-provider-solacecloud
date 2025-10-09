# Resource: solacecloud_connection_endpoint_dns_name

This resource allows you to create and manage custom DNS names (hostnames) for Solace Cloud connection endpoints.

## About Custom Hostnames

Each connection endpoint has a unique, randomly generated hostname, such as `mr1egxydp8fguv.messaging.solace.cloud`. You can configure custom hostnames to provide personalized names in the `messaging.solace.com` domain or use your own custom domains.

Custom hostnames are useful for:
- **Migration** - Maintain consistent hostnames when migrating from other messaging platforms.
- **Client integration** - Use simpler, memorable names that align with your naming conventions.
- **Application partitioning** - Assign distinct hostnames to different applications or environments.

You can access the event broker service using any configured custom hostname, as well as the initial generated hostname.

**Prerequisites:**
- An existing event broker service (`solacecloud_service`)
- A configured connection endpoint (`solacecloud_connection_endpoint`)
- For custom domains outside `messaging.solace.com`, you must own and control the domain 


## Example Usage

### Basic DNS Name

```hcl
resource "solacecloud_connection_endpoint_dns_name" "custom_hostname" {
  service_id             = solacecloud_service.broker_service.id
  connection_endpoint_id = solacecloud_connection_endpoint.endpoint.id
  dns_name               = "broker.messaging.solace.cloud"
}
```

## Argument Reference

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
* Keep total length under 230 characters.
* Include at least one dot (e.g., `broker.example.com` is valid, but `broker` is not).

**Valid examples:**
* `broker.example.com`
* `my-broker.prod.company.com`
* `event-mesh-1.us-east.example.org`

**Invalid examples:**
* `broker` (missing domain)
* `-broker.example.com` (label starts with hyphen)
* `broker-.example.com` (label ends with hyphen)
* `broker..example.com` (empty label)

## Attribute Reference

* `id` - The unique identifier for the DNS name.
* `domain_type` - Indicates who owns the domain. Valid values:
  * `CustomerManaged` - You own and control the domain (e.g., `broker.example.com`).
  * `SolaceManaged` - Solace owns the domain (e.g., `broker.messaging.solace.cloud`).

## Limitations

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

