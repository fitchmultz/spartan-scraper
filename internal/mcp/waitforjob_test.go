package mcp

// Tests for waitForJob function.
// Verifies that waitForJob correctly polls job status, handles terminal states,
// and respects both explicit timeout and context cancellation.
//
// Does NOT handle:
// - Actual job execution or worker pool behavior
// - Store implementation details beyond the Get method
// - Job creation or scheduling
//
// Invariants:
// - waitForJob uses independent timeout timer (time.After)
// - Context cancellation/deadline takes precedence over explicit timeout
// - Returns apperrors.Internal on timeout or job failure
// - Polls every 200ms while waiting
import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

type mockStore struct {
	jobs   map[string]model.Job
	getErr error
}

func (m *mockStore) Get(_ context.Context, id string) (model.Job, error) {
	if m.getErr != nil {
		return model.Job{}, m.getErr
	}

	job, ok := m.jobs[id]
	if !ok {
		return model.Job{}, errors.New("job not found")
	}

	return job, nil
}

func TestWaitForJob_Success(t *testing.T) {
	store := &mockStore{jobs: make(map[string]model.Job)}
	jobID := "test-job-1"
	store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusQueued}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusRunning}
		time.Sleep(100 * time.Millisecond)
		store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusSucceeded}
	}()

	err := waitForJob(ctx, store, jobID, 10)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWaitForJob_Failed(t *testing.T) {
	store := &mockStore{jobs: make(map[string]model.Job)}
	jobID := "test-job-2"
	store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusQueued}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusRunning}
		time.Sleep(100 * time.Millisecond)
		store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusFailed, Error: "connection timeout"}
	}()

	err := waitForJob(ctx, store, jobID, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedErr := "job failed: connection timeout"
	if err.Error() != expectedErr {
		t.Fatalf("expected error message '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestWaitForJob_Timeout(t *testing.T) {
	store := &mockStore{jobs: make(map[string]model.Job)}
	jobID := "test-job-3"
	store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusRunning}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	timeoutSeconds := 1
	start := time.Now()
	err := waitForJob(ctx, store, jobID, timeoutSeconds)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Fatalf("expected internal error kind, got: %v", err)
	}

	expectedMsg := "job timed out after 1 seconds"
	if err.Error() != expectedMsg {
		t.Fatalf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	if elapsed < time.Duration(timeoutSeconds)*time.Second {
		t.Fatalf("expected timeout after at least %d seconds, got %v", timeoutSeconds, elapsed)
	}

	if elapsed > time.Duration(timeoutSeconds+2)*time.Second {
		t.Fatalf("expected timeout soon after %d seconds, got %v", timeoutSeconds, elapsed)
	}
}

func TestWaitForJob_ContextCancellation(t *testing.T) {
	store := &mockStore{jobs: make(map[string]model.Job)}
	jobID := "test-job-4"
	store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusRunning}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := waitForJob(ctx, store, jobID, 30)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got: %v", err)
	}

	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected context cancellation to happen quickly, took %v", elapsed)
	}
}

func TestWaitForJob_ContextDeadlineExceeded(t *testing.T) {
	store := &mockStore{jobs: make(map[string]model.Job)}
	jobID := "test-job-5"
	store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusRunning}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := waitForJob(ctx, store, jobID, 30)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected context deadline exceeded error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded error, got: %v", err)
	}

	if elapsed > 300*time.Millisecond {
		t.Fatalf("expected context deadline to be respected, took %v", elapsed)
	}
}

func TestWaitForJob_StoreError(t *testing.T) {
	store := &mockStore{jobs: make(map[string]model.Job)}
	jobID := "test-job-6"
	store.getErr = errors.New("database connection failed")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := waitForJob(ctx, store, jobID, 10)
	if err == nil {
		t.Fatal("expected store error, got nil")
	}

	expectedErr := "database connection failed"
	if err.Error() != expectedErr {
		t.Fatalf("expected store error '%v', got '%v'", expectedErr, err)
	}
}

func TestWaitForJob_FailedWithoutError(t *testing.T) {
	store := &mockStore{jobs: make(map[string]model.Job)}
	jobID := "test-job-7"
	store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusQueued}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusFailed}
	}()

	err := waitForJob(ctx, store, jobID, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Fatalf("expected internal error kind, got: %v", err)
	}

	expectedMsg := "job failed"
	if err.Error() != expectedMsg {
		t.Fatalf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestWaitForJob_TimeoutZero(t *testing.T) {
	store := &mockStore{jobs: make(map[string]model.Job)}
	jobID := "test-job-8"
	store.jobs[jobID] = model.Job{ID: jobID, Status: model.StatusRunning}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := waitForJob(ctx, store, jobID, 0)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got: %v", err)
	}

	if elapsed < 200*time.Millisecond {
		t.Fatalf("expected waitForJob to wait indefinitely until context cancel, got %v", elapsed)
	}

	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected context cancellation to happen quickly, took %v", elapsed)
	}
}
