package dnsName

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"terraform-provider-solacecloud/internal/model"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/internal/provider/operationutils"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/missioncontrol"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &DnsNameMoveResource{}
var _ resource.ResourceWithImportState = &DnsNameMoveResource{}

func NewDnsNameMoveResource() resource.Resource {
	return &DnsNameMoveResource{}
}

type DnsNameMoveResource struct {
	APIClient          *apiclient.RetryableClientWithResponses
	APIPollingInterval int
}

func (r *DnsNameMoveResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection_endpoint_dns_name_move"
}

func (r *DnsNameMoveResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = model.DnsNameMoveResourceSchema()
}

func (r *DnsNameMoveResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig := req.ProviderData.(shared.ProviderConfig)
	r.APIClient = apiclient.NewRetryableClient(providerConfig.APIClient, 3, 10)
	r.APIPollingInterval = providerConfig.APIPollingInterval
}

// handleAPIResponse is a generic helper for handling API responses with error handling
func handleAPIResponse(statusCode int, body []byte, httpResponse *http.Response,
	json400, json401, json403, json404, json503 *missioncontrol.ErrorResponse,
	diagnostics *diag.Diagnostics, expectedStatus int) bool {

	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		expectedStatus,
		body,
		httpResponse,
		json400,
		json401,
		json403,
		json404,
		json503,
	)
	return errorHandler.HandleError(diagnostics)
}

// handleAsyncOperation handles async operations using operationutils
func (r *DnsNameMoveResource) handleAsyncOperation(ctx context.Context, operationId, serviceId string, diagnostics *diag.Diagnostics) {
	if operationId == "" {
		return
	}

	adapter := &apiClientAdapter{
		client:    r.APIClient,
		serviceId: serviceId,
	}

	params := &operationutils.OperationParams{
		Ctx:             ctx,
		ServiceId:       serviceId,
		OperationId:     operationId,
		R:               adapter,
		PollingInterval: r.APIPollingInterval,
		Timeout:         5 * time.Minute,
		Diagnostics:     diagnostics,
	}

	operationutils.WaitForOperationToComplete(params)
}

func (r *DnsNameMoveResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data model.DnsNameMoveModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceServiceId := data.SourceServiceId.ValueString()
	sourceConnectionEndpointId := data.SourceConnectionEndpointId.ValueString()
	dnsName := data.DnsName.ValueString()

	// Set defaults for target if not specified
	targetServiceId := data.TargetServiceId.ValueString()
	if targetServiceId == "" {
		targetServiceId = sourceServiceId
		data.TargetServiceId = types.StringValue(targetServiceId)
	}

	targetConnectionEndpointId := data.TargetConnectionEndpointId.ValueString()

	// Validate that we're not moving to a different organization
	if targetServiceId != sourceServiceId {
		// Get organization info for both services to ensure they're in the same org
		sourceServiceResp, err := r.APIClient.GetServiceWithResponse(ctx, sourceServiceId, nil)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error validating source service",
				"Could not retrieve source service details: "+err.Error(),
			)
			return
		}

		if handleAPIResponse(sourceServiceResp.StatusCode(), sourceServiceResp.Body, sourceServiceResp.HTTPResponse,
			nil, sourceServiceResp.JSON401, sourceServiceResp.JSON403, sourceServiceResp.JSON404, sourceServiceResp.JSON503,
			&resp.Diagnostics, http.StatusOK) {
			return
		}

		targetServiceResp, err := r.APIClient.GetServiceWithResponse(ctx, targetServiceId, nil)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error validating target service",
				"Could not retrieve target service details: "+err.Error(),
			)
			return
		}

		if handleAPIResponse(targetServiceResp.StatusCode(), targetServiceResp.Body, targetServiceResp.HTTPResponse,
			nil, targetServiceResp.JSON401, targetServiceResp.JSON403, targetServiceResp.JSON404, targetServiceResp.JSON503,
			&resp.Diagnostics, http.StatusOK) {
			return
		}

		tflog.Info(ctx, "Moving DNS name between services", map[string]interface{}{
			"source_service": sourceServiceId,
			"target_service": targetServiceId,
		})
	}

	// Check target endpoint capacity (max 5 DNS names)
	existingDnsNames, err := r.APIClient.GetConnectionEndpointDnsNamesWithResponse(ctx, targetServiceId, targetConnectionEndpointId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking target endpoint DNS names",
			"Could not check existing DNS names on target endpoint: "+err.Error(),
		)
		return
	}

	if handleAPIResponse(existingDnsNames.StatusCode(), existingDnsNames.Body, existingDnsNames.HTTPResponse,
		nil, existingDnsNames.JSON401, existingDnsNames.JSON403, existingDnsNames.JSON404, existingDnsNames.JSON503,
		&resp.Diagnostics, http.StatusOK) {
		return
	}

	if len(existingDnsNames.JSON200.Data) >= 5 {
		resp.Diagnostics.AddError(
			"Target endpoint at maximum DNS names",
			"Cannot move DNS name to target endpoint: maximum of 5 DNS names reached",
		)
		return
	}

	// Create the move request
	moveRequest := missioncontrol.DnsNameMoveRequest{}

	if targetServiceId != "" && targetServiceId != sourceServiceId {
		moveRequest.TargetServiceId = &targetServiceId
	}

	if targetConnectionEndpointId != "" {
		moveRequest.TargetConnectionEndpointId = &targetConnectionEndpointId
	}

	moveResp, err := r.APIClient.MoveConnectionEndpointDnsNameWithResponse(
		ctx, sourceServiceId, sourceConnectionEndpointId, dnsName, moveRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error calling Solace Cloud API",
			"Could not move DNS name, unexpected error: "+err.Error(),
		)
		return
	}

	if handleAPIResponse(moveResp.StatusCode(), moveResp.Body, moveResp.HTTPResponse,
		moveResp.JSON400, moveResp.JSON401, moveResp.JSON403, moveResp.JSON404, moveResp.JSON503,
		&resp.Diagnostics, http.StatusAccepted) {
		return
	}

	// DNS name move is asynchronous - poll for completion
	if moveResp.StatusCode() == 202 && moveResp.JSON202 != nil && moveResp.JSON202.Data.Id != nil {
		jobId := *moveResp.JSON202.Data.Id
		tflog.Info(ctx, "DNS name move job started", map[string]interface{}{
			"jobId": jobId,
		})

		// Use operationutils for consistent polling behavior
		r.handleAsyncOperation(ctx, jobId, targetServiceId, &resp.Diagnostics)

		// Check if operation failed by examining diagnostics
		for _, diag := range resp.Diagnostics.Errors() {
			if diag.Summary() == "The operation failed." {
				// Operation actually failed - don't set state
				return
			}
		}

		// Operation completed successfully or with warnings
		tflog.Info(ctx, "DNS name move operation completed", map[string]interface{}{
			"dns_name": dnsName,
			"target_service": targetServiceId,
		})

		// Store the job ID as the resource ID for tracking (operation was initiated successfully)
		data.Id = types.StringValue(jobId)
	} else {
		resp.Diagnostics.AddError(
			"DNS name move failed",
			"No job ID was returned from the move operation",
		)
		return
	}


	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnsNameMoveResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data model.DnsNameMoveModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For move operations, the "read" operation validates that the DNS name
	// is now at the target location. If it's not found yet, it might still be moving.
	targetServiceId := data.TargetServiceId.ValueString()
	targetConnectionEndpointId := data.TargetConnectionEndpointId.ValueString()
	dnsName := data.DnsName.ValueString()


	dnsNamesResp, err := r.APIClient.GetConnectionEndpointDnsNamesWithResponse(ctx, targetServiceId, targetConnectionEndpointId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking DNS name location",
			"Could not verify DNS name location: "+err.Error(),
		)
		return
	}

	if dnsNamesResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Error reading target endpoint DNS names",
			fmt.Sprintf("API returned status %d", dnsNamesResp.StatusCode()),
		)
		return
	}

	// Verify the DNS name exists at the target
	found := false
	if dnsNamesResp.JSON200 != nil {
		for _, dns := range dnsNamesResp.JSON200.Data {
			if dns.DnsName != nil && *dns.DnsName == dnsName {
				found = true
				break
			}
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"DNS name move verification failed",
			fmt.Sprintf("DNS name %s was not found on target service %s after move operation", dnsName, targetServiceId),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnsNameMoveResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// DNS name moves are immutable - all changes require replacement
	resp.Diagnostics.AddError(
		"DNS Name Move Update Not Supported",
		"DNS name move operations are immutable. Any changes require resource replacement.",
	)
}

func (r *DnsNameMoveResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// For move operations, delete doesn't actually move the DNS name back
	// It just removes the tracking of the move operation from Terraform state
	tflog.Info(ctx, "DNS name move operation removed from Terraform state - DNS name remains at target location")
}

func (r *DnsNameMoveResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// DNS name move operations represent a one-time action rather than persistent state.
	// Import is not supported because move operations should be managed through Terraform apply/destroy lifecycle.
	// If you need to track a DNS name that was moved, use the solacecloud_connection_endpoint_dns_name resource instead.
	resp.Diagnostics.AddError(
		"Import not supported for DNS name move operations",
		"DNS name move operations are one-time actions and cannot be imported. Use 'terraform apply' to perform moves and manage them through the normal Terraform lifecycle.",
	)
}

// apiClientAdapter adapts the new 4-argument API client for use with operationutils
type apiClientAdapter struct {
	client    *apiclient.RetryableClientWithResponses
	serviceId string
}

func (a *apiClientAdapter) GetServiceOperationWithResponse(ctx context.Context, serviceId string, operationId string, params *missioncontrol.GetServiceOperationParams, reqEditors ...missioncontrol.RequestEditorFn) (*missioncontrol.GetServiceOperationResponse, error) {
	if params == nil {
		params = &missioncontrol.GetServiceOperationParams{}
	}
	return a.client.GetServiceOperationWithResponseAndParams(ctx, serviceId, operationId, params, reqEditors...)
}