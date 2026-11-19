---
page_title: "solacecloud_connection_endpoint_dns_name_move Resource - terraform-provider-solacecloud"
subcategory: ""
---

# Resource: solacecloud_connection_endpoint_dns_name_move

This resource moves a custom DNS name (hostname) from one connection endpoint to another, on the same event broker service or on another service in the same organization. The DNS name cannot be moved to a different organization.

A move is a one-time action. Creating the resource starts the move and waits for it to finish; the resource then tracks that the DNS name is on the target connection endpoint.

## Example Usage

The following examples show how to configure the resource.

### Moving a DNS Name to Another Service

```hcl
resource "solacecloud_connection_endpoint_dns_name_move" "move" {
  source_service_id             = solacecloud_service.old_service.id
  source_connection_endpoint_id = solacecloud_connection_endpoint.old_endpoint.id
  dns_name                      = "broker.messaging.solace.cloud"
  target_service_id             = solacecloud_service.new_service.id
  target_connection_endpoint_id = solacecloud_connection_endpoint.new_endpoint.id
}
```

### Moving a DNS Name Between Connection Endpoints of the Same Service

```hcl
resource "solacecloud_connection_endpoint_dns_name_move" "move" {
  source_service_id             = solacecloud_service.broker_service.id
  source_connection_endpoint_id = solacecloud_connection_endpoint.public_endpoint.id
  dns_name                      = "broker.messaging.solace.cloud"
  target_connection_endpoint_id = solacecloud_connection_endpoint.private_endpoint.id
}
```

## Argument Reference

The following arguments are supported.

### Required Arguments

* `source_service_id` - (Required) The unique identifier of the source event broker service. Cannot be changed after creation.
* `source_connection_endpoint_id` - (Required) The unique identifier of the source connection endpoint. Cannot be changed after creation.
* `dns_name` - (Required) The DNS name to move. Cannot be changed after creation.

### Optional Arguments

* `target_service_id` - (Optional) The unique identifier of the target event broker service. If not specified, defaults to the source service ID. Cannot be changed after creation.
* `target_connection_endpoint_id` - (Optional) The unique identifier of the target connection endpoint. Cannot be changed after creation. Always set it: the provider does not look up a default endpoint, and it reads the DNS name back from this endpoint on every refresh.

## Attribute Reference

* `id` - The operation identifier of the move.
* `target_service_id` - The target service ID. When you omit it, the provider sets it to `source_service_id`.
* `target_connection_endpoint_id` - The target connection endpoint ID, as set in the configuration.

## DNS Name Move Limitations

* The target connection endpoint can have at most 5 DNS names. The move fails if the target already has 5.
* The source and target services must be in the same organization.

## Update Behavior

A move cannot be updated. Any change to an argument replaces the resource: Terraform removes the old move from the state and starts a new move from `source_service_id` and `source_connection_endpoint_id`. After a move the DNS name is no longer on the source connection endpoint, so to move it again, set the `source_*` arguments to where the DNS name is now.

## Delete Behavior

Destroying the resource removes it only from the Terraform state. The DNS name stays on the target connection endpoint; it is not moved back. If the DNS name or the target connection endpoint is gone, the resource is removed from the state on the next refresh.

## Import

Import is not supported. To manage a DNS name that was already moved, use the [`solacecloud_connection_endpoint_dns_name`](connection_endpoint_dns_name.md) resource.
