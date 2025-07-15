package shared

import (
	"terraform-provider-solacecloud/missioncontrol"
	"terraform-provider-solacecloud/platform"
)

// ProviderConfig maps provider schema data to a Go type.
// This type is shared between the provider and data sources/resources.
type ProviderConfig struct {
	APIClient          *missioncontrol.ClientWithResponses
	APIPollingInterval int
	PlatformClient     *platform.ClientWithResponses
}
