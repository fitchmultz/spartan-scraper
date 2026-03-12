/**
 * Tests for results loading utilities.
 *
 * Tests error handling behavior and JSONL parsing logic in isolation.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  buildResultsUrl,
  loadResults,
  parseErrorResponse,
  parseJsonArrayResults,
  parseJsonlResults,
} from "./results";

const validNDJSON = JSON.stringify({
  url: "https://example.com",
  status: 200,
  title: "Test",
  text: "Content",
  links: [],
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("parseErrorResponse", () => {
  it("should parse valid error JSON", async () => {
    const mockResponse = {
      ok: false,
      status: 404,
      statusText: "Not Found",
      json: async () => ({ error: "job not found" }),
    } as unknown as Response;

    const errorMessage = await parseErrorResponse(
      mockResponse,
      404,
      "Not Found",
    );
    expect(errorMessage).toBe("job not found");
  });

  it("should return default message when error field is missing", async () => {
    const mockResponse = {
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      json: async () => ({ message: "internal error" }),
    } as unknown as Response;

    const errorMessage = await parseErrorResponse(
      mockResponse,
      500,
      "Internal Server Error",
    );
    expect(errorMessage).toBe(
      "Failed to load results (500 Internal Server Error)",
    );
  });

  it("should return default message when JSON parsing fails", async () => {
    const mockResponse = {
      ok: false,
      status: 404,
      statusText: "Not Found",
      json: async () => {
        throw new Error("Invalid JSON");
      },
    } as unknown as Response;

    const errorMessage = await parseErrorResponse(
      mockResponse,
      404,
      "Not Found",
    );
    expect(errorMessage).toBe("Failed to load results (404 Not Found)");
  });

  it("should return correct default message format with status and statusText", async () => {
    const mockResponse = {
      ok: false,
      status: 400,
      statusText: "Bad Request",
      json: async () => {
        throw new Error("Invalid JSON");
      },
    } as unknown as Response;

    const errorMessage = await parseErrorResponse(
      mockResponse,
      400,
      "Bad Request",
    );
    expect(errorMessage).toBe("Failed to load results (400 Bad Request)");
  });
});

describe("parseJsonlResults", () => {
  it("should parse valid JSONL", () => {
    const result = parseJsonlResults(validNDJSON);

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([JSON.parse(validNDJSON)]);
    expect(result.raw).toEqual(validNDJSON);
  });

  it("should skip malformed lines", () => {
    const mixedNDJSON = `invalid line\n${validNDJSON}\nalso invalid`;
    const result = parseJsonlResults(mixedNDJSON);

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([JSON.parse(validNDJSON)]);
    expect(result.raw).toEqual(mixedNDJSON);
  });

  it("should handle empty input", () => {
    const result = parseJsonlResults("");

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([]);
    expect(result.raw).toEqual("");
  });

  it("should handle all malformed lines", () => {
    const corrupted = "invalid json\nmore invalid json";
    const result = parseJsonlResults(corrupted);

    expect(result.error).toBe(
      "No valid results found. Results file may be corrupted.",
    );
    expect(result.data).toBeUndefined();
    expect(result.raw).toEqual(corrupted);
  });

  it("should preserve raw text in all cases", () => {
    const testText = "some text";
    const result = parseJsonlResults(testText);
    expect(result.raw).toEqual(testText);
  });

  it("should filter empty lines", () => {
    const withEmptyLines = `\n\n${validNDJSON}\n\n\n`;
    const result = parseJsonlResults(withEmptyLines);

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([JSON.parse(validNDJSON)]);
    expect(result.raw).toEqual(withEmptyLines);
  });

  it("should parse multiple valid lines", () => {
    const line2 = JSON.stringify({
      url: "https://example2.com",
      status: 200,
      title: "Test2",
      text: "Content2",
      links: [],
    });
    const multiLineNDJSON = `${validNDJSON}\n${line2}`;
    const result = parseJsonlResults(multiLineNDJSON);

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([JSON.parse(validNDJSON), JSON.parse(line2)]);
    expect(result.raw).toEqual(multiLineNDJSON);
  });
});

describe("parseJsonArrayResults", () => {
  it("should parse a paginated JSON array response", () => {
    const responseText = `[${validNDJSON}]`;

    const result = parseJsonArrayResults(responseText);

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([JSON.parse(validNDJSON)]);
    expect(result.raw).toEqual(
      JSON.stringify([JSON.parse(validNDJSON)], null, 2),
    );
  });

  it("should reject non-array JSON payloads", () => {
    const result = parseJsonArrayResults(validNDJSON);

    expect(result.error).toBe("Expected paginated results to be a JSON array.");
    expect(result.data).toBeUndefined();
  });
});

describe("buildResultsUrl", () => {
  it("should return relative path when base URL is empty", () => {
    const url = buildResultsUrl("test-id", "jsonl");
    expect(url).toBe("/v1/jobs/test-id/results?format=jsonl");
  });

  it("should include format parameter", () => {
    const url = buildResultsUrl("test-id", "json");
    expect(url).toContain("?format=json");
  });

  it("should include pagination params for jsonl format", () => {
    const url = buildResultsUrl("test-id", "jsonl", 100, 0);
    expect(url).toContain("?format=jsonl&limit=100&offset=0");
  });

  it("should calculate correct offset for page 1", () => {
    const url = buildResultsUrl("test-id", "jsonl", 100, 0);
    expect(url).toContain("offset=0");
  });

  it("should calculate correct offset for page 2", () => {
    const url = buildResultsUrl("test-id", "jsonl", 100, 100);
    expect(url).toContain("offset=100");
  });

  it("should calculate correct offset for page 5", () => {
    const url = buildResultsUrl("test-id", "jsonl", 100, 400);
    expect(url).toContain("offset=400");
  });

  it("should not include pagination for non-jsonl formats", () => {
    const url = buildResultsUrl("test-id", "json", 100, 0);
    expect(url).toBe("/v1/jobs/test-id/results?format=json");
    expect(url).not.toContain("limit");
    expect(url).not.toContain("offset");
  });

  it("should handle csv format", () => {
    const url = buildResultsUrl("test-id", "csv");
    expect(url).toBe("/v1/jobs/test-id/results?format=csv");
  });

  it("should handle md format", () => {
    const url = buildResultsUrl("test-id", "md");
    expect(url).toBe("/v1/jobs/test-id/results?format=md");
  });

  it("should handle json format", () => {
    const url = buildResultsUrl("test-id", "json");
    expect(url).toBe("/v1/jobs/test-id/results?format=json");
  });

  it("should include transform parameters when provided", () => {
    const url = buildResultsUrl(
      "test-id",
      "json",
      undefined,
      undefined,
      "{title: title}",
      "jmespath",
    );
    expect(url).toContain("format=json");
    expect(url).toContain("transform_expression=");
    expect(url).toContain("transform_language=jmespath");
  });

  it("should encode transform expression", () => {
    const url = buildResultsUrl(
      "test-id",
      "json",
      undefined,
      undefined,
      "{title: title, url: url}",
      "jmespath",
    );
    expect(url).toContain(
      "transform_expression=%7Btitle%3A%20title%2C%20url%3A%20url%7D",
    );
  });

  it("should handle jsonata language", () => {
    const url = buildResultsUrl(
      "test-id",
      "json",
      undefined,
      undefined,
      '{"name": name}',
      "jsonata",
    );
    expect(url).toContain("transform_language=jsonata");
  });

  it("should not include transform_language if transform_expression is not provided", () => {
    const url = buildResultsUrl(
      "test-id",
      "json",
      undefined,
      undefined,
      undefined,
      "jmespath",
    );
    expect(url).not.toContain("transform_language");
  });
});

describe("loadResults", () => {
  it("returns X-Total-Count from the original jsonl response", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({
        "X-Total-Count": "42",
        "Content-Type": "application/json",
      }),
      text: async () => `[${validNDJSON}]`,
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await loadResults("job-1", "jsonl", 1, 100);

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(result.error).toBeUndefined();
    expect(result.totalCount).toBe(42);
    expect(result.data).toEqual([JSON.parse(validNDJSON)]);
    expect(result.raw).toEqual(
      JSON.stringify([JSON.parse(validNDJSON)], null, 2),
    );
  });
});
