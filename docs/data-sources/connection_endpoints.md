# Data Source: solacecloud_connection_endpoints

This data source provides information about connection endpoints for a Solace Cloud service. Connection endpoints represent network interfaces through which clients can connect to the service, providing various protocols and ports for communication.

## Example Usage

### List all connection endpoints for a service

```hcl
data "solacecloud_connection_endpoints" "all_endpoints" {
  service_id = solacecloud_service.broker_service.id
}
```

### Filter by endpoint name

```hcl
data "solacecloud_connection_endpoints" "public_endpoint" {
  service_id = solacecloud_service.broker_service.id
  name       = "Default Public"
}
```

### Get a specific endpoint by ID

```hcl
data "solacecloud_connection_endpoints" "specific_endpoint" {
  service_id = solacecloud_service.broker_service.id
  id         = "4cs6600zrgj"
}
```

## Argument Reference

* `service_id` - (Required) The unique identifier of the event broker service.
* `id` - (Optional) The unique identifier of a specific connection endpoint to fetch.
* `name` - (Optional) The name of a specific connection endpoint to filter by.

## Attribute Reference

* `endpoints` - A list of connection endpoint objects. Each endpoint contains:
  * `id` - The unique identifier for the connection endpoint.
  * `type` - The type of object (typically "connectionEndpoint").
  * `name` - The name of the connection endpoint.
  * `description` - A description of the connection endpoint.
  * `access_type` - The access type, either "PUBLIC" or "PRIVATE".
  * `k8s_service_type` - The Kubernetes service type (e.g., "LOADBALANCER").
  * `k8s_service_id` - The Kubernetes service identifier.
  * `ports` - An object containing the available protocol ports. Available protocols include: `amqp`, `amqp_tls`, `management_tls`, `mqtt`, `mqtt_tls`, `mqtt_websocket`, `mqtt_websocket_tls`, `rest_incoming`, `rest_incoming_tls`, `smf`, `smf_compressed`, `smf_tls`, `ssh_tls`, `web`, `web_tls`.