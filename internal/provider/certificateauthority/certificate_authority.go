package certificateauthority

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/internal/provider/operationutils"
	"terraform-provider-solacecloud/internal/shared"
	"terraform-provider-solacecloud/missioncontrol"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	operationTimeout = 5 * time.Minute
	errCallingAPI    = "Error calling Solace Cloud API"
)

var nameValidators = []validator.String{
	stringvalidator.LengthBetween(1, 64),
	stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9._-]+$`), "must contain only letters, digits, periods, underscores, and hyphens"),
	stringvalidator.NoneOf(".", ".."),
}

type serviceClient struct {
	api             *apiclient.RetryableClientWithResponses
	pollingInterval int
}

func (c *serviceClient) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig := req.ProviderData.(shared.ProviderConfig)
	c.api = apiclient.NewRetryableClient(providerConfig.APIClient, 3, 10)
	c.pollingInterval = providerConfig.APIPollingInterval
}

func (c *serviceClient) waitForOperation(ctx context.Context, serviceId string, operation *missioncontrol.OperationResponse, diagnostics *diag.Diagnostics) {
	if operation == nil || operation.Data.Id == nil {
		diagnostics.AddError(
			"Unexpected Solace Cloud API response",
			"The certificate authority request was accepted without an operation identifier, so its progress cannot be tracked.",
		)
		return
	}

	operationutils.WaitForOperationToComplete(&operationutils.OperationParams{
		Ctx:             ctx,
		ServiceId:       serviceId,
		OperationId:     *operation.Data.Id,
		R:               c.api,
		PollingInterval: c.pollingInterval,
		Timeout:         operationTimeout,
		Diagnostics:     diagnostics,
	})
}

func idOf(serviceId, name string) types.String {
	return types.StringValue(serviceId + "/" + name)
}

func importState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := shared.ParseImportId(req.ID, 2)
	if parts == nil || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected <service_id>/<name>, got "+req.ID)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

// The API's 409 text tells REST callers to send a PATCH, so a create conflict gets the Terraform next steps instead.
func addCreateConflict(diagnostics *diag.Diagnostics, kind, serviceId, name string, body []byte) {
	var apiError struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &apiError) != nil || apiError.Message == "" {
		apiError.Message = string(body)
	}
	diagnostics.AddError("Resource Conflict", fmt.Sprintf(
		"Could not create %s certificate authority %s on event broker service %s.\n\n"+
			"If it already exists, import it so that Terraform manages it:\n  terraform import <resource address> %s/%s\n\n"+
			"If another operation is running on the event broker service, wait for it to finish, then apply again.\n\n"+
			"Solace Cloud API: %s",
		kind, name, serviceId, serviceId, name, apiError.Message))
}

// certificateOnly rejects PEM blocks other than certificates at plan time. cert_content is not sensitive,
// so a private key pasted with the certificate would otherwise be printed in the plan before the API rejects it.
// Its errors are not attached to the attribute, because Terraform then quotes the attribute's source lines,
// which would print a key written inline in the configuration.
type certificateOnly struct{}

func (certificateOnly) Description(context.Context) string {
	return "must contain only CERTIFICATE PEM blocks"
}

func (v certificateOnly) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (certificateOnly) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	// pem.Decode skips blocks it cannot parse, so a damaged key would pass the loop below.
	if strings.Contains(value, "PRIVATE KEY-----") {
		resp.Diagnostics.AddError("Not a certificate",
			"cert_content contains a private key. Provide only the certificate of the certificate authority, never a private key.")
		return
	}
	rest := []byte(value)
	for {
		var block *pem.Block
		if block, rest = pem.Decode(rest); block == nil {
			return
		}
		if block.Type != "CERTIFICATE" {
			resp.Diagnostics.AddError("Not a certificate", fmt.Sprintf(
				"cert_content contains a %q PEM block. Provide only the certificate of the certificate authority, never a private key.", block.Type))
			return
		}
	}
}

// CertificateAuthorityModel holds the attributes that both resources have. Each resource's model embeds it.
type CertificateAuthorityModel struct {
	Id          types.String `tfsdk:"id"`
	ServiceId   types.String `tfsdk:"service_id"`
	Name        types.String `tfsdk:"name"`
	CertContent types.String `tfsdk:"cert_content"`
}

// commonAttributes returns the schema of the attributes in CertificateAuthorityModel. kind is "client" or "domain".
func commonAttributes(kind string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The identifier of the certificate authority, `<service_id>/<name>`.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"service_id": schema.StringAttribute{
			MarkdownDescription: "The identifier of the event broker service. Changing it replaces the resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the certificate authority, 1 to 64 letters, digits, periods, underscores, and hyphens, " +
				"unique among the " + kind + " certificate authorities of the event broker service. Changing it replaces the resource.",
			Required:      true,
			Validators:    nameValidators,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"cert_content": schema.StringAttribute{
			MarkdownDescription: "The X.509 certificate of the certificate authority, encoded in privacy enhanced mail (PEM) format. Provide exactly one certificate and no other PEM block, such as a private key. " +
				"The API never returns it, so Terraform keeps the configured value and cannot detect a certificate replaced outside Terraform.",
			Required:   true,
			Validators: []validator.String{certificateOnly{}},
		},
	}
}

// apiResult holds the parts of an API response that the shared flow checks.
type apiResult struct {
	status    int
	body      []byte
	errors    *shared.ErrorResponseAdaptor
	operation *missioncontrol.OperationResponse
}

// errorResponses holds the typed error bodies of a write response.
type errorResponses struct {
	json400, json401, json403, json404, json503 *missioncontrol.ErrorResponse
}

// accepted builds the apiResult of a write, which the API answers with 202 and an operation.
func accepted(status int, body []byte, httpResponse *http.Response, errs errorResponses, operation *missioncontrol.OperationResponse) apiResult {
	return apiResult{status, body, shared.NewMissionControlErrorResponseAdaptor(http.StatusAccepted, body, httpResponse,
		errs.json400, errs.json401, errs.json403, errs.json404, errs.json503), operation}
}

// fetched builds the apiResult of a GET, which the API answers with 200.
func fetched(status int, body []byte, httpResponse *http.Response, json401, json403, json503 *missioncontrol.ErrorResponse) apiResult {
	return apiResult{status, body, shared.NewMissionControlErrorResponseAdaptor(http.StatusOK, body, httpResponse,
		nil, json401, json403, nil, json503), nil}
}

// certificateAuthorityKind is what differs between the client and the domain certificate authority resources.
// M is the resource's model. The CRUD flow both resources share is written once against it, below.
type certificateAuthorityKind[M any] struct {
	name   string
	common func(data *M) *CertificateAuthorityModel
	create func(ctx context.Context, data *M) (apiResult, error)
	update func(ctx context.Context, plan, state *M) (apiResult, error)
	// get reads the certificate authority into data and reports whether the response contained it.
	get    func(ctx context.Context, data *M) (apiResult, bool, error)
	remove func(ctx context.Context, serviceId, name string) (apiResult, error)
}

func createCertificateAuthority[M any](ctx context.Context, c *serviceClient, k certificateAuthorityKind[M], req resource.CreateRequest, resp *resource.CreateResponse) {
	var data M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common := k.common(&data)
	serviceId, name := common.ServiceId.ValueString(), common.Name.ValueString()
	unlock, ok := shared.LockService(ctx, serviceId, &resp.Diagnostics)
	if !ok {
		return
	}
	defer unlock()

	result, err := k.create(ctx, &data)
	if err != nil {
		addCallError(&resp.Diagnostics, "create", k.name, err)
		return
	}
	if result.status == http.StatusConflict {
		addCreateConflict(&resp.Diagnostics, k.name, serviceId, name, result.body)
		return
	}
	if result.errors.HandleError(&resp.Diagnostics) {
		return
	}

	common.Id = idOf(serviceId, name)
	c.waitForOperation(ctx, serviceId, result.operation, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() && !readCertificateAuthority(ctx, k, &data, &resp.Diagnostics) && !resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Certificate authority not found", fmt.Sprintf(
			"The %s certificate authority %s was created on event broker service %s but could not be read back.", k.name, name, serviceId))
	}
	// After the request is accepted the certificate authority may exist, so it is recorded even when an error follows.
	// Terraform then tracks it as tainted and replaces it, instead of the next apply getting a 409.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// readState refreshes the state from the API, or removes the resource from state when the API no longer has it,
// so that the next plan creates it again.
func readState[M any](ctx context.Context, k certificateAuthorityKind[M], req resource.ReadRequest, resp *resource.ReadResponse) {
	var data M
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found := readCertificateAuthority(ctx, k, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		tflog.Warn(ctx, "Certificate authority not found, removing it from state", map[string]interface{}{"id": k.common(&data).Id.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func updateCertificateAuthority[M any](ctx context.Context, c *serviceClient, k certificateAuthorityKind[M], req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common := k.common(&plan)
	serviceId, name := common.ServiceId.ValueString(), common.Name.ValueString()
	unlock, ok := shared.LockService(ctx, serviceId, &resp.Diagnostics)
	if !ok {
		return
	}
	defer unlock()

	result, err := k.update(ctx, &plan, &state)
	if err != nil {
		addCallError(&resp.Diagnostics, "update", k.name, err)
		return
	}
	if result.errors.HandleError(&resp.Diagnostics) {
		return
	}

	c.waitForOperation(ctx, serviceId, result.operation, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !readCertificateAuthority(ctx, k, &plan, &resp.Diagnostics) {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError("Certificate authority not found", fmt.Sprintf(
				"The %s certificate authority %s on event broker service %s was removed while it was being updated.", k.name, name, serviceId))
		}
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// deleteCertificateAuthority treats a 404 as success: the certificate authority is already gone.
func deleteCertificateAuthority[M any](ctx context.Context, c *serviceClient, k certificateAuthorityKind[M], req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data M
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common := k.common(&data)
	serviceId, name := common.ServiceId.ValueString(), common.Name.ValueString()
	unlock, ok := shared.LockService(ctx, serviceId, &resp.Diagnostics)
	if !ok {
		return
	}
	defer unlock()

	result, err := k.remove(ctx, serviceId, name)
	if err != nil {
		addCallError(&resp.Diagnostics, "delete", k.name, err)
		return
	}
	if result.status == http.StatusNotFound {
		tflog.Warn(ctx, "Certificate authority not found on delete, treating it as already deleted", map[string]interface{}{"id": common.Id.ValueString()})
		return
	}
	if result.errors.HandleError(&resp.Diagnostics) {
		return
	}
	c.waitForOperation(ctx, serviceId, result.operation, &resp.Diagnostics)
}

func addCallError(diagnostics *diag.Diagnostics, action, kind string, err error) {
	diagnostics.AddError(errCallingAPI, "Could not "+action+" "+kind+" certificate authority, unexpected error: "+err.Error())
}

// readCertificateAuthority reports false when the certificate authority does not exist or the read failed.
// A failed read also adds an error to diagnostics.
func readCertificateAuthority[M any](ctx context.Context, k certificateAuthorityKind[M], data *M, diagnostics *diag.Diagnostics) (found bool) {
	common := k.common(data)
	serviceId, name := common.ServiceId.ValueString(), common.Name.ValueString()

	result, contained, err := k.get(ctx, data)
	if err != nil {
		addCallError(diagnostics, "read", k.name, err)
		return false
	}
	if result.status == http.StatusNotFound {
		return false
	}
	if result.errors.HandleError(diagnostics) {
		return false
	}
	if !contained {
		diagnostics.AddError("Unexpected Solace Cloud API response",
			"Could not read "+k.name+" certificate authority: the response did not contain the certificate authority. "+string(result.body))
		return false
	}
	common.Id = idOf(serviceId, name)
	return true
}
