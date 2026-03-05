/**
 * Tests for useWebSocket hook.
 *
 * @module useWebSocket.test
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useWebSocket } from "./useWebSocket";

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;

  static reset() {
    MockWebSocket.instances = [];
  }

  static latest(): MockWebSocket {
    const instance = MockWebSocket.instances.at(-1);
    if (!instance) throw new Error("No WebSocket instances were created");
    return instance;
  }

  readonly url: string;
  readyState = MockWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  send = vi.fn();

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.();
  }

  triggerOpen() {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.();
  }
}

describe("useWebSocket", () => {
  beforeEach(() => {
    vi.stubGlobal("WebSocket", MockWebSocket);
    MockWebSocket.reset();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("does not reconnect when callback props change", () => {
    const { rerender, unmount } = renderHook(
      ({
        onConnect,
        onDisconnect,
      }: {
        onConnect: () => void;
        onDisconnect: () => void;
      }) =>
        useWebSocket({
          url: "ws://localhost:8741/v1/ws",
          onConnect,
          onDisconnect,
        }),
      {
        initialProps: {
          onConnect: vi.fn(),
          onDisconnect: vi.fn(),
        },
      },
    );

    expect(MockWebSocket.instances).toHaveLength(1);

    rerender({
      onConnect: vi.fn(),
      onDisconnect: vi.fn(),
    });

    expect(MockWebSocket.instances).toHaveLength(1);

    unmount();
  });

  it("uses the latest callback refs without reconnecting", () => {
    const initialOnConnect = vi.fn();
    const latestOnConnect = vi.fn();

    const { rerender, unmount } = renderHook(
      ({ onConnect }: { onConnect: () => void }) =>
        useWebSocket({
          url: "ws://localhost:8741/v1/ws",
          onConnect,
        }),
      {
        initialProps: {
          onConnect: initialOnConnect,
        },
      },
    );

    expect(MockWebSocket.instances).toHaveLength(1);

    rerender({ onConnect: latestOnConnect });
    expect(MockWebSocket.instances).toHaveLength(1);

    act(() => {
      MockWebSocket.latest().triggerOpen();
    });

    expect(initialOnConnect).not.toHaveBeenCalled();
    expect(latestOnConnect).toHaveBeenCalledTimes(1);

    unmount();
  });
});
