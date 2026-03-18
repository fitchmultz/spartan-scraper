/**
 * Purpose: Manage a resilient WebSocket connection for real-time web app updates.
 * Responsibilities: Handle connect/disconnect lifecycle, auto-reconnect, heartbeat messages, and callback delivery for parsed server messages.
 * Scope: Browser-side WebSocket transport only; state derivation and UI responses live in consuming hooks/components.
 * Usage: Call `useWebSocket()` from client-side hooks such as `useAppData` and pass the target URL plus callbacks.
 * Invariants/Assumptions: Only one live socket is owned per hook instance, manual disconnects suppress reconnect loops, and disabled connections should fully tear down without stale event noise.
 */

import { useCallback, useEffect, useRef, useState } from "react";

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
  enabled?: boolean;
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
  const initialConnectTimeoutRef = useRef<number | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const heartbeatIntervalRef = useRef<number | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const isManualDisconnectRef = useRef(false);

  const {
    url,
    enabled = true,
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

  const clearInitialConnectTimeout = useCallback(() => {
    if (initialConnectTimeoutRef.current !== null) {
      window.clearTimeout(initialConnectTimeoutRef.current);
      initialConnectTimeoutRef.current = null;
    }
  }, []);

  const clearHeartbeatInterval = useCallback(() => {
    if (heartbeatIntervalRef.current !== null) {
      window.clearInterval(heartbeatIntervalRef.current);
      heartbeatIntervalRef.current = null;
    }
  }, []);

  const isStaleSocket = useCallback((socket: WebSocket) => {
    return wsRef.current !== socket;
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
    clearInitialConnectTimeout();
    clearReconnectTimeout();
    clearHeartbeatInterval();

    if (wsRef.current) {
      const ws = wsRef.current;
      wsRef.current = null;
      ws.close();
      return;
    }

    setState("disconnected");
    onDisconnectRef.current?.();
  }, [
    clearInitialConnectTimeout,
    clearReconnectTimeout,
    clearHeartbeatInterval,
  ]);

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
        if (isStaleSocket(ws) || isManualDisconnectRef.current) {
          ws.close();
          return;
        }

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
        if (isStaleSocket(ws) || isManualDisconnectRef.current) {
          return;
        }

        const err = new Error("WebSocket error");
        setError(err);
        console.error("WebSocket error:", event);
      };

      ws.onclose = () => {
        if (isStaleSocket(ws)) {
          return;
        }

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
    isStaleSocket,
    clearReconnectTimeout,
    clearHeartbeatInterval,
  ]);

  useEffect(() => {
    if (!enabled) {
      disconnect();
      return;
    }

    initialConnectTimeoutRef.current = window.setTimeout(() => {
      initialConnectTimeoutRef.current = null;
      connect();
    }, 0);

    return () => {
      isManualDisconnectRef.current = true;
      clearInitialConnectTimeout();
      clearReconnectTimeout();
      clearHeartbeatInterval();

      if (wsRef.current) {
        const ws = wsRef.current;
        wsRef.current = null;
        ws.close();
      }
    };
  }, [
    connect,
    disconnect,
    enabled,
    clearInitialConnectTimeout,
    clearReconnectTimeout,
    clearHeartbeatInterval,
  ]);

  return { state, error, send, connect, disconnect };
}
