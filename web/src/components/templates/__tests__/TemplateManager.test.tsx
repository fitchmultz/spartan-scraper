import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TemplateManager } from "../TemplateManager";
import * as api from "../../../api";

vi.mock("../../../api", () => ({
  createTemplate: vi.fn(),
  deleteTemplate: vi.fn(),
  getTemplate: vi.fn(),
  updateTemplate: vi.fn(),
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
          onSave({ name: initialTemplate?.name ?? "saved-template" })
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
  const onOpenAIGenerator = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal(
      "confirm",
      vi.fn(() => true),
    );
  });

  it("loads template details and exposes built-in actions from the template library", async () => {
    vi.mocked(api.getTemplate).mockResolvedValue({
      data: {
        name: "article",
        is_built_in: true,
        template: {
          name: "article",
          selectors: [{ name: "title", selector: "article h1", attr: "text" }],
        },
      },
      error: undefined,
      request: new Request("http://127.0.0.1:8741/v1/templates/article"),
      response: new Response(),
    });

    render(
      <TemplateManager
        templateNames={["article", "custom-news"]}
        onTemplatesChanged={onTemplatesChanged}
        onOpenAIGenerator={onOpenAIGenerator}
      />,
    );

    await waitFor(() => {
      expect(
        screen.getByRole("heading", { name: "article" }),
      ).toBeInTheDocument();
    });

    expect(
      screen.getAllByText(/open details and management actions/i),
    ).toHaveLength(2);
    expect(
      screen.getByRole("button", { name: /duplicate/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /edit in visual builder/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /^delete$/i }),
    ).not.toBeInTheDocument();
  });

  it("edits a custom template through the modal editor", async () => {
    vi.mocked(api.getTemplate).mockResolvedValue({
      data: {
        name: "custom-news",
        is_built_in: false,
        template: {
          name: "custom-news",
          selectors: [{ name: "title", selector: "h1", attr: "text" }],
        },
      },
      error: undefined,
      request: new Request("http://127.0.0.1:8741/v1/templates/custom-news"),
      response: new Response(),
    });
    vi.mocked(api.updateTemplate).mockResolvedValue({
      data: {
        name: "custom-news-renamed",
        is_built_in: false,
        template: {
          name: "custom-news-renamed",
          selectors: [{ name: "title", selector: "main h1", attr: "text" }],
        },
      },
      error: undefined,
      request: new Request("http://127.0.0.1:8741/v1/templates/custom-news"),
      response: new Response(),
    });

    render(
      <TemplateManager
        templateNames={["custom-news"]}
        onTemplatesChanged={onTemplatesChanged}
        onOpenAIGenerator={onOpenAIGenerator}
      />,
    );

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /^edit$/i }),
      ).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: /^edit$/i }));
    fireEvent.change(screen.getByLabelText(/template name/i), {
      target: { value: "custom-news-renamed" },
    });

    const selectorInputs = screen.getAllByDisplayValue(/h1/);
    fireEvent.change(selectorInputs[0], {
      target: { value: "main h1" },
    });

    fireEvent.click(screen.getByRole("button", { name: /save changes/i }));

    await waitFor(() => {
      expect(api.updateTemplate).toHaveBeenCalledWith({
        baseUrl: "http://127.0.0.1:8741",
        path: { name: "custom-news" },
        body: {
          name: "custom-news-renamed",
          selectors: [{ name: "title", selector: "main h1", attr: "text" }],
        },
      });
    });
    expect(onTemplatesChanged).toHaveBeenCalled();
  });

  it("deletes a custom template from the primary management surface", async () => {
    vi.mocked(api.getTemplate).mockResolvedValue({
      data: {
        name: "custom-news",
        is_built_in: false,
        template: {
          name: "custom-news",
          selectors: [{ name: "title", selector: "h1", attr: "text" }],
        },
      },
      error: undefined,
      request: new Request("http://127.0.0.1:8741/v1/templates/custom-news"),
      response: new Response(),
    });
    vi.mocked(api.deleteTemplate).mockResolvedValue({
      data: undefined,
      error: undefined,
      request: new Request("http://127.0.0.1:8741/v1/templates/custom-news"),
      response: new Response(),
    });

    render(
      <TemplateManager
        templateNames={["custom-news"]}
        onTemplatesChanged={onTemplatesChanged}
        onOpenAIGenerator={onOpenAIGenerator}
      />,
    );

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /^delete$/i }),
      ).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: /^delete$/i }));

    await waitFor(() => {
      expect(api.deleteTemplate).toHaveBeenCalledWith({
        baseUrl: "http://127.0.0.1:8741",
        path: { name: "custom-news" },
      });
    });
    expect(onTemplatesChanged).toHaveBeenCalled();
  });
});
