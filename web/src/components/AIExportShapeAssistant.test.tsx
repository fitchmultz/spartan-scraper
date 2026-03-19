/**
 * Purpose: Verify the export-shape AI modal guards required inputs and self-defends when AI assistance is unavailable.
 * Responsibilities: Mock the AI export-shape API, exercise explicit apply flows, and cover unavailable-state rendering.
 * Scope: Component coverage for `AIExportShapeAssistant` only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Generated shapes are only applied after an operator click, and unavailable AI should disable generation clearly.
 */

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import * as api from "../api";
import { AIExportShapeAssistant } from "./AIExportShapeAssistant";

vi.mock("../api", () => ({
  aiExportShape: vi.fn(),
}));

describe("AIExportShapeAssistant", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls aiExportShape and applies the returned shape", async () => {
    const onApplyShape = vi.fn();
    const onClose = vi.fn();

    vi.mocked(api.aiExportShape).mockResolvedValue({
      data: {
        issues: ["current shape is empty"],
        inputStats: {
          fieldOptionCount: 8,
          topLevelFieldCount: 4,
          normalizedFieldCount: 2,
          evidenceFieldCount: 0,
          sampleRecordCount: 1,
        },
        shape: {
          topLevelFields: ["url", "title"],
          normalizedFields: ["field.price"],
          summaryFields: ["title", "field.price"],
          fieldLabels: {
            "field.price": "Price",
          },
          formatting: {
            emptyValue: "—",
            markdownTitle: "Pricing Export",
          },
        },
        explanation: "Focused the export on operator-facing pricing fields.",
        route_id: "openai/gpt-5.4",
        provider: "openai",
        model: "gpt-5.4",
      },
      request: new Request("http://localhost:8741/v1/ai/export-shape"),
      response: new Response(),
    });

    render(
      <AIExportShapeAssistant
        isOpen
        onClose={onClose}
        format="md"
        currentShape={{ topLevelFields: ["url"] }}
        onApplyShape={onApplyShape}
      />,
    );

    fireEvent.change(screen.getByLabelText(/representative job id/i), {
      target: { value: "job-123" },
    });
    fireEvent.change(screen.getByLabelText(/instructions/i), {
      target: { value: "Prioritize pricing fields" },
    });
    fireEvent.click(screen.getByRole("button", { name: /generate shape/i }));

    await waitFor(() => {
      expect(api.aiExportShape).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        body: {
          job_id: "job-123",
          format: "md",
          currentShape: {
            topLevelFields: ["url"],
          },
          instructions: "Prioritize pricing fields",
        },
      });
    });

    expect(await screen.findByText(/Generated Shape/i)).toBeInTheDocument();
    expect(screen.getByText(/current shape is empty/i)).toBeInTheDocument();
    expect(screen.getByText(/Pricing Export/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /apply shape/i }));

    expect(onApplyShape).toHaveBeenCalledWith({
      topLevelFields: ["url", "title"],
      normalizedFields: ["field.price"],
      summaryFields: ["title", "field.price"],
      fieldLabels: {
        "field.price": "Price",
      },
      formatting: {
        emptyValue: "—",
        markdownTitle: "Pricing Export",
      },
    });
    expect(onClose).toHaveBeenCalled();
  });

  it("requires a representative job id", async () => {
    render(
      <AIExportShapeAssistant
        isOpen
        onClose={vi.fn()}
        format="csv"
        onApplyShape={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /generate shape/i }));

    expect(
      await screen.findByText(/representative job id is required/i),
    ).toBeInTheDocument();
    expect(api.aiExportShape).not.toHaveBeenCalled();
  });

  it("shows an unavailable notice and disables generation when ai is off", () => {
    render(
      <AIExportShapeAssistant
        isOpen
        onClose={vi.fn()}
        aiStatus={{
          status: "disabled",
          message: "AI helpers are disabled.",
        }}
        format="csv"
        onApplyShape={vi.fn()}
      />,
    );

    expect(screen.getByText(/AI helpers are disabled\./i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /generate shape/i }),
    ).toBeDisabled();
  });
});
