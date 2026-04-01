/**
 * Purpose: Verify TemplateManager preview, AI-assist, and builder tools remain wired to the inline workspace.
 * Responsibilities: Cover promotion save-and-preview flows, built-in duplication saves, selector preview, AI generation, and visual-builder handoff.
 * Scope: TemplateManager tool integrations only.
 * Usage: Run under Vitest as part of the web test suite.
 * Invariants/Assumptions: Tool surfaces remain inline, API calls are mocked, and preview state reflects the latest workspace draft.
 */

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  buildGetTemplateResponse,
  buildNamedTemplatePromotionSeed,
  renderTemplateManager,
  setupTemplateManagerTest,
  getTemplateApiMocks,
} from "./templateManagerTestHarness";

setupTemplateManagerTest();

describe("TemplateManagerTools", () => {
  it("saves a promotion-seeded template draft and previews the saved selectors", async () => {
    getTemplateApiMocks().getTemplate.mockImplementation(async ({ path }) => ({
      data:
        path.name === "article-copy"
          ? {
              name: "article-copy",
              is_built_in: false,
              template: {
                name: "article-copy",
                selectors: [
                  { name: "title", selector: "article h1", attr: "text" },
                ],
              },
            }
          : {
              name: "article",
              is_built_in: true,
              template: {
                name: "article",
                selectors: [
                  { name: "title", selector: "article h1", attr: "text" },
                ],
              },
            },
      error: undefined as never,
      request: new Request(`http://127.0.0.1:8741/v1/templates/${path.name}`),
      response: new Response(),
    }));
    getTemplateApiMocks().createTemplate.mockResolvedValue({
      data: {
        name: "article-copy",
        is_built_in: false,
        template: {
          name: "article-copy",
          selectors: [{ name: "title", selector: "article h1", attr: "text" }],
        },
      },
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates"),
      response: new Response(),
    });
    getTemplateApiMocks().testSelector.mockResolvedValue({
      data: {
        selector: "article h1",
        matches: 1,
        elements: [{ tag: "h1", text: "Promoted headline" }],
      },
      error: undefined as never,
      request: new Request(
        "http://127.0.0.1:8741/v1/template-preview/test-selector",
      ),
      response: new Response(),
    });

    await renderTemplateManager({
      templateNames: ["article"],
      promotionSeed: buildNamedTemplatePromotionSeed(),
    });

    expect(await screen.findByLabelText(/template name/i)).toHaveValue(
      "article-copy",
    );
    expect(screen.getByDisplayValue("article h1")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /save template/i }));

    await waitFor(() => {
      expect(getTemplateApiMocks().createTemplate).toHaveBeenCalledWith(
        expect.objectContaining({
          body: expect.objectContaining({
            name: "article-copy",
            selectors: [
              expect.objectContaining({
                name: "title",
                selector: "article h1",
              }),
            ],
          }),
        }),
      );
    });

    await waitFor(() => {
      expect(getTemplateApiMocks().getTemplate).toHaveBeenCalledWith(
        expect.objectContaining({ path: { name: "article-copy" } }),
      );
    });
    expect(screen.getByLabelText(/preview target url/i)).toHaveValue(
      "https://example.com/article",
    );

    fireEvent.click(screen.getByRole("button", { name: /run preview/i }));

    await waitFor(() => {
      expect(getTemplateApiMocks().testSelector).toHaveBeenCalledWith(
        expect.objectContaining({
          body: expect.objectContaining({
            url: "https://example.com/article",
            selector: "article h1",
          }),
        }),
      );
    });

    expect(await screen.findByText(/1 match/i)).toBeInTheDocument();
    expect(screen.getByText(/promoted headline/i)).toBeInTheDocument();
  });

  it("duplicates a built-in template into an editable draft instead of opening a modal", async () => {
    getTemplateApiMocks().getTemplate.mockResolvedValue(
      buildGetTemplateResponse({
        name: "article",
        isBuiltIn: true,
        selector: "article h1",
      }),
    );
    getTemplateApiMocks().createTemplate.mockResolvedValue({
      data: undefined,
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates"),
      response: new Response(),
    });

    await renderTemplateManager({ templateNames: ["article"] });

    await screen.findByRole("button", { name: /duplicate to edit/i });
    fireEvent.click(screen.getByRole("button", { name: /duplicate to edit/i }));

    await waitFor(() => {
      expect(screen.getByLabelText(/template name/i)).toHaveValue(
        "article-copy",
      );
    });

    fireEvent.click(screen.getByRole("button", { name: /save template/i }));

    await waitFor(() => {
      expect(getTemplateApiMocks().createTemplate).toHaveBeenCalledWith(
        expect.objectContaining({
          body: expect.objectContaining({ name: "article-copy" }),
        }),
      );
    });
  });

  it("runs selector preview in the right rail", async () => {
    getTemplateApiMocks().getTemplate.mockResolvedValue(
      buildGetTemplateResponse({
        name: "custom-news",
        selector: "h1",
      }),
    );
    getTemplateApiMocks().testSelector.mockResolvedValue({
      data: {
        selector: "h1",
        matches: 1,
        elements: [{ tag: "h1", text: "Headline" }],
      },
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/test-selector"),
      response: new Response(),
    });

    await renderTemplateManager({ templateNames: ["custom-news"] });

    await screen.findByLabelText(/preview target url/i);

    fireEvent.change(screen.getByLabelText(/preview target url/i), {
      target: { value: "https://example.com/article" },
    });

    fireEvent.click(screen.getByRole("button", { name: /run preview/i }));

    expect(await screen.findByText(/1 match/i)).toBeInTheDocument();
    expect(screen.getByText(/headline/i)).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/preview target url/i), {
      target: { value: "https://example.com/updated" },
    });

    await waitFor(() => {
      expect(screen.queryByText(/1 match/i)).not.toBeInTheDocument();
    });
    expect(
      screen.getByText(/run preview to inspect current selector matches/i),
    ).toBeInTheDocument();
  });

  it("applies AI-generated templates into the inline workspace", async () => {
    getTemplateApiMocks().getTemplate.mockResolvedValue(
      buildGetTemplateResponse({
        name: "custom-news",
        selector: "h1",
      }),
    );
    getTemplateApiMocks().aiTemplateGenerate.mockResolvedValue({
      data: {
        template: {
          name: "generated-template",
          is_built_in: false,
          template: {
            name: "generated-template",
            selectors: [{ name: "price", selector: ".price", attr: "text" }],
          },
        },
        explanation: "Generated a price selector.",
      },
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/ai/template-generate"),
      response: new Response(),
    });

    await renderTemplateManager({ templateNames: ["custom-news"] });

    fireEvent.click(screen.getByRole("button", { name: /^generate$/i }));
    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/product" },
    });
    fireEvent.change(screen.getByLabelText(/description/i), {
      target: { value: "Extract price" },
    });

    fireEvent.click(screen.getByRole("button", { name: /generate template/i }));
    await screen.findByRole("button", { name: /apply to workspace/i });

    fireEvent.click(
      screen.getByRole("button", { name: /apply to workspace/i }),
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/template name/i)).toHaveValue(
        "generated-template",
      );
    });
    expect(screen.getByDisplayValue(".price")).toBeInTheDocument();
  });

  it("opens the visual builder inline inside the workspace", async () => {
    getTemplateApiMocks().getTemplate.mockResolvedValue(
      buildGetTemplateResponse({
        name: "custom-news",
        selector: "h1",
      }),
    );

    await renderTemplateManager({ templateNames: ["custom-news"] });

    await screen.findByText(/custom-news/i);
    fireEvent.click(
      screen.getByRole("button", { name: /open visual builder/i }),
    );

    expect(screen.getByText(/visual builder mock/i)).toBeInTheDocument();
  });
});
