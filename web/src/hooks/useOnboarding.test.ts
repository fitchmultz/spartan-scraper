/**
 * Purpose: Verify the progressive onboarding state model persists first-run hints, route visits, and tour restart semantics.
 * Responsibilities: Cover first-run hint dismissal, route visitation persistence, immediate tour restart, and full-tour completion bookkeeping.
 * Scope: `useOnboarding` state behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Tests run in jsdom with localStorage stubbed and the onboarding step count sourced from shared onboarding config.
 */

import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ONBOARDING_TOTAL_STEPS } from "../lib/onboarding";
import { useOnboarding } from "./useOnboarding";

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

  it("shows a lightweight first-run hint for first-time users", () => {
    const { result } = renderHook(() => useOnboarding());

    expect(result.current.shouldShowFirstRunHint).toBe(true);
    expect(result.current.isTourActive).toBe(false);

    act(() => {
      result.current.dismissFirstRunHint();
    });

    expect(result.current.shouldShowFirstRunHint).toBe(false);
  });

  it("persists visited routes for contextual onboarding", () => {
    const { result, unmount } = renderHook(() => useOnboarding());

    act(() => {
      result.current.markRouteVisited("templates");
    });

    expect(result.current.hasVisitedRoute("templates")).toBe(true);

    unmount();

    const { result: refreshed } = renderHook(() => useOnboarding());
    expect(refreshed.current.hasVisitedRoute("templates")).toBe(true);
  });

  it("resetOnboarding restarts the full tour immediately", () => {
    const { result } = renderHook(() => useOnboarding());

    act(() => {
      result.current.resetOnboarding();
    });

    expect(result.current.isTourActive).toBe(true);
    expect(result.current.currentStep).toBe(0);
  });

  it("finishOnboarding completes all guided steps", () => {
    const { result } = renderHook(() => useOnboarding());

    act(() => {
      result.current.finishOnboarding();
    });

    expect(result.current.completedSteps).toHaveLength(ONBOARDING_TOTAL_STEPS);
    expect(result.current.shouldShowFirstRunHint).toBe(false);
  });
});
