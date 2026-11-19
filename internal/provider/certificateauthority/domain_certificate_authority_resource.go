package certificateauthority

import (
	"context"
	"terraform-provider-solacecloud/missioncontrol"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var _ resource.Resource = &DomainCertificateAuthorityResource{}
var _ resource.ResourceWithImportState = &DomainCertificateAuthorityResource{}

func NewDomainCertificateAuthorityResource() resource.Resource {
	return &DomainCertificateAuthorityResource{}
}

type DomainCertificateAuthorityResource struct {
	serviceClient
}

type DomainCertificateAuthorityModel struct {
	CertificateAuthorityModel
}

func (r *DomainCertificateAuthorityResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_certificate_authority"
}

func (r *DomainCertificateAuthorityResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a domain certificate authority of an event broker service: a certificate authority the " +
			"event broker service trusts when it connects to other servers, for example over a bridge or an LDAP connection.",
		Attributes: commonAttributes("domain"),
	}
}

func (r *DomainCertificateAuthorityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	createCertificateAuthority(ctx, &r.serviceClient, r.kind(), req, resp)
}

func (r *DomainCertificateAuthorityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	readState(ctx, r.kind(), req, resp)
}

func (r *DomainCertificateAuthorityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	updateCertificateAuthority(ctx, &r.serviceClient, r.kind(), req, resp)
}

func (r *DomainCertificateAuthorityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	deleteCertificateAuthority(ctx, &r.serviceClient, r.kind(), req, resp)
}

func (r *DomainCertificateAuthorityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importState(ctx, req, resp)
}

func (r *DomainCertificateAuthorityResource) kind() certificateAuthorityKind[DomainCertificateAuthorityModel] {
	return certificateAuthorityKind[DomainCertificateAuthorityModel]{
		name: "domain",
		common: func(data *DomainCertificateAuthorityModel) *CertificateAuthorityModel {
			return &data.CertificateAuthorityModel
		},
		create: func(ctx context.Context, data *DomainCertificateAuthorityModel) (apiResult, error) {
			apiResp, err := r.api.CreateDomainCertificateAuthorityWithResponse(ctx, data.ServiceId.ValueString(), missioncontrol.CreateDomainCertificateAuthorityJSONRequestBody{
				Name:        data.Name.ValueString(),
				CertContent: data.CertContent.ValueStringPointer(),
			})
			if err != nil {
				return apiResult{}, err
			}
			return accepted(apiResp.StatusCode(), apiResp.Body, apiResp.HTTPResponse,
				errorResponses{apiResp.JSON400, apiResp.JSON401, apiResp.JSON403, apiResp.JSON404, apiResp.JSON503}, apiResp.JSON202), nil
		},
		// A domain certificate authority has only its certificate to change, and PUT replaces it.
		update: func(ctx context.Context, plan, _ *DomainCertificateAuthorityModel) (apiResult, error) {
			apiResp, err := r.api.UpdateDomainCertificateAuthorityWithResponse(ctx, plan.ServiceId.ValueString(), plan.Name.ValueString(),
				missioncontrol.UpdateDomainCertificateAuthorityJSONRequestBody{CertContent: plan.CertContent.ValueStringPointer()})
			if err != nil {
				return apiResult{}, err
			}
			return accepted(apiResp.StatusCode(), apiResp.Body, apiResp.HTTPResponse,
				errorResponses{apiResp.JSON400, apiResp.JSON401, apiResp.JSON403, apiResp.JSON404, apiResp.JSON503}, apiResp.JSON202), nil
		},
		get: func(ctx context.Context, data *DomainCertificateAuthorityModel) (apiResult, bool, error) {
			apiResp, err := r.api.GetDomainCertificateAuthorityWithResponse(ctx, data.ServiceId.ValueString(), data.Name.ValueString())
			if err != nil {
				return apiResult{}, false, err
			}
			return fetched(apiResp.StatusCode(), apiResp.Body, apiResp.HTTPResponse, apiResp.JSON401, apiResp.JSON403, apiResp.JSON503),
				apiResp.JSON200 != nil, nil
		},
		remove: func(ctx context.Context, serviceId, name string) (apiResult, error) {
			apiResp, err := r.api.DeleteDomainCertificateAuthorityWithResponse(ctx, serviceId, name)
			if err != nil {
				return apiResult{}, err
			}
			return accepted(apiResp.StatusCode(), apiResp.Body, apiResp.HTTPResponse,
				errorResponses{apiResp.JSON400, apiResp.JSON401, apiResp.JSON403, apiResp.JSON404, apiResp.JSON503}, apiResp.JSON202), nil
		},
	}
}
