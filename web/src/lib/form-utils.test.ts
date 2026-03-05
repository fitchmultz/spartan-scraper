/**
 * Tests for form parsing functions and request builders.
 *
 * Tests form building logic for scrape, crawl, and research requests in isolation.
 */
import { describe, it, expect } from "vitest";
import {
  buildAuth,
  parseProcessors,
  buildPipelineOptions,
  buildScrapeRequest,
  buildCrawlRequest,
  buildResearchRequest,
} from "./form-utils";

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
