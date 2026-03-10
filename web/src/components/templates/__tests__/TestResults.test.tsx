/**
 * Template A/B test results component tests.
 *
 * Purpose:
 * - Verify the extracted results panel preserves key behaviors.
 *
 * Responsibilities:
 * - Confirm the auto-select action is shown only when appropriate.
 * - Confirm successful auto-select refreshes through the parent callback.
 *
 * Scope:
 * - Unit tests for the template A/B test results panel only.
 *
 * Usage:
 * - Run through Vitest as part of the web component test suite.
 *
 * Invariants/Assumptions:
 * - Auto-select should refresh through callbacks rather than a hard reload.
 */

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TestResults } from "../ab-tests/TestResults";
import * as api from "../../../api";

vi.mock("../../../api", () => ({
  getV1TemplateAbTestsByIdResults: vi.fn(),
  postV1TemplateAbTestsByIdAutoSelect: vi.fn(),
}));

vi.mock("../../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

describe("TestResults", () => {
  const onUpdate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(window, "confirm").mockReturnValue(true);
  });

  it("renders auto-select winner when results are significant and no winner exists", async () => {
    vi.mocked(api.getV1TemplateAbTestsByIdResults).mockResolvedValue({
      data: {
        baseline_template: "baseline",
        variant_template: "variant",
        baseline_metrics: {
          success_rate: 82.5,
          field_coverage: 0.91,
          sample_size: 125,
        },
        variant_metrics: {
          success_rate: 89.1,
          field_coverage: 0.95,
          sample_size: 130,
        },
        statistical_test: {
          p_value: 0.0123,
          is_significant: true,
        },
        recommendation: "Select the variant template.",
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/template-ab-tests/test"),
      response: new Response(),
    });
    vi.mocked(api.postV1TemplateAbTestsByIdAutoSelect).mockResolvedValue({
      data: undefined,
      error: undefined as never,
      request: new Request(
        "http://localhost:8741/v1/template-ab-tests/test/auto-select",
      ),
      response: new Response(),
    });

    render(<TestResults testId="test-id" onUpdate={onUpdate} />);

    expect(await screen.findByText("Test Results")).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: /auto-select winner/i }),
    );

    await waitFor(() => {
      expect(api.postV1TemplateAbTestsByIdAutoSelect).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
        path: { id: "test-id" },
      });
      expect(onUpdate).toHaveBeenCalled();
    });
  });

  it("hides auto-select when a winner already exists", async () => {
    vi.mocked(api.getV1TemplateAbTestsByIdResults).mockResolvedValue({
      data: {
        baseline_template: "baseline",
        variant_template: "variant",
        winner: "variant",
        baseline_metrics: {
          success_rate: 82.5,
          field_coverage: 0.91,
          sample_size: 125,
        },
        variant_metrics: {
          success_rate: 89.1,
          field_coverage: 0.95,
          sample_size: 130,
        },
        statistical_test: {
          p_value: 0.0123,
          is_significant: true,
        },
        recommendation: "Variant already selected.",
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/template-ab-tests/test"),
      response: new Response(),
    });

    render(<TestResults testId="test-id" onUpdate={onUpdate} />);

    expect(await screen.findByText("Test Results")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /auto-select winner/i }),
    ).not.toBeInTheDocument();
  });
});
