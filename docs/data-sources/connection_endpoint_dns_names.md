# Data Source: solacecloud_connection_endpoint_dns_names

This data source provides information about DNS names for a specific connection endpoint. DNS names are the fully qualified domain names that can be used to connect to the event broker service through the specified connection endpoint.

## Example Usage

```hcl
data "solacecloud_connection_endpoint_dns_names" "endpoint_dns" {
  service_id            = solacecloud_service.broker_service.id
  connection_endpoint_id = "4cs6600zrgj"
}
```

## Argument Reference

* `service_id` - (Required) The unique identifier of the event broker service.
* `connection_endpoint_id` - (Required) The unique identifier of the connection endpoint.

## Attribute Reference

* `dns_names` - A list of DNS name objects. Each DNS name contains:
  * `id` - The identifier of the DNS name.
  * `dns_name` - The fully qualified domain name.
  * `domain_type` - The domain type, either "CustomerManaged" or "SolaceManaged".