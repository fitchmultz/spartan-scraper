/**
 * Purpose: Verify the results assistant can generate export shapes and expose explicit apply actions from the persistent assistant rail.
 * Responsibilities: Mock assistant API responses, mount the assistant inside the shared provider, and assert shape application behavior.
 * Scope: Component coverage for `ResultsAssistantSection` only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Shape generation stays explicit and route-scoped, and the assistant never mutates export state without an apply click.
 */

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import * as api from "../../../api";
import { AIAssistantProvider } from "../AIAssistantProvider";
import { ResultsAssistantSection } from "../ResultsAssistantSection";

vi.mock("../../../api", () => ({
  aiExportShape: vi.fn(),
  aiResearchRefine: vi.fn(),
}));

vi.mock("../../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

describe("ResultsAssistantSection", () => {
  it("generates and applies a shape", async () => {
    const onApplyShape = vi.fn();

    vi.mocked(api.aiExportShape).mockResolvedValue({
      data: {
        shape: {
          summaryFields: ["title", "url"],
          normalizedFields: ["field.price"],
        },
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/ai/export-shape"),
      response: new Response(),
    });

    render(
      <AIAssistantProvider>
        <ResultsAssistantSection
          jobId="job-123"
          jobType="scrape"
          resultFormat="jsonl"
          selectedResultIndex={0}
          resultSummary="Saved output"
          selectedResult={null}
          mode="shape"
          onModeChange={() => {}}
          shapeFormat="md"
          onShapeFormatChange={() => {}}
          currentShape={undefined}
          onApplyShape={onApplyShape}
        />
      </AIAssistantProvider>,
    );

    fireEvent.click(screen.getByText(/generate shape/i));

    await waitFor(() => {
      expect(api.aiExportShape).toHaveBeenCalled();
    });

    fireEvent.click(screen.getByText(/apply shape/i));

    expect(onApplyShape).toHaveBeenCalledWith({
      summaryFields: ["title", "url"],
      normalizedFields: ["field.price"],
    });
  });
});
