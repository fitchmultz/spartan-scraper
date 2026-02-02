/**
 * Tests for useExportScheduleForm hook.
 *
 * Tests form state management, validation, and submission handling
 * for export schedule forms.
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useExportScheduleForm } from "./useExportScheduleForm";
import type { ExportSchedule } from "../api";

describe("useExportScheduleForm", () => {
  describe("initial state", () => {
    it("should initialize with default form data", () => {
      const { result } = renderHook(() => useExportScheduleForm());

      expect(result.current.formData.name).toBe("");
      expect(result.current.formData.enabled).toBe(true);
      expect(result.current.formData.format).toBe("json");
      expect(result.current.formData.destinationType).toBe("local");
      expect(result.current.formError).toBeNull();
      expect(result.current.formSubmitting).toBe(false);
      expect(result.current.editingId).toBeNull();
    });
  });

  describe("setFormField", () => {
    it("should update a single form field", () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test Schedule");
      });

      expect(result.current.formData.name).toBe("Test Schedule");
    });

    it("should update nested form fields", () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("format", "csv");
        result.current.setFormField("destinationType", "s3");
      });

      expect(result.current.formData.format).toBe("csv");
      expect(result.current.formData.destinationType).toBe("s3");
    });
  });

  describe("setFormDataPartial", () => {
    it("should update multiple form fields at once", () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormDataPartial({
          name: "New Name",
          format: "jsonl",
        });
      });

      expect(result.current.formData.name).toBe("New Name");
      expect(result.current.formData.format).toBe("jsonl");
      expect(result.current.formData.enabled).toBe(true); // unchanged
    });
  });

  describe("resetForm", () => {
    it("should reset form to defaults", () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test");
        result.current.setFormField("format", "csv");
      });

      act(() => {
        result.current.resetForm();
      });

      expect(result.current.formData.name).toBe("");
      expect(result.current.formData.format).toBe("json");
      expect(result.current.editingId).toBeNull();
      expect(result.current.formError).toBeNull();
    });
  });

  describe("initFormForEdit", () => {
    it("should populate form from schedule", () => {
      const { result } = renderHook(() => useExportScheduleForm());

      const schedule: ExportSchedule = {
        id: "123",
        name: "Edit Test",
        enabled: false,
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
        filters: {},
        export: {
          format: "csv",
          destination_type: "s3",
        },
      };

      act(() => {
        result.current.initFormForEdit(schedule);
      });

      expect(result.current.formData.name).toBe("Edit Test");
      expect(result.current.formData.enabled).toBe(false);
      expect(result.current.formData.format).toBe("csv");
      expect(result.current.editingId).toBe("123");
    });
  });

  describe("submitForm validation", () => {
    it("should fail validation when name is empty", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(false);
      expect(result.current.formError).toBe("Name is required");
      expect(onCreate).not.toHaveBeenCalled();
    });

    it("should fail validation when local path is empty for local destination", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test");
        result.current.setFormField("destinationType", "local");
        result.current.setFormField("localPath", "");
      });

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(false);
      expect(result.current.formError).toBe(
        "Local path is required for local destination",
      );
    });

    it("should fail validation when webhook URL is empty for webhook destination", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test");
        result.current.setFormField("destinationType", "webhook");
        result.current.setFormField("webhookUrl", "");
      });

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(false);
      expect(result.current.formError).toBe(
        "Webhook URL is required for webhook destination",
      );
    });

    it("should fail validation when webhook URL is invalid", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test");
        result.current.setFormField("destinationType", "webhook");
        result.current.setFormField("webhookUrl", "not-a-valid-url");
      });

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(false);
      expect(result.current.formError).toBe("Webhook URL must be a valid URL");
    });

    it("should fail validation when cloud bucket is empty for cloud destination", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test");
        result.current.setFormField("destinationType", "s3");
        result.current.setFormField("cloudBucket", "");
      });

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(false);
      expect(result.current.formError).toBe(
        "Bucket/container name is required for cloud destinations",
      );
    });

    it("should fail validation when maxRetries is negative", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test");
        result.current.setFormField("destinationType", "local");
        result.current.setFormField("localPath", "/data");
        result.current.setFormField("maxRetries", -1);
      });

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(false);
      expect(result.current.formError).toBe("Max retries must be 0 or greater");
    });

    it("should fail validation when baseDelayMs is negative", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test");
        result.current.setFormField("destinationType", "local");
        result.current.setFormField("localPath", "/data");
        result.current.setFormField("baseDelayMs", -100);
      });

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(false);
      expect(result.current.formError).toBe("Base delay must be 0 or greater");
    });
  });

  describe("submitForm success cases", () => {
    it("should call onCreate for new schedule", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "New Schedule");
        result.current.setFormField("destinationType", "local");
        result.current.setFormField("localPath", "/data");
      });

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(true);
      expect(onCreate).toHaveBeenCalledTimes(1);
      expect(onCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "New Schedule",
          enabled: true,
        }),
      );
      expect(onUpdate).not.toHaveBeenCalled();
    });

    it("should call onUpdate for existing schedule", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      const schedule: ExportSchedule = {
        id: "123",
        name: "Existing",
        enabled: true,
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
        filters: {},
        export: {
          format: "json",
          destination_type: "local",
          local_path: "/data",
        },
      };

      act(() => {
        result.current.initFormForEdit(schedule);
        result.current.setFormField("name", "Updated Name");
      });

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(true);
      expect(onUpdate).toHaveBeenCalledTimes(1);
      expect(onUpdate).toHaveBeenCalledWith(
        "123",
        expect.objectContaining({
          name: "Updated Name",
        }),
      );
      expect(onCreate).not.toHaveBeenCalled();
    });

    it("should reset form after successful submission", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test");
        result.current.setFormField("localPath", "/data");
      });

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      await act(async () => {
        await result.current.submitForm(onCreate, onUpdate);
      });

      expect(result.current.formData.name).toBe("");
      expect(result.current.editingId).toBeNull();
    });

    it("should handle webhook destination with valid URL", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Webhook Schedule");
        result.current.setFormField("destinationType", "webhook");
        result.current.setFormField(
          "webhookUrl",
          "https://example.com/webhook",
        );
      });

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(true);
      expect(onCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          export: expect.objectContaining({
            destination_type: "webhook",
            webhook_url: "https://example.com/webhook",
          }),
        }),
      );
    });

    it("should handle cloud destination with bucket", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Cloud Schedule");
        result.current.setFormField("destinationType", "gcs");
        result.current.setFormField("cloudProvider", "gcs");
        result.current.setFormField("cloudBucket", "my-bucket");
        result.current.setFormField("cloudRegion", "us-central1");
      });

      const onCreate = vi.fn().mockResolvedValue(undefined);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(true);
      expect(onCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          export: expect.objectContaining({
            destination_type: "gcs",
            cloud_config: expect.objectContaining({
              provider: "gcs",
              bucket: "my-bucket",
              region: "us-central1",
            }),
          }),
        }),
      );
    });
  });

  describe("submitForm error handling", () => {
    it("should set error message on create failure", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test");
        result.current.setFormField("localPath", "/data");
      });

      const error = new Error("API Error");
      const onCreate = vi.fn().mockRejectedValue(error);
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(false);
      expect(result.current.formError).toBe("API Error");
      expect(result.current.formSubmitting).toBe(false);
    });

    it("should handle non-Error exceptions", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test");
        result.current.setFormField("localPath", "/data");
      });

      const onCreate = vi.fn().mockRejectedValue("String error");
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      let success: boolean | undefined;

      await act(async () => {
        success = await result.current.submitForm(onCreate, onUpdate);
      });

      expect(success).toBe(false);
      expect(result.current.formError).toBe("Failed to save export schedule");
    });

    it("should set submitting state during submission", async () => {
      const { result } = renderHook(() => useExportScheduleForm());

      act(() => {
        result.current.setFormField("name", "Test");
        result.current.setFormField("localPath", "/data");
      });

      let resolveFn: (() => void) | undefined;
      const onCreate = vi.fn().mockImplementation(
        () =>
          new Promise<void>((resolve) => {
            resolveFn = resolve;
          }),
      );
      const onUpdate = vi.fn().mockResolvedValue(undefined);

      act(() => {
        result.current.submitForm(onCreate, onUpdate);
      });

      expect(result.current.formSubmitting).toBe(true);

      await act(async () => {
        resolveFn?.();
      });

      expect(result.current.formSubmitting).toBe(false);
    });
  });
});
