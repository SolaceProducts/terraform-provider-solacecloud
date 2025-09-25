package apiclient

import (
	"context"
	"errors"
	"io"
	mc "terraform-provider-solacecloud/missioncontrol"
	"time"
)

type RetryableClientWithResponses struct {
	api         CRUDClientWithResponses
	maxRetries  int
	waitSeconds int
}

type ApiError interface {
	StatusCode() int
}

type CRUDClientWithResponses interface {
	CreateServiceWithResponse(ctx context.Context, body mc.CreateServiceJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.CreateServiceResponse, error)
	GetServiceWithResponse(ctx context.Context, id string, params *mc.GetServiceParams, reqEditors ...mc.RequestEditorFn) (*mc.GetServiceResponse, error)
	DeleteServiceWithResponse(ctx context.Context, id string, reqEditors ...mc.RequestEditorFn) (*mc.DeleteServiceResponse, error)
	UpdateServiceWithBodyWithResponse(ctx context.Context, id string, contentType string, body io.Reader, reqEditors ...mc.RequestEditorFn) (*mc.UpdateServiceResponse, error)
	UpdateMessageSpoolWithBodyWithResponse(ctx context.Context, serviceId string, contentType string, body io.Reader, reqEditors ...mc.RequestEditorFn) (*mc.UpdateMessageSpoolResponse, error)
	GetServiceOperationWithResponse(ctx context.Context, serviceId string, operationId string, params *mc.GetServiceOperationParams, reqEditors ...mc.RequestEditorFn) (*mc.GetServiceOperationResponse, error)
	
	// Connection Endpoint operations
	CreateConnectionEndpointWithResponse(ctx context.Context, serviceId string, body mc.CreateConnectionEndpointJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.CreateConnectionEndpointResponse, error)
	GetConnectionEndpointsWithResponse(ctx context.Context, serviceId string, reqEditors ...mc.RequestEditorFn) (*mc.GetConnectionEndpointsResponse, error)
	GetConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...mc.RequestEditorFn) (*mc.GetConnectionEndpointResponse, error)
	UpdateConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, body mc.UpdateConnectionEndpointJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.UpdateConnectionEndpointResponse, error)
	DeleteConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...mc.RequestEditorFn) (*mc.DeleteConnectionEndpointResponse, error)

	// DNS Name operations
	GetConnectionEndpointDnsNamesWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...mc.RequestEditorFn) (*mc.GetConnectionEndpointDnsNamesResponse, error)
	CreateConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, body mc.CreateConnectionEndpointDnsNameJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.CreateConnectionEndpointDnsNameResponse, error)
	DeleteConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, dnsName string, reqEditors ...mc.RequestEditorFn) (*mc.DeleteConnectionEndpointDnsNameResponse, error)
	MoveConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, dnsName string, body mc.MoveConnectionEndpointDnsNameJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.MoveConnectionEndpointDnsNameResponse, error)

}


func NewRetryableClient(api CRUDClientWithResponses, maxRetries, waitSeconds int) *RetryableClientWithResponses {
	return &RetryableClientWithResponses{api, maxRetries, waitSeconds}
}

func isRetryableError(err error) bool {
	var apiErr ApiError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode() >= 500 && apiErr.StatusCode() < 600
	}
	return false
}

func retry[T any](ctx context.Context, fn func() (T, error), maxRetries, waitSec int) (T, error) {
	var lastErr error
	var zero T
	for attempt := 0; attempt < maxRetries; attempt++ {
		var result T
		result, err := fn()
		if err == nil || !isRetryableError(err) {
			return result, err
		}
		lastErr = err
		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(time.Duration(waitSec) * time.Second):
				// retry
			}
		}
	}
	return zero, lastErr
}

func (w *RetryableClientWithResponses) CreateServiceWithResponse(ctx context.Context, body mc.CreateServiceJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.CreateServiceResponse, error) {
	return retry(ctx, func() (*mc.CreateServiceResponse, error) {
		return w.api.CreateServiceWithResponse(ctx, body)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) GetServiceWithResponse(ctx context.Context, id string, params *mc.GetServiceParams, reqEditors ...mc.RequestEditorFn) (*mc.GetServiceResponse, error) {
	return retry(ctx, func() (*mc.GetServiceResponse, error) {
		return w.api.GetServiceWithResponse(ctx, id, params, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) DeleteServiceWithResponse(ctx context.Context, id string, reqEditors ...mc.RequestEditorFn) (*mc.DeleteServiceResponse, error) {
	return retry(ctx, func() (*mc.DeleteServiceResponse, error) {
		return w.api.DeleteServiceWithResponse(ctx, id, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) UpdateServiceWithBodyWithResponse(ctx context.Context, id string, contentType string, body io.Reader, reqEditors ...mc.RequestEditorFn) (*mc.UpdateServiceResponse, error) {
	return retry(ctx, func() (*mc.UpdateServiceResponse, error) {
		return w.api.UpdateServiceWithBodyWithResponse(ctx, id, contentType, body, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) UpdateMessageSpoolWithBodyWithResponse(ctx context.Context, serviceId string, contentType string, body io.Reader, reqEditors ...mc.RequestEditorFn) (*mc.UpdateMessageSpoolResponse, error) {
	return retry(ctx, func() (*mc.UpdateMessageSpoolResponse, error) {
		return w.api.UpdateMessageSpoolWithBodyWithResponse(ctx, serviceId, contentType, body, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) GetServiceOperationWithResponse(ctx context.Context, serviceId string, operationId string, params *mc.GetServiceOperationParams, reqEditors ...mc.RequestEditorFn) (*mc.GetServiceOperationResponse, error) {
	return retry(ctx, func() (*mc.GetServiceOperationResponse, error) {
		return w.api.GetServiceOperationWithResponse(ctx, serviceId, operationId, params, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

// GetServiceOperationWithResponseAndParams is the new version that accepts params for DNS operations
func (w *RetryableClientWithResponses) GetServiceOperationWithResponseAndParams(ctx context.Context, serviceId string, operationId string, params *mc.GetServiceOperationParams, reqEditors ...mc.RequestEditorFn) (*mc.GetServiceOperationResponse, error) {
	return retry(ctx, func() (*mc.GetServiceOperationResponse, error) {
		return w.api.GetServiceOperationWithResponse(ctx, serviceId, operationId, params, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

// Connection Endpoint operations
func (w *RetryableClientWithResponses) CreateConnectionEndpointWithResponse(ctx context.Context, serviceId string, body mc.CreateConnectionEndpointJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.CreateConnectionEndpointResponse, error) {
	return retry(ctx, func() (*mc.CreateConnectionEndpointResponse, error) {
		return w.api.CreateConnectionEndpointWithResponse(ctx, serviceId, body, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) GetConnectionEndpointsWithResponse(ctx context.Context, serviceId string, reqEditors ...mc.RequestEditorFn) (*mc.GetConnectionEndpointsResponse, error) {
	return retry(ctx, func() (*mc.GetConnectionEndpointsResponse, error) {
		return w.api.GetConnectionEndpointsWithResponse(ctx, serviceId, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) GetConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...mc.RequestEditorFn) (*mc.GetConnectionEndpointResponse, error) {
	return retry(ctx, func() (*mc.GetConnectionEndpointResponse, error) {
		return w.api.GetConnectionEndpointWithResponse(ctx, serviceId, connectionEndpointId, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) DeleteConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...mc.RequestEditorFn) (*mc.DeleteConnectionEndpointResponse, error) {
	return retry(ctx, func() (*mc.DeleteConnectionEndpointResponse, error) {
		return w.api.DeleteConnectionEndpointWithResponse(ctx, serviceId, connectionEndpointId, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) UpdateConnectionEndpointWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, body mc.UpdateConnectionEndpointJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.UpdateConnectionEndpointResponse, error) {
	return retry(ctx, func() (*mc.UpdateConnectionEndpointResponse, error) {
		return w.api.UpdateConnectionEndpointWithResponse(ctx, serviceId, connectionEndpointId, body, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

// DNS Name wrapper methods
func (w *RetryableClientWithResponses) GetConnectionEndpointDnsNamesWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, reqEditors ...mc.RequestEditorFn) (*mc.GetConnectionEndpointDnsNamesResponse, error) {
	return retry(ctx, func() (*mc.GetConnectionEndpointDnsNamesResponse, error) {
		return w.api.GetConnectionEndpointDnsNamesWithResponse(ctx, serviceId, connectionEndpointId, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) CreateConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, body mc.CreateConnectionEndpointDnsNameJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.CreateConnectionEndpointDnsNameResponse, error) {
	return retry(ctx, func() (*mc.CreateConnectionEndpointDnsNameResponse, error) {
		return w.api.CreateConnectionEndpointDnsNameWithResponse(ctx, serviceId, connectionEndpointId, body, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) DeleteConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, dnsName string, reqEditors ...mc.RequestEditorFn) (*mc.DeleteConnectionEndpointDnsNameResponse, error) {
	return retry(ctx, func() (*mc.DeleteConnectionEndpointDnsNameResponse, error) {
		return w.api.DeleteConnectionEndpointDnsNameWithResponse(ctx, serviceId, connectionEndpointId, dnsName, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}

func (w *RetryableClientWithResponses) MoveConnectionEndpointDnsNameWithResponse(ctx context.Context, serviceId string, connectionEndpointId string, dnsName string, body mc.MoveConnectionEndpointDnsNameJSONRequestBody, reqEditors ...mc.RequestEditorFn) (*mc.MoveConnectionEndpointDnsNameResponse, error) {
	return retry(ctx, func() (*mc.MoveConnectionEndpointDnsNameResponse, error) {
		return w.api.MoveConnectionEndpointDnsNameWithResponse(ctx, serviceId, connectionEndpointId, dnsName, body, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}
