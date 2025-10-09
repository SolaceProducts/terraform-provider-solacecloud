# Resource: solacecloud_connection_endpoint

This resource allows you to create and manage connection endpoints for a Solace Cloud service. Connection endpoints define how clients can connect to your event broker service using various messaging protocols.

## Example Usage

### Basic Connection Endpoint

```hcl
resource "solacecloud_connection_endpoint" "endpoint" {
  service_id  = "myserviceid"
  access_type = "PRIVATE"
  name        = "Public Endpoint2"
  description = "MyCoolEndpoint"

  ports = {
    smf_tls = {
      port = 5543
    }
    mqtt_tls = {
      port = 1000
    }
  }
}
```

### Public Connection Endpoint with Multiple Protocols

```hcl
resource "solacecloud_connection_endpoint" "public_endpoint" {
  service_id  = solacecloud_service.broker_service.id
  access_type = "PUBLIC"
  name        = "Public Multi-Protocol Endpoint"
  description = "Public endpoint with multiple protocol support"

  ports = {
    smf = {
      port = 55555
    }
    smf_tls = {
      port = 55443
    }
    mqtt = {
      port = 1883
    }
    mqtt_tls = {
      port = 8883
    }
    web = {
      port = 80
    }
    web_tls = {
      port = 443
    }
  }
}
```

## Argument Reference

### Required Arguments

* `service_id` - (Required) The ID of the Solace Cloud service that the connection endpoint is associated to.
* `access_type` - (Required) The access type of the connection endpoint. Valid values are `PUBLIC` or `VPC_PEERING`. This cannot be changed after creation.
* `name` - (Required) The name of the connection endpoint.
* `ports` - (Required) A map defining the ports for various protocols. Each protocol can be configured with a port number.

### Optional Arguments

* `description` - (Optional) A description for the connection endpoint.

### Port Configuration

The `ports` argument accepts a map with the following protocol keys. Each protocol is optional, and you can configure only the protocols you need:

* `smf` - Standard Message Format (SMF) port configuration.
* `smf_tls` - SMF over TLS port configuration.
* `smf_compressed` - SMF compressed port configuration.
* `amqp` - AMQP plain-text port configuration.
* `amqp_tls` - AMQP over TLS port configuration.
* `mqtt` - MQTT plain-text port configuration.
* `mqtt_tls` - MQTT over TLS port configuration.
* `mqtt_websocket` - MQTT WebSocket plain-text port configuration.
* `mqtt_websocket_tls` - MQTT WebSocket over TLS port configuration.
* `web` - WebSocket over HTTP port configuration.
* `web_tls` - WebSocket over HTTPS port configuration.
* `rest_incoming` - REST incoming plain-text port configuration.
* `rest_incoming_tls` - REST incoming over TLS port configuration.
* `management_tls` - Management over TLS port configuration (SEMP).
* `ssh_tls` - SSH over TLS port configuration (CLI access).

#### Protocol to REST API Mapping

These protocols map to the REST API protocols as follows:

| Terraform Protocol    | REST API Protocol                   |
|-----------------------|-------------------------------------|
| `smf`                 | `serviceSmfPlainTextListenPort`     |
| `smf_tls`             | `serviceSmfTlsListenPort`           |
| `smf_compressed`      | `serviceSmfCompressedListenPort`    |
| `amqp`                | `serviceAmqpPlainTextListenPort`    |
| `amqp_tls`            | `serviceAmqpTlsListenPort`          |
| `mqtt`                | `serviceMqttPlainTextListenPort`    |
| `mqtt_tls`            | `serviceMqttTlsListenPort`          |
| `mqtt_websocket`      | `serviceMqttWebSocketListenPort`    |
| `mqtt_websocket_tls`  | `serviceMqttTlsWebSocketListenPort` |
| `web`                 | `serviceWebPlainTextListenPort`     |
| `web_tls`             | `serviceWebTlsListenPort`           |
| `rest_incoming`       | `serviceRestIncomingPlainTextListenPort` |
| `rest_incoming_tls`   | `serviceRestIncomingTlsListenPort`  |
| `management_tls`      | `serviceManagementTlsListenPort`    |
| `ssh_tls`             | `managementSshTlsListenPort`        |

## Attribute Reference

* `id` - The unique identifier for the connection endpoint.

## Update Behavior

All parameters except `access_type` can be updated after creation. Applying changes will start a job that may take several minutes to complete.

## Delete Behavior

Deleting the connection endpoint removes it from the associated service. This action may also take several minutes to complete.

## Import

Connection endpoints can be imported using a combination of the service ID and connection endpoint ID separated by a slash.

The format is: `${service_id}/connectionEndpoints/${connection_endpoint_id}`

### Example

```hcl
import {
  to = solacecloud_connection_endpoint.endpoint
  id = "myserviceid/connectionEndpoints/myendpointid"
}

resource "solacecloud_connection_endpoint" "endpoint" {
  service_id  = "myserviceid"
  access_type = "PRIVATE"
  name        = "Public Endpoint2"
  description = "MyCoolEndpoint"

  ports = {
    smf_tls = {
      port = 5543
    }
    mqtt_tls = {
      port = 1000
    }
  }
}
```

Alternatively, using the command line:

```bash
terraform import solacecloud_connection_endpoint.endpoint myserviceid/connectionEndpoints/myendpointid
```
