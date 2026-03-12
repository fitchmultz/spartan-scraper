/**
 * Tests for useOnboarding hook.
 *
 * @module useOnboarding.test
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useOnboarding } from "./useOnboarding";
import { ONBOARDING_TOTAL_STEPS } from "../lib/onboarding";

describe("useOnboarding", () => {
  const storage = new Map<string, string>();

  const localStorageMock = {
    getItem: vi.fn((key: string) => storage.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => {
      storage.set(key, value);
    }),
    removeItem: vi.fn((key: string) => {
      storage.delete(key);
    }),
    clear: vi.fn(() => {
      storage.clear();
    }),
  };

  beforeEach(() => {
    storage.clear();
    vi.clearAllMocks();
    vi.stubGlobal("localStorage", localStorageMock);
    window.history.replaceState({}, "", "/");
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows the welcome modal for first-time users", async () => {
    const { result } = renderHook(() => useOnboarding());

    await waitFor(() => {
      expect(result.current.shouldShowWelcome).toBe(true);
    });

    expect(result.current.isTourActive).toBe(false);
  });

  it("hides welcome and starts tour when onboarding starts", async () => {
    const { result } = renderHook(() => useOnboarding());

    await waitFor(() => {
      expect(result.current.shouldShowWelcome).toBe(true);
    });

    act(() => {
      result.current.startOnboarding();
    });

    expect(result.current.shouldShowWelcome).toBe(false);
    expect(result.current.isTourActive).toBe(true);
    expect(result.current.currentStep).toBe(0);
    expect(localStorageMock.setItem).toHaveBeenCalled();
    expect(storage.get("spartan-onboarding")).toContain(
      '"hasStartedOnboarding":true',
    );
  });

  it("persists skip immediately and respects it on refresh", async () => {
    const { result, unmount } = renderHook(() => useOnboarding());

    await waitFor(() => {
      expect(result.current.shouldShowWelcome).toBe(true);
    });

    act(() => {
      result.current.skipOnboarding();
    });

    expect(result.current.shouldShowWelcome).toBe(false);
    expect(result.current.hasSkippedOnboarding).toBe(true);
    expect(localStorageMock.setItem).toHaveBeenCalled();
    expect(storage.get("spartan-onboarding")).toContain(
      '"hasSkippedOnboarding":true',
    );

    unmount();

    const { result: refreshed } = renderHook(() => useOnboarding());

    expect(refreshed.current.shouldShowWelcome).toBe(false);
    expect(refreshed.current.hasSkippedOnboarding).toBe(true);
  });

  it("resetOnboarding returns to welcome state instead of auto-starting", async () => {
    const { result } = renderHook(() => useOnboarding());

    await waitFor(() => {
      expect(result.current.shouldShowWelcome).toBe(true);
    });

    act(() => {
      result.current.startOnboarding();
    });

    expect(result.current.isTourActive).toBe(true);

    act(() => {
      result.current.resetOnboarding();
    });

    expect(result.current.shouldShowWelcome).toBe(true);
    expect(result.current.isTourActive).toBe(false);
    expect(result.current.currentStep).toBe(0);
    expect(storage.get("spartan-onboarding")).toContain(
      '"hasSkippedOnboarding":false',
    );
  });

  it("keeps total steps aligned with onboarding constants", async () => {
    const { result } = renderHook(() => useOnboarding());

    await waitFor(() => {
      expect(result.current.shouldShowWelcome).toBe(true);
    });

    expect(result.current.totalSteps).toBe(ONBOARDING_TOTAL_STEPS);

    act(() => {
      result.current.finishOnboarding();
    });

    expect(result.current.completedSteps).toHaveLength(ONBOARDING_TOTAL_STEPS);
  });
});
