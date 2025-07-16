# Environment Data Source Example

This example demonstrates how to use the `solacecloud_environment` data source to fetch environment information and use it when creating a Solace Cloud service.

## Usage

1. Set your Solace Cloud API token as an environment variable:

```bash
export SOLACECLOUD_API_TOKEN=your-api-token
```

2. Initialize the Terraform configuration:

```bash
terraform init
```

3. Apply the Terraform configuration:

```bash
terraform apply
```

## What This Example Does

This example:

1. Fetches the "Default" environment using the `solacecloud_environment` data source
2. Creates a Solace Cloud service in the Default environment
3. Outputs the environment ID, environment type, and service ID

## Notes

- Currently, only the "Default" environment is supported by the data source
- The service is created in the "eks-eu-central-1a" datacenter with the "ENTERPRISE_1K_STANDALONE" service class
