package provider_test

import (
	"terraform-provider-solacecloud/internal/provider"
	
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

/*
const (
	// providerConfig is a shared configuration to combine with the actual
	// test configuration so the HashiCups client is properly configured.
	// It is also possible to use the HASHICUPS_ environment variables instead,
	// such as updating the Makefile and running the testing through that tool.
	// providerConfig = `
// provider "solacecloud" {
// 	base_url             = "https://05f9317c-aa20-4ae5-9274-6ba5e8b67266.mock.pstmn.io/"
// 	api_polling_interval = 40
// }
// `
)
*/

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"solacecloud": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
)
