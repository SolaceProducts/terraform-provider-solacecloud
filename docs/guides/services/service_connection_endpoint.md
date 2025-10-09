# Service Connection Endpoint Guide

When you create a service, you have the option of defining a connection endpoint during service creation. 

For example:

```hcl
resource "solacecloud_service" "myservice" {
  datacenter_id = "gke-gcp-us-central1-a"
  name          = "cool-service"
  connection_endpoint = {
    access_type = "PUBLIC"
    name        = "Public Endpoint2"
    description = "MyCoolEndpoint"
    ports = {
      smf_tls = {
        port : 5543
      },
      management_tls = {
        port : 9443
      },
    }
  }
}
```

This creates a service with a connection endpoint named "Public Endpoint2" with the specified ports.

The connection endpoint can be modified after creation by changing any of the attributes except for `access_type`.

The application of changes starts a job that may take several minutes to complete.

