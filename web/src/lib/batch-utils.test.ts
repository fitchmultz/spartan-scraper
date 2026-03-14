import { describe, expect, it } from "vitest";

import {
  buildBatchResearchRequest,
  buildBatchScrapeRequest,
} from "./batch-utils";

describe("batch-utils AI extraction support", () => {
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
});
