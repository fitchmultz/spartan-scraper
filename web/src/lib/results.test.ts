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
      new Response('{"title":"Example"}', {
        status: 200,
        headers: {
          "Content-Type": "application/json",
          "Content-Disposition": 'attachment; filename="job-1.json"',
        },
      }),
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
      content: '{"title":"Example"}',
      filename: "job-1.json",
      contentType: "application/json",
      isBinary: false,
    });
  });

  it("exports binary xlsx results as base64 payloads", async () => {
    const bytes = new Uint8Array([0x50, 0x4b, 0x03, 0x04]);
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(bytes, {
        status: 200,
        headers: {
          "Content-Type":
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
          "Content-Disposition": 'attachment; filename="job-1.xlsx"',
        },
      }),
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
