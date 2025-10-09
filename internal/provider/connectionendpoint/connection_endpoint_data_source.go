package connectionendpoint

import (
	"context"
	"fmt"
	"net/http"

	"terraform-provider-solacecloud/internal/model"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/missioncontrol"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ConnectionEndpointDataSource{}
	_ datasource.DataSourceWithConfigure = &ConnectionEndpointDataSource{}
)

// NewConnectionEndpointDataSource is a helper function to simplify the provider implementation.
func NewConnectionEndpointDataSource() datasource.DataSource {
	return &ConnectionEndpointDataSource{}
}

// ConnectionEndpointDataSource is the data source implementation.
type ConnectionEndpointDataSource struct {
	APIClient       *missioncontrol.ClientWithResponses
	endpointManager *EndpointManager
}

// ConnectionEndpointDataSourceModel maps the data source schema data.
type ConnectionEndpointDataSourceModel struct {
	ServiceId string                    `tfsdk:"service_id"`
	Id        types.String              `tfsdk:"id"`
	Name      types.String              `tfsdk:"name"`
	Endpoints []ConnectionEndpointModel `tfsdk:"endpoints"`
}

// ConnectionEndpointModel represents a single connection endpoint
type ConnectionEndpointModel struct {
	Id             types.String `tfsdk:"id"`
	Type           types.String `tfsdk:"type"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	AccessType     types.String `tfsdk:"access_type"`
	K8sServiceType types.String `tfsdk:"k8s_service_type"`
	K8sServiceId   types.String `tfsdk:"k8s_service_id"`
	Ports          types.Object `tfsdk:"ports"`
}

// Configure adds the provider configured client to the data source.
func (d *ConnectionEndpointDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig, ok := req.ProviderData.(shared.ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected shared.ProviderConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.APIClient = providerConfig.APIClient
	d.endpointManager = NewEndpointManager(apiclient.NewRetryableClient(providerConfig.APIClient, 3, 10), providerConfig.APIPollingInterval)
}

// Metadata returns the data source type name.
func (d *ConnectionEndpointDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection_endpoints"
}

// Schema defines the schema for the data source.
func (d *ConnectionEndpointDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches connection endpoints for a service. Can list all endpoints, filter by ID, or filter by name.",
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Description: "The ID of the service to fetch connection endpoints for.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "Optional: The ID of a specific connection endpoint to fetch.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "Optional: The name of a specific connection endpoint to fetch.",
				Optional:    true,
			},
			"endpoints": schema.ListNestedAttribute{
				Description: "List of connection endpoints matching the criteria.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier for this connection endpoint.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of object.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the connection endpoint.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the connection endpoint.",
							Computed:    true,
						},
						"access_type": schema.StringAttribute{
							Description: "The access type (PUBLIC or PRIVATE).",
							Computed:    true,
						},
						"k8s_service_type": schema.StringAttribute{
							Description: "The Kubernetes service type.",
							Computed:    true,
						},
						"k8s_service_id": schema.StringAttribute{
							Description: "The Kubernetes service ID.",
							Computed:    true,
						},
						"ports": model.EndpointProtocolSchema(),
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *ConnectionEndpointDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	diags := d.readDataInternal(ctx, req, resp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *ConnectionEndpointDataSource) readDataInternal(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) diag.Diagnostics {
	var diags diag.Diagnostics
	var state ConnectionEndpointDataSourceModel

	diags.Append(req.Config.Get(ctx, &state)...)
	if diags.HasError() {
		return diags
	}

	serviceId := state.ServiceId
	endpointId := state.Id.ValueString()
	endpointName := state.Name.ValueString()

	tflog.Debug(ctx, fmt.Sprintf("Looking for connection endpoints for service: %s", serviceId))

	// Case 1: Get by ID
	if endpointId != "" {
		endpoint, diagnostics, done := d.findEndpointById(ctx, endpointId, serviceId, diags)
		if done {
			return diagnostics
		}
		state.Endpoints = []ConnectionEndpointModel{*endpoint}
	} else {
		// Case 2 & 3: List all or filter by name
		endpointsFromAPI := d.endpointManager.ListEndpoints(ctx, serviceId, &diags)
		if diags.HasError() {
			return diags
		}

		// Convert to model
		var endpoints []ConnectionEndpointModel
		endpoints = d.convertEndpointsToModel(ctx, endpointsFromAPI, diags, endpoints)

		diagnostics, done := d.filterByNameIfProvided(ctx, endpointName, endpoints, diags, serviceId, &state)
		if done {
			return diagnostics
		}
	}

	// Set state
	diags.Append(resp.State.Set(ctx, &state)...)
	return diags
}

func (d *ConnectionEndpointDataSource) convertEndpointsToModel(ctx context.Context, endpointsFromAPI []missioncontrol.GetConnectionEndpoint, diags diag.Diagnostics, endpoints []ConnectionEndpointModel) []ConnectionEndpointModel {
	for _, ep := range endpointsFromAPI {
		endpoint := mapEndpointToModel(ctx, &ep, &diags)
		if endpoint != nil {
			endpoints = append(endpoints, *endpoint)
		}
	}
	return endpoints
}

func (d *ConnectionEndpointDataSource) filterByNameIfProvided(ctx context.Context, endpointName string, endpoints []ConnectionEndpointModel, diags diag.Diagnostics, serviceId string, state *ConnectionEndpointDataSourceModel) (diag.Diagnostics, bool) {
	if endpointName != "" {
		tflog.Debug(ctx, fmt.Sprintf("Filtering endpoints by name: %s", endpointName))
		var filtered []ConnectionEndpointModel
		for _, ep := range endpoints {
			if ep.Name.ValueString() == endpointName {
				filtered = append(filtered, ep)
			}
		}
		if len(filtered) == 0 {
			diags.AddError(
				"Connection Endpoint Not Found",
				fmt.Sprintf("Could not find connection endpoint with name '%s' for service '%s'", endpointName, serviceId),
			)
			return diags, true
		}
		state.Endpoints = filtered
	} else {
		state.Endpoints = endpoints
	}
	return nil, false
}

func (d *ConnectionEndpointDataSource) findEndpointById(ctx context.Context, endpointId string, serviceId string, diags diag.Diagnostics) (*ConnectionEndpointModel, diag.Diagnostics, bool) {
	tflog.Debug(ctx, fmt.Sprintf("Fetching connection endpoint by ID: %s", endpointId))
	endpoint, diagsFromGet := d.getEndpointById(ctx, serviceId, endpointId)
	diags.Append(diagsFromGet...)
	if diags.HasError() {
		return nil, diags, true
	}
	return endpoint, nil, false
}

// getEndpointById fetches a single connection endpoint by ID
func (d *ConnectionEndpointDataSource) getEndpointById(ctx context.Context, serviceId, endpointId string) (*ConnectionEndpointModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	apiResp, err := d.APIClient.GetConnectionEndpointWithResponse(ctx, serviceId, endpointId)
	if err != nil {
		diags.AddError(
			"Error Reading Connection Endpoint",
			fmt.Sprintf("Could not read connection endpoint %s: %s", endpointId, err),
		)
		return nil, diags
	}

	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		http.StatusOK,
		apiResp.Body,
		apiResp.HTTPResponse,
		apiResp.JSON400,
		apiResp.JSON401,
		apiResp.JSON403,
		apiResp.JSON404,
		apiResp.JSON503,
	)

	if errorHandler.HandleError(&diags) {
		return nil, diags
	}

	tflog.Debug(ctx, fmt.Sprintf("Connection Endpoint API Response: %s", string(apiResp.Body)))

	// Use the structured response from the API client
	if apiResp.JSON200 != nil {
		endpoint := mapEndpointToModel(ctx, &apiResp.JSON200.Data, &diags)
		return endpoint, diags
	}

	diags.AddError(
		"Empty Connection Endpoint Response",
		"The connection endpoint response data is empty",
	)
	return nil, diags
}

// mapEndpointToModel maps API response to the data source model
func mapEndpointToModel(ctx context.Context, ep *missioncontrol.GetConnectionEndpoint, diags *diag.Diagnostics) *ConnectionEndpointModel {
	if ep == nil {
		return nil
	}

	endpointModel := &ConnectionEndpointModel{
		Id:          types.StringPointerValue(ep.Id),
		Type:        types.StringPointerValue(ep.Type),
		Name:        types.StringValue(ep.Name),
		Description: types.StringPointerValue(ep.Description),
		AccessType:  types.StringValue(string(ep.AccessType)),
	}

	if ep.K8sServiceType != nil {
		endpointModel.K8sServiceType = types.StringValue(string(*ep.K8sServiceType))
	}

	if ep.K8sServiceId != nil {
		endpointModel.K8sServiceId = types.StringPointerValue(ep.K8sServiceId)
	}

	// Map ports using the shared protocol mapping logic
	if len(ep.Ports) > 0 {
		portsObj, portsDiags := model.ToObjectValue(ep.Ports)
		diags.Append(portsDiags...)
		endpointModel.Ports = portsObj
	} else {
		endpointModel.Ports = types.ObjectNull(model.EndpointProtocolsTypes())
	}

	return endpointModel
}
