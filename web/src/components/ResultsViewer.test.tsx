import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { ResearchResultItem } from "../types";
import { ResultsViewer } from "./ResultsViewer";

describe("ResultsViewer agentic research rendering", () => {
  it("renders bounded agentic research details for research results", () => {
    const result: ResearchResultItem = {
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

    const agenticPanel = screen
      .getByRole("heading", { name: "Agentic Research" })
      .closest(".panel");

    expect(agenticPanel).not.toBeNull();
    const panel = within(agenticPanel as HTMLElement);

    expect(panel.getByText(/Status completed/)).toBeInTheDocument();
    expect(
      panel.getByText(
        "The vendor uses enterprise pricing and provides dedicated SLA-backed support.",
      ),
    ).toBeInTheDocument();
    expect(panel.getByText("Round 1")).toBeInTheDocument();
    expect(
      panel.getByRole("link", { name: "https://example.com/support" }),
    ).toBeInTheDocument();
  });
});
