package environment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"terraform-provider-solacecloud/internal/shared"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-solacecloud/missioncontrol"
	"terraform-provider-solacecloud/platform"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &EnvironmentDataSource{}
	_ datasource.DataSourceWithConfigure = &EnvironmentDataSource{}
)

// NewEnvironmentDataSource is a helper function to simplify the provider implementation.
func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

// EnvironmentDataSource is the data source implementation.
type EnvironmentDataSource struct {
	APIClient      *missioncontrol.ClientWithResponses
	PlatformClient *platform.ClientWithResponses
}

// EnvironmentDataSourceModel maps the data source schema data.
type EnvironmentDataSourceModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

// Configure adds the provider configured client to the data source.
func (d *EnvironmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	// Use the shared provider config type
	providerConfig, ok := req.ProviderData.(shared.ProviderConfig)
	if !ok {
		// Log the actual type for debugging
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected shared.ProviderConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.APIClient = providerConfig.APIClient
	d.PlatformClient = providerConfig.PlatformClient
}

// Metadata returns the data source type name.
func (d *EnvironmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

// Schema defines the schema for the data source.
func (d *EnvironmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a Solace Cloud environment by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this environment.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the environment to fetch.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of object for informational purposes.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	diags := d.readDataInternal(ctx, req, resp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *EnvironmentDataSource) readDataInternal(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) diag.Diagnostics {
	var diags diag.Diagnostics
	var state EnvironmentDataSourceModel

	diags.Append(req.Config.Get(ctx, &state)...)
	if diags.HasError() {
		return diags
	}

	// Get the requested environment name
	requestedName := state.Name.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Looking for environment with name: %s", requestedName))

	// First, try to find the environment by name using the platform API
	params := &platform.SearchEnvironmentsParams{
		Name: &requestedName,
	}

	searchResp, err := d.PlatformClient.SearchEnvironmentsWithResponse(ctx, params)
	if err != nil {
		diags.AddError(
			"Error Searching Environments",
			fmt.Sprintf("Could not search environments: %s", err),
		)
		return diags
	}

	errorHandler := shared.NewPlatformErrorResponseAdaptor(
		http.StatusOK,
		searchResp.Body,
		searchResp.HTTPResponse,
		searchResp.JSON400, // JSON400
		searchResp.JSON401, // JSON401
		searchResp.JSON403, // JSON403
		searchResp.JSON404, // JSON404
		nil,                // JSON503
	)

	if errorHandler.HandleError(&diags) {
		return diags
	}

	// Log the API response
	tflog.Debug(ctx, fmt.Sprintf("Environments Search API Response: %s", string(searchResp.Body)))

	// Parse the response body
	var environmentsResponse struct {
		Data *[]struct {
			Id   *string `json:"id,omitempty"`
			Name string  `json:"name"`
		} `json:"data,omitempty"`
	}

	err = json.Unmarshal(searchResp.Body, &environmentsResponse)
	if err != nil {
		diags.AddError(
			"Error Parsing Environments Response",
			fmt.Sprintf("Could not parse environments response: %s", err),
		)
		return diags
	}

	// Find the environment with the requested name
	var environmentID string
	var found bool

	if environmentsResponse.Data != nil && len(*environmentsResponse.Data) > 0 {
		for _, env := range *environmentsResponse.Data {
			if env.Name == requestedName && env.Id != nil {
				environmentID = *env.Id
				found = true
				tflog.Debug(ctx, fmt.Sprintf("Found environment with name '%s', ID: %s", requestedName, environmentID))
				break
			}
		}
	}

	// If not found by name, return an error
	if !found {
		diags.AddError(
			"Environment Not Found",
			fmt.Sprintf("Could not find environment with name '%s'", requestedName),
		)
		return diags
	}

	// Get environment details from API using the platform client
	apiResp, err := d.PlatformClient.GetEnvironmentByIdWithResponse(ctx, environmentID)
	if err != nil {
		diags.AddError(
			"Error Reading Environment",
			fmt.Sprintf("Could not read environment %s: %s", environmentID, err),
		)
		return diags
	}

	errorHandler = shared.NewPlatformErrorResponseAdaptor(
		http.StatusOK,
		apiResp.Body,
		apiResp.HTTPResponse,
		apiResp.JSON400, // JSON400
		apiResp.JSON401, // JSON401
		apiResp.JSON403, // JSON403
		apiResp.JSON404, // JSON404
		nil,             // JSON503
	)

	if errorHandler.HandleError(&diags) {
		return diags
	}

	// Log the API response
	tflog.Debug(ctx, fmt.Sprintf("Environment API Response: %s", string(apiResp.Body)))

	// Parse response body into EnvironmentResponseEnvelope
	var environmentResponse platform.EnvironmentResponseEnvelope
	err = json.Unmarshal(apiResp.Body, &environmentResponse)
	if err != nil {
		diags.AddError(
			"Error Parsing Environment Response",
			fmt.Sprintf("Could not parse environment response: %s", err),
		)
		return diags
	}

	// Map response data to model
	if environmentResponse.Data != nil {
		state.Id = types.StringPointerValue(environmentResponse.Data.Id)

		// Handle the Type field
		if environmentResponse.Data.Type != nil {
			state.Type = types.StringValue(*environmentResponse.Data.Type)
		} else {
			state.Type = types.StringValue("environment")
		}
	} else {
		diags.AddError(
			"Empty Environment Response",
			"The environment response data is empty",
		)
		return diags
	}

	// Set state
	diags.Append(resp.State.Set(ctx, &state)...)
	return diags
}
