# Terraform Provider for Solace Cloud

[![Staging Acceptance Test](https://github.com/SolaceDev/terraform-provider-solacecloud/actions/workflows/staging-acceptance.yml/badge.svg)](https://github.com/SolaceDev/terraform-provider-solacecloud/actions/workflows/staging-acceptance.yml)

This repository contains the official Terraform provider for Solace Cloud. This provider allows you to manage Solace Cloud resources using HashiCorp Terraform, enabling infrastructure as code practices for your Solace event streaming and management services.

## Repository Structure

This repository is organized as follows:

- `internal/`: Contains the core logic for the Terraform provider.
- `examples/`: Includes example Terraform configurations demonstrating how to use the provider.
- `scripts/`: Contains build and test automation scripts.
- `docs/`: Provides documentation for the provider's resources and data sources.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22

## Building The Provider

To build the provider from source:

1. Clone the repository.
2. Navigate to the repository directory.
3. Build the provider using the Go `install` command:

```shell
go install
```
This command compiles the provider and installs the binary to your Go environment.

## Updating the Solace Cloud API Module

The provider utilizes an OpenAPI-generated library for interacting with the Solace Cloud API. To incorporate new API features or updates:

1. Refer to the instructions in [`OpenAPI-howto.md`](./OpenAPI-howto.md) to regenerate the API client.
2. After updating, ensure to test the provider thoroughly.

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

Fill this in for each provider

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

### Useful Commands:

Run acceptance tests against staging
```shell
bash scripts/staging-acceptance.sh
````
*Note:* Acceptance tests create real resources, and often cost money to run.

Run local tests (Including acceptance tests against mocks)
```shell
make test
```

###  (Optional) Terraform dev install config

This is optional because you can execute the provider from the acceptance tests exactly like you would in a real Terraform environment.

To install the provider locally, you can copy the following terraform configuration to your `~/.terraformrc` file:

```hcl
provider_installation {

  dev_overrides {
      "hashicorp.com/edu/solacecloud" = "path-to-terraform-provider/terraform-provider-solacecloud"
      "registry.terraform.io/solaceproducts/solacebroker" = "${GOBIN_OR_PATH_TO_BROKER_PROVIDER}"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```


