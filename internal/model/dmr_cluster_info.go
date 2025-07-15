package model

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Package model contains the DmrClusterInfo schema which is an object nested in the service resource.  It describes
// the service's DMR cluster information, which would be needed to configure a DMR link from another service to the one
// described here.

type DmrClusterInfoModel struct {
	Name                         types.String        `tfsdk:"name"`
	Password                     types.String        `tfsdk:"password"`
	RemoteAddress                types.String        `tfsdk:"remote_address"`
	PrimaryRouterName            types.String        `tfsdk:"primary_router_name"`
	SupportedAuthenticationModes basetypes.ListValue `tfsdk:"supported_authentication_modes"`
}

func DmrClusterInfoAttributeSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "The DMR cluster details.",
		Computed:            true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the DMR cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password for the cluster.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"remote_address": schema.StringAttribute{
				MarkdownDescription: "The address of the remote node in the cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"primary_router_name": schema.StringAttribute{
				MarkdownDescription: "The name of the primary router in the DMR cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"supported_authentication_modes": schema.ListAttribute{
				MarkdownDescription: "The authentication mode between the nodes in the DMR cluster.",
				ElementType:         types.StringType,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func DmrClusterInfoObjectType() types.ObjectType {
	return basetypes.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":                           types.StringType,
			"password":                       types.StringType,
			"remote_address":                 types.StringType,
			"primary_router_name":            types.StringType,
			"supported_authentication_modes": basetypes.ListType{ElemType: types.StringType},
		},
	}
}

func (m DmrClusterInfoModel) ToObjectValue() (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValue(
		DmrClusterInfoObjectType().AttrTypes,
		map[string]attr.Value{
			"name":                           m.Name,
			"password":                       m.Password,
			"remote_address":                 m.RemoteAddress,
			"primary_router_name":            m.PrimaryRouterName,
			"supported_authentication_modes": m.SupportedAuthenticationModes,
		})
}
