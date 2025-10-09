 # Connection Endpoint Resource
 
The connection endpoint resource allows you to create and manage connection endpoints for a Solace Cloud service.

## Configuration
- `service_id` - (Required) The ID of the Solace Cloud service to which the connection endpoint will be associated.
- `access_type` - (Required) The access type of the connection endpoint. Valid values are `PUBLIC` or `VPC_PEERING`.
- `name` - (Required) The name of the connection endpoint.
- `description` - (Optional) A description for the connection endpoint.
- `ports` - (Required) A map defining the ports for various protocols.

The available protocols and their corresponding port configurations are:
- `smf` - Standard Message Format (SMF) port configuration.
- `smf_tls` - SMF over TLS port configuration.
- `smf_compressed` - SMF compressed port configuration.
- `amqp` - AMQP plain-text port configuration.
- `amqp_tls` - AMQP over TLS port configuration.
- `mqtt` - MQTT plain-text port configuration.
- `mqtt_tls` - MQTT over TLS port configuration.
- `mqtt_websocket` - MQTT WebSocket plain-text port configuration.
- `mqtt_websocket_tls` - MQTT WebSocket over TLS port configuration.
- `web` - WebSocket over HTTP port configuration.
- `web_tls` - WebSocket over HTTPS port configuration.
- `rest_incoming` - REST incoming plain-text port configuration.
- `rest_incoming_tls` - REST incoming over TLS port configuration.
- `management_tls` - Management over TLS port configuration (SEMP).
- `ssh_tls` - SSH over TLS port configuration (CLI access).

These protocols can be mapped to the REST API protocols as follows:
- `smf` -> `serviceSmfPlainTextListenPort`
- `smf_tls` -> `serviceSmfTlsListenPort`
- `smf_compressed` -> `serviceSmfCompressedListenPort`
- `amqp` -> `serviceAmqpPlainTextListenPort`
- `amqp_tls` -> `serviceAmqpTlsListenPort`
- `mqtt` -> `serviceMqttPlainTextListenPort`
- `mqtt_tls` -> `serviceMqttTlsListenPort`
- `mqtt_websocket` -> `serviceMqttWebSocketListenPort`
- `mqtt_websocket_tls` -> `serviceMqttTlsWebSocketListenPort`
- `web` -> `serviceWebPlainTextListenPort`
- `web_tls` -> `serviceWebTlsListenPort`
- `rest_incoming` -> `serviceRestIncomingPlainTextListenPort`
- `rest_incoming_tls` -> `serviceRestIncomingTlsListenPort`
- `management_tls` -> `serviceManagementTlsListenPort`
- `ssh_tls` -> `managementSshTlsListenPort`

### Example Usage

```hcl
resource "solacecloud_connection_endpoint" "endpoint" {
  access_type = "PRIVATE"
  name        = "Public Endpoint2"
  service_id  = "myserviceid" 
  description = "MyCoolEndpoint"
  ports = {
    smf_tls = {
      port : 5543
    },
    mqtt_tls = {
      port : 1000
    }
  }
}
```

## Update
All the parameters except for `access_type` can be updated. Applying changes starts a job that may take several minutes to complete.

## Delete
Deleting the connection endpoint removes it from the associated service. This action may also take several minutes.

## Import

The import ID is not a normal connection endpoint ID. It is a combination of the service ID and the connection endpoint ID separated by a slash.

The exact format is: `${service_id}/connectioneEndpoints/${connection_endpoint_id}`


### Example
```hcl
import {
  to =solacecloud_service_connection_endpoint.endpoint
  id = "myserviceid/connectionEndpoints/myendpointid"
}

resource "solacecloud_connection_endpoint" "endpoint" {
  access_type = "PRIVATE"
  name        = "Public Endpoint2"
  service_id  = "myserviceid" 
  description = "MyCoolEndpoint"
  ports = {
    smf_tls = {
      port : 5543
    },
    mqtt_tls = {
      port : 1000
    }
  }
}





```










