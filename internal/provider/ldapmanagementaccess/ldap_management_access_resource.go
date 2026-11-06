package ldapmanagementaccess

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/internal/provider/operationutils"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/missioncontrol"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	operationTimeout = 5 * time.Minute
	errCallingAPI    = "Error calling Solace Cloud API"
)

var _ resource.Resource = &LdapManagementAccessResource{}
var _ resource.ResourceWithImportState = &LdapManagementAccessResource{}

func NewLdapManagementAccessResource() resource.Resource {
	return &LdapManagementAccessResource{}
}

type LdapManagementAccessResource struct {
	APIClient          *apiclient.RetryableClientWithResponses
	APIPollingInterval int
}

type LdapManagementAccessModel struct {
	Id                    types.String `tfsdk:"id"`
	ServiceId             types.String `tfsdk:"service_id"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	GroupAttributeName    types.String `tfsdk:"group_attribute_name"`
	ReadOnlyGroupNames    types.Set    `tfsdk:"read_only_group_names"`
	ReadWriteGroupNames   types.Set    `tfsdk:"read_write_group_names"`
	MeshManagerGroupNames types.Set    `tfsdk:"mesh_manager_group_names"`
	ManagementProfileName types.String `tfsdk:"management_profile_name"`
}

func (r *LdapManagementAccessResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ldap_management_access"
}

func (r *LdapManagementAccessResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// The API always reports every group list, empty or not, so an omitted list defaults to an empty set.
	groupNames := func(description string) schema.SetAttribute {
		return schema.SetAttribute{
			MarkdownDescription: description,
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages LDAP authentication of management users for an event broker service: " +
			"which LDAP groups can manage the event broker service, and with which Mission Control role. " +
			"The event broker service must already have an LDAP authentication profile named `management`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the event broker service.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the event broker service. Changing it replaces the resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether management users are authenticated against the `management` LDAP authentication profile.",
				Required:            true,
			},
			"group_attribute_name": schema.StringAttribute{
				MarkdownDescription: "The name of the LDAP attribute that carries the group membership of a management user, for example `memberOf`.",
				Required:            true,
			},
			"read_only_group_names":    groupNames("The LDAP groups whose members are granted Mission Control Viewer access to the event broker service. At least one of the three group lists must contain a group."),
			"read_write_group_names":   groupNames("The LDAP groups whose members are granted Mission Control Editor access to the event broker service. At least one of the three group lists must contain a group."),
			"mesh_manager_group_names": groupNames("The LDAP groups whose members are granted Mission Control Manager access to the event broker service. The event broker service must support the Mission Control Manager role. At least one of the three group lists must contain a group."),
			"management_profile_name": schema.StringAttribute{
				MarkdownDescription: "The name of the LDAP authentication profile used to authenticate management users. " +
					"It is `management` for configurations made with this resource. An event broker service configured through an older API " +
					"may report another name; this resource can then read the configuration but not change or delete it.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *LdapManagementAccessResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig := req.ProviderData.(shared.ProviderConfig)
	r.APIClient = apiclient.NewRetryableClient(providerConfig.APIClient, 3, 10)
	r.APIPollingInterval = providerConfig.APIPollingInterval
}

func (r *LdapManagementAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LdapManagementAccessModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := data.ServiceId.ValueString()
	body := missioncontrol.CreateLdapManagementAccessJSONRequestBody{
		Enabled:               data.Enabled.ValueBool(),
		GroupAttributeName:    data.GroupAttributeName.ValueString(),
		ReadOnlyGroupNames:    toGroupNames(ctx, data.ReadOnlyGroupNames, &resp.Diagnostics),
		ReadWriteGroupNames:   toGroupNames(ctx, data.ReadWriteGroupNames, &resp.Diagnostics),
		MeshManagerGroupNames: toGroupNames(ctx, data.MeshManagerGroupNames, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.APIClient.CreateLdapManagementAccessWithResponse(ctx, serviceId, body)
	if err != nil {
		resp.Diagnostics.AddError(
			errCallingAPI,
			"Could not create LDAP management access, unexpected error: "+err.Error(),
		)
		return
	}

	// The API's "already configured" message names its PATCH and DELETE APIs, which mean nothing in Terraform.
	if apiResp.StatusCode() == http.StatusConflict {
		if isAlreadyConfigured(apiResp.JSON409) {
			resp.Diagnostics.AddError(
				"LDAP management access is already configured",
				fmt.Sprintf("Event broker service %s already has LDAP management access configured. "+
					"Import it to manage it with Terraform: terraform import <resource address> %s", serviceId, serviceId),
			)
		} else {
			resp.Diagnostics.AddError("Resource Conflict",
				(&shared.MissionControlErrorResponse{ErrorResponse: apiResp.JSON409}).GetMessage())
		}
		return
	}

	if shared.NewMissionControlErrorResponseAdaptor(http.StatusAccepted, apiResp.Body, apiResp.HTTPResponse,
		apiResp.JSON400, apiResp.JSON401, apiResp.JSON403, apiResp.JSON404, apiResp.JSON503,
	).HandleError(&resp.Diagnostics) {
		return
	}

	r.waitForOperation(ctx, serviceId, apiResp.JSON202, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		// The configuration may exist. Record it so Terraform tracks it (as tainted) instead of the next apply getting a 409.
		data.Id = data.ServiceId
		data.ManagementProfileName = types.StringValue("management")
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	if !r.readAccess(ctx, &data, &resp.Diagnostics) {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError(
				"LDAP management access not found",
				fmt.Sprintf("LDAP management access was created on event broker service %s but could not be read back.", serviceId),
			)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LdapManagementAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LdapManagementAccessModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found := r.readAccess(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		// Removed outside Terraform: drop it from state so it is planned for re-creation
		tflog.Warn(ctx, "LDAP management access not found, removing it from state", map[string]interface{}{
			"serviceId": data.ServiceId.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update sends only the fields that differ from state; the API rejects an empty PATCH.
func (r *LdapManagementAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state LdapManagementAccessModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := plan.ServiceId.ValueString()
	body, changed := updateBody(ctx, plan, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if changed {
		apiResp, err := r.APIClient.UpdateLdapManagementAccessWithResponse(ctx, serviceId, body)
		if err != nil {
			resp.Diagnostics.AddError(
				errCallingAPI,
				"Could not update LDAP management access, unexpected error: "+err.Error(),
			)
			return
		}

		if shared.NewMissionControlErrorResponseAdaptor(http.StatusAccepted, apiResp.Body, apiResp.HTTPResponse,
			apiResp.JSON400, apiResp.JSON401, apiResp.JSON403, apiResp.JSON404, apiResp.JSON503,
		).HandleError(&resp.Diagnostics) {
			return
		}

		r.waitForOperation(ctx, serviceId, apiResp.JSON202, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !r.readAccess(ctx, &plan, &resp.Diagnostics) {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError(
				"LDAP management access not found",
				fmt.Sprintf("LDAP management access on event broker service %s was removed while it was being updated.", serviceId),
			)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LdapManagementAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LdapManagementAccessModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := data.ServiceId.ValueString()
	apiResp, err := r.APIClient.DeleteLdapManagementAccessWithResponse(ctx, serviceId)
	if err != nil {
		resp.Diagnostics.AddError(
			errCallingAPI,
			"Could not delete LDAP management access, unexpected error: "+err.Error(),
		)
		return
	}

	// Already gone (for example removed in the console): nothing left to delete
	if apiResp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "LDAP management access not found on delete, treating it as already deleted", map[string]interface{}{
			"serviceId": serviceId,
		})
		return
	}

	if shared.NewMissionControlErrorResponseAdaptor(http.StatusAccepted, apiResp.Body, apiResp.HTTPResponse,
		apiResp.JSON400, apiResp.JSON401, apiResp.JSON403, nil, apiResp.JSON503,
	).HandleError(&resp.Diagnostics) {
		return
	}

	r.waitForOperation(ctx, serviceId, apiResp.JSON202, &resp.Diagnostics)
}

func (r *LdapManagementAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("service_id"), req, resp)
}

// Returns false without an error when the service has no configuration; callers decide if that is an error.
func (r *LdapManagementAccessResource) readAccess(ctx context.Context, data *LdapManagementAccessModel, diagnostics *diag.Diagnostics) bool {
	serviceId := data.ServiceId.ValueString()

	apiResp, err := r.APIClient.GetLdapManagementAccessWithResponse(ctx, serviceId)
	if err != nil {
		diagnostics.AddError(
			errCallingAPI,
			"Could not read LDAP management access, unexpected error: "+err.Error(),
		)
		return false
	}

	if apiResp.StatusCode() == http.StatusNotFound {
		return false
	}

	if shared.NewMissionControlErrorResponseAdaptor(http.StatusOK, apiResp.Body, apiResp.HTTPResponse,
		nil, apiResp.JSON401, apiResp.JSON403, nil, apiResp.JSON503,
	).HandleError(diagnostics) {
		return false
	}
	if apiResp.JSON200 == nil {
		diagnostics.AddError(
			"Unexpected Solace Cloud API response",
			"Could not read LDAP management access: the response did not contain the configuration. "+string(apiResp.Body),
		)
		return false
	}

	access := apiResp.JSON200.Data
	data.Id = data.ServiceId
	data.Enabled = types.BoolPointerValue(access.Enabled)
	data.GroupAttributeName = types.StringPointerValue(access.GroupAttributeName)
	data.ReadOnlyGroupNames = fromGroupNames(ctx, access.ReadOnlyGroupNames, diagnostics)
	data.ReadWriteGroupNames = fromGroupNames(ctx, access.ReadWriteGroupNames, diagnostics)
	data.MeshManagerGroupNames = fromGroupNames(ctx, access.MeshManagerGroupNames, diagnostics)
	data.ManagementProfileName = types.StringPointerValue(access.ManagementProfileName)
	return !diagnostics.HasError()
}

func (r *LdapManagementAccessResource) waitForOperation(ctx context.Context, serviceId string, operation *missioncontrol.OperationResponse, diagnostics *diag.Diagnostics) {
	if operation == nil || operation.Data.Id == nil {
		diagnostics.AddError(
			"Unexpected Solace Cloud API response",
			"The LDAP management access request was accepted without an operation identifier, so its progress cannot be tracked.",
		)
		return
	}

	operationutils.WaitForOperationToComplete(&operationutils.OperationParams{
		Ctx:             ctx,
		ServiceId:       serviceId,
		OperationId:     *operation.Data.Id,
		R:               r.APIClient,
		PollingInterval: r.APIPollingInterval,
		Timeout:         operationTimeout,
		Diagnostics:     diagnostics,
	})
}

// The spec's example omits validationDetails, so the message prefix is checked as well.
func isAlreadyConfigured(conflict *missioncontrol.ErrorResponse) bool {
	if conflict == nil {
		return false
	}
	if conflict.ValidationDetails != nil && (*conflict.ValidationDetails)["ldapManagementAccess"] != nil {
		return true
	}
	return conflict.Message != nil && strings.HasPrefix(*conflict.Message, "LDAP management access is already configured")
}

func updateBody(ctx context.Context, plan, state LdapManagementAccessModel, diagnostics *diag.Diagnostics) (missioncontrol.UpdateLdapManagementAccessJSONRequestBody, bool) {
	var body missioncontrol.UpdateLdapManagementAccessJSONRequestBody
	changed := false
	if !plan.Enabled.Equal(state.Enabled) {
		body.Enabled = plan.Enabled.ValueBoolPointer()
		changed = true
	}
	if !plan.GroupAttributeName.Equal(state.GroupAttributeName) {
		body.GroupAttributeName = plan.GroupAttributeName.ValueStringPointer()
		changed = true
	}
	if !plan.ReadOnlyGroupNames.Equal(state.ReadOnlyGroupNames) {
		body.ReadOnlyGroupNames = toGroupNames(ctx, plan.ReadOnlyGroupNames, diagnostics)
		changed = true
	}
	if !plan.ReadWriteGroupNames.Equal(state.ReadWriteGroupNames) {
		body.ReadWriteGroupNames = toGroupNames(ctx, plan.ReadWriteGroupNames, diagnostics)
		changed = true
	}
	if !plan.MeshManagerGroupNames.Equal(state.MeshManagerGroupNames) {
		body.MeshManagerGroupNames = toGroupNames(ctx, plan.MeshManagerGroupNames, diagnostics)
		changed = true
	}
	return body, changed
}

func toGroupNames(ctx context.Context, set types.Set, diagnostics *diag.Diagnostics) *[]string {
	names := []string{}
	diagnostics.Append(set.ElementsAs(ctx, &names, false)...)
	return &names
}

func fromGroupNames(ctx context.Context, names *[]string, diagnostics *diag.Diagnostics) types.Set {
	values := []string{}
	if names != nil {
		values = *names
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, values)
	diagnostics.Append(diags...)
	return set
}
