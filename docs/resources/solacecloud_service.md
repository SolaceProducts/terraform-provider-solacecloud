# Resource: solacecloud_service

This resource allows you to create and manage event broker services. A service represents an instance of a Solace event broker instance running in the cloud.

## Example Usage

### Basic Service Creation

```hcl
resource "solacecloud_service" "broker_service" {
  name             = "my-broker-service"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}
```

### Service with Environment

```hcl
# Fetch the Default environment
data "solacecloud_environment" "environment" {
  name = "Default"
}

resource "solacecloud_service" "broker_service" {
  name             = "my-broker-service"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
  environment_id   = data.solacecloud_environment.environment.id
}
```

### Service with Custom Configuration

```hcl
resource "solacecloud_service" "broker_service" {
  name                 = "my-broker-service"
  datacenter_id        = "eks-eu-central-1a"
  service_class_id     = "ENTERPRISE_1K_STANDALONE"
  message_vpn_name     = "my-custom-vpn"
  max_spool_usage      = 20
  mate_link_encryption = true
  locked               = false
}
```

### Using Service Credentials with Solace Broker Provider

```hcl
terraform {
  required_providers {
    solacecloud = {
      source = "registry.terraform.io/solaceproducts/solacecloud"
    }
    solacebroker = {
      source  = "SolaceProducts/solacebroker"
      version = "1.1.0"
    }
  }
}

resource "solacecloud_service" "broker_service" {
  name             = "my-broker-service"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

provider "solacebroker" {
  url      = "https://${solacecloud_service.broker_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service.connection_endpoints[0].ports.management_tls.port}"
  username = solacecloud_service.broker_service.message_vpn.manager_management_credential.username
  password = solacecloud_service.broker_service.message_vpn.manager_management_credential.password
}

resource "solacebroker_msg_vpn_queue" "queue1" {
  queue_name     = "my-queue"
  msg_vpn_name   = "msgvpn-${solacecloud_service.broker_service.id}"
  ingress_enabled = true
  egress_enabled  = true
  max_msg_size    = 10000
  partition_count = 1
}
```

## Argument Reference

### Required Arguments

* `name` - (Required) The event broker service name. Must be between 1 and 50 characters.
* `datacenter_id` - (Required) The identifier of the datacenter where the service will be deployed. Must be between 1 and 50 characters.

### Optional Arguments

* `service_class_id` - (Optional) The identifier of the service class. Default is "DEVELOPER". Valid values include:
  * `DEVELOPER`
  * `ENTERPRISE_250_HIGHAVAILABILITY`
  * `ENTERPRISE_1K_HIGHAVAILABILITY`
  * `ENTERPRISE_50K_HIGHAVAILABILITY`
  * `ENTERPRISE_100K_HIGHAVAILABILITY`
  * `ENTERPRISE_5K_HIGHAVAILABILITY`
  * `ENTERPRISE_10K_HIGHAVAILABILITY`
  * `ENTERPRISE_250_STANDALONE`
  * `ENTERPRISE_1K_STANDALONE`
  * `ENTERPRISE_5K_STANDALONE`
  * `ENTERPRISE_10K_STANDALONE`
  * `ENTERPRISE_50K_STANDALONE`
  * `ENTERPRISE_100K_STANDALONE`

* `event_broker_version` - (Optional) The event broker version. A default version is provided when this is not specified. The format is release.year or release.year.release type.build number-revision. For more information, see [Release and Versioning Scheme for Event Broker Services](https://docs.solace.com/Cloud/broker-version-conventions.htm).

* `message_vpn_name` - (Optional) The message VPN name. A default message VPN name is provided when this is not specified. Must be between 1 and 26 characters, may only contain alphanumeric, - or _ characters, must begin with alphabetic or _ characters, and cannot be 'default'. For more information, see [Viewing and Managing the Message VPN](https://docs.solace.com/Cloud/Broker-Manager/message-vpn-settings.htm).

* `max_spool_usage` - (Optional) The message spool size, in gigabytes (GB). A default message spool size is provided if this is not specified. Must be between 10 and 6000. For more information, see [Configuring Message Spool Sizes](https://docs.solace.com/Cloud/Configure-Message-Spools.htm).

* `cluster_name` - (Optional) The name of the DMR cluster where the service will be created. Must be between 1 and 64 characters, may only contain alphanumeric, - or _ characters, must begin with alphabetic or _ characters, and cannot be 'default'.

* `owned_by` - (Optional, Computed) The unique identifier representing the user who owns the event broker service.

* `locked` - (Optional, Computed) Indicates if you can delete the event broker service after creating it. For more information, see [Using Deletion Protection](https://docs.solace.com/Cloud/ght_service_deletion_protection.htm). The default value is false. The valid values are:
  * `true` - You cannot delete this service
  * `false` - You can delete this service

* `mate_link_encryption` - (Optional, Computed) For high-availability (HA) services, you can enable encryption of the mate-link connection between the primary and backup brokers, also known as redundancyGroupSSL in the V2 REST API documentation. For more information, see [HA-Link Security](https://docs.solace.com/Cloud/ha_concept.htm?Highlight=mate-link#ha-link-security). The default value is true. The valid values are:
  * `true` - Enabled
  * `false` - Disabled

* `custom_router_name` - (Optional) The unique prefix for the name of the router for the event broker service. If left undefined, the service ID will be used. Defining this is useful when replacing a part of a DMR Cluster or DR setup. The value should be left undefined for most use cases.

* `environment_id` - (Optional, Computed) The unique identifier of the environment where you want to create the service. You can only specify an environment identifier when creating services in a Public Region. You cannot specify an environment identifier when creating a service in a Dedicated Region. Creating a service in a Public Region without specifying an environment identifier places it in the default environment.

## Attribute Reference

* `id` - The unique identifier for the event broker service.

* `message_vpn` - The Message VPN details. This is a complex object with the following attributes:
  * `name` - The name of the Message VPN.
  * `authentication_basic_enabled` - Indicates whether basic authentication is enabled.
  * `authentication_basic_type` - The authentication type. One of: "INTERNAL", "LDAP", "RADIUS", "NONE".
  * `authentication_client_cert_enabled` - Indicates whether client certificate authentication is enabled.
  * `authentication_client_cert_validate_date_enabled` - Indicates whether the validation of the 'Not Before' and 'Not After' dates in a client certificate is enabled.
  * `max_connection_count` - The maximum number of clients that are permitted to simultaneously connect to the Message VPN.
  * `max_egress_flow_count` - The total permitted number of egress flows for a Message VPN.
  * `max_endpoint_count` - The maximum number of flows that can bind to a non-exclusive durable topic endpoint.
  * `max_ingress_flow_count` - The total permitted number of ingress flows for a Message VPN.
  * `max_msg_spool_usage` - The maximum message spool usage.
  * `max_subscription_count` - The maximum number of unique subscriptions.
  * `max_transacted_session_count` - The maximum number of simultaneous transacted sessions and/or XA Sessions allowed for the given Message VPN.
  * `max_transaction_count` - The total number of simultaneous transactions in a Message VPN.
  * `truststore_uri` - The URI for the TLS trust store.
  * `manager_management_credential` - The credentials for the manager management user.
    * `username` - The username.
    * `password` - The password (sensitive).
  * `editor_management_credential` - The credentials for the editor management user.
    * `username` - The username.
    * `password` - The password (sensitive).
  * `viewer_management_credential` - The credentials for the viewer management user.
    * `username` - The username.
    * `password` - The password (sensitive).
  * `messaging_client_credential` - The credentials for the messaging client.
    * `username` - The username.
    * `password` - The password (sensitive).

* `connection_endpoints` - The list of Connection Endpoints for this service. Each connection endpoint has the following attributes:
  * `id` - The identifier of the connection endpoint.
  * `name` - The name of the connection endpoint.
  * `description` - The description for the connection endpoint.
  * `access_type` - The connectivity for the connection endpoint. Either "PRIVATE" (private IP) or "PUBLIC" (public Internet IP).
  * `k8s_service_type` - The connectivity configuration that is used in the Kubernetes cluster. One of: "NodePort", "LoadBalancer", "ClusterIP".
  * `k8s_service_id` - The identifier for the Kubernetes service.
  * `hostnames` - The hostnames assigned to the connection endpoint.
  * `ports` - The protocols and port numbers of the connection endpoint. This is a complex object with the following possible attributes:
    * `web` - WebSocket over HTTP (plain-text).
    * `web_tls` - WebSocket over secured HTTP.
    * `management_tls` - Secured management connection using SEMP.
    * `rest_incoming_tls` - Secure REST messaging.
    * `amqp` - AMQP (plain-text).
    * `mqtt_websocket` - MQTT WebSocket (plain-text).
    * `rest_incoming` - REST messaging (plain-text).
    * `smf_compressed` - SMF (plain-text) in a compressed format over TCP.
    * `mqtt` - MQTT (plain-text).
    * `smf` - SMF Host (plain-text) over TCP.
    * `amqp_tls` - AMQP over a secure TCP connection.
    * `mqtt_tls` - Secure MQTT.
    * `smf_tls` - Secure SMF using TLS over TCP.
    * `mqtt_websocket_tls` - WebSocket secured MQTT.
    * `ssh_tls` - Secure port for Solace Command Line Interface (CLI).

    Each protocol, when enabled, contains:
    * `port` - The port number for the protocol.

* `dmr_cluster` - The DMR cluster details. This is a complex object with the following attributes:
  * `name` - The name of the DMR cluster.
  * `password` - The password for the cluster (sensitive).
  * `remote_address` - The address of the remote node in the cluster.
  * `primary_router_name` - The name of the primary router in the DMR cluster.
  * `supported_authentication_modes` - The authentication mode between the nodes in the DMR cluster.

## Import

You can import event broker services using the service ID:

```bash
terraform import solacecloud_service.broker_service service-id
```
