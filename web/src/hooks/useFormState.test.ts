/**
 * Purpose: Verify shared job-form browser mode state stays normalized when headless and Playwright toggles change.
 * Responsibilities: Cover direct setter transitions, preset normalization, and the invariant that Playwright cannot remain enabled while headless is off.
 * Scope: useFormState hook behavior only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Headless false always forces usePlaywright false, and preset application should never introduce invalid browser-mode combinations.
 */

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useFormState } from "./useFormState";

describe("useFormState", () => {
  it("keeps Playwright disabled when a setter tries to enable it without headless", () => {
    const { result } = renderHook(() => useFormState());

    act(() => {
      result.current.setUsePlaywright(true);
    });

    expect(result.current.headless).toBe(false);
    expect(result.current.usePlaywright).toBe(false);
  });

  it("clears Playwright when headless is turned off", () => {
    const { result } = renderHook(() => useFormState());

    act(() => {
      result.current.setHeadless(true);
      result.current.setUsePlaywright(true);
      result.current.setHeadless(false);
    });

    expect(result.current.headless).toBe(false);
    expect(result.current.usePlaywright).toBe(false);
  });

  it("normalizes presets that try to keep Playwright on while headless is off", () => {
    const { result } = renderHook(() => useFormState());

    act(() => {
      result.current.applyPreset({ headless: false, usePlaywright: true });
    });

    expect(result.current.headless).toBe(false);
    expect(result.current.usePlaywright).toBe(false);
  });
});
