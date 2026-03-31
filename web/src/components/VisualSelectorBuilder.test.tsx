/**
 * Purpose: Verify user-visible validation, invalidation, and error handling in the visual template authoring flow.
 * Responsibilities: Assert fetch, selector-test, and save failures use normalized API error copy, and confirm stale preview state does not survive input changes.
 * Scope: `VisualSelectorBuilder` behavior only.
 * Usage: Run under Vitest as part of the web test suite.
 * Invariants/Assumptions: API calls are mocked, formatted error copy should surface instead of raw object stringification, and stale async selector-test responses must be ignored.
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

  it("exposes the fetch action with an accessible name", () => {
    render(<VisualSelectorBuilder onSave={vi.fn()} onCancel={vi.fn()} />);

    expect(
      screen.getByRole("button", { name: /^fetch page$/i }),
    ).toBeInTheDocument();
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

  it("trims URLs and clears stale selector state after refetching", async () => {
    vi.mocked(api.getTemplatePreview)
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
      })
      .mockResolvedValueOnce({
        data: {
          dom_tree: {
            tag: "main",
            id: "content",
            path: "main",
            depth: 0,
            children: [],
          },
        } as never,
        error: undefined as never,
        request: new Request("http://127.0.0.1:8741/v1/template-preview"),
        response: new Response(),
      });

    vi.mocked(api.testSelector).mockResolvedValue({
      data: {
        selector: "article",
        matches: 1,
        elements: [{ tag: "article", text: "Story", path: "article" }],
      } as never,
      error: undefined as never,
      request: new Request(
        "http://127.0.0.1:8741/v1/template-preview/test-selector",
      ),
      response: new Response(),
    });

    render(<VisualSelectorBuilder onSave={vi.fn()} onCancel={vi.fn()} />);

    fireEvent.change(screen.getByLabelText(/url to analyze/i), {
      target: { value: " https://example.com/article " },
    });
    fireEvent.click(screen.getByText(/^Fetch Page$/));

    await waitFor(() => {
      expect(api.getTemplatePreview).toHaveBeenNthCalledWith(
        1,
        expect.objectContaining({
          query: expect.objectContaining({
            url: "https://example.com/article",
          }),
        }),
      );
    });

    fireEvent.click(
      await screen.findByRole("button", { name: /select article element/i }),
    );
    fireEvent.click(screen.getByRole("button", { name: /^test$/i }));

    expect(await screen.findByText(/1 match/i)).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/url to analyze/i), {
      target: { value: " https://example.com/next " },
    });

    await waitFor(() => {
      expect(
        screen.queryByLabelText(/generated selector/i),
      ).not.toBeInTheDocument();
    });
    expect(screen.queryByText(/1 match/i)).not.toBeInTheDocument();

    fireEvent.click(screen.getByText(/^Fetch Page$/));

    await waitFor(() => {
      expect(api.getTemplatePreview).toHaveBeenNthCalledWith(
        2,
        expect.objectContaining({
          query: expect.objectContaining({ url: "https://example.com/next" }),
        }),
      );
    });
    expect(
      await screen.findByRole("button", { name: /select main element/i }),
    ).toBeInTheDocument();
  });

  it("ignores stale selector test responses after the selector changes", async () => {
    vi.mocked(api.getTemplatePreview).mockResolvedValue({
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

    let resolveSelectorTest: ((value: unknown) => void) | undefined;
    vi.mocked(api.testSelector).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveSelectorTest = resolve;
        }) as never,
    );

    render(<VisualSelectorBuilder onSave={vi.fn()} onCancel={vi.fn()} />);

    fireEvent.change(screen.getByLabelText(/url to analyze/i), {
      target: { value: "https://example.com/article" },
    });
    fireEvent.click(screen.getByText(/^Fetch Page$/));
    fireEvent.click(
      await screen.findByRole("button", { name: /select article element/i }),
    );
    fireEvent.click(screen.getByRole("button", { name: /^test$/i }));

    fireEvent.change(screen.getByLabelText(/generated selector/i), {
      target: { value: "main" },
    });

    resolveSelectorTest?.({
      data: {
        selector: "article",
        matches: 1,
        elements: [{ tag: "article", text: "Story", path: "article" }],
      },
      error: undefined,
      request: new Request(
        "http://127.0.0.1:8741/v1/template-preview/test-selector",
      ),
      response: new Response(),
    });

    await waitFor(() => {
      expect(screen.queryByText(/1 match/i)).not.toBeInTheDocument();
    });
    expect(screen.getByDisplayValue("main")).toBeInTheDocument();
  });

  it("keeps selector-row focus stable while editing", () => {
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

    const fieldInput = screen.getByDisplayValue("title");
    fieldInput.focus();
    expect(document.activeElement).toBe(fieldInput);

    fireEvent.change(fieldInput, {
      target: { value: "headline" },
    });

    expect(screen.getByDisplayValue("headline")).toBe(document.activeElement);
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
