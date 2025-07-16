package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"terraform-provider-solacecloud/internal/provider/environment"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/missioncontrol"
	"terraform-provider-solacecloud/platform"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &solaceCloudProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &solaceCloudProvider{
			version: version,
		}
	}
}

// solaceCloudProvider is the provider implementation.
type solaceCloudProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// solaceCloudProviderModel maps provider schema data to a Go type.
type solaceCloudProviderModel struct {
	BaseURL            types.String `tfsdk:"base_url"`
	APIToken           types.String `tfsdk:"api_token"`
	APIPollingInterval types.Int64  `tfsdk:"api_polling_interval"`
}

// For backward compatibility, keep the SolaceCloudProviderConfig type
// but use the shared.ProviderConfig type internally
type SolaceCloudProviderConfig = shared.ProviderConfig

// Metadata returns the provider type name.
func (p *solaceCloudProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "solacecloud"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *solaceCloudProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Required:    false,
				Optional:    true,
				Sensitive:   false,
				Description: "Base URL for REST API Endpoints. The PubSub+ Home Cloud your account is located determines the base URL you use.  ex: https://api.solace.cloud/",
			},
			"api_token": schema.StringAttribute{
				Required:    false,
				Optional:    true,
				Sensitive:   true,
				Description: "Token for authenticating with the Solace Cloud API. Can be set as Env Variable SOLACE_APITOKEN",
			},
			"api_polling_interval": schema.Int64Attribute{
				Required:    false,
				Optional:    true,
				Sensitive:   false,
				Description: "Polling Interval in seconds for API calls that need to wait untill a process changes status. For example wait until a SC service is marked as COMPLETED. Default value is 30 seconds",
			},
		},
	}
}

func (p *solaceCloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config solaceCloudProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if resp.Diagnostics.HasError() {
		return
	}

	baseUrl := config.BaseURL.ValueString()

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.
	if baseUrl == "" {
		baseUrl = "https://production-api.solace.cloud"
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	apiToken := os.Getenv("SOLACECLOUD_API_TOKEN")

	if config.APIToken.ValueString() != "" {
		apiToken = config.APIToken.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.
	if apiToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing Solace Cloud API Token",
			"The provider cannot create the API client as there is an unknown configuration value for the Solace API Token. "+
				"Set the api_token value in the configuration or use the SOLACECLOUD_API_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	apiPollingInterval := 30

	if config.APIPollingInterval.ValueInt64() != 0 {
		apiPollingInterval = int(config.APIPollingInterval.ValueInt64())
	} else {
		tflog.Debug(ctx, fmt.Sprintf("No api_polling_interval value was configured on the provider, using default value of = %d ", apiPollingInterval))
	}

	tflog.Debug(ctx, fmt.Sprintf("api_polling_interval = %d ", apiPollingInterval))

	if resp.Diagnostics.HasError() {
		return
	}

	///////////////////////////////////////////////
	//Create Solace Cloud API Client using the generated OpenAPIv3 code
	///////////////////////////////////////////////

	// Use the NewSecurityProviderBearerToken to pass the SC APIToken
	tokenAuth, err := securityprovider.NewSecurityProviderBearerToken(apiToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set the API_Token on the securityprovider.NewSecurityProviderBearerToken ",
			"An unexpected error occurred when creating the SolaceCloud API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Solace Cloud API Client Error: "+err.Error(),
		)
	}

	//Create the Solace Cloud API Client we'll be using to make all the requests on this TF Provider
	//Use tokenAuth.Intercept to set the Bearer Token before sending Requests
	apiClient, err := missioncontrol.NewClientWithResponses(
		baseUrl,
		missioncontrol.WithRequestEditorFn(tokenAuth.Intercept),
		// Add a Request Editor to set the Content-Type to JSON for all requests
		func(c *missioncontrol.Client) error {
			c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
				req.Header.Set("Accept", "application/json")
				req.Header.Set("x-issuer", "terraform-provider-solacecloud")
				return nil
			})
			return nil
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set the API_Token on the securityprovider.NewSecurityProviderBearerToken ",
			"An unexpected error occurred when creating the SolaceCloud API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Solace Cloud API Client Error: "+err.Error(),
		)
	}

	// Request JSON content-type for all requests.  This is required to get error payload to be encoded as JSON (Instead of XML).

	// Create the Platform API client
	platformClient, err := platform.NewClientWithResponses(
		baseUrl,
		platform.WithRequestEditorFn(tokenAuth.Intercept),
		// Add a Request Editor to set the Content-Type to JSON for all requests
		func(c *platform.Client) error {
			c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
				req.Header.Set("Accept", "application/json")
				req.Header.Set("x-issuer", "terraform-provider-solacecloud")
				return nil
			})
			return nil
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Platform API client",
			"An unexpected error occurred when creating the Platform API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Platform API Client Error: "+err.Error(),
		)
	}

	//	Make the Solace Cloud API client & other config params available during DataSource and Resource as a shared.ProviderConfig
	providerConfig := shared.ProviderConfig{
		APIClient:          apiClient,
		APIPollingInterval: apiPollingInterval,
		PlatformClient:     platformClient,
	}

	// Make the TOKEN client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = providerConfig
	resp.ResourceData = providerConfig

}

// DataSources defines the data sources implemented in the provider.
func (p *solaceCloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		environment.NewEnvironmentDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *solaceCloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewServiceResource,
	}

	// SCService....
	// Nameconst
	// Regionconst
	// Typeconst

}
