/**
 * Purpose: Verify shared diff helpers keep operator-facing comparisons deterministic and safe.
 * Responsibilities: Cover nested equality checks, research diff classification, and AI candidate fallback behavior for unsupported fields.
 * Scope: `diff-utils` only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Research evidence matches by URL, cluster changes match by ID, and unsupported AI candidate fields should force raw JSON fallback instead of partial summaries.
 */

import { describe, expect, it } from "vitest";

import type { RenderProfile } from "../api";
import type { ResearchResultItem } from "../types";
import {
  deepEqual,
  diffResearchResults,
  summarizeRenderProfileCandidateDiff,
} from "./diff-utils";

describe("diff-utils", () => {
  it("compares nested arrays and objects deterministically", () => {
    expect(
      deepEqual(
        {
          title: "Example",
          links: ["/a", "/b"],
          metadata: { status: 200, tags: ["news"] },
        },
        {
          title: "Example",
          links: ["/a", "/b"],
          metadata: { status: 200, tags: ["news"] },
        },
      ),
    ).toBe(true);

    expect(
      deepEqual(
        { title: "Example", metadata: { status: 200 } },
        { title: "Example", metadata: { status: 404 } },
      ),
    ).toBe(false);
  });

  it("classifies research evidence, cluster, citation, and summary changes", () => {
    const before: ResearchResultItem = {
      summary: "Initial summary",
      confidence: 0.52,
      evidence: [
        {
          url: "https://example.com/pricing",
          title: "Pricing",
          snippet: "Contact sales",
          score: 0.4,
        },
      ],
      clusters: [
        {
          id: "cluster-1",
          label: "Pricing",
          confidence: 0.6,
          evidence: [],
        },
      ],
      citations: [{ canonical: "https://example.com/pricing" }],
    };

    const after: ResearchResultItem = {
      summary: "Updated summary",
      confidence: 0.93,
      evidence: [
        {
          url: "https://example.com/pricing",
          title: "Pricing",
          snippet: "Enterprise pricing now published",
          score: 0.8,
        },
        {
          url: "https://example.com/support",
          title: "Support",
          snippet: "24/7 support",
          score: 0.7,
        },
      ],
      clusters: [
        {
          id: "cluster-1",
          label: "Pricing and packaging",
          confidence: 0.8,
          evidence: [],
        },
        {
          id: "cluster-2",
          label: "Support",
          confidence: 0.7,
          evidence: [],
        },
      ],
      citations: [
        { canonical: "https://example.com/pricing" },
        { canonical: "", url: "https://example.com/support" },
      ],
    };

    const diff = diffResearchResults(before, after);

    expect(diff.summaryChanges).toEqual({
      oldValue: "Initial summary",
      newValue: "Updated summary",
    });
    expect(diff.confidenceChanges).toEqual({
      oldValue: 0.52,
      newValue: 0.93,
    });
    expect(diff.evidenceModified).toHaveLength(1);
    expect(diff.evidenceModified[0]?.url).toBe("https://example.com/pricing");
    expect(diff.evidenceAdded).toEqual([
      expect.objectContaining({ url: "https://example.com/support" }),
    ]);
    expect(diff.clustersModified).toEqual([
      expect.objectContaining({ id: "cluster-1" }),
    ]);
    expect(diff.clustersAdded).toEqual([
      expect.objectContaining({ id: "cluster-2" }),
    ]);
    expect(diff.citationsAdded).toEqual([
      expect.objectContaining({ url: "https://example.com/support" }),
    ]);
  });

  it("falls back to raw JSON when unsupported render-profile fields change", () => {
    const previous: RenderProfile = {
      name: "default",
      hostPatterns: ["example.com"],
      wait: { mode: "dom_ready" },
    };
    const latest: RenderProfile = {
      name: "default",
      hostPatterns: ["example.com"],
      wait: { mode: "selector", selector: "main" },
      assumeJsHeavy: true,
    };

    const summary = summarizeRenderProfileCandidateDiff(previous, latest);

    expect(summary.changes).toEqual([
      expect.objectContaining({
        field: "Wait mode",
        path: "wait.mode",
        oldValue: "dom_ready",
        newValue: "selector",
      }),
      expect.objectContaining({
        field: "Wait selector",
        path: "wait.selector",
        oldValue: undefined,
        newValue: "main",
      }),
    ]);
    expect(summary.latestFields).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ label: "Profile name", value: "default" }),
        expect.objectContaining({ label: "Wait mode", value: "selector" }),
      ]),
    );
    expect(summary.shouldShowRawJsonByDefault).toBe(true);
    expect(summary.rawJsonReason).toContain("assumeJsHeavy");
  });
});
