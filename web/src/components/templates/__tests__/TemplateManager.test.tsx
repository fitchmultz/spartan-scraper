/**
 * Purpose: Verify the template workspace behavior for inline editing, preview, AI assistance, and builder handoffs.
 * Responsibilities: Cover TemplateManager route-level interactions without relying on live API calls or the real visual builder implementation.
 * Scope: Component behavior for `TemplateManager` only.
 * Usage: Run under Vitest as part of the web test suite.
 * Invariants/Assumptions: API calls are mocked, built-in templates are non-destructive, and the `/templates` route no longer depends on modal-first editing.
 */

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TemplateManager } from "../TemplateManager";
import { AIAssistantProvider } from "../../ai-assistant";
import { ToastProvider } from "../../toast";
import * as api from "../../../api";

vi.mock("../../../api", () => ({
  getTemplate: vi.fn(),
  createTemplate: vi.fn(),
  updateTemplate: vi.fn(),
  deleteTemplate: vi.fn(),
  testSelector: vi.fn(),
  aiTemplateGenerate: vi.fn(),
  aiTemplateDebug: vi.fn(),
}));

vi.mock("../../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://127.0.0.1:8741"),
}));

vi.mock("../../VisualSelectorBuilder", () => ({
  VisualSelectorBuilder: ({
    initialTemplate,
    onSave,
    onCancel,
  }: {
    initialTemplate?: { name?: string };
    onSave: (template: { name?: string }) => void;
    onCancel: () => void;
  }) => (
    <div>
      <div>Visual Builder Mock</div>
      <div>{initialTemplate?.name ?? "new template"}</div>
      <button
        type="button"
        onClick={() =>
          onSave({ name: initialTemplate?.name ?? "builder-saved" })
        }
      >
        Save Builder
      </button>
      <button type="button" onClick={onCancel}>
        Cancel Builder
      </button>
    </div>
  ),
}));

describe("TemplateManager", () => {
  const onTemplatesChanged = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    if (typeof window.localStorage.clear === "function") {
      window.localStorage.clear();
    }
    if (typeof window.sessionStorage.clear === "function") {
      window.sessionStorage.clear();
    }
    vi.stubGlobal(
      "confirm",
      vi.fn(() => true),
    );
  });

  it("duplicates a named-template promotion seed into the workspace", async () => {
    vi.mocked(api.getTemplate).mockResolvedValue({
      data: {
        name: "article",
        is_built_in: true,
        template: {
          name: "article",
          selectors: [{ name: "title", selector: "article h1", attr: "text" }],
        },
      },
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates/article"),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <AIAssistantProvider>
          <TemplateManager
            templateNames={["article"]}
            onTemplatesChanged={onTemplatesChanged}
            promotionSeed={{
              kind: "template",
              mode: "named-template",
              source: {
                jobId: "job-123",
                jobKind: "scrape",
                jobStatus: "succeeded",
                label: "Source URL",
                value: "https://example.com/article",
              },
              suggestedName: "article-copy",
              previewUrl: "https://example.com/article",
              templateName: "article",
              carriedForward: [
                "The saved extraction rules from template “article”.",
              ],
              remainingDecisions: [
                "Review the duplicated template name before saving.",
              ],
              unsupportedCarryForward: [
                "Runtime execution settings and auth do not become part of the duplicated template automatically.",
              ],
            }}
          />
        </AIAssistantProvider>
      </ToastProvider>,
    );

    expect(await screen.findByLabelText(/template name/i)).toHaveValue(
      "article-copy",
    );
    expect(
      screen.getByText(/template draft seeded from a verified job/i),
    ).toBeInTheDocument();
    expect(screen.getByDisplayValue("article h1")).toBeInTheDocument();
  });

  it("renders a custom template as an inline editable workspace", async () => {
    vi.mocked(api.getTemplate).mockResolvedValue({
      data: {
        name: "custom-news",
        is_built_in: false,
        template: {
          name: "custom-news",
          selectors: [{ name: "title", selector: "h1", attr: "text" }],
        },
      },
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates/custom-news"),
      response: new Response(),
    });

    vi.mocked(api.updateTemplate).mockResolvedValue({
      data: undefined,
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates/custom-news"),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <AIAssistantProvider>
          <TemplateManager
            templateNames={["custom-news"]}
            onTemplatesChanged={onTemplatesChanged}
          />
        </AIAssistantProvider>
      </ToastProvider>,
    );

    expect(await screen.findByLabelText(/template name/i)).toHaveValue(
      "custom-news",
    );
    expect(screen.queryByText(/modal-overlay/i)).not.toBeInTheDocument();

    fireEvent.change(screen.getByDisplayValue("h1"), {
      target: { value: "main h1" },
    });

    fireEvent.click(screen.getByRole("button", { name: /save template/i }));

    await waitFor(() => {
      expect(api.updateTemplate).toHaveBeenCalledWith(
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

  it("saves a promotion-seeded template draft and previews the saved selectors", async () => {
    vi.mocked(api.getTemplate).mockImplementation(async ({ path }) => ({
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

    vi.mocked(api.createTemplate).mockResolvedValue({
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

    vi.mocked(api.testSelector).mockResolvedValue({
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

    render(
      <ToastProvider>
        <AIAssistantProvider>
          <TemplateManager
            templateNames={["article"]}
            onTemplatesChanged={onTemplatesChanged}
            promotionSeed={{
              kind: "template",
              mode: "named-template",
              source: {
                jobId: "job-123",
                jobKind: "scrape",
                jobStatus: "succeeded",
                label: "Source URL",
                value: "https://example.com/article",
              },
              suggestedName: "article-copy",
              previewUrl: "https://example.com/article",
              templateName: "article",
              carriedForward: [
                "The saved extraction rules from template “article”.",
              ],
              remainingDecisions: [
                "Review the duplicated template name before saving.",
              ],
              unsupportedCarryForward: [
                "Runtime execution settings and auth do not become part of the duplicated template automatically.",
              ],
            }}
          />
        </AIAssistantProvider>
      </ToastProvider>,
    );

    expect(await screen.findByLabelText(/template name/i)).toHaveValue(
      "article-copy",
    );
    expect(
      screen.getByText(/template draft seeded from a verified job/i),
    ).toBeInTheDocument();
    expect(screen.getByDisplayValue("article h1")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /save template/i }));

    await waitFor(() => {
      expect(api.createTemplate).toHaveBeenCalledWith(
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
      expect(api.getTemplate).toHaveBeenCalledWith(
        expect.objectContaining({ path: { name: "article-copy" } }),
      );
    });
    expect(screen.getByLabelText(/preview target url/i)).toHaveValue(
      "https://example.com/article",
    );

    fireEvent.click(screen.getByRole("button", { name: /run preview/i }));

    await waitFor(() => {
      expect(api.testSelector).toHaveBeenCalledWith(
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
    vi.mocked(api.getTemplate).mockResolvedValue({
      data: {
        name: "article",
        is_built_in: true,
        template: {
          name: "article",
          selectors: [{ name: "title", selector: "article h1", attr: "text" }],
        },
      },
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates/article"),
      response: new Response(),
    });

    vi.mocked(api.createTemplate).mockResolvedValue({
      data: undefined,
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates"),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <AIAssistantProvider>
          <TemplateManager
            templateNames={["article"]}
            onTemplatesChanged={onTemplatesChanged}
          />
        </AIAssistantProvider>
      </ToastProvider>,
    );

    await screen.findByRole("button", { name: /duplicate to edit/i });
    fireEvent.click(screen.getByRole("button", { name: /duplicate to edit/i }));

    await waitFor(() => {
      expect(screen.getByLabelText(/template name/i)).toHaveValue(
        "article-copy",
      );
    });

    fireEvent.click(screen.getByRole("button", { name: /save template/i }));

    await waitFor(() => {
      expect(api.createTemplate).toHaveBeenCalledWith(
        expect.objectContaining({
          body: expect.objectContaining({
            name: "article-copy",
          }),
        }),
      );
    });
  });

  it("runs selector preview in the right rail", async () => {
    vi.mocked(api.getTemplate).mockResolvedValue({
      data: {
        name: "custom-news",
        is_built_in: false,
        template: {
          name: "custom-news",
          selectors: [{ name: "title", selector: "h1", attr: "text" }],
        },
      },
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates/custom-news"),
      response: new Response(),
    });

    vi.mocked(api.testSelector).mockResolvedValue({
      data: {
        selector: "h1",
        matches: 1,
        elements: [{ tag: "h1", text: "Headline" }],
      },
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/test-selector"),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <AIAssistantProvider>
          <TemplateManager
            templateNames={["custom-news"]}
            onTemplatesChanged={onTemplatesChanged}
          />
        </AIAssistantProvider>
      </ToastProvider>,
    );

    await screen.findByLabelText(/preview target url/i);

    fireEvent.change(screen.getByLabelText(/preview target url/i), {
      target: { value: "https://example.com/article" },
    });

    fireEvent.click(screen.getByRole("button", { name: /run preview/i }));

    expect(await screen.findByText(/1 match/i)).toBeInTheDocument();
    expect(screen.getByText(/headline/i)).toBeInTheDocument();
  });

  it("applies AI-generated templates into the inline workspace", async () => {
    vi.mocked(api.getTemplate).mockResolvedValue({
      data: {
        name: "custom-news",
        is_built_in: false,
        template: {
          name: "custom-news",
          selectors: [{ name: "title", selector: "h1", attr: "text" }],
        },
      },
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates/custom-news"),
      response: new Response(),
    });

    vi.mocked(api.aiTemplateGenerate).mockResolvedValue({
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

    render(
      <ToastProvider>
        <AIAssistantProvider>
          <TemplateManager
            templateNames={["custom-news"]}
            onTemplatesChanged={onTemplatesChanged}
          />
        </AIAssistantProvider>
      </ToastProvider>,
    );

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

    expect(screen.getByLabelText(/template name/i)).toHaveValue(
      "generated-template",
    );
    expect(screen.getByDisplayValue(".price")).toBeInTheDocument();
  });

  it("opens the visual builder inline inside the workspace", async () => {
    vi.mocked(api.getTemplate).mockResolvedValue({
      data: {
        name: "custom-news",
        is_built_in: false,
        template: {
          name: "custom-news",
          selectors: [{ name: "title", selector: "h1", attr: "text" }],
        },
      },
      error: undefined as never,
      request: new Request("http://127.0.0.1:8741/v1/templates/custom-news"),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <AIAssistantProvider>
          <TemplateManager
            templateNames={["custom-news"]}
            onTemplatesChanged={onTemplatesChanged}
          />
        </AIAssistantProvider>
      </ToastProvider>,
    );

    await screen.findByText(/custom-news/i);
    fireEvent.click(
      screen.getByRole("button", { name: /open visual builder/i }),
    );

    expect(screen.getByText(/visual builder mock/i)).toBeInTheDocument();
  });
});
