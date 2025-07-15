package shared

import (
	"net/http"
	"terraform-provider-solacecloud/missioncontrol"
	"terraform-provider-solacecloud/platform"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// ErrorResponseProvider is an interface for error responses from different APIs
type ErrorResponseProvider interface {
	GetMessage() string
	GetErrorId() string
}

// MissionControlErrorResponse adapts missioncontrol.ErrorResponse to ErrorResponseProvider
type MissionControlErrorResponse struct {
	ErrorResponse *missioncontrol.ErrorResponse
}

func (m *MissionControlErrorResponse) GetMessage() string {
	if m.ErrorResponse != nil && m.ErrorResponse.Message != nil {
		return *m.ErrorResponse.Message
	}
	return ""
}

func (m *MissionControlErrorResponse) GetErrorId() string {
	if m.ErrorResponse != nil && m.ErrorResponse.ErrorId != nil {
		return *m.ErrorResponse.ErrorId
	}
	return ""
}

// PlatformErrorResponse adapts platform.ErrorResponse to ErrorResponseProvider
type PlatformErrorResponse struct {
	ErrorResponse *platform.ErrorResponse
}

func (p *PlatformErrorResponse) GetMessage() string {
	if p.ErrorResponse != nil && p.ErrorResponse.Message != nil {
		return *p.ErrorResponse.Message
	}
	return ""
}

func (p *PlatformErrorResponse) GetErrorId() string {
	if p.ErrorResponse != nil && p.ErrorResponse.ErrorId != nil {
		return *p.ErrorResponse.ErrorId
	}
	return ""
}

// ErrorResponseAdaptor provides centralized error handling for API responses
type ErrorResponseAdaptor struct {
	ExpectedStatusCode int
	Body               []byte
	HTTPResponse       *http.Response
	JSON400            ErrorResponseProvider
	JSON401            ErrorResponseProvider
	JSON403            ErrorResponseProvider
	JSON404            ErrorResponseProvider
	JSON503            ErrorResponseProvider
}

func (r ErrorResponseAdaptor) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r ErrorResponseAdaptor) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// NewErrorResponseAdaptor creates a new error handler with the expected success status code
func NewErrorResponseAdaptor(
	expectedStatusCode int,
	body []byte,
	httpResponse *http.Response,
	json400 ErrorResponseProvider,
	json401 ErrorResponseProvider,
	json403 ErrorResponseProvider,
	json404 ErrorResponseProvider,
	json503 ErrorResponseProvider,
) *ErrorResponseAdaptor {
	return &ErrorResponseAdaptor{
		ExpectedStatusCode: expectedStatusCode,
		Body:               body,
		HTTPResponse:       httpResponse,
		JSON400:            json400,
		JSON401:            json401,
		JSON403:            json403,
		JSON404:            json404,
		JSON503:            json503,
	}
}

// NewMissionControlErrorResponseAdaptor creates a new error handler for MissionControl API responses
func NewMissionControlErrorResponseAdaptor(
	expectedStatusCode int,
	body []byte,
	httpResponse *http.Response,
	json400 *missioncontrol.ErrorResponse,
	json401 *missioncontrol.ErrorResponse,
	json403 *missioncontrol.ErrorResponse,
	json404 *missioncontrol.ErrorResponse,
	json503 *missioncontrol.ErrorResponse,
) *ErrorResponseAdaptor {
	return &ErrorResponseAdaptor{
		ExpectedStatusCode: expectedStatusCode,
		Body:               body,
		HTTPResponse:       httpResponse,
		JSON400:            &MissionControlErrorResponse{ErrorResponse: json400},
		JSON401:            &MissionControlErrorResponse{ErrorResponse: json401},
		JSON403:            &MissionControlErrorResponse{ErrorResponse: json403},
		JSON404:            &MissionControlErrorResponse{ErrorResponse: json404},
		JSON503:            &MissionControlErrorResponse{ErrorResponse: json503},
	}
}

// NewPlatformErrorResponseAdaptor creates a new error handler for Platform API responses
func NewPlatformErrorResponseAdaptor(
	expectedStatusCode int,
	body []byte,
	httpResponse *http.Response,
	json400 *platform.ErrorResponse,
	json401 *platform.ErrorResponse,
	json403 *platform.ErrorResponse,
	json404 *platform.ErrorResponse,
	json503 *platform.ErrorResponse,
) *ErrorResponseAdaptor {
	return &ErrorResponseAdaptor{
		ExpectedStatusCode: expectedStatusCode,
		Body:               body,
		HTTPResponse:       httpResponse,
		JSON400:            &PlatformErrorResponse{ErrorResponse: json400},
		JSON401:            &PlatformErrorResponse{ErrorResponse: json401},
		JSON403:            &PlatformErrorResponse{ErrorResponse: json403},
		JSON404:            &PlatformErrorResponse{ErrorResponse: json404},
		JSON503:            &PlatformErrorResponse{ErrorResponse: json503},
	}
}

func (h *ErrorResponseAdaptor) HandleError(diagnostics *diag.Diagnostics) bool {
	if h.StatusCode() == h.ExpectedStatusCode {
		return false // No error
	}

	switch h.StatusCode() {
	case http.StatusUnauthorized:
		diagnostics.AddError(
			"Authentication Failed",
			"Received HTTP 401 Unauthorized. Please check your authentication configuration:\n\n"+
				"1. Verify your API token is correct and not expired\n"+
				"2. Set the api_token in your provider configuration or use the SOLACECLOUD_API_TOKEN environment variable\n"+
				"3. Ensure your API token has the necessary permissions to delete services\n"+
				"4. Check that the base_url is correct for your Solace Cloud region\n\n"+
				"Example provider configuration:\n"+
				"provider \"solacecloud\" {\n"+
				"  base_url  = \"https://api.solace.cloud/\"\n"+
				"  api_token = \"your-api-token-here\"\n"+
				"}\n\n"+
				"Or set environment variable: export SOLACECLOUD_API_TOKEN=\"your-api-token-here\"",
		)
	case http.StatusBadRequest:
		if h.JSON400 != nil && h.JSON400.GetMessage() != "" {
			diagnostics.AddError("Bad Request", h.JSON400.GetMessage())
		} else {
			diagnostics.AddError("Bad Request", "Received HTTP 400 Bad Request. "+
				"This usually indicates a malformed request or missing required parameters. "+
				"Check your request body and parameters.")
		}
	case http.StatusForbidden:
		if h.JSON403 != nil && h.JSON403.GetMessage() != "" {
			diagnostics.AddError("Forbidden", h.JSON403.GetMessage())
		} else {
			diagnostics.AddError("Forbidden", "Received HTTP 403 Forbidden. "+
				"This usually indicates that your API token does not have the necessary permissions to perform this action. "+
				"Check your API token's permissions and ensure it has access to the requested resource.")
		}
	case http.StatusNotFound:
		if h.JSON404 != nil && h.JSON404.GetMessage() != "" {
			diagnostics.AddError("Not Found", h.JSON404.GetMessage())
		} else {
			diagnostics.AddError("Not Found", "Received HTTP 404 Not Found. "+
				"This usually indicates that the requested resource does not exist or has already been deleted. "+
				"Check the resource ID and ensure it is correct.")
		}
	case http.StatusServiceUnavailable:
		if h.JSON503 != nil && h.JSON503.GetMessage() != "" {
			diagnostics.AddError("Service Unavailable", h.JSON503.GetMessage())
		} else {
			diagnostics.AddError("Service Unavailable", "Received HTTP 503 Service Unavailable. "+
				"This usually indicates that the Solace Cloud API is temporarily unavailable. "+
				"Try again later.")
		}
	case http.StatusConflict:
		diagnostics.AddError("Resource Conflict", "Received HTTP 409 Conflict. "+
			"This usually indicates that a resource with the same name already exists. "+
			string(h.Body))
	default:
		diagnostics.AddError(string(h.Body), "Unexpected Error")
	}

	return true // Error occurred
}
