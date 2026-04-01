/**
 * Purpose: Verify promotion and inline authoring entry points for TemplateManager.
 * Responsibilities: Cover promotion-seeded drafts, inline editing, and delete-visibility rules for local drafts.
 * Scope: Route-level TemplateManager promotion and basic authoring behavior only.
 * Usage: Run under Vitest as part of the web test suite.
 * Invariants/Assumptions: API calls stay mocked, built-in templates remain non-destructive, and promotion seeds preserve canonical workspace copy.
 */

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  buildGetTemplateResponse,
  buildGuidedBlankPromotionSeed,
  buildNamedTemplatePromotionSeed,
  renderTemplateManager,
  setupTemplateManagerTest,
  getTemplateApiMocks,
} from "./templateManagerTestHarness";

setupTemplateManagerTest();

describe("TemplateManager", () => {
  it("duplicates a named-template promotion seed into the workspace", async () => {
    getTemplateApiMocks().getTemplate.mockResolvedValue(
      buildGetTemplateResponse({
        name: "article",
        isBuiltIn: true,
        selector: "article h1",
      }),
    );

    await renderTemplateManager({
      templateNames: ["article"],
      promotionSeed: buildNamedTemplatePromotionSeed(),
    });

    expect(await screen.findByLabelText(/template name/i)).toHaveValue(
      "article-copy",
    );
    expect(
      screen.getByText(/template draft seeded from a verified job/i),
    ).toBeInTheDocument();
    expect(screen.getByDisplayValue("article h1")).toBeInTheDocument();
  });

  it("guides a blank promoted template draft to the first reusable rule", async () => {
    getTemplateApiMocks().createTemplate.mockResolvedValue({
      data: undefined,
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates"),
      response: new Response(),
    });

    await renderTemplateManager({
      promotionSeed: buildGuidedBlankPromotionSeed(),
    });

    expect(await screen.findByLabelText(/template name/i)).toHaveValue(
      "example-com-template",
    );
    expect(
      screen.getByText(/it did not include reusable selector rules/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/save stays disabled until you finish/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/name the first selector rule/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/add a css selector for the first selector rule/i),
    ).toBeInTheDocument();

    const saveButton = screen.getByRole("button", { name: /save template/i });
    expect(saveButton).toBeDisabled();

    fireEvent.click(screen.getByRole("button", { name: /use title starter/i }));

    expect(screen.getByLabelText(/field name/i)).toHaveValue("title");
    expect(screen.getByLabelText(/css selector/i)).toHaveValue("h1");
    await waitFor(() => {
      expect(saveButton).toBeEnabled();
    });

    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(getTemplateApiMocks().createTemplate).toHaveBeenCalledWith(
        expect.objectContaining({
          body: expect.objectContaining({
            name: "example-com-template",
            selectors: [
              expect.objectContaining({ name: "title", selector: "h1" }),
            ],
          }),
        }),
      );
    });
  });

  it("renders a custom template as an inline editable workspace", async () => {
    getTemplateApiMocks().getTemplate.mockResolvedValue(
      buildGetTemplateResponse({
        name: "custom-news",
        selector: "h1",
      }),
    );
    getTemplateApiMocks().updateTemplate.mockResolvedValue({
      data: undefined,
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates/custom-news"),
      response: new Response(),
    });

    await renderTemplateManager({ templateNames: ["custom-news"] });

    expect(await screen.findByLabelText(/template name/i)).toHaveValue(
      "custom-news",
    );
    expect(screen.queryByText(/modal-overlay/i)).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /^delete$/i }),
    ).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/css selector/i), {
      target: { value: "main h1" },
    });

    await waitFor(() => {
      expect(screen.getByLabelText(/css selector/i)).toHaveValue("main h1");
    });

    fireEvent.click(screen.getByRole("button", { name: /save template/i }));

    await waitFor(() => {
      expect(getTemplateApiMocks().updateTemplate).toHaveBeenCalledWith(
        expect.objectContaining({
          path: { name: "custom-news" },
          body: expect.objectContaining({
            name: "custom-news",
            selectors: [
              expect.objectContaining({
                name: "title",
                selector: "main h1",
              }),
            ],
          }),
        }),
      );
    });
  });

  it("does not show delete for a new local draft seeded from a saved selection", async () => {
    getTemplateApiMocks().getTemplate.mockResolvedValue(
      buildGetTemplateResponse({
        name: "article",
        isBuiltIn: true,
        selector: "article h1",
      }),
    );

    await renderTemplateManager({ templateNames: ["article"] });

    expect(await screen.findByLabelText(/template name/i)).toHaveValue(
      "article",
    );

    fireEvent.click(screen.getByRole("button", { name: /new template/i }));

    await waitFor(() => {
      expect(
        screen.getByRole("heading", { name: /new template/i }),
      ).toBeInTheDocument();
    });
    expect(
      screen.queryByRole("button", { name: /^delete$/i }),
    ).not.toBeInTheDocument();
  });

  it("does not show delete for a duplicated built-in draft", async () => {
    getTemplateApiMocks().getTemplate.mockResolvedValue(
      buildGetTemplateResponse({
        name: "article",
        isBuiltIn: true,
        selector: "article h1",
      }),
    );

    await renderTemplateManager({ templateNames: ["article"] });

    expect(await screen.findByLabelText(/template name/i)).toHaveValue(
      "article",
    );

    fireEvent.click(screen.getByRole("button", { name: /duplicate to edit/i }));

    await waitFor(() => {
      expect(screen.getByLabelText(/template name/i)).toHaveValue(
        "article-copy",
      );
    });
    expect(
      screen.queryByRole("button", { name: /^delete$/i }),
    ).not.toBeInTheDocument();
  });
});
