// Package queue provides a pluggable queue backend abstraction for job distribution.
//
// This file implements the Redis queue backend using Redis Streams (XADD/XREADGROUP).
// It supports consumer groups for distributed worker coordination.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/redis/go-redis/v9"
)

// Redis is a Redis-backed queue backend using Redis Streams.
// It supports consumer groups for distributed worker coordination.
type Redis struct {
	client       *redis.Client
	streamKey    string
	groupName    string
	consumerName string
	maxLen       int64

	// Message tracking for ack/nack
	pendingMu sync.Mutex
	pending   map[string]string // msgID -> stream message ID

	closed  bool
	closeMu sync.RWMutex
	wg      sync.WaitGroup
}

// RedisOptions contains configuration for the Redis backend.
type RedisOptions struct {
	Addr       string
	Password   string
	DB         int
	KeyPrefix  string
	StreamName string
	GroupName  string
	ConsumerID string // Unique identifier for this consumer instance
	MaxLen     int64  // Maximum stream length (approximate), 0 = unlimited
}

// NewRedis creates a new Redis queue backend.
func NewRedis(opts RedisOptions) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to connect to Redis", err)
	}

	streamKey := opts.KeyPrefix + opts.StreamName
	groupName := opts.GroupName
	if groupName == "" {
		groupName = "spartan-workers"
	}
	consumerName := opts.ConsumerID
	if consumerName == "" {
		consumerName = fmt.Sprintf("consumer-%d", time.Now().UnixNano())
	}

	r := &Redis{
		client:       client,
		streamKey:    streamKey,
		groupName:    groupName,
		consumerName: consumerName,
		maxLen:       opts.MaxLen,
		pending:      make(map[string]string),
	}

	// Create consumer group (idempotent)
	if err := r.createGroup(ctx); err != nil {
		return nil, err
	}

	return r, nil
}

// createGroup creates the consumer group if it doesn't exist.
func (r *Redis) createGroup(ctx context.Context) error {
	// Try to create group from beginning ($ means only new messages)
	// We use "0" to read all pending messages from the past
	err := r.client.XGroupCreateMkStream(ctx, r.streamKey, r.groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create consumer group", err)
	}
	return nil
}

// Publish enqueues a message using XADD.
func (r *Redis) Publish(ctx context.Context, msg Message) error {
	r.closeMu.RLock()
	if r.closed {
		r.closeMu.RUnlock()
		return ErrQueueClosed
	}
	r.closeMu.RUnlock()

	// Serialize message
	data, err := json.Marshal(msg)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal message", err)
	}

	// Build XADD args
	args := &redis.XAddArgs{
		Stream: r.streamKey,
		Values: map[string]interface{}{
			"body":    data,
			"msg_id":  msg.ID,
			"headers": serializeHeaders(msg.Headers),
		},
	}

	if r.maxLen > 0 {
		args.MaxLen = r.maxLen
		// Approximate max length is the default behavior for MaxLen
	}

	if err := r.client.XAdd(ctx, args).Err(); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to publish message", err)
	}

	return nil
}

// Subscribe starts consuming messages using XREADGROUP.
func (r *Redis) Subscribe(ctx context.Context, handler Handler) error {
	for {
		r.closeMu.RLock()
		closed := r.closed
		r.closeMu.RUnlock()
		if closed {
			return nil
		}

		// Read from consumer group
		streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    r.groupName,
			Consumer: r.consumerName,
			Streams:  []string{r.streamKey, ">"}, // ">" means only undelivered messages
			Count:    1,
			Block:    5 * time.Second,
		}).Result()

		if err == redis.Nil {
			// Timeout, continue
			continue
		}
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				slog.Error("redis stream read error", "error", err)
				time.Sleep(time.Second)
				continue
			}
		}

		// Process messages
		for _, stream := range streams {
			for _, xmsg := range stream.Messages {
				msg, err := r.parseMessage(xmsg)
				if err != nil {
					slog.Error("failed to parse message", "error", err, "msgID", xmsg.ID)
					// Acknowledge bad messages to avoid reprocessing
					r.client.XAck(ctx, r.streamKey, r.groupName, xmsg.ID)
					continue
				}

				// Track pending message
				r.pendingMu.Lock()
				r.pending[msg.ID] = xmsg.ID
				r.pendingMu.Unlock()

				// Process message
				r.wg.Add(1)
				handlerErr := handler(ctx, msg)
				r.wg.Done()

				if handlerErr != nil {
					// Message failed, don't ack - it will be redelivered
					slog.Error("message handler failed", "msgID", msg.ID, "error", handlerErr)
				} else {
					// Success - ack the message
					if err := r.Ack(ctx, msg.ID); err != nil {
						slog.Error("failed to ack message", "msgID", msg.ID, "error", err)
					}
				}
			}
		}
	}
}

// Ack acknowledges a message using XACK.
func (r *Redis) Ack(ctx context.Context, msgID string) error {
	r.pendingMu.Lock()
	streamMsgID, ok := r.pending[msgID]
	if ok {
		delete(r.pending, msgID)
	}
	r.pendingMu.Unlock()

	if !ok {
		return ErrMessageNotFound
	}

	return r.client.XAck(ctx, r.streamKey, r.groupName, streamMsgID).Err()
}

// Nack rejects a message.
// With requeue=true, the message will be redelivered to another consumer.
// With requeue=false, the message is acknowledged and discarded.
func (r *Redis) Nack(ctx context.Context, msgID string, requeue bool) error {
	r.pendingMu.Lock()
	streamMsgID, ok := r.pending[msgID]
	if ok {
		delete(r.pending, msgID)
	}
	r.pendingMu.Unlock()

	if !ok {
		return ErrMessageNotFound
	}

	if requeue {
		// For Redis, we don't ack the message - it will be redelivered
		// when the consumer group claim timeout expires (default 30 min)
		// or when another consumer calls XAUTOCLAIM
		return nil
	}

	// Acknowledge to discard
	return r.client.XAck(ctx, r.streamKey, r.groupName, streamMsgID).Err()
}

// QueueDepth returns the approximate number of messages in the stream.
func (r *Redis) QueueDepth(ctx context.Context) (int64, error) {
	info, err := r.client.XLen(ctx, r.streamKey).Result()
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to get queue depth", err)
	}
	return info, nil
}

// Close cleanly shuts down the Redis backend.
func (r *Redis) Close() error {
	r.closeMu.Lock()
	if r.closed {
		r.closeMu.Unlock()
		return nil
	}
	r.closed = true
	r.closeMu.Unlock()

	// Wait for in-flight messages
	r.wg.Wait()

	return r.client.Close()
}

// parseMessage converts a Redis stream message to a Message.
func (r *Redis) parseMessage(xmsg redis.XMessage) (Message, error) {
	var msg Message

	// Try to get the serialized body first
	if bodyData, ok := xmsg.Values["body"].(string); ok {
		if err := json.Unmarshal([]byte(bodyData), &msg); err != nil {
			return Message{}, err
		}
	} else {
		// Fallback: parse individual fields
		msg.ID = getString(xmsg.Values, "msg_id")
		if msg.ID == "" {
			// Use Redis message ID as fallback
			msg.ID = xmsg.ID
		}
		if headersStr := getString(xmsg.Values, "headers"); headersStr != "" {
			msg.Headers = deserializeHeaders(headersStr)
		}
	}

	return msg, nil
}

// Helper functions

func serializeHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	data, _ := json.Marshal(headers)
	return string(data)
}

func deserializeHeaders(s string) map[string]string {
	if s == "" {
		return nil
	}
	var headers map[string]string
	json.Unmarshal([]byte(s), &headers)
	return headers
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
