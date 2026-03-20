/**
 * Purpose: Verify job-to-automation promotion mapping stays aligned with the audited source-job reuse contract.
 * Responsibilities: Assert template, watch, and export schedule seeds preserve supported fields, flag unsupported carry-forward, and keep destination semantics explicit.
 * Scope: Pure promotion helper coverage only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Promotion consumes sanitized jobs, watch promotion is scrape-first, and export schedules automate future matching completed jobs rather than rerunning the source job.
 */

import { describe, expect, it } from "vitest";
import type { Job } from "../api";
import {
  buildExportSchedulePromotionSeed,
  buildTemplatePromotionSeed,
  buildWatchPromotionSeed,
} from "./promotion";

function makeJob(overrides: Partial<Job> = {}): Job {
  return {
    id: "job-1",
    kind: "scrape",
    status: "succeeded",
    createdAt: "2026-03-20T10:00:00Z",
    updatedAt: "2026-03-20T10:05:00Z",
    finishedAt: "2026-03-20T10:05:00Z",
    specVersion: 1,
    spec: {
      version: 1,
      url: "https://example.com/pricing",
      execution: {
        headless: true,
        playwright: true,
        screenshot: {
          enabled: true,
          fullPage: true,
          format: "png",
        },
      },
    },
    run: {
      waitMs: 0,
      runMs: 1000,
      totalMs: 1000,
    },
    ...overrides,
  };
}

describe("promotion", () => {
  it("builds an inline-template seed when reusable extraction rules exist", () => {
    const seed = buildTemplatePromotionSeed(
      makeJob({
        spec: {
          version: 1,
          url: "https://example.com/article",
          execution: {
            extract: {
              inline: {
                name: "inline-article",
                selectors: [{ name: "title", selector: "article h1" }],
              },
            },
          },
        },
      }),
    );

    expect(seed.mode).toBe("inline-template");
    expect(seed.template?.selectors?.[0]?.selector).toBe("article h1");
    expect(seed.previewUrl).toBe("https://example.com/article");
  });

  it("builds a scrape-first watch seed and flags unsupported auth carry-forward", () => {
    const seed = buildWatchPromotionSeed(
      makeJob({
        spec: {
          version: 1,
          url: "https://example.com/pricing",
          execution: {
            headless: true,
            playwright: true,
            authProfile: "private-site",
            screenshot: { enabled: true, fullPage: true, format: "png" },
          },
        },
      }),
    );

    expect(seed.eligible).toBe(true);
    expect(seed.formData.url).toBe("https://example.com/pricing");
    expect(seed.formData.usePlaywright).toBe(true);
    expect(seed.unsupportedCarryForward).toContain(
      "Authentication settings are not carried into watches in this cut.",
    );
  });

  it("rejects non-scrape watch promotion in the first cut", () => {
    const seed = buildWatchPromotionSeed(
      makeJob({
        kind: "research",
        spec: {
          version: 1,
          query: "Find pricing changes",
          urls: ["https://example.com/pricing"],
          execution: {},
        },
      }),
    );

    expect(seed.eligible).toBe(false);
    expect(seed.eligibilityMessage).toMatch(/scrape-first/i);
  });

  it("seeds future-job export intent instead of replay semantics", () => {
    const seed = buildExportSchedulePromotionSeed(makeJob(), "md");

    expect(seed.formData.filterJobKinds).toEqual(["scrape"]);
    expect(seed.formData.filterJobStatus).toEqual(["succeeded"]);
    expect(seed.formData.format).toBe("md");
    expect(seed.unsupportedCarryForward[0]).toMatch(
      /do not rerun this source job/i,
    );
  });
});
