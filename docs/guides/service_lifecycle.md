# Service Lifecycle Management

This guide explains how to manage the lifecycle of Solace Cloud services using the Terraform provider, including service upscaling, upgrading, and understanding the relationship between Terraform attributes and the Solace Cloud V2 API.

## Service Upscaling

Upscaling a service involves increasing its capacity to handle more messages, connections, or storage. In the Solace Cloud Terraform Provider, you can upscale a service by modifying certain attributes.

### Modifiable Attributes for Upscaling

The following attributes can be modified to upscale a service:

1. **max_spool_usage** - Increase the message spool size to handle more guaranteed messages. For more information, see [Configuring Message Spool Sizes](https://docs.solace.com/Cloud/Configure-Message-Spools.htm).

Example of increasing the message spool size:

```hcl
resource "solacecloud_service" "broker_service" {
  name             = "my-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
  max_spool_usage  = 50  # Increase from the default or previous value
}
```

### Upscaling Behavior

When you increase the `max_spool_usage` attribute and apply the changes:

1. Terraform will call the Solace Cloud API to update the service.
2. The service will remain online during the upscaling process.
3. The operation is performed without message loss.
4. The process may take several minutes to complete, depending on the size of the change.

### Upscaling Limitations

- You cannot decrease the `max_spool_usage` value once it has been increased.
- The maximum value for `max_spool_usage` is 6000 GB.
- The minimum value for `max_spool_usage` is 10 GB.

## Service Class Changes and Immutable Attributes

Some attributes of a service cannot be changed after creation and require creating a new service. These immutable attributes include:

1. **service_class_id** - The service class (e.g., DEVELOPER, ENTERPRISE_1K_STANDALONE).
2. **datacenter_id** - The datacenter where the service is deployed.
3. **message_vpn_name** - The name of the Message VPN.
4. **event_broker_version** - The version of the event broker.
5. **cluster_name** - The name of the DMR cluster.
6. **custom_router_name** - The custom router name prefix.
7. **environment_id** - The environment where the service is deployed.

If you need to change any of these attributes, you must:

1. Create a new service with the desired attributes.
2. Migrate your data and configuration to the new service.
3. Delete the old service once migration is complete.

Example of creating a new service with a different service class:

```hcl
# Original service
resource "solacecloud_service" "old_service" {
  name             = "old-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

# New service with upgraded service class
resource "solacecloud_service" "new_service" {
  name             = "new-service"
  datacenter_id    = "aws-us-east-1a"
  service_class_id = "ENTERPRISE_10K_STANDALONE"  # Upgraded service class
  message_vpn_name = "my-vpn"                     # Same VPN name for easier migration
}
```

## Service Upgrading (Event Broker Version)

The Solace Cloud Terraform Provider does not directly support upgrading the event broker version of an existing service. The `event_broker_version` attribute is immutable after service creation.

To upgrade the event broker version:

1. Create a new service with the desired version.
2. Migrate your data and configuration to the new service.
3. Delete the old service once migration is complete.

Example of specifying an event broker version:

```hcl
resource "solacecloud_service" "broker_service" {
  name                = "my-service"
  datacenter_id       = "aws-us-east-1a"
  service_class_id    = "ENTERPRISE_1K_STANDALONE"
  event_broker_version = "10.0.1.7-3"  # Specify the desired version
}
```

## Terraform to V2 API Parameter Mapping

The Solace Cloud Terraform Provider uses the Solace Cloud V2 API to manage services. The following table maps Terraform attributes to their corresponding V2 API parameters:

| Terraform Attribute | V2 API Parameter | Description |
|---------------------|------------------|-------------|
| `name` | `name` | The name of the service |
| `datacenter_id` | `datacenterId` | The ID of the datacenter |
| `service_class_id` | `serviceClassId` | The service class ID |
| `message_vpn_name` | `msgVpnName` | The name of the Message VPN |
| `max_spool_usage` | `maxSpoolUsage` | The maximum message spool usage in GB |
| `event_broker_version` | `eventBrokerVersion` | The version of the event broker |
| `cluster_name` | `clusterName` | The name of the DMR cluster |
| `custom_router_name` | `customRouterName` | The custom router name prefix |
| `environment_id` | `environmentId` | The ID of the environment |
| `locked` | `locked` | Whether the service is locked against deletion |
| `mate_link_encryption` | `redundancyGroupSslEnabled` | Whether mate link encryption is enabled |
| `owned_by` | `ownedBy` | The ID of the user who owns the service |

### API Response to Terraform Attribute Mapping

When reading a service from the API, the provider maps the API response fields to Terraform attributes:

| V2 API Response Field | Terraform Attribute | Description |
|------------------------|---------------------|-------------|
| `id` | `id` | The unique identifier for the service |
| `name` | `name` | The name of the service |
| `datacenterId` | `datacenter_id` | The ID of the datacenter |
| `serviceClassId` | `service_class_id` | The service class ID |
| `eventBrokerServiceVersion` | `event_broker_version` | The version of the event broker |
| `broker.msgVpns[0].msgVpnName` | `message_vpn_name` | The name of the Message VPN |
| `broker.maxSpoolUsage` | `max_spool_usage` | The maximum message spool usage in GB |
| `broker.cluster.name` | `cluster_name` | The name of the DMR cluster |
| `locked` | `locked` | Whether the service is locked against deletion |
| `environmentId` | `environment_id` | The ID of the environment |
| `broker.redundancyGroupSslEnabled` | `mate_link_encryption` | Whether mate link encryption is enabled |
| `ownedBy` | `owned_by` | The ID of the user who owns the service |

## V2 API Features Not Implemented in Terraform

Some features of the Solace Cloud V2 API are not currently implemented in the Terraform provider. These include:

1. **Service Upgrades**: The API supports upgrading the event broker version, but this is not implemented in the provider.

2. **Service Downgrades**: The API supports downgrading certain service attributes, but this is not implemented in the provider.

3. **Service Maintenance Windows**: The API supports configuring maintenance windows for services, but this is not exposed in the provider.

4. **Service Metrics**: The API provides access to service metrics, but this is not available in the provider.

5. **User Management**: The API supports managing users and their permissions, but this is not implemented in the provider.

6. **Service Plugins**: The API supports managing service plugins, but this is not exposed in the provider.

7. **Service Backup and Restore**: While the API supports backup and restore operations, the provider does not directly implement these. Instead, you can use the Solace Broker Provider to implement backup and restore functionality as described in the [Importing Services](./importing_services.md) guide.

## Best Practices for Service Lifecycle Management

1. **Plan for Immutability**: Design your infrastructure with the understanding that certain service attributes cannot be changed after creation.

2. **Use Terraform Modules**: Create reusable modules for service creation to ensure consistency across your infrastructure.

3. **Version Control**: Keep your Terraform configurations in version control to track changes.

4. **Test Changes in Non-Production**: Always test service modifications in a non-production environment first.

5. **Document Service Attributes**: Maintain documentation of your service attributes, especially those that cannot be changed.

6. **Implement Backup and Restore**: Use the Solace Broker Provider to implement backup and restore functionality for your services.

7. **Monitor Service Status**: Monitor the status of your services during and after modifications to ensure they are functioning correctly.

## Conclusion

Understanding the lifecycle management of your event broker services is essential for effectively using the Terraform provider. By knowing which attributes can be modified and which are immutable, you can plan your infrastructure accordingly and implement strategies for service upgrades and migrations.
