/**
 * useExportScheduleForm.test
 *
 * Purpose:
 * - Verify the reduced export schedule form hook matches the 1.0 local/webhook
 *   destination model.
 *
 * Responsibilities:
 * - Cover basic field updates, validation, edit initialization, and request submission.
 *
 * Scope:
 * - Hook behavior only.
 *
 * Usage:
 * - Run with Vitest.
 *
 * Invariants/Assumptions:
 * - Only `local` and `webhook` destinations are supported.
 */

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { ExportSchedule } from "../api";
import { useExportScheduleForm } from "./useExportScheduleForm";

describe("useExportScheduleForm", () => {
  it("initializes with local defaults", () => {
    const { result } = renderHook(() => useExportScheduleForm());

    expect(result.current.formData.format).toBe("json");
    expect(result.current.formData.destinationType).toBe("local");
    expect(result.current.formError).toBeNull();
    expect(result.current.editingId).toBeNull();
  });

  it("updates individual fields", () => {
    const { result } = renderHook(() => useExportScheduleForm());

    act(() => {
      result.current.setFormField("name", "Export everything");
      result.current.setFormField("format", "csv");
      result.current.setFormField("destinationType", "webhook");
    });

    expect(result.current.formData.name).toBe("Export everything");
    expect(result.current.formData.format).toBe("csv");
    expect(result.current.formData.destinationType).toBe("webhook");
  });

  it("loads form state for editing an existing local schedule", () => {
    const { result } = renderHook(() => useExportScheduleForm());

    const schedule: ExportSchedule = {
      id: "sched-1",
      name: "Local export",
      enabled: false,
      created_at: "2026-03-10T00:00:00Z",
      updated_at: "2026-03-10T00:00:00Z",
      filters: {},
      export: {
        format: "jsonl",
        destination_type: "local",
        local_path: "exports/{job_id}.jsonl",
      },
    };

    act(() => {
      result.current.initFormForEdit(schedule);
    });

    expect(result.current.editingId).toBe("sched-1");
    expect(result.current.formData.name).toBe("Local export");
    expect(result.current.formData.destinationType).toBe("local");
    expect(result.current.formData.localPath).toBe("exports/{job_id}.jsonl");
  });

  it("rejects missing local paths", async () => {
    const { result } = renderHook(() => useExportScheduleForm());

    act(() => {
      result.current.setFormField("name", "Broken local export");
      result.current.setFormField("destinationType", "local");
      result.current.setFormField("localPath", "");
    });

    const onCreate = vi.fn().mockResolvedValue(undefined);
    const onUpdate = vi.fn().mockResolvedValue(undefined);

    let success = false;
    await act(async () => {
      success = await result.current.submitForm(onCreate, onUpdate);
    });

    expect(success).toBe(false);
    expect(result.current.formError).toBe(
      "Local path is required for local destination",
    );
  });

  it("rejects invalid webhook URLs", async () => {
    const { result } = renderHook(() => useExportScheduleForm());

    act(() => {
      result.current.setFormField("name", "Webhook export");
      result.current.setFormField("destinationType", "webhook");
      result.current.setFormField("webhookUrl", "not-a-url");
    });

    const onCreate = vi.fn().mockResolvedValue(undefined);
    const onUpdate = vi.fn().mockResolvedValue(undefined);

    let success = false;
    await act(async () => {
      success = await result.current.submitForm(onCreate, onUpdate);
    });

    expect(success).toBe(false);
    expect(result.current.formError).toBe("Webhook URL must be a valid URL");
  });

  it("submits new local schedules through onCreate", async () => {
    const { result } = renderHook(() => useExportScheduleForm());

    act(() => {
      result.current.setFormField("name", "Local export");
      result.current.setFormField("destinationType", "local");
      result.current.setFormField("localPath", "exports/{job_id}.json");
      result.current.setFormField("format", "json");
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
        name: "Local export",
        export: expect.objectContaining({
          destination_type: "local",
          local_path: "exports/{job_id}.json",
        }),
      }),
    );
    expect(onUpdate).not.toHaveBeenCalled();
  });

  it("submits edited schedules through onUpdate", async () => {
    const { result } = renderHook(() => useExportScheduleForm());

    const schedule: ExportSchedule = {
      id: "sched-2",
      name: "Webhook export",
      enabled: true,
      created_at: "2026-03-10T00:00:00Z",
      updated_at: "2026-03-10T00:00:00Z",
      filters: {},
      export: {
        format: "json",
        destination_type: "webhook",
        webhook_url: "https://example.com/hook",
      },
    };

    act(() => {
      result.current.initFormForEdit(schedule);
      result.current.setFormField("webhookUrl", "https://example.com/next");
    });

    const onCreate = vi.fn().mockResolvedValue(undefined);
    const onUpdate = vi.fn().mockResolvedValue(undefined);

    let success = false;
    await act(async () => {
      success = await result.current.submitForm(onCreate, onUpdate);
    });

    expect(success).toBe(true);
    expect(onUpdate).toHaveBeenCalledWith(
      "sched-2",
      expect.objectContaining({
        export: expect.objectContaining({
          destination_type: "webhook",
          webhook_url: "https://example.com/next",
        }),
      }),
    );
  });

  it("includes export shaping when submitting supported formats", async () => {
    const { result } = renderHook(() => useExportScheduleForm());

    act(() => {
      result.current.setFormField("name", "Shaped export");
      result.current.setFormField("destinationType", "local");
      result.current.setFormField("localPath", "exports/{job_id}.md");
      result.current.setFormField("pathTemplate", "exports/{job_id}.md");
      result.current.setFormField("format", "md");
      result.current.setFormField("shapeTopLevelFields", "url\ntitle");
      result.current.setFormField("shapeSummaryFields", "title\nfield.price");
      result.current.setFormField("shapeFieldLabels", "field.price=Price");
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
        export: expect.objectContaining({
          format: "md",
          shape: {
            topLevelFields: ["url", "title"],
            summaryFields: ["title", "field.price"],
            fieldLabels: {
              "field.price": "Price",
            },
          },
        }),
      }),
    );
  });
});
