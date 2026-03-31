/**
 * Tests for form parsing functions and request builders.
 *
 * Tests form building logic for scrape, crawl, and research requests in isolation.
 */
import { describe, it, expect } from "vitest";
import {
  buildAIExtractOptions,
  buildAuth,
  buildPipelineOptions,
  buildCrawlRequest,
  buildResearchAgenticOptions,
  buildResearchRequest,
  buildScrapeRequest,
  getHttpUrlValidationState,
  getInvalidHttpUrls,
  isValidHttpUrl,
  parseAIExtractSchemaText,
  parseProcessors,
  parseUrlList,
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

describe("AI extract helpers", () => {
  it("parses schema-guided JSON into an object", () => {
    expect(
      parseAIExtractSchemaText('{"title":"Example","price":"$19.99"}'),
    ).toEqual({
      title: "Example",
      price: "$19.99",
    });
  });

  it("rejects non-object AI schema JSON", () => {
    expect(() => parseAIExtractSchemaText('["bad"]')).toThrow(
      "AI schema must be a JSON object",
    );
  });

  it("builds natural-language AI extraction options", () => {
    expect(
      buildAIExtractOptions(
        true,
        "natural_language",
        "Extract title and price",
        "",
        "title, price",
      ),
    ).toEqual({
      enabled: true,
      mode: "natural_language",
      prompt: "Extract title and price",
      fields: ["title", "price"],
    });
  });

  it("builds schema-guided AI extraction options", () => {
    expect(
      buildAIExtractOptions(
        true,
        "schema_guided",
        "ignored",
        '{"title":"Example","price":"$19.99"}',
        "title, price",
      ),
    ).toEqual({
      enabled: true,
      mode: "schema_guided",
      schema: {
        title: "Example",
        price: "$19.99",
      },
      fields: ["title", "price"],
    });
  });
});

describe("isValidHttpUrl", () => {
  it("accepts trimmed http and https URLs with hosts", () => {
    expect(isValidHttpUrl(" https://example.com/path ")).toBe(true);
    expect(isValidHttpUrl("http://localhost:8741/health")).toBe(true);
  });

  it("rejects non-http schemes and missing hosts", () => {
    expect(isValidHttpUrl("ftp://example.com/file.txt")).toBe(false);
    expect(isValidHttpUrl("https:")).toBe(false);
    expect(isValidHttpUrl("not-a-url")).toBe(false);
  });
});

describe("getHttpUrlValidationState", () => {
  it("distinguishes missing URLs from malformed ones", () => {
    expect(getHttpUrlValidationState("   ")).toBe("missing");
    expect(getHttpUrlValidationState("notaurl")).toBe("invalid");
    expect(getHttpUrlValidationState("https://example.com")).toBeNull();
  });
});

describe("getInvalidHttpUrls", () => {
  it("returns only malformed HTTP(S) URLs from a parsed list", () => {
    expect(
      getInvalidHttpUrls([
        "https://example.com",
        "not-a-url",
        "http://localhost:8741/health",
        "ftp://example.com/file.txt",
      ]),
    ).toEqual(["not-a-url", "ftp://example.com/file.txt"]);
  });
});

describe("buildAuth", () => {
  it("builds direct proxy auth overrides", () => {
    expect(
      buildAuth(
        "",
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        "http://proxy.example:8080",
        "user",
        "pass",
      ),
    ).toEqual({
      proxy: {
        url: "http://proxy.example:8080",
        username: "user",
        password: "pass",
      },
    });
  });

  it("builds proxy-pool selection hints", () => {
    expect(
      buildAuth(
        "",
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        "us-east",
        ["residential", "sticky"],
        ["proxy-2"],
      ),
    ).toEqual({
      proxyHints: {
        preferred_region: "us-east",
        required_tags: ["residential", "sticky"],
        exclude_proxy_ids: ["proxy-2"],
      },
    });
  });

  it("rejects conflicting direct proxy and proxy-pool hints", () => {
    expect(() =>
      buildAuth(
        "",
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        "http://proxy.example:8080",
        undefined,
        undefined,
        "us-east",
      ),
    ).toThrow(/mutually exclusive/i);
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

  it("should merge AI extraction options into crawl extract config", () => {
    const request = buildCrawlRequest(
      "https://example.com",
      2,
      10,
      true,
      false,
      30,
      undefined,
      undefined,
      { template: "article", validate: true },
      "",
      "",
      "",
      false,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      {
        enabled: true,
        mode: "natural_language",
        prompt: "Extract title and price",
        fields: ["title", "price"],
      },
    );

    expect(request.extract).toEqual({
      template: "article",
      validate: true,
      ai: {
        enabled: true,
        mode: "natural_language",
        prompt: "Extract title and price",
        fields: ["title", "price"],
      },
    });
  });
});

describe("buildResearchAgenticOptions", () => {
  it("returns undefined when disabled", () => {
    expect(buildResearchAgenticOptions(false, "", 1, 3)).toBeUndefined();
  });

  it("builds bounded agentic research config", () => {
    expect(
      buildResearchAgenticOptions(
        true,
        "Prioritize pricing and support commitments",
        2,
        4,
      ),
    ).toEqual({
      enabled: true,
      instructions: "Prioritize pricing and support commitments",
      maxRounds: 2,
      maxFollowUpUrls: 4,
    });
  });
});

describe("parseUrlList", () => {
  it("accepts comma-separated and newline-separated URLs", () => {
    expect(
      parseUrlList(
        "https://example.com,\nhttps://example.org\nhttps://example.net",
      ),
    ).toEqual([
      "https://example.com",
      "https://example.org",
      "https://example.net",
    ]);
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

  it("should merge AI extraction options into research extract config", () => {
    const request = buildResearchRequest(
      "pricing model",
      ["https://example.com"],
      2,
      10,
      false,
      false,
      30,
      undefined,
      undefined,
      { template: "article", validate: true },
      "",
      "",
      "",
      undefined,
      undefined,
      undefined,
      undefined,
      {
        enabled: true,
        mode: "natural_language",
        prompt: "Extract the pricing model and contract terms",
        fields: ["pricing_model", "contract_terms"],
      },
    );

    expect(request.extract).toEqual({
      template: "article",
      validate: true,
      ai: {
        enabled: true,
        mode: "natural_language",
        prompt: "Extract the pricing model and contract terms",
        fields: ["pricing_model", "contract_terms"],
      },
    });
  });

  it("should include agentic research config when provided", () => {
    const request = buildResearchRequest(
      "pricing model",
      ["https://example.com"],
      2,
      10,
      false,
      false,
      30,
      undefined,
      undefined,
      undefined,
      "",
      "",
      "",
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      {
        enabled: true,
        instructions: "Prioritize pricing and support commitments",
        maxRounds: 2,
        maxFollowUpUrls: 4,
      },
    );

    expect(request.agentic).toEqual({
      enabled: true,
      instructions: "Prioritize pricing and support commitments",
      maxRounds: 2,
      maxFollowUpUrls: 4,
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
