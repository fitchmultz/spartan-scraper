/**
 * Purpose: Verify selector preview invalidation behavior in the inline template preview rail.
 * Responsibilities: Ensure stale async preview responses do not overwrite newer editor state.
 * Scope: `TemplatePreviewPane` only.
 * Usage: Run with Vitest as part of the web component suite.
 * Invariants/Assumptions: Preview requests are mocked and changing template inputs should invalidate in-flight preview results.
 */

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import * as api from "../../../api";
import { TemplatePreviewPane } from "../TemplatePreviewPane";

vi.mock("../../../api", () => ({
  testSelector: vi.fn(),
}));

describe("TemplatePreviewPane", () => {
  it("ignores stale preview responses after selector changes", async () => {
    let resolvePreview: ((value: unknown) => void) | undefined;
    vi.mocked(api.testSelector).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolvePreview = resolve;
        }) as never,
    );

    const onUrlChange = vi.fn();
    const firstTemplate = {
      name: "article",
      selectors: [{ name: "title", selector: "h1", attr: "text" }],
    };
    const nextTemplate = {
      name: "article",
      selectors: [{ name: "title", selector: "h2", attr: "text" }],
    };

    const view = render(
      <TemplatePreviewPane
        template={firstTemplate}
        url="https://example.com/article"
        onUrlChange={onUrlChange}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /run preview/i }));

    view.rerender(
      <TemplatePreviewPane
        template={nextTemplate}
        url="https://example.com/article"
        onUrlChange={onUrlChange}
      />,
    );

    resolvePreview?.({
      data: {
        selector: "h1",
        matches: 1,
        elements: [{ tag: "h1", text: "Headline", path: "h1" }],
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
    expect(
      screen.getByText(/run preview to inspect current selector matches/i),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /run preview/i })).toBeEnabled();
  });
});
