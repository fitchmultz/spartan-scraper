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
});
