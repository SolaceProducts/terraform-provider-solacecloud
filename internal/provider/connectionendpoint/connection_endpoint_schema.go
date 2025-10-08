package connectionendpoint

import (
	"terraform-provider-solacecloud/internal/model"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ConnectionEndpointSchema defines the schema for the Solace Cloud Connection Endpoint resource
func ConnectionEndpointSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The identifier of the connection endpoint.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"service_id": schema.StringAttribute{
			Required:    true,
			Description: "The ID of the Solace Cloud Service this connection endpoint belongs to.",
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
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
			Computed: true,
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
		"ports": model.ConnectionEndpointProtocolSchema(),
	}
}

// ConnectionEndpointResourceModel describes the resource data model for the standalone connection endpoint resource.
// This model corresponds to the schema defined in the connection endpoint resource and is used for
// managing individual connection endpoints as a separate resource.
type ConnectionEndpointResourceModel struct {
	Id             types.String          `tfsdk:"id"`
	ServiceId      types.String          `tfsdk:"service_id"`
	Name           types.String          `tfsdk:"name"`
	Description    types.String          `tfsdk:"description"`
	AccessType     types.String          `tfsdk:"access_type"`
	K8SServiceType types.String          `tfsdk:"k8s_service_type"`
	K8SServiceId   types.String          `tfsdk:"k8s_service_id"`
	Ports          basetypes.ObjectValue `tfsdk:"ports"`
}
