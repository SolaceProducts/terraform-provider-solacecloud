package provider

import (
	"context"
	"regexp"
	"strings"
	"terraform-provider-solacecloud/internal/model"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

// ServiceResource defines the resource implementation.
type ServiceResource struct {
	APIClient          *RetryableClientWithResponses
	APIPollingInterval int
	APIToken           string
}

type ServiceResourceModel struct {
	Id                  types.String          `tfsdk:"id"`
	Name                types.String          `tfsdk:"name"`
	EventBrokerVersion  types.String          `tfsdk:"event_broker_version"`
	MessageVpnName      types.String          `tfsdk:"message_vpn_name"`
	MaxSpoolUsage       types.Int64           `tfsdk:"max_spool_usage"`
	ServiceClassId      types.String          `tfsdk:"service_class_id"`
	DatacenterId        types.String          `tfsdk:"datacenter_id"`
	ClusterName         types.String          `tfsdk:"cluster_name"`
	OwnedBy             types.String          `tfsdk:"owned_by"`
	Locked              types.Bool            `tfsdk:"locked"`
	MateLinkEncryption  types.Bool            `tfsdk:"mate_link_encryption"`
	ConnectionEndpoints types.List            `tfsdk:"connection_endpoints"`
	CustomRouterName    types.String          `tfsdk:"custom_router_name"`
	EnvironmentId       types.String          `tfsdk:"environment_id"`
	MessageVpn          basetypes.ObjectValue `tfsdk:"message_vpn"`
	DmrClusterInfo      basetypes.ObjectValue `tfsdk:"dmr_cluster"`
}

type NameNotDefaultValidator struct{}

func (v NameNotDefaultValidator) Description(ctx context.Context) string {
	return "Validates that the string is not 'default'"
}

func (v NameNotDefaultValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates that the string is not 'default'"
}

func (v NameNotDefaultValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if strings.ToLower(req.ConfigValue.ValueString()) == "default" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Name",
			"Name cannot be 'default'",
		)
	}
}

func (r *ServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Solace Cloud Service resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The event broker service name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(50),
				},
			},
			"event_broker_version": schema.StringAttribute{
				MarkdownDescription: "The event broker version. A default version is provided when this is not specified.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+-\d+$`),
						"event broker version format is major.minor.load.build-cloudRevision",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					NewImmutableStringPlanModifier(),
				},
			},
			"message_vpn_name": schema.StringAttribute{
				MarkdownDescription: "The message VPN name. A default message VPN name is provided when this is not specified.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 26),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9\-_]*$`),
						"may only contain alphanumeric, - or _ characters",
					),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z_].*`),
						"must begin with alphabetic or _ characters",
					),
					NameNotDefaultValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					NewImmutableStringPlanModifier(),
				},
			},
			"max_spool_usage": schema.Int64Attribute{
				MarkdownDescription: "The message spool size, in gigabytes (GB). A default message spool size is provided if this is not specified.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(10),
					int64validator.AtMost(6000),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"service_class_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the service class.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("DEVELOPER"),
				Validators: []validator.String{
					stringvalidator.OneOf(
						"DEVELOPER",
						"ENTERPRISE_250_HIGHAVAILABILITY",
						"ENTERPRISE_1K_HIGHAVAILABILITY",
						"ENTERPRISE_50K_HIGHAVAILABILITY",
						"ENTERPRISE_100K_HIGHAVAILABILITY",
						"ENTERPRISE_5K_HIGHAVAILABILITY",
						"ENTERPRISE_10K_HIGHAVAILABILITY",
						"ENTERPRISE_250_STANDALONE",
						"ENTERPRISE_1K_STANDALONE",
						"ENTERPRISE_5K_STANDALONE",
						"ENTERPRISE_10K_STANDALONE",
						"ENTERPRISE_50K_STANDALONE",
						"ENTERPRISE_100K_STANDALONE",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					NewImmutableStringPlanModifier(),
				},
			},
			"datacenter_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the datacenter.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 50),
				},
				PlanModifiers: []planmodifier.String{
					NewImmutableStringPlanModifier(),
				},
			},
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "The name of the DMR cluster where the service will be created",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9\-_]*$`),
						"may only contain alphanumeric, - or _ characters",
					),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z_].*`),
						"must begin with alphabetic or _ characters",
					),
					NameNotDefaultValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					NewImmutableStringPlanModifier(),
				},
			},
			"owned_by": schema.StringAttribute{
				MarkdownDescription: "The unique identifier representing the user who owns the event broker service.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"locked": schema.BoolAttribute{
				MarkdownDescription: "Indicates if you can delete the event broker service after creating it. " +
					"The default value is false, and the valid values are: <p><ul><li>'true' - " +
					"you cannot delete this service</li><li>'false' - you can delete this service</li></ul></p>",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown()},
			},
			"mate_link_encryption": schema.BoolAttribute{
				MarkdownDescription: "Enable or disable SSL for the redundancy group (for mate-link encryption). " +
					"The default value is false and the valid values are: <p><ul><li>'true' - enabled</li><li>'false' - disabled</li></ul></p>",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"connection_endpoints": model.ConnectionEndpointListSchema(),
			"custom_router_name": schema.StringAttribute{
				MarkdownDescription: "The unique prefix for the name of the router for the event broker service. " +
					"If left undefined, the service ID will be used.  Defining this is useful when replacing a " +
					"part of a DMR Cluster or DR setup.  Should be left undefined for most use cases.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					NewCustomRouterNamePlanModifier(),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the environment where you want to create the service." +
					"You can only specify an environment identifier when creating services in a Public Region. " +
					"You cannot specify an environment identifier when creating a service in a Dedicated Region." +
					"Creating a service in a Public Region without specifying an environment identifier places it" +
					"in the default environment.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(),
					NewImmutableStringPlanModifier(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Solace Cloud Service ID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"message_vpn": model.MessageVpnAttributeSchema(),
			"dmr_cluster": model.DmrClusterInfoAttributeSchema(),
		},
	}
}
