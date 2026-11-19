package shared

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// An event broker service runs one job at a time and answers a second write with 409 while the first
// is in progress. The certificate authority, LDAP management access, connection endpoint and DNS name resources
// take the service's one slot before they start a job. The service and DNS name move resources do not take it yet.
var serviceJobs sync.Map

// LockService waits for the service's slot. Call the returned unlock once the job has finished.
func LockService(ctx context.Context, serviceId string, diagnostics *diag.Diagnostics) (unlock func(), ok bool) {
	slot, _ := serviceJobs.LoadOrStore(serviceId, make(chan struct{}, 1))
	jobs := slot.(chan struct{})
	select {
	case jobs <- struct{}{}:
		return func() { <-jobs }, true
	case <-ctx.Done():
		diagnostics.AddError(
			"Cancelled",
			"Cancelled while waiting for an earlier operation on event broker service "+serviceId+" to finish: "+ctx.Err().Error(),
		)
		return nil, false
	}
}
