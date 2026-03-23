/**
 * Purpose: Verify user-visible error handling in the visual template authoring flow.
 * Responsibilities: Assert fetch, selector-test, and save failures use normalized API error copy.
 * Scope: `VisualSelectorBuilder` behavior only.
 * Usage: Run under Vitest as part of the web test suite.
 * Invariants/Assumptions: API calls are mocked, and formatted error copy should surface instead of raw object stringification.
 */

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import * as api from "../api";
import { VisualSelectorBuilder } from "./VisualSelectorBuilder";

vi.mock("../api", () => ({
  createTemplate: vi.fn(),
  getTemplatePreview: vi.fn(),
  testSelector: vi.fn(),
  updateTemplate: vi.fn(),
}));

describe("VisualSelectorBuilder", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows formatted fetch and selector-test errors", async () => {
    vi.mocked(api.getTemplatePreview)
      .mockResolvedValueOnce({
        data: undefined as never,
        error: { message: "Preview request failed" } as never,
        request: new Request("http://127.0.0.1:8741/v1/template-preview"),
        response: new Response(),
      })
      .mockResolvedValueOnce({
        data: {
          dom_tree: {
            tag: "article",
            id: "story",
            path: "article",
            depth: 0,
            children: [],
          },
        } as never,
        error: undefined as never,
        request: new Request("http://127.0.0.1:8741/v1/template-preview"),
        response: new Response(),
      });

    vi.mocked(api.testSelector).mockResolvedValue({
      data: undefined as never,
      error: { detail: "Selector test failed" } as never,
      request: new Request(
        "http://127.0.0.1:8741/v1/template-preview/test-selector",
      ),
      response: new Response(),
    });

    render(<VisualSelectorBuilder onSave={vi.fn()} onCancel={vi.fn()} />);

    fireEvent.change(screen.getByLabelText(/url to analyze/i), {
      target: { value: "https://example.com/article" },
    });
    fireEvent.click(screen.getByText(/^Fetch Page$/));

    expect(
      await screen.findByText("Preview request failed"),
    ).toBeInTheDocument();
    expect(screen.queryByText("[object Object]")).not.toBeInTheDocument();

    fireEvent.click(screen.getByText(/^Fetch Page$/));

    const selectNodeButton = await screen.findByRole("button", {
      name: /select article element/i,
    });
    fireEvent.click(selectNodeButton);
    fireEvent.click(screen.getByRole("button", { name: /^test$/i }));

    expect(await screen.findByText("Selector test failed")).toBeInTheDocument();
    expect(screen.queryByText("[object Object]")).not.toBeInTheDocument();
  });

  it("shows formatted save errors", async () => {
    vi.mocked(api.updateTemplate).mockResolvedValue({
      data: undefined as never,
      error: { error: "Template save blocked" } as never,
      request: new Request("http://127.0.0.1:8741/v1/templates/article"),
      response: new Response(),
    });

    render(
      <VisualSelectorBuilder
        initialTemplate={{
          name: "article",
          selectors: [{ name: "title", selector: "h1", attr: "text" }],
        }}
        onSave={vi.fn()}
        onCancel={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /save template/i }));

    await waitFor(() => {
      expect(api.updateTemplate).toHaveBeenCalledWith(
        expect.objectContaining({
          path: { name: "article" },
        }),
      );
    });
    expect(
      await screen.findByText("Template save blocked"),
    ).toBeInTheDocument();
    expect(screen.queryByText("[object Object]")).not.toBeInTheDocument();
  });
});
