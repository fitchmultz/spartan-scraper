// Package jobs provides regression coverage for manager event publication.
//
// Purpose:
// - Prove job-event subscriber bookkeeping is no longer blocked by side-effect handlers.
//
// Responsibilities:
// - Verify publishEvent snapshots subscribers before invoking blocking side effects.
//
// Scope:
// - Event publication locking behavior only.
//
// Usage:
// - Run with `go test ./internal/jobs`.
//
// Invariants/Assumptions:
// - Slow export-trigger side effects must not keep the subscriber lock held.
package jobs

import (
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

type blockingExportTrigger struct {
	started chan struct{}
	release chan struct{}
}

func (b *blockingExportTrigger) HandleJobEvent(event JobEvent) {
	close(b.started)
	<-b.release
}

func TestPublishEventReleasesSubscriberLockBeforeSideEffects(t *testing.T) {
	trigger := &blockingExportTrigger{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	manager := &Manager{exportTrigger: trigger}
	subscriber := make(chan JobEvent)
	manager.SubscribeToEvents(subscriber)

	done := make(chan struct{})
	go func() {
		manager.publishEvent(JobEvent{Job: model.Job{ID: "job-1"}})
		close(done)
	}()

	select {
	case <-trigger.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blocking export trigger")
	}

	unsubscribed := make(chan struct{})
	go func() {
		manager.UnsubscribeFromEvents(subscriber)
		close(unsubscribed)
	}()

	select {
	case <-unsubscribed:
	case <-time.After(time.Second):
		close(trigger.release)
		<-done
		t.Fatal("unsubscribe blocked behind side-effect handler")
	}

	close(trigger.release)
	<-done
}
