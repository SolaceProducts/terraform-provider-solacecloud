package model

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Package model contains the MessageVpn schema which is an object nested in the service resource.  It describes
// all the settings of the Message VPN that has been configured on the service.
// Is also contains the three management user credentials that are used to authenticate with the broker's SEMP
// management interface.  Each one of them ties directly to a Mission Control role: Manager, Editor, and Viewer.
// Their attribute field name refer to which role they map to.
// It also describes how messaging clients authentication has been configured and which authentication methods are
// enabled.
// Finally, this schema also describes the service's limits for the Message VPN.

type MessageVpnModel struct {
	Name                                        types.String          `tfsdk:"name"`
	AuthenticationBasicEnabled                  types.Bool            `tfsdk:"authentication_basic_enabled"`
	AuthenticationBasicType                     types.String          `tfsdk:"authentication_basic_type"`
	AuthenticationClientCertEnabled             types.Bool            `tfsdk:"authentication_client_cert_enabled"`
	AuthenticationClientCertValidateDateEnabled types.Bool            `tfsdk:"authentication_client_cert_validate_date_enabled"`
	MaxConnectionCount                          types.Int64           `tfsdk:"max_connection_count"`
	MaxEgressFlowCount                          types.Int64           `tfsdk:"max_egress_flow_count"`
	MaxEndpointCount                            types.Int64           `tfsdk:"max_endpoint_count"`
	MaxIngressFlowCount                         types.Int64           `tfsdk:"max_ingress_flow_count"`
	MaxMsgSpoolUsage                            types.Int64           `tfsdk:"max_msg_spool_usage"`
	MaxSubscriptionCount                        types.Int64           `tfsdk:"max_subscription_count"`
	MaxTransactedSessionCount                   types.Int64           `tfsdk:"max_transacted_session_count"`
	MaxTransactionCount                         types.Int64           `tfsdk:"max_transaction_count"`
	TruststoreUri                               types.String          `tfsdk:"truststore_uri"`
	ManagerManagementCredential                 basetypes.ObjectValue `tfsdk:"manager_management_credential"`
	EditorManagementCredential                  basetypes.ObjectValue `tfsdk:"editor_management_credential"`
	ViewerManagementCredential                  basetypes.ObjectValue `tfsdk:"viewer_management_credential"`
	MessagingClientCredential                   basetypes.ObjectValue `tfsdk:"messaging_client_credential"`
}

func MessageVpnAttributeSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "The Message VPN details",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Message VPN.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"authentication_basic_enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether basic authentication is enabled.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"authentication_basic_type": schema.StringAttribute{
				MarkdownDescription: "The authentication type.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"INTERNAL",
						"LDAP",
						"RADIUS",
						"NONE",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"authentication_client_cert_enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether client certificate authentication is enabled.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"authentication_client_cert_validate_date_enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the validation of the 'Not Before' and 'Not After' dates in a client certificate is enabled.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"max_connection_count": schema.Int64Attribute{
				MarkdownDescription: "The maximum number of clients that are permitted to simultaneously connect to the Message VPN.\n",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_egress_flow_count": schema.Int64Attribute{
				MarkdownDescription: "The total permitted number of ingress flows (that is, Guaranteed Message client publish flows) for a Message VPN.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_endpoint_count": schema.Int64Attribute{
				MarkdownDescription: "The maximum number of flows that can bind to a non-exclusive durable topic endpoint.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_ingress_flow_count": schema.Int64Attribute{
				MarkdownDescription: "The total permitted number of ingress flows (that is, Guaranteed Message client publish flows) for a Message VPN.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_msg_spool_usage": schema.Int64Attribute{
				MarkdownDescription: "The maximum message spool usage",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					NewMaxSpoolUsagePlanModifier(),
				},
			},
			"max_subscription_count": schema.Int64Attribute{
				MarkdownDescription: "The maximum number of unique subscriptions.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_transacted_session_count": schema.Int64Attribute{
				MarkdownDescription: "The maximum number of simultaneous transacted sessions and/or XA Sessions allowed for the given Message VPN.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_transaction_count": schema.Int64Attribute{
				MarkdownDescription: "The total number of simultaneous transactions (both local transactions and transactions " +
					"within distributed/XA transaction branches) in a Message VPN.",
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"truststore_uri": schema.StringAttribute{
				MarkdownDescription: "The URI for the TLS trust store.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"manager_management_credential": BasicAuthCredentialAttributeSchema(),
			"editor_management_credential":  BasicAuthCredentialAttributeSchema(),
			"viewer_management_credential":  BasicAuthCredentialAttributeSchema(),
			"messaging_client_credential":   BasicAuthCredentialAttributeSchema(),
		},
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
	}
}

func (m MessageVpnModel) ToObjectValue() (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValue(
		map[string]attr.Type{
			"name":                                             types.StringType,
			"authentication_basic_enabled":                     types.BoolType,
			"authentication_basic_type":                        types.StringType,
			"authentication_client_cert_enabled":               types.BoolType,
			"authentication_client_cert_validate_date_enabled": types.BoolType,
			"max_connection_count":                             types.Int64Type,
			"max_egress_flow_count":                            types.Int64Type,
			"max_endpoint_count":                               types.Int64Type,
			"max_ingress_flow_count":                           types.Int64Type,
			"max_msg_spool_usage":                              types.Int64Type,
			"max_subscription_count":                           types.Int64Type,
			"max_transacted_session_count":                     types.Int64Type,
			"max_transaction_count":                            types.Int64Type,
			"truststore_uri":                                   types.StringType,
			"manager_management_credential":                    BasicAuthCredentialObjectType(),
			"editor_management_credential":                     BasicAuthCredentialObjectType(),
			"viewer_management_credential":                     BasicAuthCredentialObjectType(),
			"messaging_client_credential":                      BasicAuthCredentialObjectType(),
		},
		map[string]attr.Value{
			"name":                                             m.Name,
			"authentication_basic_enabled":                     m.AuthenticationBasicEnabled,
			"authentication_basic_type":                        m.AuthenticationBasicType,
			"authentication_client_cert_enabled":               m.AuthenticationClientCertEnabled,
			"authentication_client_cert_validate_date_enabled": m.AuthenticationClientCertValidateDateEnabled,
			"max_connection_count":                             m.MaxConnectionCount,
			"max_egress_flow_count":                            m.MaxEgressFlowCount,
			"max_endpoint_count":                               m.MaxEndpointCount,
			"max_ingress_flow_count":                           m.MaxIngressFlowCount,
			"max_msg_spool_usage":                              m.MaxMsgSpoolUsage,
			"max_subscription_count":                           m.MaxSubscriptionCount,
			"max_transacted_session_count":                     m.MaxTransactedSessionCount,
			"max_transaction_count":                            m.MaxTransactionCount,
			"truststore_uri":                                   m.TruststoreUri,
			"manager_management_credential":                    m.ManagerManagementCredential,
			"editor_management_credential":                     m.EditorManagementCredential,
			"viewer_management_credential":                     m.ViewerManagementCredential,
			"messaging_client_credential":                      m.MessagingClientCredential,
		})
}
