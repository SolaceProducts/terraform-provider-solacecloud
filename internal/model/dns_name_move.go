package model

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// DnsNameMoveModel represents a DNS name move operation
type DnsNameMoveModel struct {
	Id                         types.String `tfsdk:"id"`
	SourceServiceId            types.String `tfsdk:"source_service_id"`
	SourceConnectionEndpointId types.String `tfsdk:"source_connection_endpoint_id"`
	DnsName                    types.String `tfsdk:"dns_name"`
	TargetServiceId            types.String `tfsdk:"target_service_id"`
	TargetConnectionEndpointId types.String `tfsdk:"target_connection_endpoint_id"`
}

func DnsNameMoveResourceSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Moves a DNS name from one connection endpoint to another. " +
			"The DNS name cannot be moved to a different organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The operation identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_service_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the source event broker service.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_connection_endpoint_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the source connection endpoint.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dns_name": schema.StringAttribute{
				MarkdownDescription: "The DNS name to move.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_service_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the target event broker service. " +
					"If not specified, defaults to the source service ID.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_connection_endpoint_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the target connection endpoint. " +
					"If not specified, defaults to the target service's only connection endpoint.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func DnsNameMoveTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                            types.StringType,
		"source_service_id":             types.StringType,
		"source_connection_endpoint_id": types.StringType,
		"dns_name":                      types.StringType,
		"target_service_id":             types.StringType,
		"target_connection_endpoint_id": types.StringType,
	}
}

func (m DnsNameMoveModel) ToObjectValue() (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValue(
		DnsNameMoveTypes(),
		map[string]attr.Value{
			"id":                            m.Id,
			"source_service_id":             m.SourceServiceId,
			"source_connection_endpoint_id": m.SourceConnectionEndpointId,
			"dns_name":                      m.DnsName,
			"target_service_id":             m.TargetServiceId,
			"target_connection_endpoint_id": m.TargetConnectionEndpointId,
		})
}
