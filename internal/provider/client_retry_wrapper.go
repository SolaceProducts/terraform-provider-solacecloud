package provider

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
	GetServiceOperationWithResponse(ctx context.Context, serviceId string, operationId string, reqEditors ...mc.RequestEditorFn) (*mc.GetServiceOperationResponse, error)
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

func (w *RetryableClientWithResponses) GetServiceOperationWithResponse(ctx context.Context, serviceId string, operationId string, reqEditors ...mc.RequestEditorFn) (*mc.GetServiceOperationResponse, error) {
	return retry(ctx, func() (*mc.GetServiceOperationResponse, error) {
		return w.api.GetServiceOperationWithResponse(ctx, serviceId, operationId, reqEditors...)
	}, w.maxRetries, w.waitSeconds)
}
