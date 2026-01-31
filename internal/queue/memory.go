// Package queue provides a pluggable queue backend abstraction for job distribution.
//
// This file implements the in-memory queue backend using Go channels.
// It provides backward compatibility for single-node deployments.
package queue

import (
	"context"
	"sync"
)

// Memory is an in-memory queue backend using buffered channels.
// This is the default backend for single-node deployments.
//
// Memory maintains backward compatibility with the original
// channel-based job queue implementation.
type Memory struct {
	ch       chan Message
	closed   bool
	closeMu  sync.RWMutex
	wg       sync.WaitGroup
	ackMu    sync.Mutex
	unacked  map[string]Message
	maxDepth int64
}

// NewMemory creates a new in-memory queue backend with the specified capacity.
func NewMemory(capacity int) *Memory {
	return &Memory{
		ch:       make(chan Message, capacity),
		unacked:  make(map[string]Message),
		maxDepth: int64(capacity),
	}
}

// Publish enqueues a message.
// Returns ErrQueueFull if the queue is at capacity.
// Returns ErrQueueClosed if the queue has been closed.
func (m *Memory) Publish(ctx context.Context, msg Message) error {
	m.closeMu.RLock()
	if m.closed {
		m.closeMu.RUnlock()
		return ErrQueueClosed
	}
	m.closeMu.RUnlock()

	select {
	case m.ch <- msg:
		return nil
	default:
		return ErrQueueFull
	}
}

// Subscribe starts consuming messages.
// Calls handler for each message received.
// Blocks until context is cancelled or Close is called.
func (m *Memory) Subscribe(ctx context.Context, handler Handler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-m.ch:
			if !ok {
				return nil
			}

			// Track unacked message
			m.ackMu.Lock()
			m.unacked[msg.ID] = msg
			m.ackMu.Unlock()

			// Process the message
			m.wg.Add(1)
			err := handler(ctx, msg)
			m.wg.Done()

			// If handler failed and we want to requeue, put it back
			if err != nil {
				// In memory backend, we don't auto-requeue
				// The handler should decide whether to retry
			}

			// Remove from unacked (auto-ack for memory backend)
			m.ackMu.Lock()
			delete(m.unacked, msg.ID)
			m.ackMu.Unlock()
		}
	}
}

// Ack acknowledges successful message processing.
// For the memory backend, this is a no-op since messages are auto-acked.
func (m *Memory) Ack(ctx context.Context, msgID string) error {
	m.ackMu.Lock()
	delete(m.unacked, msgID)
	m.ackMu.Unlock()
	return nil
}

// Nack rejects a message.
// For the memory backend with requeue=true, the message is re-enqueued.
// With requeue=false, the message is discarded.
func (m *Memory) Nack(ctx context.Context, msgID string, requeue bool) error {
	m.ackMu.Lock()
	msg, ok := m.unacked[msgID]
	if !ok {
		m.ackMu.Unlock()
		return ErrMessageNotFound
	}
	delete(m.unacked, msgID)
	m.ackMu.Unlock()

	if requeue {
		// Try to requeue, but don't block
		select {
		case m.ch <- msg:
		default:
			// Queue is full, message is lost
		}
	}
	return nil
}

// QueueDepth returns the current number of messages in queue.
func (m *Memory) QueueDepth(ctx context.Context) (int64, error) {
	return int64(len(m.ch)), nil
}

// Close cleanly shuts down the backend.
// Waits for in-flight messages to complete.
func (m *Memory) Close() error {
	m.closeMu.Lock()
	if m.closed {
		m.closeMu.Unlock()
		return nil
	}
	m.closed = true
	m.closeMu.Unlock()

	// Wait for in-flight messages
	m.wg.Wait()

	close(m.ch)
	return nil
}
