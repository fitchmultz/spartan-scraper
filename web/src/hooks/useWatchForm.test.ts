/**
 * Purpose: Verify watch-form state supports both normal create/edit flows and promotion-seeded drafts.
 * Responsibilities: Cover default state, promotion draft initialization, validation, and create submission.
 * Scope: Hook behavior only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Promotion drafts must reset editing state, preserve explicit review fields, and submit through the same watch input conversion path as manual creates.
 */

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Watch } from "../api";
import { useWatchForm } from "./useWatchForm";

describe("useWatchForm", () => {
  it("initializes with monitoring defaults", () => {
    const { result } = renderHook(() => useWatchForm());

    expect(result.current.formData.intervalSeconds).toBe(3600);
    expect(result.current.formData.diffFormat).toBe("unified");
    expect(result.current.editingId).toBeNull();
    expect(result.current.formError).toBeNull();
  });

  it("loads a promotion draft as a fresh create form", () => {
    const { result } = renderHook(() => useWatchForm());

    act(() => {
      result.current.initFormFromDraft({
        ...result.current.formData,
        url: "https://example.com/pricing",
        headless: true,
        usePlaywright: true,
        screenshotEnabled: true,
      });
    });

    expect(result.current.editingId).toBeNull();
    expect(result.current.formData.url).toBe("https://example.com/pricing");
    expect(result.current.formData.usePlaywright).toBe(true);
    expect(result.current.formData.screenshotEnabled).toBe(true);
  });

  it("loads existing watches for editing", () => {
    const { result } = renderHook(() => useWatchForm());

    const watch: Watch = {
      id: "watch-1",
      url: "https://example.com/pricing",
      intervalSeconds: 900,
      enabled: true,
      createdAt: "2026-03-20T10:00:00Z",
      changeCount: 0,
      diffFormat: "unified",
      notifyOnChange: false,
      headless: false,
      usePlaywright: false,
      status: "active",
    };

    act(() => {
      result.current.initFormForEdit(watch);
    });

    expect(result.current.editingId).toBe("watch-1");
    expect(result.current.formData.intervalSeconds).toBe(900);
  });

  it("submits promotion-seeded watch drafts through onCreate", async () => {
    const { result } = renderHook(() => useWatchForm());

    act(() => {
      result.current.initFormFromDraft({
        ...result.current.formData,
        url: "https://example.com/pricing",
        headless: true,
        usePlaywright: true,
        screenshotEnabled: true,
      });
    });

    const onCreate = vi.fn().mockResolvedValue(undefined);
    const onUpdate = vi.fn().mockResolvedValue(undefined);

    let success = false;
    await act(async () => {
      success = await result.current.submitForm(onCreate, onUpdate);
    });

    expect(success).toBe(true);
    expect(onCreate).toHaveBeenCalledWith(
      expect.objectContaining({
        url: "https://example.com/pricing",
        headless: true,
        usePlaywright: true,
        screenshotEnabled: true,
      }),
    );
    expect(onUpdate).not.toHaveBeenCalled();
  });
});
