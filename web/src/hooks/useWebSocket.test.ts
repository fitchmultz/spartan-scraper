/**
 * Tests for useWebSocket hook.
 *
 * @module useWebSocket.test
 */

import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
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
  onerror: ((event: Event) => void) | null = null;
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

  triggerError() {
    this.onerror?.(new Event("error"));
  }
}

describe("useWebSocket", () => {
  beforeEach(() => {
    vi.stubGlobal("WebSocket", MockWebSocket);
    MockWebSocket.reset();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
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

    act(() => {
      vi.runAllTimers();
    });

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

    act(() => {
      vi.runAllTimers();
    });

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

  it("ignores stale socket errors after manual disconnect", () => {
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});

    const { unmount } = renderHook(() =>
      useWebSocket({
        url: "ws://localhost:8741/v1/ws",
      }),
    );

    act(() => {
      vi.runAllTimers();
    });

    const ws = MockWebSocket.latest();

    unmount();

    act(() => {
      ws.triggerError();
    });

    expect(consoleError).not.toHaveBeenCalled();
  });

  it("cancels the initial connect during immediate unmount", () => {
    const { unmount } = renderHook(() =>
      useWebSocket({
        url: "ws://localhost:8741/v1/ws",
      }),
    );

    unmount();

    act(() => {
      vi.runAllTimers();
    });

    expect(MockWebSocket.instances).toHaveLength(0);
  });
});
