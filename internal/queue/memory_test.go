// Package queue provides a pluggable queue backend abstraction for job distribution.
//
// This file contains tests for the in-memory queue backend.
package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemory_Publish(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		capacity  int
		msgCount  int
		wantError bool
	}{
		{
			name:      "publish single message",
			capacity:  10,
			msgCount:  1,
			wantError: false,
		},
		{
			name:      "publish multiple messages",
			capacity:  10,
			msgCount:  5,
			wantError: false,
		},
		{
			name:      "publish at capacity",
			capacity:  5,
			msgCount:  5,
			wantError: false,
		},
		{
			name:      "publish beyond capacity",
			capacity:  3,
			msgCount:  5,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewMemory(tt.capacity)
			defer m.Close()

			ctx := context.Background()
			var lastErr error

			for i := 0; i < tt.msgCount; i++ {
				msg := Message{
					ID:   string(rune('a' + i)),
					Body: []byte("message " + string(rune('a'+i))),
				}
				lastErr = m.Publish(ctx, msg)
			}

			if tt.wantError {
				assert.Equal(t, ErrQueueFull, lastErr)
			} else {
				assert.NoError(t, lastErr)
			}
		})
	}
}

func TestMemory_Publish_Closed(t *testing.T) {
	t.Parallel()

	m := NewMemory(10)
	m.Close()

	ctx := context.Background()
	msg := Message{ID: "1", Body: []byte("test")}

	err := m.Publish(ctx, msg)
	assert.Equal(t, ErrQueueClosed, err)
}

func TestMemory_Subscribe(t *testing.T) {
	t.Parallel()

	m := NewMemory(10)
	defer m.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Publish some messages
	for i := 0; i < 3; i++ {
		msg := Message{
			ID:   string(rune('a' + i)),
			Body: []byte("message " + string(rune('a'+i))),
		}
		require.NoError(t, m.Publish(ctx, msg))
	}

	// Subscribe and collect messages
	var received []Message
	var mu sync.Mutex

	done := make(chan struct{})
	go func() {
		_ = m.Subscribe(ctx, func(msgCtx context.Context, msg Message) error {
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()

			if len(received) >= 3 {
				cancel() // Stop after receiving all messages
			}
			return nil
		})
		close(done)
	}()

	select {
	case <-done:
		// Expected when context is cancelled
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for subscription")
	}

	mu.Lock()
	assert.Len(t, received, 3)
	mu.Unlock()
}

func TestMemory_QueueDepth(t *testing.T) {
	t.Parallel()

	m := NewMemory(10)
	defer m.Close()

	ctx := context.Background()

	// Initially empty
	depth, err := m.QueueDepth(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), depth)

	// Add messages
	for i := 0; i < 5; i++ {
		msg := Message{ID: string(rune('a' + i)), Body: []byte("test")}
		require.NoError(t, m.Publish(ctx, msg))
	}

	depth, err = m.QueueDepth(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), depth)
}

func TestMemory_Ack(t *testing.T) {
	t.Parallel()

	m := NewMemory(10)
	defer m.Close()

	// Ack is a no-op for memory backend
	ctx := context.Background()
	err := m.Ack(ctx, "msg-1")
	assert.NoError(t, err)
}

func TestMemory_Nack(t *testing.T) {
	t.Parallel()

	m := NewMemory(10)
	defer m.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Publish a message
	msg := Message{ID: "msg-1", Body: []byte("test")}
	require.NoError(t, m.Publish(ctx, msg))

	// Start a subscriber that will receive the message and track it as unacked
	received := make(chan Message, 1)
	go func() {
		_ = m.Subscribe(ctx, func(msgCtx context.Context, receivedMsg Message) error {
			received <- receivedMsg
			// Don't ack - leave it in unacked state
			<-msgCtx.Done() // Wait for context cancellation
			return nil
		})
	}()

	// Wait for message to be received
	select {
	case receivedMsg := <-received:
		assert.Equal(t, "msg-1", receivedMsg.ID)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	// Now Nack should work since the message is in unacked state
	err := m.Nack(ctx, "msg-1", true)
	assert.NoError(t, err)
}

func TestMemory_Nack_NotFound(t *testing.T) {
	t.Parallel()

	m := NewMemory(10)
	defer m.Close()

	ctx := context.Background()

	// Nack unknown message
	err := m.Nack(ctx, "unknown", true)
	assert.Equal(t, ErrMessageNotFound, err)
}

func TestMemory_Close(t *testing.T) {
	t.Parallel()

	m := NewMemory(10)

	// Close should be idempotent
	assert.NoError(t, m.Close())
	assert.NoError(t, m.Close())
}

func TestMemory_ConcurrentPublishSubscribe(t *testing.T) {
	t.Parallel()

	m := NewMemory(100)
	defer m.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const numMessages = 100
	const numPublishers = 5

	// Start subscribers
	var receivedCount int
	var mu sync.Mutex
	done := make(chan struct{})

	for i := 0; i < 3; i++ {
		go func() {
			_ = m.Subscribe(ctx, func(msgCtx context.Context, msg Message) error {
				mu.Lock()
				receivedCount++
				if receivedCount >= numMessages*numPublishers {
					mu.Unlock()
					cancel()
					return nil
				}
				mu.Unlock()
				return nil
			})
		}()
	}

	// Start publishers
	var wg sync.WaitGroup
	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				msg := Message{
					ID:   string(rune('a'+id)) + string(rune('0'+j%10)),
					Body: []byte("test message"),
				}
				// Retry on full queue
				for {
					err := m.Publish(ctx, msg)
					if err == nil {
						break
					}
					if err == ErrQueueFull {
						time.Sleep(time.Millisecond)
						continue
					}
					t.Errorf("unexpected error: %v", err)
					return
				}
			}
		}(i)
	}

	// Wait for publishers
	wg.Wait()

	// Wait for all messages to be received or timeout
	select {
	case <-done:
	case <-ctx.Done():
	}

	mu.Lock()
	assert.GreaterOrEqual(t, receivedCount, numMessages*numPublishers)
	mu.Unlock()
}
