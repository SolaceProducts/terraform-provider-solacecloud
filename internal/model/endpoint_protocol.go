package model

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Package model contains the EndpointProtocol schema which is nested into the EndpointProtocols object.
// Each instance of EndpointProtocol represents a single protocol that is served by the service.  When null, it means
// the protocol is disabled.  Otherwise, the protocol is enabled and the port number is specified.

// EndpointProtocolModel represents one TCP Port of a connection endpoints, and which protocol it provides service for.
type EndpointProtocolModel struct {
	Port types.Int64 `tfsdk:"port"`
}

func NullEndpointProtocol() basetypes.ObjectValue {
	return types.ObjectNull(map[string]attr.Type{
		"port": types.Int64Type,
	})
}

func EndpointProtocolModelType() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"port": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (m EndpointProtocolModel) ToObjectValue() (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValue(
		map[string]attr.Type{
			"port": types.Int64Type,
		},
		map[string]attr.Value{
			"port": m.Port,
		})
}
