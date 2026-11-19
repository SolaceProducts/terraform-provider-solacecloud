package shared

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestLockServiceQueuesWritesPerService(t *testing.T) {
	var diags diag.Diagnostics
	unlockA, ok := LockService(context.Background(), "svc-lock-a", &diags)
	if !ok {
		t.Fatal("first lock on svc-lock-a should succeed")
	}

	if unlockB, ok := LockService(context.Background(), "svc-lock-b", &diags); !ok {
		t.Fatal("another service must not wait for svc-lock-a")
	} else {
		unlockB()
	}

	acquired := make(chan func())
	go func() {
		unlock, _ := LockService(context.Background(), "svc-lock-a", &diag.Diagnostics{})
		acquired <- unlock
	}()
	select {
	case <-acquired:
		t.Fatal("a second write to svc-lock-a ran before the first finished")
	case <-time.After(50 * time.Millisecond):
	}
	unlockA()
	select {
	case unlock := <-acquired:
		unlock()
	case <-time.After(time.Second):
		t.Fatal("the second write to svc-lock-a never ran after the first finished")
	}
}

func TestLockServiceStopsWaitingWhenCancelled(t *testing.T) {
	unlock, _ := LockService(context.Background(), "svc-lock-cancel", &diag.Diagnostics{})
	defer unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var diags diag.Diagnostics
	if _, ok := LockService(ctx, "svc-lock-cancel", &diags); ok || !diags.HasError() {
		t.Fatalf("a cancelled wait should fail with an error, got ok=%v diags=%v", ok, diags)
	}
}
