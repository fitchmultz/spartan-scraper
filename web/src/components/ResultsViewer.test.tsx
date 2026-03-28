/**
 * Purpose: Verify the result viewer keeps research detail readable while demoting deeper inspection behind disclosures.
 * Responsibilities: Assert agentic research content remains reachable from the new primary-reader layout.
 * Scope: Component coverage for `ResultsViewer`.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Agentic detail is secondary to the main research summary but must remain accessible without route changes.
 */

import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { ResearchResultItem } from "../types";
import { ResultsViewer } from "./ResultsViewer";

describe("ResultsViewer agentic research rendering", () => {
  it("keeps agentic research details available behind a secondary disclosure", async () => {
    const user = userEvent.setup();

    const result: ResearchResultItem = {
      query: "support model",
      summary: "Deterministic summary",
      confidence: 0.73,
      evidence: [],
      clusters: [],
      citations: [],
      agentic: {
        status: "completed",
        summary:
          "The vendor uses enterprise pricing and provides dedicated SLA-backed support.",
        focusAreas: ["pricing model", "support commitments"],
        keyFindings: ["Pricing is handled through enterprise contracts."],
        followUpUrls: ["https://example.com/support"],
        rounds: [
          {
            round: 1,
            goal: "Inspect support details",
            selectedUrls: ["https://example.com/support"],
            addedEvidenceCount: 1,
          },
        ],
        confidence: 0.84,
        provider: "openai",
        model: "gpt-5.4",
      },
    };

    render(
      <ResultsViewer
        jobId="job-123"
        jobKind="research"
        resultItems={[result]}
        selectedResultIndex={0}
        setSelectedResultIndex={vi.fn()}
        resultSummary={result.summary ?? null}
        resultConfidence={result.confidence ?? null}
        resultEvidence={[]}
        resultClusters={[]}
        resultCitations={[]}
        resultAgentic={result.agentic ?? null}
        rawResult={JSON.stringify(result)}
        resultFormat="jsonl"
        currentPage={1}
        totalResults={1}
        resultsPerPage={100}
        onLoadPage={vi.fn()}
      />,
    );

    expect(screen.getAllByText("Deterministic summary").length).toBeGreaterThan(
      0,
    );

    await user.click(screen.getByText(/Agentic research details/i));

    const link = screen.getByRole("link", {
      name: "https://example.com/support",
    });
    expect(link).toBeInTheDocument();

    const disclosure = link.closest("details");
    expect(disclosure).not.toBeNull();

    const detailRegion = within(disclosure as HTMLElement);
    expect(detailRegion.getByText(/Status completed/)).toBeInTheDocument();
    expect(
      detailRegion.getByText(
        "The vendor uses enterprise pricing and provides dedicated SLA-backed support.",
      ),
    ).toBeInTheDocument();
    expect(detailRegion.getByText("Round 1")).toBeInTheDocument();
  });

  it("keeps research navigator copy compact and query-led", () => {
    const longSummary =
      "This summary starts with a readable thesis about how the product positions enterprise support, but it keeps going with extra detail about migrations, procurement, onboarding, and renewal timing that should stay inside the detail panel instead of dominating the navigator card.";
    const firstResult: ResearchResultItem = {
      query: "Vendor support differences",
      summary: longSummary,
      confidence: 0.61,
      evidence: [
        {
          url: "https://example.com/support",
          title: "Support overview",
          snippet: "Support summary",
          score: 12,
        },
        {
          url: "https://example.com/pricing",
          title: "Pricing overview",
          snippet: "Pricing summary",
          score: 8,
        },
      ],
      clusters: [
        {
          id: "cluster-1",
          label: "Support",
          confidence: 0.82,
          evidence: [],
        },
      ],
      citations: [
        {
          canonical: "https://example.com/support",
        },
      ],
    };
    const selectedResult: ResearchResultItem = {
      query: "Selected research result",
      summary: "Stable selected summary",
      confidence: 0.74,
      evidence: [],
      clusters: [],
      citations: [],
    };

    render(
      <ResultsViewer
        jobId="job-123"
        jobKind="research"
        resultItems={[firstResult, selectedResult]}
        selectedResultIndex={1}
        setSelectedResultIndex={vi.fn()}
        resultSummary={selectedResult.summary ?? null}
        resultConfidence={selectedResult.confidence ?? null}
        resultEvidence={selectedResult.evidence ?? []}
        resultClusters={selectedResult.clusters ?? []}
        resultCitations={selectedResult.citations ?? []}
        resultAgentic={selectedResult.agentic ?? null}
        rawResult={JSON.stringify([firstResult, selectedResult])}
        resultFormat="jsonl"
        currentPage={1}
        totalResults={2}
        resultsPerPage={100}
        onLoadPage={vi.fn()}
      />,
    );

    const navigatorButton = screen.getByRole("button", {
      name: /Vendor support differences/i,
    });
    const navigatorRegion = within(navigatorButton);

    expect(
      navigatorRegion.getByText(/This summary starts with a readable thesis/i),
    ).toBeInTheDocument();
    expect(navigatorRegion.queryByText(longSummary)).not.toBeInTheDocument();
    expect(
      navigatorRegion.getByText("2 evidence · 1 clusters · 1 citations"),
    ).toBeInTheDocument();
  });
});
