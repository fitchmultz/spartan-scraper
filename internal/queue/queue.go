// Package queue provides a pluggable queue backend abstraction for job distribution.
//
// This package defines:
// - Backend interface for queue implementations (memory, Redis, RabbitMQ)
// - Message type for queue entries with metadata
// - Handler type for message processing
// - Common errors (ErrQueueFull, ErrQueueClosed)
//
// This package does NOT handle:
// - Job execution logic (jobs package handles this)
// - Message serialization (callers must marshal/unmarshal)
// - Distributed locking (distributed package handles this)
//
// Invariants:
// - Publish is non-blocking and returns ErrQueueFull if at capacity
// - Subscribe blocks until context cancellation or Close()
// - Ack/Nack are idempotent (safe to call multiple times)
// - Close is idempotent and graceful (waits for in-flight messages)
package queue

import (
	"context"
	"errors"
)

// Common errors returned by queue backends.
var (
	// ErrQueueFull is returned when the queue is at capacity.
	ErrQueueFull = errors.New("queue is full")
	// ErrQueueClosed is returned when operating on a closed queue.
	ErrQueueClosed = errors.New("queue is closed")
	// ErrMessageNotFound is returned when acknowledging an unknown message.
	ErrMessageNotFound = errors.New("message not found")
)

// Message represents a queue message with delivery semantics.
type Message struct {
	ID      string
	Body    []byte
	Headers map[string]string
}

// Handler processes a queue message.
// The handler should return an error if the message cannot be processed.
// If the handler returns an error, the message may be requeued based on
// the backend's retry policy.
type Handler func(ctx context.Context, msg Message) error

// Backend defines the queue backend interface.
// Supports both in-memory (channel) and external (Redis/RabbitMQ) implementations.
//
// Implementations must be safe for concurrent use.
type Backend interface {
	// Publish enqueues a message.
	// Returns ErrQueueFull if the queue is at capacity.
	// Returns ErrQueueClosed if the queue has been closed.
	Publish(ctx context.Context, msg Message) error

	// Subscribe starts consuming messages.
	// Calls handler for each message received.
	// Blocks until context is cancelled or Close is called.
	// The handler must call Ack or Nack for each message.
	Subscribe(ctx context.Context, handler Handler) error

	// Ack acknowledges successful message processing.
	// The message will be permanently removed from the queue.
	Ack(ctx context.Context, msgID string) error

	// Nack rejects a message.
	// If requeue=true, the message goes back to the queue for retry.
	// If requeue=false, the message is discarded or moved to a dead letter queue.
	Nack(ctx context.Context, msgID string, requeue bool) error

	// QueueDepth returns the current number of messages in queue.
	QueueDepth(ctx context.Context) (int64, error)

	// Close cleanly shuts down the backend.
	// Waits for in-flight messages to complete.
	Close() error
}
