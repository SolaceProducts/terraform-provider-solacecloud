package certificateauthority

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"terraform-provider-solacecloud/missioncontrol"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ClientCertificateAuthorityResource{}
var _ resource.ResourceWithImportState = &ClientCertificateAuthorityResource{}

func NewClientCertificateAuthorityResource() resource.Resource {
	return &ClientCertificateAuthorityResource{}
}

type ClientCertificateAuthorityResource struct {
	serviceClient
}

type ClientCertificateAuthorityModel struct {
	CertificateAuthorityModel
	RevocationCheckEnabled      types.Bool   `tfsdk:"revocation_check_enabled"`
	OcspOverrideUrl             types.String `tfsdk:"ocsp_override_url"`
	OcspTimeout                 types.Int32  `tfsdk:"ocsp_timeout"`
	OcspNonResponderCertEnabled types.Bool   `tfsdk:"ocsp_non_responder_cert_enabled"`
}

func (r *ClientCertificateAuthorityResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_certificate_authority"
}

func (r *ClientCertificateAuthorityResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := commonAttributes("client")
	maps.Copy(attributes, map[string]schema.Attribute{
		"revocation_check_enabled": schema.BoolAttribute{
			MarkdownDescription: "Indicates whether the event broker service uses OCSP to check the revocation status of client certificates issued by this certificate authority. Defaults to `false`.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"ocsp_override_url": schema.StringAttribute{
			MarkdownDescription: "The URL of the OCSP responder to use instead of the one named in the client certificate, including the scheme, for example `http://ocsp.example.com`. HTTPS is not supported.",
			Optional:            true,
			Validators:          []validator.String{ocspUrl{}},
		},
		"ocsp_timeout": schema.Int32Attribute{
			MarkdownDescription: "The number of seconds to wait for an OCSP response, 1 to 86,400 (24 hours). Defaults to `5`.",
			Optional:            true,
			Computed:            true,
			Default:             int32default.StaticInt32(5),
			Validators:          []validator.Int32{int32validator.Between(1, 86400)},
		},
		"ocsp_non_responder_cert_enabled": schema.BoolAttribute{
			MarkdownDescription: "Indicates whether the event broker service accepts an OCSP response signed by a certificate other than the certificate of the OCSP responder. Defaults to `false`.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
	})
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a client certificate authority of an event broker service: a certificate authority whose " +
			"certificates clients can present to authenticate, and how the event broker service checks their revocation with Online Certificate Status Protocol (OCSP).",
		Attributes: attributes,
	}
}

func (r *ClientCertificateAuthorityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	createCertificateAuthority(ctx, &r.serviceClient, r.kind(), req, resp)
}

func (r *ClientCertificateAuthorityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	readState(ctx, r.kind(), req, resp)
}

func (r *ClientCertificateAuthorityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	updateCertificateAuthority(ctx, &r.serviceClient, r.kind(), req, resp)
}

func (r *ClientCertificateAuthorityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	deleteCertificateAuthority(ctx, &r.serviceClient, r.kind(), req, resp)
}

func (r *ClientCertificateAuthorityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importState(ctx, req, resp)
}

func (r *ClientCertificateAuthorityResource) kind() certificateAuthorityKind[ClientCertificateAuthorityModel] {
	return certificateAuthorityKind[ClientCertificateAuthorityModel]{
		name: "client",
		common: func(data *ClientCertificateAuthorityModel) *CertificateAuthorityModel {
			return &data.CertificateAuthorityModel
		},
		create: func(ctx context.Context, data *ClientCertificateAuthorityModel) (apiResult, error) {
			apiResp, err := r.api.CreateClientCertificateAuthorityWithResponse(ctx, data.ServiceId.ValueString(), missioncontrol.CreateClientCertificateAuthorityJSONRequestBody{
				Name:                        data.Name.ValueString(),
				CertContent:                 data.CertContent.ValueStringPointer(),
				RevocationCheckEnabled:      data.RevocationCheckEnabled.ValueBool(),
				OcspOverrideUrl:             data.OcspOverrideUrl.ValueStringPointer(),
				OcspTimeout:                 data.OcspTimeout.ValueInt32(),
				OcspNonResponderCertEnabled: data.OcspNonResponderCertEnabled.ValueBool(),
			})
			if err != nil {
				return apiResult{}, err
			}
			return accepted(apiResp.StatusCode(), apiResp.Body, apiResp.HTTPResponse,
				errorResponses{apiResp.JSON400, apiResp.JSON401, apiResp.JSON403, apiResp.JSON404, apiResp.JSON503}, apiResp.JSON202), nil
		},
		update: func(ctx context.Context, plan, state *ClientCertificateAuthorityModel) (apiResult, error) {
			apiResp, err := r.api.UpdateClientCertificateAuthorityWithResponse(ctx, plan.ServiceId.ValueString(), plan.Name.ValueString(), updateBody(*plan, *state))
			if err != nil {
				return apiResult{}, err
			}
			return accepted(apiResp.StatusCode(), apiResp.Body, apiResp.HTTPResponse,
				errorResponses{apiResp.JSON400, apiResp.JSON401, apiResp.JSON403, apiResp.JSON404, apiResp.JSON503}, apiResp.JSON202), nil
		},
		get: func(ctx context.Context, data *ClientCertificateAuthorityModel) (apiResult, bool, error) {
			apiResp, err := r.api.GetClientCertificateAuthorityWithResponse(ctx, data.ServiceId.ValueString(), data.Name.ValueString())
			if err != nil {
				return apiResult{}, false, err
			}
			if apiResp.JSON200 != nil {
				readOcsp(data, apiResp.JSON200.Data)
			}
			return fetched(apiResp.StatusCode(), apiResp.Body, apiResp.HTTPResponse, apiResp.JSON401, apiResp.JSON403, apiResp.JSON503),
				apiResp.JSON200 != nil, nil
		},
		remove: func(ctx context.Context, serviceId, name string) (apiResult, error) {
			apiResp, err := r.api.DeleteClientCertificateAuthorityWithResponse(ctx, serviceId, name)
			if err != nil {
				return apiResult{}, err
			}
			return accepted(apiResp.StatusCode(), apiResp.Body, apiResp.HTTPResponse,
				errorResponses{apiResp.JSON400, apiResp.JSON401, apiResp.JSON403, apiResp.JSON404, apiResp.JSON503}, apiResp.JSON202), nil
		},
	}
}

func readOcsp(data *ClientCertificateAuthorityModel, ca missioncontrol.ClientCertificateAuthority) {
	data.RevocationCheckEnabled = types.BoolPointerValue(ca.RevocationCheckEnabled)
	// Config rejects "", so a cleared URL is null whether GET omits it or returns "".
	data.OcspOverrideUrl = types.StringNull()
	if url := ca.OcspOverrideUrl; url != nil && *url != "" {
		data.OcspOverrideUrl = types.StringValue(*url)
	}
	data.OcspTimeout = types.Int32PointerValue(ca.OcspTimeout)
	data.OcspNonResponderCertEnabled = types.BoolPointerValue(ca.OcspNonResponderCertEnabled)
}

func updateBody(plan, state ClientCertificateAuthorityModel) missioncontrol.UpdateClientCertificateAuthorityJSONRequestBody {
	var body missioncontrol.UpdateClientCertificateAuthorityJSONRequestBody
	if !plan.CertContent.Equal(state.CertContent) {
		body.CertContent = plan.CertContent.ValueStringPointer()
	}
	if !plan.RevocationCheckEnabled.Equal(state.RevocationCheckEnabled) {
		body.RevocationCheckEnabled = plan.RevocationCheckEnabled.ValueBoolPointer()
	}
	if !plan.OcspOverrideUrl.Equal(state.OcspOverrideUrl) {
		// The API takes "" as "no override URL", which is how a removed attribute is cleared.
		url := plan.OcspOverrideUrl.ValueString()
		body.OcspOverrideUrl = &url
	}
	if !plan.OcspTimeout.Equal(state.OcspTimeout) {
		body.OcspTimeout = plan.OcspTimeout.ValueInt32Pointer()
	}
	if !plan.OcspNonResponderCertEnabled.Equal(state.OcspNonResponderCertEnabled) {
		body.OcspNonResponderCertEnabled = plan.OcspNonResponderCertEnabled.ValueBoolPointer()
	}
	return body
}

// ocspUrlPattern is the API's pattern for ocspOverrideUrl without its lookahead, which Go's regexp lacks.
var ocspUrlPattern = regexp.MustCompile(`^([^:/]+)://.+$`)

// ocspUrl rejects at plan time what the API rejects on apply: a URL without a scheme, or an https URL.
type ocspUrl struct{}

func (ocspUrl) Description(context.Context) string {
	return "must include a scheme other than https, for example http://ocsp.example.com"
}

func (v ocspUrl) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ocspUrl) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	match := ocspUrlPattern.FindStringSubmatch(req.ConfigValue.ValueString())
	if match == nil || strings.EqualFold(match[1], "https") {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid OCSP override URL",
			fmt.Sprintf("%q %s. HTTPS is not supported.", req.ConfigValue.ValueString(), v.Description(ctx)))
	}
}
