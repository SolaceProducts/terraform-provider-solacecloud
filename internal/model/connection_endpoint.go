package model

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Package model contains the Terraform Schemas and all utility methods to work with them.
// Each nested object types are defined in their own file.
// The connection endpoint model defines the schema for the connection endpoint object, which is nested as a list
// in the service resource.
// Each connection endpoint represents an address over which the broker is accessible.  A connection endpoint exposes
// a set of protocols which is modeled by the endpoint protocols schema.
// Finally, each connection endpoint is also associated with a set of hostnames.  These hostnames resolve to this
// connection endpoint.  The first hostname of the list is the preferred hostname.

type ConnectionEndpointModel struct {
	Id             types.String          `tfsdk:"id"`
	Name           types.String          `tfsdk:"name"`
	Description    types.String          `tfsdk:"description"`
	AccessType     types.String          `tfsdk:"access_type"`
	K8SServiceType types.String          `tfsdk:"k8s_service_type"`
	K8SServiceId   types.String          `tfsdk:"k8s_service_id"`
	Hostnames      basetypes.ListValue   `tfsdk:"hostnames"`
	Ports          basetypes.ObjectValue `tfsdk:"ports"`
}

func ConnectionEndpointSchema() schema.NestedAttributeObject {
	return schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the connection endpoint.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the connection endpoint.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 50),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description for the connection endpoint.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"access_type": schema.StringAttribute{
				MarkdownDescription: "The connectivity for the connection endpoint. This can be either PRIVATE (private IP) " +
					"or PUBLIC (public Internet IP)",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"PRIVATE",
						"PUBLIC",
					),
				},
			},
			"k8s_service_type": schema.StringAttribute{
				MarkdownDescription: "The connectivity configuration that is used in the Kubernetes cluster.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"NodePort",
						"LoadBalancer",
						"ClusterIP",
					),
				},
			},
			"k8s_service_id": schema.StringAttribute{
				MarkdownDescription: "The identifier for the Kubernetes service.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostnames": schema.ListAttribute{
				MarkdownDescription: "The hostnames assigned to the connection endpoint.",
				ElementType:         types.StringType,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"ports": EndpointProtocolSchema(),
		},
	}
}

func ConnectionEndpointListSchema() schema.Attribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: "The list of Connection Endpoints for this service.  If left empty, all TLS protocols are enabled on their default ports, and the connection endpoint access type uses the datacenter's default access type.",
		Computed:            true,
		NestedObject:        ConnectionEndpointSchema(),
		PlanModifiers: []planmodifier.List{
			listplanmodifier.UseStateForUnknown(),
		},
	}
}

func ConnectionEndpointTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":               types.StringType,
		"name":             types.StringType,
		"description":      types.StringType,
		"access_type":      types.StringType,
		"k8s_service_type": types.StringType,
		"k8s_service_id":   types.StringType,
		"hostnames":        types.ListType{ElemType: types.StringType},
		"ports":            EndpointProtocolSchema().GetType(),
	}
}

func (m ConnectionEndpointModel) ToObjectValue() (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValue(
		ConnectionEndpointTypes(),
		map[string]attr.Value{
			"id":               m.Id,
			"name":             m.Name,
			"description":      m.Description,
			"access_type":      m.AccessType,
			"k8s_service_type": m.K8SServiceType,
			"k8s_service_id":   m.K8SServiceId,
			"hostnames":        m.Hostnames,
			"ports":            m.Ports,
		})
}
