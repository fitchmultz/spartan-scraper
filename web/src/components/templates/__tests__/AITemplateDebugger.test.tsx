import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AITemplateDebugger } from "../../AITemplateDebugger";
import * as api from "../../../api";

vi.mock("../../../api", () => ({
  aiTemplateDebug: vi.fn(),
  updateTemplate: vi.fn(),
}));

vi.mock("../../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

describe("AITemplateDebugger", () => {
  const onClose = vi.fn();
  const onTemplateSaved = vi.fn();
  const template = {
    name: "product-template",
    selectors: [{ name: "title", selector: ".missing", attr: "text" }],
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("validates required URL in URL mode", async () => {
    render(
      <AITemplateDebugger
        isOpen={true}
        template={template}
        onClose={onClose}
        onTemplateSaved={onTemplateSaved}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /debug template/i }));

    await waitFor(() => {
      expect(screen.getByText(/url is required/i)).toBeInTheDocument();
    });
  });

  it("calls aiTemplateDebug with visual URL mode options", async () => {
    vi.mocked(api.aiTemplateDebug).mockResolvedValue({
      data: {
        issues: ["selector title matched no elements"],
        explanation: "Use the visible h1 instead.",
        suggested_template: {
          name: "product-template",
          selectors: [{ name: "title", selector: "h1", attr: "text" }],
        },
        route_id: "openai/gpt-5.4",
        provider: "openai",
        model: "gpt-5.4",
        visual_context_used: true,
      },
      error: undefined,
      request: new Request(
        "http://localhost:8741/v1/extract/ai-template-debug",
      ),
      response: new Response(),
    });

    render(
      <AITemplateDebugger
        isOpen={true}
        template={template}
        onClose={onClose}
        onTemplateSaved={onTemplateSaved}
      />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/product" },
    });
    fireEvent.change(screen.getByLabelText(/repair instructions/i), {
      target: { value: "Prefer visible headings" },
    });
    fireEvent.click(screen.getByLabelText(/use headless browser/i));
    fireEvent.click(screen.getByLabelText(/include screenshot context/i));
    fireEvent.click(screen.getByRole("button", { name: /debug template/i }));

    await waitFor(() => {
      expect(api.aiTemplateDebug).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
        body: {
          url: "https://example.com/product",
          template,
          instructions: "Prefer visible headings",
          headless: true,
          playwright: false,
          visual: true,
        },
      });
    });

    expect(screen.getByText(/detected issues/i)).toBeInTheDocument();
    expect(
      screen.getByText(/selector title matched no elements/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/^Used$/)).toBeInTheDocument();
  });

  it("saves the suggested template over the current template", async () => {
    vi.mocked(api.aiTemplateDebug).mockResolvedValue({
      data: {
        issues: ["selector title matched no elements"],
        suggested_template: {
          name: "product-template",
          selectors: [{ name: "title", selector: "h1", attr: "text" }],
        },
      },
      error: undefined,
      request: new Request(
        "http://localhost:8741/v1/extract/ai-template-debug",
      ),
      response: new Response(),
    });
    vi.mocked(api.updateTemplate).mockResolvedValue({
      data: {
        name: "product-template",
        template: {
          name: "product-template",
          selectors: [{ name: "title", selector: "h1", attr: "text" }],
        },
      },
      error: undefined,
      request: new Request(
        "http://localhost:8741/v1/templates/product-template",
      ),
      response: new Response(),
    });

    render(
      <AITemplateDebugger
        isOpen={true}
        template={template}
        onClose={onClose}
        onTemplateSaved={onTemplateSaved}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /paste html/i }));
    fireEvent.change(screen.getByLabelText(/^html/i), {
      target: { value: "<html><body><h1>Widget</h1></body></html>" },
    });
    fireEvent.click(screen.getByRole("button", { name: /debug template/i }));

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /save suggested template/i }),
      ).toBeInTheDocument();
    });

    fireEvent.click(
      screen.getByRole("button", { name: /save suggested template/i }),
    );

    await waitFor(() => {
      expect(api.updateTemplate).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
        path: { name: "product-template" },
        body: {
          name: "product-template",
          selectors: [{ name: "title", selector: "h1", attr: "text" }],
        },
      });
    });
    expect(onTemplateSaved).toHaveBeenCalledTimes(1);
  });
});
