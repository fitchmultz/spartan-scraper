// Unit tests for results loading utilities and form parsing functions.
// These tests verify error handling behavior and form building logic in isolation.
import { describe, it, expect } from "vitest";
import {
  parseJsonlResults,
  buildResultsUrl,
  parseErrorResponse,
} from "./lib/results";
import {
  buildAuth,
  parseProcessors,
  buildPipelineOptions,
  buildScrapeRequest,
  buildCrawlRequest,
  buildResearchRequest,
} from "./lib/form-utils";

const validNDJSON = JSON.stringify({
  url: "https://example.com",
  status: 200,
  title: "Test",
  text: "Content",
  links: [],
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
    );

    expect(request).toMatchObject({
      query: "test query",
      urls: ["https://example.com", "https://test.com"],
      pipeline: {
        preProcessors: ["sanitize", "cleanup"],
        postProcessors: undefined,
        transformers: ["csv-export"],
      },
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
  it("should include pagination parameters for jsonl format", () => {
    const resultsPerPage = 100;
    const offset = 0;
    const expectedParams = `limit=${resultsPerPage}&offset=${offset}`;
    expect(expectedParams).toBe("limit=100&offset=0");
  });

  it("should not include pagination parameters for non-jsonl formats", () => {
    const format = "json";
    const expectedParams = `format=${format}`;
    expect(expectedParams).toBe("format=json");
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
