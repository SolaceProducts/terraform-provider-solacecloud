package model

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"regexp"
)

// DnsNameModel represents a DNS name configuration for a connection endpoint
type DnsNameModel struct {
	Id                   types.String `tfsdk:"id"`
	ServiceId            types.String `tfsdk:"service_id"`
	ConnectionEndpointId types.String `tfsdk:"connection_endpoint_id"`
	DnsName              types.String `tfsdk:"dns_name"`
	DomainType           types.String `tfsdk:"domain_type"`
}

func DnsNameResourceSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Creates and manages a custom DNS name (hostname) for a Solace Cloud event broker service connection endpoint.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the DNS name.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the event broker service.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"connection_endpoint_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the connection endpoint.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dns_name": schema.StringAttribute{
				MarkdownDescription: "The fully qualified domain name (FQDN) to use for the DNS name. Each label must be between 1-63 characters.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$`),
						"DNS name must be a valid FQDN with alphanumeric characters, dots, and hyphens only. Each label must be between 1-63 characters and cannot start or end with hyphens.",
					),
					stringvalidator.LengthAtMost(230),
				},
			},
			"domain_type": schema.StringAttribute{
				MarkdownDescription: "The domain type: CustomerManaged or SolaceManaged.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func DnsNameTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                     types.StringType,
		"service_id":             types.StringType,
		"connection_endpoint_id": types.StringType,
		"dns_name":               types.StringType,
		"domain_type":            types.StringType,
	}
}

func (m DnsNameModel) ToObjectValue() (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValue(
		DnsNameTypes(),
		map[string]attr.Value{
			"id":                     m.Id,
			"service_id":             m.ServiceId,
			"connection_endpoint_id": m.ConnectionEndpointId,
			"dns_name":               m.DnsName,
			"domain_type":            m.DomainType,
		})
}