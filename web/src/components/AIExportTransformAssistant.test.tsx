/**
 * Purpose: Verify the export-transform AI modal guards required inputs and self-defends when AI assistance is unavailable.
 * Responsibilities: Mock the AI transform API, exercise explicit apply flows, and cover unavailable-state rendering.
 * Scope: Component coverage for `AIExportTransformAssistant` only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Generated transforms are only applied after an operator click, and unavailable AI should disable generation clearly.
 */

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import * as api from "../api";
import { AIExportTransformAssistant } from "./AIExportTransformAssistant";

vi.mock("../api", () => ({
  aiTransformGenerate: vi.fn(),
}));

describe("AIExportTransformAssistant", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls aiTransformGenerate and applies the returned transform", async () => {
    const onApplyTransform = vi.fn();
    const onClose = vi.fn();

    vi.mocked(api.aiTransformGenerate).mockResolvedValue({
      data: {
        issues: ["current transform is invalid: invalid transform.expression"],
        inputStats: {
          sampleRecordCount: 2,
          fieldPathCount: 5,
          currentTransformProvided: true,
        },
        transform: {
          expression: "{title: title, url: url}",
          language: "jmespath",
        },
        preview: [{ title: "Example", url: "https://example.com" }],
        explanation: "Projected the title and URL fields for export.",
        route_id: "openai/gpt-5.4",
        provider: "openai",
        model: "gpt-5.4",
      },
      request: new Request("http://localhost:8741/v1/ai/transform-generate"),
      response: new Response(),
    });

    render(
      <AIExportTransformAssistant
        isOpen
        onClose={onClose}
        currentTransform={{ expression: "[", language: "jmespath" }}
        onApplyTransform={onApplyTransform}
      />,
    );

    fireEvent.change(screen.getByLabelText(/representative job id/i), {
      target: { value: "job-123" },
    });
    fireEvent.change(screen.getByLabelText(/instructions/i), {
      target: { value: "Project the URL and title for export" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: /generate transform/i }),
    );

    await waitFor(() => {
      expect(api.aiTransformGenerate).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        body: {
          job_id: "job-123",
          currentTransform: {
            expression: "[",
            language: "jmespath",
          },
          preferredLanguage: "jmespath",
          instructions: "Project the URL and title for export",
        },
      });
    });

    expect(await screen.findByText(/generated transform/i)).toBeInTheDocument();
    expect(
      screen.getByText(/current transform is invalid/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/sample records 2/i)).toBeInTheDocument();
    expect(screen.getByText(/field paths 5/i)).toBeInTheDocument();
    expect(
      screen.getByText(/projected the title and url fields/i),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /apply transform/i }));

    expect(onApplyTransform).toHaveBeenCalledWith({
      expression: "{title: title, url: url}",
      language: "jmespath",
    });
    expect(onClose).toHaveBeenCalled();
  });

  it("requires a representative job id", async () => {
    render(
      <AIExportTransformAssistant
        isOpen
        onClose={vi.fn()}
        onApplyTransform={vi.fn()}
      />,
    );

    fireEvent.click(
      screen.getByRole("button", { name: /generate transform/i }),
    );

    expect(
      await screen.findByText(/representative job id is required/i),
    ).toBeInTheDocument();
    expect(api.aiTransformGenerate).not.toHaveBeenCalled();
  });

  it("shows an unavailable notice and disables generation when ai is off", () => {
    render(
      <AIExportTransformAssistant
        isOpen
        onClose={vi.fn()}
        aiStatus={{
          status: "disabled",
          message: "AI helpers are disabled.",
        }}
        onApplyTransform={vi.fn()}
      />,
    );

    expect(screen.getByText(/AI helpers are disabled\./i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /generate transform/i }),
    ).toBeDisabled();
  });
});
