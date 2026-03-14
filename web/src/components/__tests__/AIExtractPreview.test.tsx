import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AIExtractPreview } from "../AIExtractPreview";
import * as api from "../../api";
import type { AiExtractPreviewResponse, ErrorResponse } from "../../api";

vi.mock("../../api", () => ({
  aiExtractPreview: vi.fn(),
}));

vi.mock("../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

describe("AIExtractPreview", () => {
  const onClose = vi.fn();

  const mockPreviewResponse = {
    fields: {
      title: {
        values: ["Example Product"],
        source: "llm",
      },
      price: {
        values: ["$19.99"],
        source: "llm",
      },
      metadata: {
        source: "llm",
        rawObject: '{"currency":"USD","availability":"in_stock"}',
      },
    },
    confidence: 0.93,
    explanation: "Captured the main product facts from the hero region.",
    tokens_used: 321,
    cached: true,
    route_id: "openai/gpt-5.4",
    provider: "openai",
    model: "gpt-5.4",
    visual_context_used: false,
  } satisfies AiExtractPreviewResponse;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not render when closed", () => {
    render(<AIExtractPreview isOpen={false} onClose={onClose} initialUrl="" />);

    expect(
      screen.queryByText(/preview extraction with ai/i),
    ).not.toBeInTheDocument();
  });

  it("seeds the initial URL when opened", () => {
    render(
      <AIExtractPreview
        isOpen={true}
        onClose={onClose}
        initialUrl="https://example.com/products/widget"
      />,
    );

    expect(screen.getByLabelText(/target url/i)).toHaveValue(
      "https://example.com/products/widget",
    );
  });

  it("validates required fields before running preview", async () => {
    render(<AIExtractPreview isOpen={true} onClose={onClose} initialUrl="" />);

    fireEvent.click(screen.getByRole("button", { name: /run ai preview/i }));

    await waitFor(() => {
      expect(screen.getByText(/url is required/i)).toBeInTheDocument();
    });
  });

  it("validates schema-guided JSON before sending the request", async () => {
    render(
      <AIExtractPreview
        isOpen={true}
        onClose={onClose}
        initialUrl="https://example.com/products"
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /schema guided/i }));
    fireEvent.change(screen.getByLabelText(/schema example json/i), {
      target: { value: "{not-json}" },
    });
    fireEvent.click(screen.getByRole("button", { name: /run ai preview/i }));

    await waitFor(() => {
      expect(
        screen.getByText(/schema example must be valid json/i),
      ).toBeInTheDocument();
    });
    expect(api.aiExtractPreview).not.toHaveBeenCalled();
  });

  it("calls aiExtractPreview with URL-mode options", async () => {
    vi.mocked(api.aiExtractPreview).mockResolvedValue({
      data: mockPreviewResponse,
      error: undefined,
      request: new Request("http://localhost:8741/v1/ai/extract-preview"),
      response: new Response(),
    });

    render(
      <AIExtractPreview
        isOpen={true}
        onClose={onClose}
        initialUrl="https://example.com/products"
      />,
    );

    fireEvent.change(screen.getByLabelText(/specific fields/i), {
      target: { value: "title, price, metadata" },
    });
    const image = new File(["fake"], "preview.png", { type: "image/png" });
    fireEvent.change(screen.getByLabelText(/upload images/i), {
      target: { files: [image] },
    });
    await screen.findByText("preview.png");
    fireEvent.change(screen.getByLabelText(/extraction instructions/i), {
      target: { value: "Extract the main product facts" },
    });
    fireEvent.click(screen.getByLabelText(/use headless browser/i));
    fireEvent.click(screen.getByLabelText(/use playwright/i));
    fireEvent.click(screen.getByLabelText(/include screenshot context/i));
    fireEvent.click(screen.getByRole("button", { name: /run ai preview/i }));

    await waitFor(() => {
      expect(api.aiExtractPreview).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
        body: {
          url: "https://example.com/products",
          mode: "natural_language",
          prompt: "Extract the main product facts",
          fields: ["title", "price", "metadata"],
          images: [{ data: "ZmFrZQ==", mime_type: "image/png" }],
          headless: true,
          playwright: true,
          visual: true,
        },
      });
    });
  });

  it("supports pasted HTML mode with schema-guided extraction", async () => {
    vi.mocked(api.aiExtractPreview).mockResolvedValue({
      data: mockPreviewResponse,
      error: undefined,
      request: new Request("http://localhost:8741/v1/ai/extract-preview"),
      response: new Response(),
    });

    render(<AIExtractPreview isOpen={true} onClose={onClose} initialUrl="" />);

    fireEvent.click(screen.getByRole("button", { name: /paste html/i }));
    fireEvent.change(screen.getByLabelText(/page url/i), {
      target: { value: "https://example.com/saved-page" },
    });
    fireEvent.change(screen.getByLabelText(/^html/i), {
      target: { value: "<html><body><h1>Example</h1></body></html>" },
    });
    fireEvent.click(screen.getByRole("button", { name: /schema guided/i }));
    fireEvent.change(screen.getByLabelText(/schema example json/i), {
      target: { value: '{"title":"Example Product","price":"$19.99"}' },
    });

    fireEvent.click(screen.getByRole("button", { name: /run ai preview/i }));

    await waitFor(() => {
      expect(api.aiExtractPreview).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
        body: {
          url: "https://example.com/saved-page",
          html: "<html><body><h1>Example</h1></body></html>",
          mode: "schema_guided",
          schema: {
            title: "Example Product",
            price: "$19.99",
          },
        },
      });
    });
  });

  it("renders the preview results with route diagnostics", async () => {
    vi.mocked(api.aiExtractPreview).mockResolvedValue({
      data: mockPreviewResponse,
      error: undefined,
      request: new Request("http://localhost:8741/v1/ai/extract-preview"),
      response: new Response(),
    });

    render(
      <AIExtractPreview
        isOpen={true}
        onClose={onClose}
        initialUrl="https://example.com/products"
      />,
    );

    fireEvent.change(screen.getByLabelText(/extraction instructions/i), {
      target: { value: "Extract the main product facts" },
    });
    fireEvent.click(screen.getByRole("button", { name: /run ai preview/i }));

    await waitFor(() => {
      expect(screen.getByText(/preview results/i)).toBeInTheDocument();
    });

    expect(screen.getByText(/ai explanation/i)).toBeInTheDocument();
    expect(
      screen.getAllByText(/captured the main product facts/i).length,
    ).toBeGreaterThan(0);
    expect(screen.getByText(/ai route/i)).toBeInTheDocument();
    expect(screen.getByText("openai/gpt-5.4")).toBeInTheDocument();
    expect(screen.getByText("openai")).toBeInTheDocument();
    expect(screen.getByText("gpt-5.4")).toBeInTheDocument();
    expect(screen.getByText("Example Product")).toBeInTheDocument();
    expect(screen.getByText("$19.99")).toBeInTheDocument();
    expect(screen.getByText(/structured value/i)).toBeInTheDocument();
  });

  it("surfaces missing AI configuration errors clearly", async () => {
    vi.mocked(api.aiExtractPreview).mockResolvedValue({
      data: undefined,
      error: {
        error: "AI extraction is not configured",
      } as ErrorResponse,
      request: new Request("http://localhost:8741/v1/ai/extract-preview"),
      response: new Response(),
    });

    render(
      <AIExtractPreview
        isOpen={true}
        onClose={onClose}
        initialUrl="https://example.com/products"
      />,
    );

    fireEvent.change(screen.getByLabelText(/extraction instructions/i), {
      target: { value: "Extract the main product facts" },
    });
    fireEvent.click(screen.getByRole("button", { name: /run ai preview/i }));

    await waitFor(() => {
      expect(
        screen.getByText(
          /enable the pi bridge and ensure your pi credentials are available/i,
        ),
      ).toBeInTheDocument();
    });
  });
});
