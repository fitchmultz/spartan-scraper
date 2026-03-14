import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AIResearchRefiner } from "./AIResearchRefiner";
import * as api from "../api";

vi.mock("../api", () => ({
  aiResearchRefine: vi.fn(),
}));

describe("AIResearchRefiner", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls aiResearchRefine and renders the structured refinement", async () => {
    vi.mocked(api.aiResearchRefine).mockResolvedValue({
      data: {
        issues: ["research query is missing"],
        inputStats: {
          evidenceCount: 1,
          evidenceUsedCount: 1,
          clusterCount: 0,
          citationCount: 1,
          hasAgentic: false,
        },
        refined: {
          summary:
            "Enterprise pricing appears to be sales-led, with support commitments documented across the supplied evidence.",
          conciseSummary:
            "Sales-led pricing with documented support commitments.",
          keyFindings: [
            "Pricing is handled through direct sales rather than self-serve checkout.",
          ],
          evidenceHighlights: [
            {
              url: "https://example.com/pricing",
              title: "Pricing",
              finding: "The pricing page routes buyers to contact sales.",
            },
          ],
          recommendedNextSteps: [
            "Confirm final SLA language with the vendor sales team.",
          ],
          confidence: 0.81,
        },
        markdown: "# Refined Research Brief\n",
        explanation:
          "Condensed the supplied research result into an operator brief.",
        route_id: "openai/gpt-5.4",
        provider: "openai",
        model: "gpt-5.4",
      },
      request: new Request("http://localhost:8741/v1/ai/research-refine"),
      response: new Response(),
    });

    render(
      <AIResearchRefiner
        isOpen
        onClose={vi.fn()}
        result={{
          query: "pricing and support commitments",
          summary: "Original summary",
          confidence: 0.78,
          evidence: [
            {
              url: "https://example.com/pricing",
              title: "Pricing",
              snippet: "Contact sales for enterprise pricing.",
              score: 0.91,
              citationUrl: "https://example.com/pricing",
            },
          ],
          citations: [
            {
              canonical: "https://example.com/pricing",
              url: "https://example.com/pricing",
            },
          ],
          clusters: [],
        }}
      />,
    );

    fireEvent.change(screen.getByLabelText(/instructions/i), {
      target: { value: "Condense this into a concise operator brief" },
    });
    fireEvent.click(screen.getByRole("button", { name: /refine result/i }));

    await waitFor(() => {
      expect(api.aiResearchRefine).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        body: {
          result: expect.objectContaining({
            query: "pricing and support commitments",
          }),
          instructions: "Condense this into a concise operator brief",
        },
      });
    });

    expect(
      await screen.findByText(
        "Sales-led pricing with documented support commitments.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText(/Input diagnostics/i)).toBeInTheDocument();
    expect(screen.getByText(/research query is missing/i)).toBeInTheDocument();
    expect(screen.getByText(/Rendered Markdown/i)).toBeInTheDocument();
  });
});
