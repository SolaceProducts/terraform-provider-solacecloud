# Troubleshooting

This guide provides solutions to common issues you might encounter when using the Solace Cloud Terraform Provider.

## Authentication Issues

### Error: Authentication Failed

**Error Message:**

```text
Error: Authentication Failed During Service Status Check
Received HTTP 401 Unauthorized while checking service creation status.
This may indicate that your API token has expired or been revoked during the service creation process.
Verify your authentication configuration and try again.
```

**Solution:**

1. Verify that your API token is correct and has not expired.
2. Ensure the API token has the necessary permissions to create and manage services.
3. Check that the `base_url` is correct for your Solace Cloud provider.
4. Try regenerating a new API token in the Solace Cloud Console. For more information, see [Managing API Tokens](https://docs.solace.com/Cloud/ght_api_tokens.htm).

## Service Creation Issues

### Error: Service Creation Failed

**Error Message:**

```text
Error: Resource Creation FAILED
Received creationState as: FAILED from the GetService API Request
```

**Solution:**

1. Check the service class and datacenter compatibility.
2. Verify that your account has sufficient permissions to create the requested service class. For more information, see [Managing Users, Groups, Roles, and Permissions](https://docs.solace.com/Cloud/cloud-user-management.htm).
3. Ensure that the datacenter ID is valid and available in your region.
4. Check if you've reached your service quota limit. For more information, see [Requesting Additional Event Broker Services](https://docs.solace.com/Cloud/ght_capacity_increase.htm).

### Error: Service Operation Timeout

**Error Message:**

```text
Error: Service operation timeout
Message spool update operation timed out after 5 minutes
```

**Solution:**

1. Increase the `api_polling_interval` in your provider configuration.
2. For large spool size changes, the operation may take longer than expected.
3. Check the Cloud Console to verify if the operation is still in progress. For more information, see [Configuring Message Spool Sizes](https://docs.solace.com/Cloud/Configure-Message-Spools.htm).
4. Try breaking down large changes into smaller increments.

## Resource Management Issues

### Error: Cannot Delete Service

**Error Message:**

```text
Error: Cannot delete service
The service is locked and cannot be deleted
```

**Solution:**

1. Check if the `locked` attribute is set to `true` for the service. For more information, see [Using Deletion Protection](https://docs.solace.com/Cloud/ght_service_deletion_protection.htm).
2. Update the service to set `locked = false` before attempting to delete.
3. If the service is managed by another system or organization policy, contact your administrator.

### Error: Cannot Update Immutable Attributes

**Error Message:**

```text
Error: Cannot update immutable attribute
The following attributes cannot be updated: service_class_id, datacenter_id, message_vpn_name
```

**Solution:**

1. Some attributes are immutable and cannot be changed after creation.
2. To change these attributes, you must create a new service and migrate your data.
3. Review the documentation to understand which attributes are immutable.

## Environment Issues

### Error: Environment Not Found

**Error Message:**

```text
Error: Environment Not Found
Could not find environment with name 'MyEnvironment'
```

**Solution:**

1. Verify that the environment name exists and is spelled correctly. For more information, see [Creating and Managing Environments](https://docs.solace.com/Cloud/environments.htm).
2. Check that your API token has access to the specified environment. For more information, see [Managing API Tokens](https://docs.solace.com/Cloud/ght_api_tokens.htm).
3. Use the Cloud Console to list available environments.
4. If the environment was recently created, it may take some time to propagate.

## Connection Issues

### Error: Failed to Connect to Solace Cloud API

**Error Message:**

```text
Error: Failed to connect to Solace Cloud API
Could not create the API client as there is an unknown configuration value for the Solace Cloud Base URL
```

**Solution:**

1. Check your network connectivity to the Solace Cloud API.
2. Verify that the `base_url` is correct and includes the protocol (https://).
3. Ensure there are no firewall or proxy issues blocking the connection.
4. Try using a different network connection.

## Import Issues

### Error: Resource Import Failed

**Error Message:**

```text
Error: Resource Import Failed
Could not find service with ID 'service-id'
```

**Solution:**

1. Verify that the service ID is correct.
2. Ensure your API token has access to the service you're trying to import. For more information, see [Managing API Tokens](https://docs.solace.com/Cloud/ght_api_tokens.htm).
3. Check if the service still exists in the Cloud Console. For more information, see [Viewing Event Broker Services](https://docs.solace.com/Cloud/cloud-configure-messaging-services.htm).

## General Troubleshooting Tips

1. **Enable Terraform Logging**: Set the `TF_LOG` environment variable to `DEBUG` or `TRACE` to get more detailed logs:

   ```bash
   export TF_LOG=DEBUG
   ```

2. **Check API Responses**: Review the Terraform logs for API response details that might provide more information about the error.

3. **Verify Resource State**: Use `terraform state show` to examine the current state of your resources:

   ```bash
   terraform state show solacecloud_service.broker_service
   ```

4. **Refresh State**: If Terraform's state is out of sync with the actual resources, try refreshing:

   ```bash
   terraform refresh
   ```

5. **Check Solace Cloud Console**: Compare the resources in Terraform with what's visible in the Cloud Console to identify discrepancies.

6. **API Token Permissions**: Ensure your API token has all the necessary permissions for the operations you're trying to perform. For more information, see [Managing API Tokens](https://docs.solace.com/Cloud/ght_api_tokens.htm).

7. **Version Compatibility**: Make sure you're using a compatible version of the Terraform provider with your Solace Cloud environment.

## Getting Help

If you continue to experience issues after trying these troubleshooting steps, consider:

1. Opening an issue in the [Solace Cloud Terraform Provider GitHub repository](https://github.com/SolaceProducts/terraform-provider-solacecloud).
2. [Contact Solace](https://docs.solace.com/get-support.htm).
3. Posting a question on the [Solace Community Forum](https://solace.community/).
