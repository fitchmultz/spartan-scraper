// Package api provides WebSocket support for real-time job updates.
//
// This module handles:
// - WebSocket connection lifecycle (upgrade, read, write, close)
// - Client connection management (register, unregister, broadcast)
// - Message framing and JSON serialization
// - Heartbeat/ping-pong for connection health
//
// Invariants:
// - All Hub methods are goroutine-safe
// - Client send channel has fixed buffer; slow clients are disconnected
// - Hub must be started with Run() before use
package api

import (
	"encoding/json"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// WSMessageType represents the type of WebSocket message.
type WSMessageType string

const (
	// Server -> Client message types
	WSMessageJobCreated    WSMessageType = "job_created"
	WSMessageJobStarted    WSMessageType = "job_started"
	WSMessageStatusChanged WSMessageType = "job_status_changed"
	WSMessageJobCompleted  WSMessageType = "job_completed"
	WSMessageManagerStatus WSMessageType = "manager_status"
	WSMessageMetrics       WSMessageType = "metrics"
	WSMessagePing          WSMessageType = "ping"

	// Client -> Server message types
	WSMessagePong            WSMessageType = "pong"
	WSMessageSubscribeJobs   WSMessageType = "subscribe_jobs"
	WSMessageUnsubscribeJobs WSMessageType = "unsubscribe_jobs"
)

// WSMessage is the envelope for all WebSocket messages.
type WSMessage struct {
	Type      WSMessageType `json:"type"`
	Timestamp int64         `json:"timestamp"`
	Payload   any           `json:"payload"`
}

// JobEventPayload for job-related messages.
type JobEventPayload struct {
	JobID     string       `json:"jobId"`
	Kind      string       `json:"kind"`
	Status    string       `json:"status"`
	Error     string       `json:"error,omitempty"`
	Progress  *JobProgress `json:"progress,omitempty"`
	UpdatedAt int64        `json:"updatedAt"`
}

// JobProgress for crawl jobs.
type JobProgress struct {
	PagesCrawled int `json:"pagesCrawled"`
	PagesTotal   int `json:"pagesTotal,omitempty"`
	DepthCurrent int `json:"depthCurrent,omitempty"`
	DepthMax     int `json:"depthMax,omitempty"`
}

// ManagerStatusPayload for manager status updates.
type ManagerStatusPayload struct {
	QueuedJobs int `json:"queuedJobs"`
	ActiveJobs int `json:"activeJobs"`
}

// Hub manages WebSocket client connections and broadcasts messages.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan WSMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	quit       chan struct{}
	quitOnce   sync.Once
	done       sync.WaitGroup
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan WSMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		quit:       make(chan struct{}),
	}
}

// Run starts the hub's event loop. Must be called in a goroutine.
func (h *Hub) Run() {
	slog.Info("starting websocket hub")
	h.done.Add(1)
	defer h.done.Done()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			slog.Debug("websocket client registered", "client", client.conn.RemoteAddr())

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			slog.Debug("websocket client unregistered", "client", client.conn.RemoteAddr())

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mu.RUnlock()

			for _, client := range clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full, close and remove
					slog.Warn("websocket client send buffer full, disconnecting", "client", client.conn.RemoteAddr())
					client.conn.Close()
				}
			}

		case <-h.quit:
			slog.Info("stopping websocket hub")
			return
		}
	}
}

// Stop signals the hub to shut down gracefully.
// This closes the quit channel which causes Run() to exit.
// Safe to call multiple times (idempotent).
func (h *Hub) Stop() {
	h.quitOnce.Do(func() {
		close(h.quit)
	})
}

// Wait blocks until the hub's Run() goroutine has exited.
// Call this after Stop() to ensure graceful shutdown.
func (h *Hub) Wait() {
	h.done.Wait()
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg WSMessage) {
	select {
	case h.broadcast <- msg:
	default:
		// Broadcast channel full, log and drop
		slog.Warn("websocket broadcast channel full, dropping message", "type", msg.Type)
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Client represents a single WebSocket connection.
type Client struct {
	hub        *Hub
	conn       net.Conn
	send       chan WSMessage
	subscribed bool
	mu         sync.Mutex
}

// NewClient creates a new WebSocket client.
func (h *Hub) NewClient(conn net.Conn) *Client {
	return &Client{
		hub:  h,
		conn: conn,
		send: make(chan WSMessage, 16),
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// Channel closed
				c.writeClose()
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				slog.Error("failed to marshal websocket message", "error", err)
				continue
			}

			if err := wsutil.WriteServerText(c.conn, data); err != nil {
				slog.Debug("failed to write websocket message", "error", err)
				return
			}

		case <-ticker.C:
			// Send ping
			if err := c.writePing(); err != nil {
				slog.Debug("failed to write ping", "error", err)
				return
			}
		}
	}
}

// readPump pumps messages from the WebSocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		data, err := wsutil.ReadClientText(c.conn)
		if err != nil {
			// Connection closed or error
			return
		}

		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			slog.Debug("failed to unmarshal client message", "error", err)
			continue
		}

		switch msg.Type {
		case WSMessagePong:
			// Client responded to ping, connection healthy
			slog.Debug("received pong from client")
		case WSMessageSubscribeJobs:
			c.mu.Lock()
			c.subscribed = true
			c.mu.Unlock()
		case WSMessageUnsubscribeJobs:
			c.mu.Lock()
			c.subscribed = false
			c.mu.Unlock()
		default:
			slog.Debug("unknown message type from client", "type", msg.Type)
		}
	}
}

// writePing sends a WebSocket ping frame.
func (c *Client) writePing() error {
	return wsutil.WriteServerMessage(c.conn, ws.OpPing, nil)
}

// writeClose sends a WebSocket close frame.
func (c *Client) writeClose() {
	wsutil.WriteServerMessage(c.conn, ws.OpClose, nil)
}

// IsSubscribed returns whether the client is subscribed to job events.
func (c *Client) IsSubscribed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.subscribed
}

// JobEventType represents the type of job lifecycle event.
type JobEventType string

const (
	JobEventCreated   JobEventType = "created"
	JobEventStarted   JobEventType = "started"
	JobEventStatus    JobEventType = "status"
	JobEventCompleted JobEventType = "completed"
)

// JobEvent represents a job lifecycle event for WebSocket broadcasting.
type JobEvent struct {
	Type       JobEventType
	Job        model.Job
	PrevStatus model.Status
}

// BroadcastJobEvent converts a JobEvent to a WSMessage and broadcasts it.
func (h *Hub) BroadcastJobEvent(event JobEvent) {
	var msgType WSMessageType
	switch event.Type {
	case JobEventCreated:
		msgType = WSMessageJobCreated
	case JobEventStarted:
		msgType = WSMessageJobStarted
	case JobEventStatus:
		msgType = WSMessageStatusChanged
	case JobEventCompleted:
		msgType = WSMessageJobCompleted
	default:
		return
	}

	payload := JobEventPayload{
		JobID:     event.Job.ID,
		Kind:      string(event.Job.Kind),
		Status:    string(event.Job.Status),
		Error:     event.Job.Error,
		UpdatedAt: event.Job.UpdatedAt.UnixMilli(),
	}

	msg := WSMessage{
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}

	h.Broadcast(msg)
}

// BroadcastManagerStatus broadcasts the current manager status to all clients.
func (h *Hub) BroadcastManagerStatus(queuedJobs, activeJobs int) {
	msg := WSMessage{
		Type:      WSMessageManagerStatus,
		Timestamp: time.Now().UnixMilli(),
		Payload: ManagerStatusPayload{
			QueuedJobs: queuedJobs,
			ActiveJobs: activeJobs,
		},
	}
	h.Broadcast(msg)
}

// BroadcastMetrics broadcasts the current metrics snapshot to all clients.
func (h *Hub) BroadcastMetrics(snapshot MetricsSnapshot) {
	msg := WSMessage{
		Type:      WSMessageMetrics,
		Timestamp: time.Now().UnixMilli(),
		Payload:   snapshot,
	}
	h.Broadcast(msg)
}
