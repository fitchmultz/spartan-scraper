/**
 * WebSocket Hook
 *
 * Custom React hook for managing WebSocket connections with auto-reconnect.
 * Handles connection lifecycle, message handling, and graceful degradation.
 *
 * @module useWebSocket
 */

import { useState, useEffect, useRef, useCallback } from "react";

export type WSConnectionState =
  | "connecting"
  | "connected"
  | "disconnected"
  | "reconnecting";

export interface WSMessage {
  type: string;
  timestamp: number;
  payload: unknown;
}

export interface UseWebSocketOptions {
  url: string;
  onMessage?: (msg: WSMessage) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  reconnectInterval?: number;
  maxReconnectInterval?: number;
  heartbeatInterval?: number;
}

export interface UseWebSocketReturn {
  state: WSConnectionState;
  error: Error | null;
  send: (msg: WSMessage) => void;
  connect: () => void;
  disconnect: () => void;
}

const DEFAULT_RECONNECT_INTERVAL = 1000;
const DEFAULT_MAX_RECONNECT_INTERVAL = 30000;
const DEFAULT_HEARTBEAT_INTERVAL = 30000;

export function useWebSocket(options: UseWebSocketOptions): UseWebSocketReturn {
  const [state, setState] = useState<WSConnectionState>("disconnected");
  const [error, setError] = useState<Error | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const heartbeatIntervalRef = useRef<number | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const isManualDisconnectRef = useRef(false);

  const {
    url,
    onMessage,
    onConnect,
    onDisconnect,
    reconnectInterval = DEFAULT_RECONNECT_INTERVAL,
    maxReconnectInterval = DEFAULT_MAX_RECONNECT_INTERVAL,
    heartbeatInterval = DEFAULT_HEARTBEAT_INTERVAL,
  } = options;

  const onMessageRef = useRef(onMessage);
  const onConnectRef = useRef(onConnect);
  const onDisconnectRef = useRef(onDisconnect);

  // Keep latest callbacks without forcing reconnects.
  onMessageRef.current = onMessage;
  onConnectRef.current = onConnect;
  onDisconnectRef.current = onDisconnect;

  const clearReconnectTimeout = useCallback(() => {
    if (reconnectTimeoutRef.current !== null) {
      window.clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

  const clearHeartbeatInterval = useCallback(() => {
    if (heartbeatIntervalRef.current !== null) {
      window.clearInterval(heartbeatIntervalRef.current);
      heartbeatIntervalRef.current = null;
    }
  }, []);

  const send = useCallback((msg: WSMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      try {
        wsRef.current.send(JSON.stringify(msg));
      } catch (err) {
        console.error("Failed to send WebSocket message:", err);
      }
    }
  }, []);

  const disconnect = useCallback(() => {
    isManualDisconnectRef.current = true;
    clearReconnectTimeout();
    clearHeartbeatInterval();

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
      return;
    }

    setState("disconnected");
    onDisconnectRef.current?.();
  }, [clearReconnectTimeout, clearHeartbeatInterval]);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    isManualDisconnectRef.current = false;
    clearReconnectTimeout();

    if (reconnectAttemptsRef.current === 0) {
      setState("connecting");
    } else {
      setState("reconnecting");
    }

    try {
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        reconnectAttemptsRef.current = 0;
        setState("connected");
        setError(null);
        onConnectRef.current?.();

        // Subscribe to job events
        send({ type: "subscribe_jobs", timestamp: Date.now(), payload: null });

        // Start heartbeat
        clearHeartbeatInterval();
        heartbeatIntervalRef.current = window.setInterval(() => {
          send({ type: "pong", timestamp: Date.now(), payload: null });
        }, heartbeatInterval);
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data) as WSMessage;
          onMessageRef.current?.(msg);
        } catch (err) {
          console.error("Failed to parse WebSocket message:", err);
        }
      };

      ws.onerror = (event) => {
        const err = new Error("WebSocket error");
        setError(err);
        console.error("WebSocket error:", event);
      };

      ws.onclose = () => {
        clearHeartbeatInterval();
        wsRef.current = null;
        setState("disconnected");

        if (!isManualDisconnectRef.current) {
          // Auto-reconnect with exponential backoff
          reconnectAttemptsRef.current++;
          const backoff = Math.min(
            reconnectInterval * 2 ** (reconnectAttemptsRef.current - 1),
            maxReconnectInterval,
          );

          reconnectTimeoutRef.current = window.setTimeout(() => {
            connect();
          }, backoff);
        }

        onDisconnectRef.current?.();
      };
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
      setState("disconnected");

      // Retry on connection failure
      if (!isManualDisconnectRef.current) {
        reconnectAttemptsRef.current++;
        const backoff = Math.min(
          reconnectInterval * 2 ** (reconnectAttemptsRef.current - 1),
          maxReconnectInterval,
        );

        reconnectTimeoutRef.current = window.setTimeout(() => {
          connect();
        }, backoff);
      }
    }
  }, [
    url,
    reconnectInterval,
    maxReconnectInterval,
    heartbeatInterval,
    send,
    clearReconnectTimeout,
    clearHeartbeatInterval,
  ]);

  useEffect(() => {
    connect();

    return () => {
      isManualDisconnectRef.current = true;
      clearReconnectTimeout();
      clearHeartbeatInterval();

      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect, clearReconnectTimeout, clearHeartbeatInterval]);

  return { state, error, send, connect, disconnect };
}
