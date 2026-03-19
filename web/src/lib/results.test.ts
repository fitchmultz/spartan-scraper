import { afterEach, describe, expect, it, vi } from "vitest";

import { buildResultsUrl, exportResults, loadResults } from "./results";

describe("results utilities", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("builds raw results URLs with pagination only for jsonl", () => {
    expect(buildResultsUrl("job-1", "jsonl", 50, 100)).toBe(
      "/v1/jobs/job-1/results?format=jsonl&limit=50&offset=100",
    );
    expect(buildResultsUrl("job-1", "json", 50, 100)).toBe(
      "/v1/jobs/job-1/results?format=json",
    );
  });

  it("exports text results using the direct export endpoint", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          export: {
            id: "export-1",
            jobId: "job-1",
            trigger: "api",
            status: "succeeded",
            title: "Export ready",
            message: "JSON export completed successfully.",
            exportedAt: "2026-03-18T00:00:00Z",
            retryCount: 0,
            request: { format: "json" },
            artifact: {
              format: "json",
              filename: "job-1.json",
              contentType: "application/json",
              encoding: "utf8",
              content: '{"title":"Example"}',
            },
          },
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    );

    const result = await exportResults("job-1", {
      format: "json",
      transform: { expression: "{title: title}", language: "jmespath" },
    });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/v1/jobs/job-1/export",
      expect.objectContaining({ method: "POST" }),
    );
    expect(result).toEqual({
      outcome: expect.objectContaining({ id: "export-1", status: "succeeded" }),
      content: '{"title":"Example"}',
      filename: "job-1.json",
      contentType: "application/json",
      isBinary: false,
    });
  });

  it("exports binary xlsx results as base64 payloads", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          export: {
            id: "export-2",
            jobId: "job-1",
            trigger: "api",
            status: "succeeded",
            title: "Export ready",
            message: "XLSX export completed successfully.",
            exportedAt: "2026-03-18T00:00:00Z",
            retryCount: 0,
            request: { format: "xlsx" },
            artifact: {
              format: "xlsx",
              filename: "job-1.xlsx",
              contentType:
                "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
              encoding: "base64",
              content: "UEsDBA==",
            },
          },
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    );

    const result = await exportResults("job-1", { format: "xlsx" });
    expect(result.filename).toBe("job-1.xlsx");
    expect(result.contentType).toBe(
      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    );
    expect(result.isBinary).toBe(true);
    expect(result.content).toBe("UEsDBA==");
  });

  it("loads paginated jsonl results from the raw results endpoint", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response('[{"url":"https://example.com"}]', {
        status: 200,
        headers: {
          "Content-Type": "application/json",
          "X-Total-Count": "1",
        },
      }),
    );

    const result = await loadResults("job-1", "jsonl", 1, 100);
    expect(result.totalCount).toBe(1);
    expect(result.data).toEqual([{ url: "https://example.com" }]);
  });
});
