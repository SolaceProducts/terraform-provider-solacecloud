package connectionendpoint

import (
	"context"
	"fmt"
	"terraform-provider-solacecloud/internal/model"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/internal/util"
	"terraform-provider-solacecloud/missioncontrol"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ConnectionEndpointResource{}
var _ resource.ResourceWithImportState = &ConnectionEndpointResource{}

func NewConnectionEndpointResource() resource.Resource {
	return &ConnectionEndpointResource{}
}

// ConnectionEndpointResource defines the resource implementation.
type ConnectionEndpointResource struct {
	APIClient          *apiclient.RetryableClientWithResponses
	APIPollingInterval int
	endpointManager    *EndpointManager
}

func (r *ConnectionEndpointResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection_endpoint"
}

func (r *ConnectionEndpointResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Connection Endpoint resource",

		Attributes: ConnectionEndpointSchema(),
	}
}

// init struct
func (r *ConnectionEndpointResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig := req.ProviderData.(shared.ProviderConfig)
	// has more retries to handle collisions
	r.APIClient = apiclient.NewRetryableClient(providerConfig.APIClient, 10, 10)
	r.APIPollingInterval = providerConfig.APIPollingInterval
	r.endpointManager = NewEndpointManager(r.APIClient, r.APIPollingInterval)
}

func (r *ConnectionEndpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// state and plan because state is not created yet.
	var data ConnectionEndpointResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := data.ServiceId.ValueString()

	// Prepare endpoint configuration
	config := EndpointConfig{
		Name:       data.Name.ValueString(),
		AccessType: missioncontrol.ConnectionEndpointAccessType(data.AccessType.ValueString()),
	}

	if util.IsKnown(data.Description) {
		description := data.Description.ValueString()
		config.Description = &description
	}

	// Convert ports ObjectValue to the expected API format
	if util.IsKnown(data.Ports) {
		ports, diags := r.endpointManager.ConvertObjectValueToPorts(ctx, data.Ports)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		config.Ports = ports
	}

	// Create the endpoint
	endpointId := r.endpointManager.CreateEndpoint(ctx, serviceId, config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Id = types.StringValue(endpointId)
	// service id is already set from the plan

	// Read the created endpoint to get complete state
	r.readDataInternal(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectionEndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectionEndpointResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readDataInternal(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		// Check if the error is because the connection endpoint doesn't exist
		for _, diagnostic := range resp.Diagnostics {
			if diagnostic.Summary() == "Not Found" {
				// Connection endpoint no longer exists, remove it from state
				resp.State.RemoveResource(ctx)
				return
			}
		}
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectionEndpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ConnectionEndpointResourceModel
	var state ConnectionEndpointResourceModel

	// Read Terraform plan plan into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := state.ServiceId.ValueString()
	connectionEndpointId := state.Id.ValueString()

	// Prepare endpoint configuration
	config := EndpointConfig{
		Name: plan.Name.ValueString(),
	}

	if util.IsKnown(plan.Description) {
		description := plan.Description.ValueString()
		config.Description = &description
	}

	// Convert ports ObjectValue to the expected API format
	if util.IsKnown(state.Ports) {
		ports, diags := r.endpointManager.ConvertObjectValueToPorts(ctx, plan.Ports)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		config.Ports = ports
	}

	// Update the endpoint
	r.endpointManager.UpdateEndpoint(ctx, serviceId, connectionEndpointId, config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the updated endpoint to get complete state
	r.readDataInternal(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ConnectionEndpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectionEndpointResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := data.ServiceId.ValueString()
	connectionEndpointId := data.Id.ValueString()

	// Delete the endpoint using EndpointManager
	r.endpointManager.DeleteEndpoint(ctx, serviceId, connectionEndpointId, &resp.Diagnostics)
}

func (r *ConnectionEndpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID should be in the format "{serviceId}/connectionEndpoints/{connectionEndpointId}"
	// For example: rlruo7yicl8/connectionEndpoints/wshhdgbo7z0
	combinedId := req.ID
	serviceId, connectionEndpointId := r.endpointManager.SplitCombinedId(combinedId, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), serviceId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), connectionEndpointId)...)
}

func (r *ConnectionEndpointResource) readDataInternal(ctx context.Context, state *ConnectionEndpointResourceModel, diagnostics *diag.Diagnostics) {
	serviceId := state.ServiceId.ValueString()
	connectionEndpointId := state.Id.ValueString()

	// Read the endpoint using EndpointManager
	endpointData := r.endpointManager.ReadEndpoint(ctx, serviceId, connectionEndpointId, diagnostics)
	if diagnostics.HasError() || endpointData == nil {
		return
	}

	// Update the model with the response state
	state.Id = types.StringValue(*endpointData.Id)
	state.Name = types.StringValue(endpointData.Name)
	state.Description = types.StringPointerValue(endpointData.Description)
	state.AccessType = types.StringValue(string(endpointData.AccessType))
	state.K8SServiceId = types.StringPointerValue(endpointData.K8sServiceId)

	// Convert K8sServiceType to string
	if endpointData.K8sServiceType != nil {
		state.K8SServiceType = types.StringValue(string(*endpointData.K8sServiceType))
	} else {
		state.K8SServiceType = types.StringNull()
	}

	// Convert ports to ObjectValue using the endpoint protocols model
	if len(endpointData.Ports) > 0 {
		portsObjectValue, diags := model.ToObjectValue(endpointData.Ports)
		diagnostics.Append(diags...)
		if !diagnostics.HasError() {
			state.Ports = portsObjectValue
		}
	}
	tflog.Info(ctx, fmt.Sprintf("Connection Endpoint ID: %s - Name: %s - Access Type: %s - Ports: %+v",
		connectionEndpointId, state.Name.ValueString(), state.AccessType.ValueString(), endpointData.Ports))
}
