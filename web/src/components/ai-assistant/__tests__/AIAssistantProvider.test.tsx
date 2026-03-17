/**
 * Purpose: Verify the AI assistant provider deduplicates semantically identical context updates.
 * Responsibilities: Reproduce equivalent route-context pushes and assert the provider does not rerender consumers when the meaning of the context is unchanged.
 * Scope: Provider-level context state behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Route adapters may construct new context objects for the same semantic state, so the provider must ignore equivalent updates to avoid rerender loops.
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AIAssistantProvider } from "../AIAssistantProvider";
import { useAIAssistant } from "../useAIAssistant";

let renderCount = 0;

function EquivalentContextHarness() {
  const { context, setContext } = useAIAssistant();
  renderCount += 1;

  return (
    <div>
      <button
        type="button"
        onClick={() => {
          setContext({
            surface: "results",
            jobId: "job-123",
            resultFormat: "jsonl",
            selectedResultIndex: 0,
            resultSummary: "Summary",
          });
        }}
      >
        Set first context
      </button>
      <button
        type="button"
        onClick={() => {
          setContext({
            surface: "results",
            jobId: "job-123",
            resultFormat: "jsonl",
            selectedResultIndex: 0,
            resultSummary: "Summary",
          });
        }}
      >
        Set equivalent context
      </button>
      <output aria-label="assistant context">
        {context?.surface === "results" ? context.jobId : "none"}
      </output>
      <output aria-label="render count">{renderCount}</output>
    </div>
  );
}

describe("AIAssistantProvider", () => {
  it("does not rerender consumers for equivalent context objects", () => {
    renderCount = 0;

    render(
      <AIAssistantProvider>
        <EquivalentContextHarness />
      </AIAssistantProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: /set first context/i }));

    expect(screen.getByLabelText("assistant context")).toHaveTextContent(
      "job-123",
    );
    expect(screen.getByLabelText("render count")).toHaveTextContent("2");

    fireEvent.click(
      screen.getByRole("button", { name: /set equivalent context/i }),
    );

    expect(screen.getByLabelText("render count")).toHaveTextContent("2");
  });
});
