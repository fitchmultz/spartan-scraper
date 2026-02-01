// Package api provides WebSocket support for real-time job updates.
//
// This test file verifies WebSocket hub functionality including
// graceful shutdown behavior.
package api

import (
	"testing"
	"time"
)

func TestHubGracefulShutdown(t *testing.T) {
	hub := NewHub()

	// Start hub in a goroutine
	go hub.Run()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// Stop the hub
	hub.Stop()

	// Wait for shutdown with timeout
	done := make(chan struct{})
	go func() {
		hub.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - hub shut down cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("hub did not shut down within timeout")
	}
}

func TestHubBroadcastAfterStop(t *testing.T) {
	hub := NewHub()

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Stop the hub
	hub.Stop()
	hub.Wait()

	// Broadcast after stop should not panic (it may drop the message)
	hub.Broadcast(WSMessage{
		Type:      WSMessagePing,
		Timestamp: time.Now().UnixMilli(),
		Payload:   nil,
	})
}

func TestHubClientRegistrationAfterStop(t *testing.T) {
	hub := NewHub()

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Stop the hub
	hub.Stop()
	hub.Wait()

	// Client count should be 0 after shutdown
	if count := hub.ClientCount(); count != 0 {
		t.Errorf("expected 0 clients after shutdown, got %d", count)
	}
}

func TestHubMultipleStopCalls(t *testing.T) {
	hub := NewHub()

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// First stop should succeed
	hub.Stop()
	hub.Wait()

	// Second stop should be safe (idempotent) and not panic
	// This is expected behavior - Stop() uses sync.Once
	hub.Stop()
	hub.Stop()
}
