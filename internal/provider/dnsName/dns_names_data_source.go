package dnsName

import (
	"context"
	"net/http"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/internal/shared"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DnsNamesDataSource{}

func NewDnsNamesDataSource() datasource.DataSource {
	return &DnsNamesDataSource{}
}

type DnsNamesDataSource struct {
	APIClient *apiclient.RetryableClientWithResponses
}

type DnsNamesDataSourceModel struct {
	ServiceId            types.String    `tfsdk:"service_id"`
	ConnectionEndpointId types.String    `tfsdk:"connection_endpoint_id"`
	DnsNames             []DnsNameModel  `tfsdk:"dns_names"`
}

type DnsNameModel struct {
	Id         types.String `tfsdk:"id"`
	DnsName    types.String `tfsdk:"dns_name"`
	DomainType types.String `tfsdk:"domain_type"`
}

func (d *DnsNamesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection_endpoint_dns_names"
}

func (d *DnsNamesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches all DNS names for a specific connection endpoint.",
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the event broker service.",
				Required:            true,
			},
			"connection_endpoint_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the connection endpoint.",
				Required:            true,
			},
			"dns_names": schema.ListNestedAttribute{
				MarkdownDescription: "List of DNS names for the connection endpoint.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The identifier of the DNS name.",
							Computed:            true,
						},
						"dns_name": schema.StringAttribute{
							MarkdownDescription: "The fully qualified domain name.",
							Computed:            true,
						},
						"domain_type": schema.StringAttribute{
							MarkdownDescription: "The domain type: CustomerManaged or SolaceManaged.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *DnsNamesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig := req.ProviderData.(shared.ProviderConfig)
	d.APIClient = apiclient.NewRetryableClient(providerConfig.APIClient, 3, 10)
}

func (d *DnsNamesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DnsNamesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := data.ServiceId.ValueString()
	connectionEndpointId := data.ConnectionEndpointId.ValueString()

	dnsNamesResp, err := d.APIClient.GetConnectionEndpointDnsNamesWithResponse(ctx, serviceId, connectionEndpointId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error calling Solace Cloud API",
			"Could not read DNS names, unexpected error: "+err.Error(),
		)
		return
	}

	errorHandler := shared.NewMissionControlErrorResponseAdaptor(
		http.StatusOK,
		dnsNamesResp.Body,
		dnsNamesResp.HTTPResponse,
		nil, nil, nil, nil, nil,
	)

	if errorHandler.HandleError(&resp.Diagnostics) {
		return
	}

	// Process the DNS names
	var dnsNames []DnsNameModel
	if dnsNamesResp.JSON200 != nil {
		for _, dns := range dnsNamesResp.JSON200.Data {
			dnsNameModel := DnsNameModel{
				Id:         types.StringPointerValue(dns.Id),
				DnsName:    types.StringPointerValue(dns.DnsName),
				DomainType: types.StringValue(string(*dns.DomainType)),
			}
			dnsNames = append(dnsNames, dnsNameModel)
		}
	}

	data.DnsNames = dnsNames

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}