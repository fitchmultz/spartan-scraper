/**
 * Purpose: Verify export schedule utils behavior with automated regression coverage.
 * Responsibilities: Define focused test cases, fixtures, and assertions for the module under test.
 * Scope: Automated test coverage only; production logic stays in the adjacent source modules.
 * Usage: Run through the repo test entrypoints or the feature-local test command.
 * Invariants/Assumptions: Tests should describe the current contract clearly and remain deterministic under local CI settings.
 */

import { describe, expect, it } from "vitest";

import type { ExportSchedule } from "../api";
import {
  clearShapeFormData,
  clearTransformFormData,
  defaultFormData,
  formDataToScheduleRequest,
  formDataToTransformConfig,
  formatDestination,
  formatExportShapeSummary,
  formatExportTransformSummary,
  formatFileSize,
  formatFilters,
  formatShapeLabels,
  formDataToShapeConfig,
  hasShapeFormData,
  hasTransformFormData,
  scheduleToFormData,
  shapeConfigToFormData,
  supportsExportShapeFormat,
  transformConfigToFormData,
} from "./export-schedule-utils";

describe("defaultFormData", () => {
  it("reflects the 1.0 defaults", () => {
    expect(defaultFormData).toEqual({
      name: "",
      enabled: true,
      filterJobKinds: [],
      filterJobStatus: ["completed"],
      filterHasResults: true,
      format: "json",
      destinationType: "local",
      pathTemplate: "exports/{kind}/{job_id}.{format}",
      localPath: "exports/{kind}/{job_id}.{format}",
      webhookUrl: "",
      maxRetries: 3,
      baseDelayMs: 1000,
      transformExpression: "",
      transformLanguage: "jmespath",
      shapeTopLevelFields: "",
      shapeNormalizedFields: "",
      shapeEvidenceFields: "",
      shapeSummaryFields: "",
      shapeFieldLabels: "",
      shapeEmptyValue: "",
      shapeMultiValueJoin: "",
      shapeMarkdownTitle: "",
    });
  });
});

describe("formatDestination", () => {
  it("formats local and webhook destinations", () => {
    expect(
      formatDestination({
        id: "1",
        name: "local",
        export: { destination_type: "local", local_path: "/data/export.json" },
      } as ExportSchedule),
    ).toBe("/data/export.json");

    expect(
      formatDestination({
        id: "2",
        name: "webhook",
        export: {
          destination_type: "webhook",
          webhook_url: "https://example.com/webhook",
        },
      } as ExportSchedule),
    ).toBe("Webhook: https://example.com/webhook...");
  });
});

describe("formatFilters", () => {
  it("formats combined filter labels", () => {
    expect(
      formatFilters({
        job_kinds: ["scrape"],
        job_status: ["completed"],
        has_results: true,
      }),
    ).toBe("Kinds: scrape | Status: completed | Has results");
  });
});

describe("formatFileSize", () => {
  it("formats common byte ranges", () => {
    expect(formatFileSize(undefined)).toBe("-");
    expect(formatFileSize(500)).toBe("500 B");
    expect(formatFileSize(1536)).toBe("1.5 KB");
    expect(formatFileSize(2.5 * 1024 * 1024)).toBe("2.5 MB");
  });
});

describe("export shape helpers", () => {
  it("recognizes supported formats", () => {
    expect(supportsExportShapeFormat("md")).toBe(true);
    expect(supportsExportShapeFormat("csv")).toBe(true);
    expect(supportsExportShapeFormat("xlsx")).toBe(true);
    expect(supportsExportShapeFormat("json")).toBe(false);
    expect(supportsExportShapeFormat("jsonl")).toBe(false);
  });

  it("round-trips shape config to and from form fields", () => {
    const formFields = shapeConfigToFormData({
      topLevelFields: ["url", "title"],
      normalizedFields: ["field.price"],
      evidenceFields: ["evidence.url"],
      summaryFields: ["title", "field.price"],
      fieldLabels: {
        "field.price": "Price",
        title: "Page Title",
      },
      formatting: {
        emptyValue: "—",
        multiValueJoin: "; ",
        markdownTitle: "Pricing Export",
      },
    });

    expect(formFields).toEqual({
      shapeTopLevelFields: "url\ntitle",
      shapeNormalizedFields: "field.price",
      shapeEvidenceFields: "evidence.url",
      shapeSummaryFields: "title\nfield.price",
      shapeFieldLabels: "field.price=Price\ntitle=Page Title",
      shapeEmptyValue: "—",
      shapeMultiValueJoin: "; ",
      shapeMarkdownTitle: "Pricing Export",
    });

    expect(
      formDataToShapeConfig({
        ...defaultFormData,
        format: "md",
        ...formFields,
      }),
    ).toEqual({
      topLevelFields: ["url", "title"],
      normalizedFields: ["field.price"],
      evidenceFields: ["evidence.url"],
      summaryFields: ["title", "field.price"],
      fieldLabels: {
        "field.price": "Price",
        title: "Page Title",
      },
      formatting: {
        emptyValue: "—",
        multiValueJoin: ";",
        markdownTitle: "Pricing Export",
      },
    });
  });

  it("returns undefined shape config for unsupported formats", () => {
    expect(
      formDataToShapeConfig({
        ...defaultFormData,
        format: "json",
        shapeTopLevelFields: "url",
      }),
    ).toBeUndefined();
  });

  it("formats shape summaries and label text", () => {
    expect(
      formatShapeLabels({
        title: "Title",
        "field.price": "Price",
      }),
    ).toBe("field.price=Price\ntitle=Title");

    expect(
      formatExportShapeSummary({
        topLevelFields: ["url", "title"],
        fieldLabels: { title: "Title" },
        formatting: { markdownTitle: "Report" },
      }),
    ).toBe("2 fields · 1 label · formatting");
    expect(formatExportShapeSummary(undefined)).toBe("Default");
  });

  it("detects and clears staged shape fields", () => {
    expect(
      hasShapeFormData({
        ...defaultFormData,
        shapeFieldLabels: "title=Title",
      }),
    ).toBe(true);
    expect(hasShapeFormData(defaultFormData)).toBe(false);
    expect(clearShapeFormData()).toEqual({
      shapeTopLevelFields: "",
      shapeNormalizedFields: "",
      shapeEvidenceFields: "",
      shapeSummaryFields: "",
      shapeFieldLabels: "",
      shapeEmptyValue: "",
      shapeMultiValueJoin: "",
      shapeMarkdownTitle: "",
    });
  });
});

describe("export transform helpers", () => {
  it("round-trips transform config to and from form fields", () => {
    expect(
      transformConfigToFormData({
        expression: "{title: title, url: url}",
        language: "jsonata",
      }),
    ).toEqual({
      transformExpression: "{title: title, url: url}",
      transformLanguage: "jsonata",
    });

    expect(
      formDataToTransformConfig({
        ...defaultFormData,
        transformExpression: "{title: title, url: url}",
        transformLanguage: "jmespath",
      }),
    ).toEqual({
      expression: "{title: title, url: url}",
      language: "jmespath",
    });
  });

  it("formats and clears transform summaries", () => {
    expect(
      formatExportTransformSummary({
        expression: "{title: title, url: url}",
        language: "jmespath",
      }),
    ).toContain("jmespath");
    expect(formatExportTransformSummary(undefined)).toBe("Default");
    expect(
      hasTransformFormData({
        ...defaultFormData,
        transformExpression: "{title: title}",
      }),
    ).toBe(true);
    expect(hasTransformFormData(defaultFormData)).toBe(false);
    expect(clearTransformFormData()).toEqual({
      transformExpression: "",
      transformLanguage: "jmespath",
    });
  });
});

describe("scheduleToFormData", () => {
  it("maps local schedules into form state", () => {
    const schedule: ExportSchedule = {
      id: "sched-1",
      name: "Local export",
      enabled: true,
      created_at: "2026-03-10T00:00:00Z",
      updated_at: "2026-03-10T00:00:00Z",
      filters: {
        job_kinds: ["crawl"],
        has_results: true,
      },
      export: {
        format: "csv",
        destination_type: "local",
        local_path: "exports/{job_id}.csv",
        path_template: "exports/{job_id}.csv",
        shape: {
          topLevelFields: ["url", "title"],
          summaryFields: ["title"],
          fieldLabels: { title: "Page Title" },
          formatting: { emptyValue: "—" },
        },
      },
      retry: {
        max_retries: 5,
        base_delay_ms: 2000,
      },
    };

    expect(scheduleToFormData(schedule)).toEqual({
      name: "Local export",
      enabled: true,
      filterJobKinds: ["crawl"],
      filterJobStatus: ["completed"],
      filterHasResults: true,
      format: "csv",
      destinationType: "local",
      pathTemplate: "exports/{job_id}.csv",
      localPath: "exports/{job_id}.csv",
      webhookUrl: "",
      maxRetries: 5,
      baseDelayMs: 2000,
      transformExpression: "",
      transformLanguage: "jmespath",
      shapeTopLevelFields: "url\ntitle",
      shapeNormalizedFields: "",
      shapeEvidenceFields: "",
      shapeSummaryFields: "title",
      shapeFieldLabels: "title=Page Title",
      shapeEmptyValue: "—",
      shapeMultiValueJoin: "",
      shapeMarkdownTitle: "",
    });
  });
});

describe("formDataToScheduleRequest", () => {
  it("builds a local export request", () => {
    expect(
      formDataToScheduleRequest({
        name: "Local export",
        enabled: true,
        filterJobKinds: ["scrape"],
        filterJobStatus: ["completed"],
        filterHasResults: true,
        format: "json",
        destinationType: "local",
        pathTemplate: "exports/{job_id}.json",
        localPath: "exports/{job_id}.json",
        webhookUrl: "",
        maxRetries: 3,
        baseDelayMs: 1000,
        transformExpression: "",
        transformLanguage: "jmespath",
        shapeTopLevelFields: "",
        shapeNormalizedFields: "",
        shapeEvidenceFields: "",
        shapeSummaryFields: "",
        shapeFieldLabels: "",
        shapeEmptyValue: "",
        shapeMultiValueJoin: "",
        shapeMarkdownTitle: "",
      }),
    ).toEqual({
      name: "Local export",
      enabled: true,
      filters: {
        job_kinds: ["scrape"],
        job_status: ["completed"],
        has_results: true,
      },
      export: {
        format: "json",
        destination_type: "local",
        path_template: "exports/{job_id}.json",
        local_path: "exports/{job_id}.json",
      },
    });
  });

  it("builds a webhook export request with retry overrides", () => {
    expect(
      formDataToScheduleRequest({
        name: "Webhook export",
        enabled: true,
        filterJobKinds: [],
        filterJobStatus: [],
        filterHasResults: false,
        format: "jsonl",
        destinationType: "webhook",
        pathTemplate: "{job_id}.{format}",
        localPath: "",
        webhookUrl: "https://example.com/hook",
        maxRetries: 5,
        baseDelayMs: 2500,
        transformExpression: "",
        transformLanguage: "jmespath",
        shapeTopLevelFields: "",
        shapeNormalizedFields: "",
        shapeEvidenceFields: "",
        shapeSummaryFields: "",
        shapeFieldLabels: "",
        shapeEmptyValue: "",
        shapeMultiValueJoin: "",
        shapeMarkdownTitle: "",
      }),
    ).toEqual({
      name: "Webhook export",
      enabled: true,
      filters: {
        has_results: false,
      },
      export: {
        format: "jsonl",
        destination_type: "webhook",
        path_template: "{job_id}.{format}",
        webhook_url: "https://example.com/hook",
      },
      retry: {
        max_retries: 5,
        base_delay_ms: 2500,
      },
    });
  });

  it("includes export transform when configured", () => {
    expect(
      formDataToScheduleRequest({
        ...defaultFormData,
        name: "Projected export",
        format: "csv",
        localPath: "exports/{job_id}.csv",
        pathTemplate: "exports/{job_id}.csv",
        transformExpression: "{title: title, url: url}",
        transformLanguage: "jmespath",
      }),
    ).toEqual({
      name: "Projected export",
      enabled: true,
      filters: {
        job_status: ["completed"],
        has_results: true,
      },
      export: {
        format: "csv",
        destination_type: "local",
        path_template: "exports/{job_id}.csv",
        local_path: "exports/{job_id}.csv",
        transform: {
          expression: "{title: title, url: url}",
          language: "jmespath",
        },
      },
    });
  });

  it("includes export shape for supported formats", () => {
    expect(
      formDataToScheduleRequest({
        ...defaultFormData,
        name: "Shaped markdown export",
        format: "md",
        localPath: "exports/{job_id}.md",
        pathTemplate: "exports/{job_id}.md",
        shapeTopLevelFields: "url\ntitle",
        shapeSummaryFields: "title\nfield.price",
        shapeFieldLabels: "field.price=Price",
        shapeEmptyValue: "—",
        shapeMarkdownTitle: "Pricing Export",
      }),
    ).toEqual({
      name: "Shaped markdown export",
      enabled: true,
      filters: {
        job_status: ["completed"],
        has_results: true,
      },
      export: {
        format: "md",
        destination_type: "local",
        path_template: "exports/{job_id}.md",
        local_path: "exports/{job_id}.md",
        shape: {
          topLevelFields: ["url", "title"],
          summaryFields: ["title", "field.price"],
          fieldLabels: {
            "field.price": "Price",
          },
          formatting: {
            emptyValue: "—",
            markdownTitle: "Pricing Export",
          },
        },
      },
    });
  });
});
