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

// Package model contains the ManagementUserCrendential schema which is an object nested in the service resource.
// It describes the service's Management User credentials, which would be needed to login to the broker's PubSub+
// manager web UI, or to authenticate with the broker's SEMP management interface.

// BasicAuthCredentialModel represents the credentials of a Management (SEMP) user.
// This is a basic auth user/password credential.
type BasicAuthCredentialModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func BasicAuthCredentialAttributeSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func BasicAuthCredentialObjectType() types.ObjectType {
	return basetypes.ObjectType{
		AttrTypes: map[string]attr.Type{
			"username": types.StringType,
			"password": types.StringType,
		},
	}
}

func (m BasicAuthCredentialModel) ToObjectValue() (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValue(
		BasicAuthCredentialObjectType().AttrTypes,
		map[string]attr.Value{
			"username": m.Username,
			"password": m.Password,
		})
}
