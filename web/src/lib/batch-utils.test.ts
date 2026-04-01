/**
 * Purpose: Verify batch utils behavior with automated regression coverage.
 * Responsibilities: Define focused test cases, fixtures, and assertions for the module under test.
 * Scope: Automated test coverage only; production logic stays in the adjacent source modules.
 * Usage: Run through the repo test entrypoints or the feature-local test command.
 * Invariants/Assumptions: Tests should describe the current contract clearly and remain deterministic under local CI settings.
 */

import { describe, expect, it } from "vitest";

import {
  buildBatchCrawlRequest,
  buildBatchResearchRequest,
  buildBatchScrapeRequest,
  validateUrls,
} from "./batch-utils";

describe("batch-utils AI extraction support", () => {
  it("rejects non-http URLs during batch validation", () => {
    expect(
      validateUrls([
        "https://example.com",
        "ftp://example.com/file.txt",
        "https:",
      ]),
    ).toEqual(["ftp://example.com/file.txt", "https:"]);
  });

  it("normalizes browser runtime fields in batch crawl requests", () => {
    const request = buildBatchCrawlRequest(
      ["https://example.com"],
      2,
      10,
      false,
      true,
      30,
      undefined,
      undefined,
      undefined,
      undefined,
      false,
      undefined,
      undefined,
      undefined,
      undefined,
    );

    expect(request).toMatchObject({
      headless: false,
      timeoutSeconds: 30,
    });
    expect(request.playwright).toBeUndefined();
  });

  it("merges AI extraction options into batch scrape extract config", () => {
    const request = buildBatchScrapeRequest(
      ["https://example.com"],
      false,
      false,
      30,
      undefined,
      undefined,
      { template: "article", validate: true },
      undefined,
      false,
      undefined,
      undefined,
      undefined,
      undefined,
      {
        enabled: true,
        mode: "natural_language",
        prompt: "Extract the title and price",
        fields: ["title", "price"],
      },
    );

    expect(request.extract).toEqual({
      template: "article",
      validate: true,
      ai: {
        enabled: true,
        mode: "natural_language",
        prompt: "Extract the title and price",
        fields: ["title", "price"],
      },
    });
  });

  it("merges AI extraction options into batch research extract config", () => {
    const request = buildBatchResearchRequest(
      ["https://example.com", "https://example.org"],
      "pricing model",
      2,
      50,
      false,
      false,
      30,
      undefined,
      undefined,
      { template: "article", validate: true },
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      {
        enabled: true,
        mode: "schema_guided",
        schema: {
          pricing_model: "Usage-based",
          support_terms: "24/7 support",
        },
        fields: ["pricing_model", "support_terms"],
      },
    );

    expect(request.extract).toEqual({
      template: "article",
      validate: true,
      ai: {
        enabled: true,
        mode: "schema_guided",
        schema: {
          pricing_model: "Usage-based",
          support_terms: "24/7 support",
        },
        fields: ["pricing_model", "support_terms"],
      },
    });
  });

  it("includes agentic research config in batch research requests", () => {
    const request = buildBatchResearchRequest(
      ["https://example.com", "https://example.org"],
      "pricing model",
      2,
      50,
      false,
      false,
      30,
      undefined,
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
