package model

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// NestedConnectionEndpointModel represents the model for the nested connection endpoint
// used in the connection_endpoint attribute of the service resource.
// This differs from ConnectionEndpointModel by including service_id and excluding hostnames.
type NestedConnectionEndpointModel struct {
	Id             types.String          `tfsdk:"id"`
	ServiceId      types.String          `tfsdk:"service_id"`
	Name           types.String          `tfsdk:"name"`
	Description    types.String          `tfsdk:"description"`
	AccessType     types.String          `tfsdk:"access_type"`
	K8SServiceType types.String          `tfsdk:"k8s_service_type"`
	K8SServiceId   types.String          `tfsdk:"k8s_service_id"`
	Ports          basetypes.ObjectValue `tfsdk:"ports"`
}

func NestedConnectionEndpointTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":               types.StringType,
		"service_id":       types.StringType,
		"name":             types.StringType,
		"description":      types.StringType,
		"access_type":      types.StringType,
		"k8s_service_type": types.StringType,
		"k8s_service_id":   types.StringType,
		"ports":            EndpointProtocolSchema().GetType(),
	}
}

func (m NestedConnectionEndpointModel) ToObjectValue() (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValue(
		NestedConnectionEndpointTypes(),
		map[string]attr.Value{
			"id":               m.Id,
			"service_id":       m.ServiceId,
			"name":             m.Name,
			"description":      m.Description,
			"access_type":      m.AccessType,
			"k8s_service_type": m.K8SServiceType,
			"k8s_service_id":   m.K8SServiceId,
			"ports":            m.Ports,
		})
}
