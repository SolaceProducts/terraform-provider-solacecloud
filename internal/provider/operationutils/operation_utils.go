package operationutils

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"net/http"
	"terraform-provider-solacecloud/internal/shared"
	mc "terraform-provider-solacecloud/missioncontrol"
	"time"
)

type ApiClient interface {
	GetServiceOperationWithResponse(ctx context.Context, serviceId string, operationId string, params *mc.GetServiceOperationParams, reqEditors ...mc.RequestEditorFn) (*mc.GetServiceOperationResponse, error)
}
type OperationParams struct {
	Ctx             context.Context
	ServiceId       string
	OperationId     string
	R               ApiClient
	PollingInterval int
	Timeout         time.Duration
	Diagnostics     *diag.Diagnostics
}

func WaitForOperationToComplete(p *OperationParams) {
	diagnostics := p.Diagnostics
	timeout := time.NewTimer(p.Timeout)
	defer timeout.Stop()
	for {
		select {
		case <-timeout.C:
			diagnostics.AddError(
				"Service operation timeout",
				fmt.Sprintf("Service operation timeout. OperationID: %s, ServiceID: %s", p.OperationId, p.ServiceId),
			)
			return
		default:
			apiClientOperationResp, err := p.R.GetServiceOperationWithResponse(p.Ctx, p.ServiceId, p.OperationId, nil)
			if err != nil {
				diagnostics.AddError(
					"Error calling Solace Cloud API",
					"Could not get service operation status, unexpected error: "+err.Error(),
				)
				return
			}

			errHandler := shared.NewMissionControlErrorResponseAdaptor(
				http.StatusOK,
				apiClientOperationResp.Body,
				apiClientOperationResp.HTTPResponse,
				nil,
				apiClientOperationResp.JSON401,
				apiClientOperationResp.JSON403,
				apiClientOperationResp.JSON404,
				apiClientOperationResp.JSON503,
			)

			if errHandler.HandleError(diagnostics) {
				return
			}

			operationStatus := *apiClientOperationResp.JSON200.Data.Status
			if operationStatus == mc.OperationStatusFAILED {
				errMessage := apiClientOperationResp.JSON200.Data.Error.Message
				diagnostics.AddError(
					"The operation failed.",
					*errMessage)
				return
			}

			if operationStatus == mc.OperationStatusSUCCEEDED {
				tflog.Info(p.Ctx, "The operation succeeded.")
				return
			}

			tflog.Info(p.Ctx, fmt.Sprintf("I am waiting for the operation to complete. Current status: %s", operationStatus))
			time.Sleep(time.Duration(p.PollingInterval) * time.Second)
		}
	}

}
