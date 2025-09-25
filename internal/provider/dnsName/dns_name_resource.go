// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &DnsNameResource{}
var _ resource.ResourceWithImportState = &DnsNameResource{}

func NewDnsNameResource() resource.Resource {
	return &DnsNameResource{}
}

type DnsNameResource struct {
	APIClient          *apiclient.RetryableClientWithResponses
	APIPollingInterval int
}

func (r *DnsNameResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection_endpoint_dns_name"
}

func (r *DnsNameResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = model.DnsNameResourceSchema()
}

func (r *DnsNameResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig := req.ProviderData.(shared.ProviderConfig)
	r.APIClient = apiclient.NewRetryableClient(providerConfig.APIClient, 3, 10)
	r.APIPollingInterval = providerConfig.APIPollingInterval
}

func (r *DnsNameResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data model.DnsNameModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := data.ServiceId.ValueString()
	connectionEndpointId := data.ConnectionEndpointId.ValueString()
	dnsName := data.DnsName.ValueString()

	// First check if we can reach the maximum of 5 hostnames
	existingDnsNames, err := r.APIClient.GetConnectionEndpointDnsNamesWithResponse(ctx, serviceId, connectionEndpointId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking existing DNS names",
			"Could not check existing DNS names: "+err.Error(),
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
			"Maximum DNS names reached",
			"Cannot add more than 5 DNS names to a connection endpoint",
		)
		return
	}

	// Create the DNS name using the OpenAPI client
	varDnsNameBody := missioncontrol.DnsNameCreateRequest{
		DnsName: dnsName,
	}

	createResp, err := r.APIClient.CreateConnectionEndpointDnsNameWithResponse(ctx, serviceId, connectionEndpointId, varDnsNameBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error calling Solace Cloud API",
			"Could not create DNS name, unexpected error: "+err.Error(),
		)
		return
	}

	if handleAPIResponse(createResp.StatusCode(), createResp.Body, createResp.HTTPResponse,
		createResp.JSON400, createResp.JSON401, createResp.JSON403, createResp.JSON404, createResp.JSON503,
		&resp.Diagnostics, http.StatusAccepted) {
		return
	}

	// DNS name creation is asynchronous - poll for completion
	if createResp.JSON202 != nil && createResp.JSON202.Data.Id != nil {
		jobId := *createResp.JSON202.Data.Id
		tflog.Info(ctx, "DNS name creation job started", map[string]interface{}{
			"jobId": jobId,
		})

		// Use operationutils for consistent polling behavior
		adapter := &apiClientAdapter{
			client:    r.APIClient,
			serviceId: serviceId,
		}

		params := &operationutils.OperationParams{
			Ctx:             ctx,
			ServiceId:       serviceId,
			OperationId:     jobId,
			R:               adapter,
			PollingInterval: r.APIPollingInterval,
			Timeout:         5 * time.Minute,
			Diagnostics:     &resp.Diagnostics,
		}

		// Wait for operation to complete - errors are handled through Diagnostics
		operationutils.WaitForOperationToComplete(params)

		// Check if operation failed by examining diagnostics
		for _, diag := range resp.Diagnostics.Errors() {
			if diag.Summary() == "The operation failed." {
				// Operation actually failed - don't continue
				return
			}
		}

		// Operation completed successfully or with warnings
		tflog.Info(ctx, "DNS name creation operation completed")
	}

	// Read back the created DNS name
	r.readDnsName(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnsNameResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data model.DnsNameModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readDnsName(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnsNameResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// DNS names are immutable - all changes require replacement
	resp.Diagnostics.AddError(
		"DNS Name Update Not Supported",
		"DNS names are immutable. Any changes require resource replacement.",
	)
}

func (r *DnsNameResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data model.DnsNameModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := data.ServiceId.ValueString()
	connectionEndpointId := data.ConnectionEndpointId.ValueString()
	dnsName := data.DnsName.ValueString()

	// Let the backend handle default hostname validation

	deleteResp, err := r.APIClient.DeleteConnectionEndpointDnsNameWithResponse(ctx, serviceId, connectionEndpointId, dnsName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error calling Solace Cloud API",
			"Could not delete DNS name, unexpected error: "+err.Error(),
		)
		return
	}

	// Handle not found error
	if deleteResp.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddError(
			"DNS name not found",
			fmt.Sprintf("DNS name '%s' was not found on connection endpoint %s. It may have already been deleted or never existed.", dnsName, connectionEndpointId),
		)
		return
	}

	if handleAPIResponse(deleteResp.StatusCode(), deleteResp.Body, deleteResp.HTTPResponse,
		deleteResp.JSON400, deleteResp.JSON401, deleteResp.JSON403, nil, deleteResp.JSON503,
		&resp.Diagnostics, http.StatusAccepted) {
		return
	}

	// DNS name deletion is asynchronous - poll for completion
	if deleteResp.JSON202 != nil && deleteResp.JSON202.Data.Id != nil {
		jobId := *deleteResp.JSON202.Data.Id
		tflog.Info(ctx, "DNS name deletion job started", map[string]interface{}{
			"jobId": jobId,
		})

		// Use operationutils for consistent polling behavior
		adapter := &apiClientAdapter{
			client:    r.APIClient,
			serviceId: serviceId,
		}

		params := &operationutils.OperationParams{
			Ctx:             ctx,
			ServiceId:       serviceId,
			OperationId:     jobId,
			R:               adapter,
			PollingInterval: r.APIPollingInterval,
			Timeout:         5 * time.Minute,
			Diagnostics:     &resp.Diagnostics,
		}

		// Wait for operation to complete - errors are handled through Diagnostics
		operationutils.WaitForOperationToComplete(params)

		// Check if operation failed by examining diagnostics
		for _, diag := range resp.Diagnostics.Errors() {
			if diag.Summary() == "The operation failed." {
				// Operation actually failed - don't continue
				return
			}
		}

		// Operation completed successfully or with warnings
		tflog.Info(ctx, "DNS name deletion operation completed successfully")
	}
}

func (r *DnsNameResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: service_id/connection_endpoint_id/dns_name
	parts := shared.ParseImportId(req.ID, 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format: service_id/connection_endpoint_id/dns_name, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connection_endpoint_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("dns_name"), parts[2])...)
}



func (r *DnsNameResource) readDnsName(ctx context.Context, data *model.DnsNameModel, diagnostics *diag.Diagnostics) {
	serviceId := data.ServiceId.ValueString()
	connectionEndpointId := data.ConnectionEndpointId.ValueString()
	dnsName := data.DnsName.ValueString()

	dnsNamesResp, err := r.APIClient.GetConnectionEndpointDnsNamesWithResponse(ctx, serviceId, connectionEndpointId)
	if err != nil {
		diagnostics.AddError(
			"Error calling Solace Cloud API",
			"Could not read DNS names, unexpected error: "+err.Error(),
		)
		return
	}

	if handleAPIResponse(dnsNamesResp.StatusCode(), dnsNamesResp.Body, dnsNamesResp.HTTPResponse,
		nil, nil, nil, nil, nil, diagnostics, http.StatusOK) {
		return
	}

	// Find the specific DNS name
	if dnsNamesResp.JSON200 != nil {
		for _, dns := range dnsNamesResp.JSON200.Data {
			if dns.DnsName != nil && *dns.DnsName == dnsName {
				data.Id = types.StringPointerValue(dns.Id)
				data.DnsName = types.StringPointerValue(dns.DnsName)
				data.DomainType = types.StringValue(string(*dns.DomainType))
				return
			}
		}
	}

	// If we get here, the DNS name was not found
	diagnostics.AddError(
		"DNS name not found",
		fmt.Sprintf("DNS name %s not found for connection endpoint %s", dnsName, connectionEndpointId),
	)
}