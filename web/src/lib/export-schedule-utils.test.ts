/**
 * Tests for export schedule utilities.
 *
 * Tests form data transformation, formatting functions, and request building
 * for export schedule operations.
 */
import { describe, it, expect } from "vitest";
import type { ExportSchedule } from "../api";
import {
  defaultFormData,
  formatDestination,
  formatFilters,
  formatFileSize,
  scheduleToFormData,
  formDataToScheduleRequest,
} from "./export-schedule-utils";

describe("defaultFormData", () => {
  it("should have correct default values", () => {
    expect(defaultFormData).toEqual({
      name: "",
      enabled: true,
      filterJobKinds: [],
      filterJobStatus: ["completed"],
      filterTags: "",
      filterHasResults: true,
      format: "json",
      destinationType: "local",
      pathTemplate: "{job_id}.{format}",
      cloudProvider: "s3",
      cloudBucket: "",
      cloudRegion: "",
      cloudPath: "",
      localPath: "",
      webhookUrl: "",
      maxRetries: 3,
      baseDelayMs: 1000,
    });
  });
});

describe("formatDestination", () => {
  it("should return 'Unknown' for missing export config", () => {
    const schedule = { id: "1", name: "test" } as ExportSchedule;
    expect(formatDestination(schedule)).toBe("Unknown");
  });

  it("should format local destination with path", () => {
    const schedule = {
      id: "1",
      name: "test",
      export: {
        destination_type: "local",
        local_path: "/data/exports",
      },
    } as ExportSchedule;
    expect(formatDestination(schedule)).toBe("/data/exports");
  });

  it("should format local destination without path", () => {
    const schedule = {
      id: "1",
      name: "test",
      export: {
        destination_type: "local",
      },
    } as ExportSchedule;
    expect(formatDestination(schedule)).toBe("Local file");
  });

  it("should format webhook destination with URL", () => {
    const schedule = {
      id: "1",
      name: "test",
      export: {
        destination_type: "webhook",
        webhook_url: "https://example.com/webhook",
      },
    } as ExportSchedule;
    expect(formatDestination(schedule)).toBe(
      "Webhook: https://example.com/webhook...",
    );
  });

  it("should format webhook destination without URL", () => {
    const schedule = {
      id: "1",
      name: "test",
      export: {
        destination_type: "webhook",
      },
    } as ExportSchedule;
    expect(formatDestination(schedule)).toBe("Webhook");
  });

  it("should format S3 destination with bucket and path", () => {
    const schedule = {
      id: "1",
      name: "test",
      export: {
        destination_type: "s3",
        cloud_config: {
          provider: "s3",
          bucket: "my-bucket",
          path: "exports/",
        },
      },
    } as ExportSchedule;
    expect(formatDestination(schedule)).toBe("S3: my-bucket/exports/");
  });

  it("should format GCS destination with bucket only", () => {
    const schedule = {
      id: "1",
      name: "test",
      export: {
        destination_type: "gcs",
        cloud_config: {
          provider: "gcs",
          bucket: "my-bucket",
        },
      },
    } as ExportSchedule;
    expect(formatDestination(schedule)).toBe("GCS: my-bucket");
  });

  it("should format Azure destination without cloud config", () => {
    const schedule = {
      id: "1",
      name: "test",
      export: {
        destination_type: "azure",
      },
    } as ExportSchedule;
    expect(formatDestination(schedule)).toBe("AZURE");
  });
});

describe("formatFilters", () => {
  it("should return 'All jobs' for undefined filters", () => {
    expect(formatFilters(undefined)).toBe("All jobs");
  });

  it("should return 'All jobs' for empty filters", () => {
    expect(formatFilters({})).toBe("All jobs");
  });

  it("should format job kinds", () => {
    expect(formatFilters({ job_kinds: ["scrape", "crawl"] })).toBe(
      "Kinds: scrape, crawl",
    );
  });

  it("should format job status", () => {
    expect(formatFilters({ job_status: ["completed", "failed"] })).toBe(
      "Status: completed, failed",
    );
  });

  it("should format tags", () => {
    expect(formatFilters({ tags: ["production", "critical"] })).toBe(
      "Tags: production, critical",
    );
  });

  it("should format has_results", () => {
    expect(formatFilters({ has_results: true })).toBe("Has results");
  });

  it("should combine multiple filters", () => {
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
  it("should return '-' for undefined", () => {
    expect(formatFileSize(undefined)).toBe("-");
  });

  it("should format bytes", () => {
    expect(formatFileSize(500)).toBe("500 B");
  });

  it("should format kilobytes", () => {
    expect(formatFileSize(1536)).toBe("1.5 KB");
  });

  it("should format megabytes", () => {
    expect(formatFileSize(2.5 * 1024 * 1024)).toBe("2.5 MB");
  });

  it("should format gigabytes", () => {
    expect(formatFileSize(3.7 * 1024 * 1024 * 1024)).toBe("3.7 GB");
  });
});

describe("scheduleToFormData", () => {
  it("should convert schedule to form data with all fields", () => {
    const schedule: ExportSchedule = {
      id: "123",
      name: "Test Schedule",
      enabled: true,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
      filters: {
        job_kinds: ["scrape", "crawl"],
        job_status: ["completed", "succeeded"],
        tags: ["prod", "critical"],
        has_results: true,
      },
      export: {
        format: "json",
        destination_type: "s3",
        path_template: "exports/{job_id}.{format}",
        cloud_config: {
          provider: "s3",
          bucket: "my-bucket",
          region: "us-east-1",
          path: "exports/",
        },
      },
      retry: {
        max_retries: 5,
        base_delay_ms: 2000,
      },
    };

    const result = scheduleToFormData(schedule);

    expect(result).toEqual({
      name: "Test Schedule",
      enabled: true,
      filterJobKinds: ["scrape", "crawl"],
      filterJobStatus: ["completed", "succeeded"],
      filterTags: "prod\ncritical",
      filterHasResults: true,
      format: "json",
      destinationType: "s3",
      pathTemplate: "exports/{job_id}.{format}",
      cloudProvider: "s3",
      cloudBucket: "my-bucket",
      cloudRegion: "us-east-1",
      cloudPath: "exports/",
      localPath: "",
      webhookUrl: "",
      maxRetries: 5,
      baseDelayMs: 2000,
    });
  });

  it("should handle schedule with minimal fields", () => {
    const schedule: ExportSchedule = {
      id: "456",
      name: "Minimal",
      enabled: false,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
      filters: {},
      export: {
        format: "json",
        destination_type: "local",
      },
    };

    const result = scheduleToFormData(schedule);

    expect(result.name).toBe("Minimal");
    expect(result.enabled).toBe(false);
    expect(result.filterJobKinds).toEqual([]);
    expect(result.filterJobStatus).toEqual(["completed"]);
  });

  it("should convert local destination schedule", () => {
    const schedule: ExportSchedule = {
      id: "789",
      name: "Local Export",
      enabled: true,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
      filters: {},
      export: {
        format: "csv",
        destination_type: "local",
        local_path: "/data/exports",
      },
    };

    const result = scheduleToFormData(schedule);

    expect(result.destinationType).toBe("local");
    expect(result.localPath).toBe("/data/exports");
    expect(result.format).toBe("csv");
  });

  it("should convert webhook destination schedule", () => {
    const schedule: ExportSchedule = {
      id: "abc",
      name: "Webhook Export",
      enabled: true,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
      filters: {},
      export: {
        format: "json",
        destination_type: "webhook",
        webhook_url: "https://api.example.com/webhook",
      },
    };

    const result = scheduleToFormData(schedule);

    expect(result.destinationType).toBe("webhook");
    expect(result.webhookUrl).toBe("https://api.example.com/webhook");
  });
});

describe("formDataToScheduleRequest", () => {
  it("should convert form data to request with all fields", () => {
    const formData = {
      name: "Test Schedule",
      enabled: true,
      filterJobKinds: ["scrape"] as Array<"scrape" | "crawl" | "research">,
      filterJobStatus: ["completed"] as Array<
        "completed" | "failed" | "succeeded" | "canceled"
      >,
      filterTags: "tag1\ntag2",
      filterHasResults: true,
      format: "json" as const,
      destinationType: "s3" as const,
      pathTemplate: "exports/{job_id}.{format}",
      cloudProvider: "s3" as const,
      cloudBucket: "my-bucket",
      cloudRegion: "us-east-1",
      cloudPath: "exports/",
      localPath: "",
      webhookUrl: "",
      maxRetries: 5,
      baseDelayMs: 2000,
    };

    const result = formDataToScheduleRequest(formData);

    expect(result).toEqual({
      name: "Test Schedule",
      enabled: true,
      filters: {
        job_kinds: ["scrape"],
        job_status: ["completed"],
        tags: ["tag1", "tag2"],
        has_results: true,
      },
      export: {
        format: "json",
        destination_type: "s3",
        path_template: "exports/{job_id}.{format}",
        cloud_config: {
          provider: "s3",
          bucket: "my-bucket",
          region: "us-east-1",
          path: "exports/",
        },
      },
      retry: {
        max_retries: 5,
        base_delay_ms: 2000,
      },
    });
  });

  it("should omit retry config when using defaults", () => {
    const formData = {
      name: "Test",
      enabled: true,
      filterJobKinds: [] as Array<"scrape" | "crawl" | "research">,
      filterJobStatus: ["completed"] as Array<
        "completed" | "failed" | "succeeded" | "canceled"
      >,
      filterTags: "",
      filterHasResults: true,
      format: "json" as const,
      destinationType: "local" as const,
      pathTemplate: "{job_id}.{format}",
      cloudProvider: "s3" as const,
      cloudBucket: "",
      cloudRegion: "",
      cloudPath: "",
      localPath: "/data",
      webhookUrl: "",
      maxRetries: 3, // default
      baseDelayMs: 1000, // default
    };

    const result = formDataToScheduleRequest(formData);

    expect(result.retry).toBeUndefined();
  });

  it("should include local path for local destination", () => {
    const formData = {
      name: "Local Export",
      enabled: true,
      filterJobKinds: [] as Array<"scrape" | "crawl" | "research">,
      filterJobStatus: [] as Array<
        "completed" | "failed" | "succeeded" | "canceled"
      >,
      filterTags: "",
      filterHasResults: false,
      format: "csv" as const,
      destinationType: "local" as const,
      pathTemplate: "{job_id}.{format}",
      cloudProvider: "s3" as const,
      cloudBucket: "",
      cloudRegion: "",
      cloudPath: "",
      localPath: "/data/exports",
      webhookUrl: "",
      maxRetries: 3,
      baseDelayMs: 1000,
    };

    const result = formDataToScheduleRequest(formData);

    expect(result.export.destination_type).toBe("local");
    expect(result.export.local_path).toBe("/data/exports");
    expect(result.export.cloud_config).toBeUndefined();
  });

  it("should include webhook URL for webhook destination", () => {
    const formData = {
      name: "Webhook Export",
      enabled: true,
      filterJobKinds: [] as Array<"scrape" | "crawl" | "research">,
      filterJobStatus: [] as Array<
        "completed" | "failed" | "succeeded" | "canceled"
      >,
      filterTags: "",
      filterHasResults: false,
      format: "json" as const,
      destinationType: "webhook" as const,
      pathTemplate: "{job_id}.{format}",
      cloudProvider: "s3" as const,
      cloudBucket: "",
      cloudRegion: "",
      cloudPath: "",
      localPath: "",
      webhookUrl: "https://example.com/webhook",
      maxRetries: 3,
      baseDelayMs: 1000,
    };

    const result = formDataToScheduleRequest(formData);

    expect(result.export.destination_type).toBe("webhook");
    expect(result.export.webhook_url).toBe("https://example.com/webhook");
  });

  it("should handle empty filter arrays by omitting them", () => {
    const formData = {
      name: "Test",
      enabled: true,
      filterJobKinds: [] as Array<"scrape" | "crawl" | "research">,
      filterJobStatus: [] as Array<
        "completed" | "failed" | "succeeded" | "canceled"
      >,
      filterTags: "",
      filterHasResults: false,
      format: "json" as const,
      destinationType: "local" as const,
      pathTemplate: "{job_id}.{format}",
      cloudProvider: "s3" as const,
      cloudBucket: "",
      cloudRegion: "",
      cloudPath: "",
      localPath: "",
      webhookUrl: "",
      maxRetries: 3,
      baseDelayMs: 1000,
    };

    const result = formDataToScheduleRequest(formData);

    expect(result.filters.job_kinds).toBeUndefined();
    expect(result.filters.job_status).toBeUndefined();
    expect(result.filters.tags).toBeUndefined();
    expect(result.filters.has_results).toBe(false);
  });

  it("should trim and filter empty tags", () => {
    const formData = {
      name: "Test",
      enabled: true,
      filterJobKinds: [] as Array<"scrape" | "crawl" | "research">,
      filterJobStatus: [] as Array<
        "completed" | "failed" | "succeeded" | "canceled"
      >,
      filterTags: "  tag1  \n\n  \n  tag2  \n  ",
      filterHasResults: false,
      format: "json" as const,
      destinationType: "local" as const,
      pathTemplate: "{job_id}.{format}",
      cloudProvider: "s3" as const,
      cloudBucket: "",
      cloudRegion: "",
      cloudPath: "",
      localPath: "",
      webhookUrl: "",
      maxRetries: 3,
      baseDelayMs: 1000,
    };

    const result = formDataToScheduleRequest(formData);

    expect(result.filters.tags).toEqual(["tag1", "tag2"]);
  });
});
