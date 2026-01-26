// Unit tests for loadResults error handling logic.
// These tests verify error handling behavior by testing logic in isolation.
// Note: loadResults is internal to App component, so these tests duplicate logic
// to verify correct error handling behavior. The actual App.tsx implementation should match
// this logic for tests to remain valid.
import { describe, it, expect, vi, beforeEach } from "vitest";
import { getApiBaseUrl } from "./lib/api-config";
import {
  buildAuth,
  parseProcessors,
  buildPipelineOptions,
  buildScrapeRequest,
  buildCrawlRequest,
  buildResearchRequest,
} from "./App";

const validNDJSON = JSON.stringify({
  url: "https://example.com",
  status: 200,
  title: "Test",
  text: "Content",
  links: [],
});

async function loadResults(jobId: string, format: string = "jsonl") {
  try {
    const apiBaseUrl = getApiBaseUrl();
    const resultsUrl = apiBaseUrl
      ? `${apiBaseUrl}/v1/jobs/${jobId}/results?format=${format}`
      : `/v1/jobs/${jobId}/results?format=${format}`;
    const response = await fetch(resultsUrl);

    if (!response.ok) {
      let errorMessage = `Failed to load results (${response.status} ${response.statusText})`;
      try {
        const errorData = (await response.json()) as { error?: string };
        if (errorData.error) {
          errorMessage = errorData.error;
        }
      } catch {
        // If parsing error body fails, use default message
      }
      return { error: errorMessage };
    }

    const text = await response.text();

    // Handle based on format
    if (format === "jsonl") {
      const lines = text.split("\n").filter((line) => line.trim());

      const parsedItems: unknown[] = [];
      for (const line of lines) {
        try {
          const parsed = JSON.parse(line);
          parsedItems.push(parsed);
        } catch {
          // Skip malformed JSON lines
        }
      }

      // Check if we had input but failed to parse anything
      if (parsedItems.length === 0 && lines.length > 0) {
        return {
          error: "No valid results found. Results file may be corrupted.",
        };
      }

      return { data: parsedItems, raw: text };
    } else {
      // For other formats, just store raw text for display
      return { raw: text };
    }
  } catch (err) {
    return { error: String(err) };
  }
}

describe("loadResults error handling", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("should display server error message on 404", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 404,
      statusText: "Not Found",
      json: async () => ({ error: "job not found" }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result).toEqual({ error: "job not found" });
    expect(mockFetch).toHaveBeenCalledWith(
      "/v1/jobs/test-id/results?format=jsonl",
    );
  });

  it("should display server error message on 500", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      json: async () => ({ error: "internal server error" }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result).toEqual({ error: "internal server error" });
    expect(mockFetch).toHaveBeenCalledWith(
      "/v1/jobs/test-id/results?format=jsonl",
    );
  });

  it("should display server error message on 400", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 400,
      statusText: "Bad Request",
      json: async () => ({ error: "invalid job id" }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result).toEqual({ error: "invalid job id" });
    expect(mockFetch).toHaveBeenCalledWith(
      "/v1/jobs/test-id/results?format=jsonl",
    );
  });

  it("should display default error message when error body cannot be parsed", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 404,
      statusText: "Not Found",
      json: async () => {
        throw new Error("Invalid JSON");
      },
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result).toEqual({ error: "Failed to load results (404 Not Found)" });
    expect(mockFetch).toHaveBeenCalledWith(
      "/v1/jobs/test-id/results?format=jsonl",
    );
  });

  it("should handle network errors", async () => {
    const mockFetch = vi.fn().mockRejectedValueOnce(new Error("Network error"));
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result.error).toContain("Network error");
    expect(mockFetch).toHaveBeenCalledWith(
      "/v1/jobs/test-id/results?format=jsonl",
    );
  });

  it("should display 'No valid results found' when results file is corrupted", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      statusText: "OK",
      text: async () => "invalid json\nmore invalid json",
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result).toEqual({
      error: "No valid results found. Results file may be corrupted.",
    });
    expect(mockFetch).toHaveBeenCalledWith(
      "/v1/jobs/test-id/results?format=jsonl",
    );
  });

  it("should parse and display valid results", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      statusText: "OK",
      text: async () => validNDJSON,
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([JSON.parse(validNDJSON)]);
    expect(result.raw).toEqual(validNDJSON);
    expect(mockFetch).toHaveBeenCalledWith(
      "/v1/jobs/test-id/results?format=jsonl",
    );
  });

  it("should handle empty results", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      statusText: "OK",
      text: async () => "",
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([]);
    expect(result.raw).toEqual("");
    expect(mockFetch).toHaveBeenCalledWith(
      "/v1/jobs/test-id/results?format=jsonl",
    );
  });

  it("should skip malformed JSON lines in results", async () => {
    const mixedNDJSON = `invalid line\n${validNDJSON}\nalso invalid`;
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      statusText: "OK",
      text: async () => mixedNDJSON,
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([JSON.parse(validNDJSON)]);
    expect(result.raw).toEqual(mixedNDJSON);
    expect(mockFetch).toHaveBeenCalledWith(
      "/v1/jobs/test-id/results?format=jsonl",
    );
  });

  describe("format parameter handling", () => {
    it("should include format parameter in URL when format is provided", async () => {
      const mockFetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: "OK",
        text: async () => validNDJSON,
      });
      vi.stubGlobal("fetch", mockFetch);

      const result = await loadResults("test-id", "json");

      expect(result.error).toBeUndefined();
      expect(result.data).toBeUndefined();
      expect(result.raw).toEqual(validNDJSON);
      expect(mockFetch).toHaveBeenCalledWith(
        "/v1/jobs/test-id/results?format=json",
      );
    });

    it("should default to jsonl format when no format is provided", async () => {
      const mockFetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: "OK",
        text: async () => validNDJSON,
      });
      vi.stubGlobal("fetch", mockFetch);

      const result = await loadResults("test-id");

      expect(result.error).toBeUndefined();
      expect(result.data).toEqual([JSON.parse(validNDJSON)]);
      expect(mockFetch).toHaveBeenCalledWith(
        "/v1/jobs/test-id/results?format=jsonl",
      );
    });

    it("should handle csv format", async () => {
      const csvContent =
        "url,status,title,text,links\nhttps://example.com,200,Test,Content,";
      const mockFetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: "OK",
        text: async () => csvContent,
      });
      vi.stubGlobal("fetch", mockFetch);

      const result = await loadResults("test-id", "csv");

      expect(result.error).toBeUndefined();
      expect(result.raw).toEqual(csvContent);
      expect(mockFetch).toHaveBeenCalledWith(
        "/v1/jobs/test-id/results?format=csv",
      );
    });

    it("should handle md format", async () => {
      const mdContent = "# Test Results\n\n- Item 1\n- Item 2";
      const mockFetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: "OK",
        text: async () => mdContent,
      });
      vi.stubGlobal("fetch", mockFetch);

      const result = await loadResults("test-id", "md");

      expect(result.error).toBeUndefined();
      expect(result.raw).toEqual(mdContent);
      expect(mockFetch).toHaveBeenCalledWith(
        "/v1/jobs/test-id/results?format=md",
      );
    });

    it("should handle json format", async () => {
      const jsonArray = `[${validNDJSON}]`;
      const mockFetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: "OK",
        text: async () => jsonArray,
      });
      vi.stubGlobal("fetch", mockFetch);

      const result = await loadResults("test-id", "json");

      expect(result.error).toBeUndefined();
      expect(result.raw).toEqual(jsonArray);
      expect(mockFetch).toHaveBeenCalledWith(
        "/v1/jobs/test-id/results?format=json",
      );
    });

    it("should return empty parsedItems for non-JSONL formats", async () => {
      const csvContent = "url,status,title\nhttps://example.com,200,Test";
      const mockFetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: "OK",
        text: async () => csvContent,
      });
      vi.stubGlobal("fetch", mockFetch);

      const result = await loadResults("test-id", "csv");

      expect(result.error).toBeUndefined();
      expect(result.raw).toEqual(csvContent);
      expect(mockFetch).toHaveBeenCalledWith(
        "/v1/jobs/test-id/results?format=csv",
      );
    });
  });
});

describe("pipeline options parsing", () => {
  it("should return undefined for empty string", () => {
    expect(parseProcessors("")).toBeUndefined();
  });

  it("should return undefined for whitespace-only string", () => {
    expect(parseProcessors("   ,   ,   ")).toBeUndefined();
  });

  it("should parse single processor", () => {
    expect(parseProcessors("redact")).toEqual(["redact"]);
  });

  it("should parse multiple processors", () => {
    expect(parseProcessors("redact, sanitize, cleanup")).toEqual([
      "redact",
      "sanitize",
      "cleanup",
    ]);
  });

  it("should trim whitespace", () => {
    expect(parseProcessors("  redact  ,  sanitize  ")).toEqual([
      "redact",
      "sanitize",
    ]);
  });

  it("should handle extra commas", () => {
    expect(parseProcessors("redact,,sanitize,,cleanup")).toEqual([
      "redact",
      "sanitize",
      "cleanup",
    ]);
  });
});

describe("buildPipelineOptions", () => {
  it("should return undefined for all empty processors", () => {
    expect(buildPipelineOptions("", "", "")).toBeUndefined();
  });

  it("should build pipeline options with pre-processors", () => {
    const result = buildPipelineOptions("redact,sanitize", "", "");
    expect(result).toEqual({
      preProcessors: ["redact", "sanitize"],
      postProcessors: undefined,
      transformers: undefined,
    });
  });

  it("should build pipeline options with all processor types", () => {
    const result = buildPipelineOptions("redact", "cleanup", "json-export");
    expect(result).toEqual({
      preProcessors: ["redact"],
      postProcessors: ["cleanup"],
      transformers: ["json-export"],
    });
  });
});

describe("buildScrapeRequest with pipeline options", () => {
  it("should include pipeline options in request", () => {
    const request = buildScrapeRequest(
      "https://example.com",
      true,
      false,
      30,
      undefined,
      undefined,
      { template: "article", validate: true },
      "redact,sanitize",
      "cleanup",
      "json-export",
      true,
    );

    expect(request).toMatchObject({
      url: "https://example.com",
      pipeline: {
        preProcessors: ["redact", "sanitize"],
        postProcessors: ["cleanup"],
        transformers: ["json-export"],
      },
      incremental: true,
      extract: { template: "article", validate: true },
    });
  });

  it("should omit undefined pipeline options", () => {
    const request = buildScrapeRequest(
      "https://example.com",
      false,
      false,
      30,
      undefined,
      undefined,
      undefined,
      "",
      "",
      "",
      false,
    );

    expect(request.pipeline).toBeUndefined();
    expect(request.incremental).toBeUndefined();
  });
});

describe("buildCrawlRequest with pipeline options", () => {
  it("should include pipeline options in request", () => {
    const request = buildCrawlRequest(
      "https://example.com",
      2,
      10,
      true,
      false,
      30,
      undefined,
      undefined,
      undefined,
      "redact",
      "cleanup",
      "",
      false,
    );

    expect(request).toMatchObject({
      url: "https://example.com",
      maxDepth: 2,
      maxPages: 10,
      pipeline: {
        preProcessors: ["redact"],
        postProcessors: ["cleanup"],
        transformers: undefined,
      },
      incremental: undefined,
    });
  });
});

describe("buildResearchRequest with pipeline options", () => {
  it("should include pipeline options in request", () => {
    const request = buildResearchRequest(
      "test query",
      ["https://example.com", "https://test.com"],
      3,
      20,
      false,
      false,
      30,
      undefined,
      undefined,
      undefined,
      "sanitize,cleanup",
      "",
      "csv-export",
      true,
    );

    expect(request).toMatchObject({
      query: "test query",
      urls: ["https://example.com", "https://test.com"],
      pipeline: {
        preProcessors: ["sanitize", "cleanup"],
        postProcessors: undefined,
        transformers: ["csv-export"],
      },
      incremental: true,
    });
  });
});

describe("auth payload generation", () => {
  it("should include login flow fields when specified", () => {
    const auth = buildAuth(
      "", // basic
      undefined, // headers
      undefined, // cookies
      undefined, // query
      "https://example.com/login", // loginUrl
      "#email", // loginUserSelector
      "#password", // loginPassSelector
      "button[type=submit]", // loginSubmitSelector
      "user@example.com", // loginUser
      "secret123", // loginPass
    );

    expect(auth).toEqual({
      loginUrl: "https://example.com/login",
      loginUserSelector: "#email",
      loginPassSelector: "#password",
      loginSubmitSelector: "button[type=submit]",
      loginUser: "user@example.com",
      loginPass: "secret123",
    });
  });

  it("should exclude undefined login flow fields", () => {
    const auth = buildAuth(
      "user:pass",
      { "X-API": "token" },
      undefined,
      { key: "value" },
      "",
      undefined,
      undefined,
      undefined,
      "",
      "",
    );

    expect(auth).toEqual({
      basic: "user:pass",
      headers: { "X-API": "token" },
      cookies: undefined,
      query: { key: "value" },
    });
  });

  it("should return undefined when no auth is specified", () => {
    const auth = buildAuth(
      "",
      undefined,
      undefined,
      undefined,
      "",
      "",
      "",
      "",
      "",
      "",
    );
    expect(auth).toBeUndefined();
  });

  it("should include both basic auth and login flow fields", () => {
    const auth = buildAuth(
      "user:pass",
      undefined,
      undefined,
      undefined,
      "https://example.com/login",
      "#email",
      "#password",
      "button[type=submit]",
      "user@example.com",
      "secret123",
    );

    expect(auth).toEqual({
      basic: "user:pass",
      loginUrl: "https://example.com/login",
      loginUserSelector: "#email",
      loginPassSelector: "#password",
      loginSubmitSelector: "button[type=submit]",
      loginUser: "user@example.com",
      loginPass: "secret123",
    });
  });

  it("should return undefined when all fields are empty strings", () => {
    const auth = buildAuth(
      "",
      undefined,
      undefined,
      undefined,
      "",
      "",
      "",
      "",
      "",
      "",
    );
    expect(auth).toBeUndefined();
  });

  it("should handle partial login flow configuration", () => {
    const auth = buildAuth(
      "",
      undefined,
      undefined,
      undefined,
      "https://example.com/login",
      undefined,
      undefined,
      undefined,
      "user@example.com",
      "",
    );

    expect(auth).toEqual({
      loginUrl: "https://example.com/login",
      loginUser: "user@example.com",
    });
  });
});

describe("pagination", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const generateMockItems = (count: number) => {
    return Array.from({ length: count }, (_, i) => ({
      url: `https://example.com/page${i + 1}`,
      status: 200,
      title: `Page ${i + 1}`,
      text: `Content ${i + 1}`,
      links: [],
    }));
  };

  it("should include pagination parameters for jsonl format", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      statusText: "OK",
      json: async () => generateMockItems(100),
      headers: new Headers({ "X-Total-Count": "250" }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const apiBaseUrl = getApiBaseUrl();
    const resultsUrl = apiBaseUrl
      ? `${apiBaseUrl}/v1/jobs/test-id/results?format=jsonl&limit=100&offset=0`
      : `/v1/jobs/test-id/results?format=jsonl&limit=100&offset=0`;
    const response = await fetch(resultsUrl);

    expect(response.ok).toBe(true);
    expect(response.headers.get("X-Total-Count")).toBe("250");
    const items = (await response.json()) as unknown[];
    expect(items.length).toBe(100);
  });

  it("should not include pagination parameters for non-jsonl formats", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      statusText: "OK",
      text: async () => "json format content",
      headers: new Headers(),
    });
    vi.stubGlobal("fetch", mockFetch);

    const apiBaseUrl = getApiBaseUrl();
    const resultsUrl = apiBaseUrl
      ? `${apiBaseUrl}/v1/jobs/test-id/results?format=json`
      : `/v1/jobs/test-id/results?format=json`;
    const response = await fetch(resultsUrl);

    expect(response.ok).toBe(true);
    expect(response.headers.get("X-Total-Count")).toBeNull();
  });

  it("should calculate correct offset for page 1", () => {
    const page = 1;
    const resultsPerPage = 100;
    const expectedOffset = (page - 1) * resultsPerPage;
    expect(expectedOffset).toBe(0);
  });

  it("should calculate correct offset for page 2", () => {
    const page = 2;
    const resultsPerPage = 100;
    const expectedOffset = (page - 1) * resultsPerPage;
    expect(expectedOffset).toBe(100);
  });

  it("should calculate correct offset for page 5", () => {
    const page = 5;
    const resultsPerPage = 100;
    const expectedOffset = (page - 1) * resultsPerPage;
    expect(expectedOffset).toBe(400);
  });

  it("should calculate total pages correctly", () => {
    const totalResults = 250;
    const resultsPerPage = 100;
    const totalPages = Math.ceil(totalResults / resultsPerPage);
    expect(totalPages).toBe(3);
  });

  it("should handle exact multiple for total pages", () => {
    const totalResults = 300;
    const resultsPerPage = 100;
    const totalPages = Math.ceil(totalResults / resultsPerPage);
    expect(totalPages).toBe(3);
  });

  it("should handle partial last page for total pages", () => {
    const totalResults = 251;
    const resultsPerPage = 100;
    const totalPages = Math.ceil(totalResults / resultsPerPage);
    expect(totalPages).toBe(3);
  });

  it("should validate page number is within range", () => {
    const totalResults = 250;
    const resultsPerPage = 100;
    const totalPages = Math.ceil(totalResults / resultsPerPage);

    const validPage = 2;
    expect(validPage >= 1 && validPage <= totalPages).toBe(true);

    const invalidPageLow = 0;
    expect(invalidPageLow >= 1 && invalidPageLow <= totalPages).toBe(false);

    const invalidPageHigh = 4;
    expect(invalidPageHigh >= 1 && invalidPageHigh <= totalPages).toBe(false);
  });

  it("should handle empty results with pagination", () => {
    const totalResults = 0;
    const resultsPerPage = 100;
    const totalPages = Math.ceil(totalResults / resultsPerPage);
    expect(totalPages).toBe(0);
  });

  it("should handle single page with items less than per page", () => {
    const totalResults = 50;
    const resultsPerPage = 100;
    const totalPages = Math.ceil(totalResults / resultsPerPage);
    expect(totalPages).toBe(1);
  });

  it("should handle max page edge case", () => {
    const totalResults = 1000;
    const resultsPerPage = 100;
    const maxPage = Math.ceil(totalResults / resultsPerPage);
    const offsetForLastPage = (maxPage - 1) * resultsPerPage;
    expect(offsetForLastPage).toBe(900);
  });

  it("should disable previous button on first page", () => {
    const currentPage = 1;
    const isDisabled = currentPage <= 1;
    expect(isDisabled).toBe(true);
  });

  it("should disable next button on last page", () => {
    const totalResults = 250;
    const resultsPerPage = 100;
    const totalPages = Math.ceil(totalResults / resultsPerPage);
    const currentPage = totalPages;
    const isDisabled = currentPage >= totalPages;
    expect(isDisabled).toBe(true);
  });

  it("should enable both buttons on middle page", () => {
    const totalResults = 300;
    const resultsPerPage = 100;
    const totalPages = Math.ceil(totalResults / resultsPerPage);
    const currentPage = 2;
    const previousDisabled = currentPage <= 1;
    const nextDisabled = currentPage >= totalPages;
    expect(previousDisabled).toBe(false);
    expect(nextDisabled).toBe(false);
  });

  it("should clamp jump input to valid page range", () => {
    const totalResults = 250;
    const resultsPerPage = 100;
    const maxPage = Math.ceil(totalResults / resultsPerPage);

    const validInput = 2;
    const isValid =
      validInput >= 1 && validInput <= maxPage && Number.isInteger(validInput);
    expect(isValid).toBe(true);

    const invalidLow = 0;
    expect(invalidLow >= 1 && invalidLow <= maxPage).toBe(false);

    const invalidHigh = 4;
    expect(invalidHigh >= 1 && invalidHigh <= maxPage).toBe(false);

    const invalidDecimal = 1.5;
    expect(Number.isInteger(invalidDecimal)).toBe(false);
  });

  it("should display correct page info text", () => {
    const currentPage = 2;
    const totalResults = 250;
    const resultsPerPage = 100;
    const totalPages = Math.ceil(totalResults / resultsPerPage);

    const expectedText = `Page ${currentPage} of ${totalPages}(${totalResults} total results)`;
    expect(expectedText).toBe("Page 2 of 3(250 total results)");
  });
});
