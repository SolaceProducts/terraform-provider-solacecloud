# Connection Endpoint Data Source

The connection endpoint data source allows querying connection endpoints associated with a Solace Cloud service. 

A connection endpoint represents a network interface through which clients can connect to the service. 
Each endpoint provides various protocols and ports for communication.


## Arguments
- `service_id` - (Required) The ID of the Solace Cloud service that the connection endpoint is associated to.
- `name` - (Optional) The name of the connection endpoint to filter by.

## Example Usage

```hcl
data "solacecloud_connection_endpoints" "endpoints" {
    service_id = solacecloud_service.broker_service.id
}
```

## Example Output
```json
{
          "schema_version": 0,
          "attributes": {
            "endpoints": [
              {
                "access_type": "PUBLIC",
                "description": "",
                "id": "4cs6600zrgj",
                "k8s_service_id": "kilo-sa-4cs6600zrgj-solace",
                "k8s_service_type": "LOADBALANCER",
                "name": "Default Public",
                "ports": {
                  "amqp": null,
                  "amqp_tls": {
                    "port": 5671
                  },
                  "management_tls": {
                    "port": 943
                  },
                  "mqtt": null,
                  "mqtt_tls": {
                    "port": 8883
                  },
                  "mqtt_websocket": null,
                  "mqtt_websocket_tls": {
                    "port": 8443
                  },
                  "rest_incoming": null,
                  "rest_incoming_tls": {
                    "port": 9443
                  },
                  "smf": null,
                  "smf_compressed": null,
                  "smf_tls": {
                    "port": 55443
                  },
                  "ssh_tls": {
                    "port": 22
                  },
                  "web": null,
                  "web_tls": {
                    "port": 443
                  }
                },
                "type": "connectionEndpoint"
              }
            ],
            "id": null,
            "name": null,
            "service_id": "46t72f22k8t"
          },
          "sensitive_attributes": [],
          "identity_schema_version": 0
        }

```

