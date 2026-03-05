/**
 * AITemplateGenerator Component Tests
 *
 * Tests for the AI-powered template generation modal component.
 *
 * @module AITemplateGeneratorTests
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { AITemplateGenerator } from "../../AITemplateGenerator";
import * as api from "../../../api";
import type { ErrorResponse } from "../../../api";

// Mock the API module
vi.mock("../../../api", () => ({
  aiTemplateGenerate: vi.fn(),
  createTemplate: vi.fn(),
}));

// Mock the api-config module
vi.mock("../../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

describe("AITemplateGenerator", () => {
  const mockOnClose = vi.fn();
  const mockOnTemplateSaved = vi.fn();

  const mockGeneratedResponse = {
    template: {
      name: "product-template",
      selectors: [
        { name: "title", selector: "h1.product-title", attr: "text" },
        { name: "price", selector: "span.price", attr: "text" },
      ],
    },
    explanation: "Generated selectors for product page extraction",
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not render when isOpen is false", () => {
    render(
      <AITemplateGenerator
        isOpen={false}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    expect(
      screen.queryByText(/generate template with ai/i),
    ).not.toBeInTheDocument();
  });

  it("renders modal when isOpen is true", () => {
    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    expect(screen.getByText(/generate template with ai/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/target url/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
  });

  it("validates required fields before generation", async () => {
    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    const generateButton = screen.getByRole("button", {
      name: /generate template/i,
    });
    fireEvent.click(generateButton);

    await waitFor(() => {
      expect(screen.getByText(/url is required/i)).toBeInTheDocument();
    });
  });

  it("validates URL format", async () => {
    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    const urlInput = screen.getByLabelText(/target url/i);
    fireEvent.change(urlInput, { target: { value: "invalid-url" } });

    const descriptionInput = screen.getByLabelText(/description/i);
    fireEvent.change(descriptionInput, {
      target: { value: "Extract product data" },
    });

    const generateButton = screen.getByRole("button", {
      name: /generate template/i,
    });
    fireEvent.click(generateButton);

    await waitFor(() => {
      expect(screen.getByText(/please enter a valid url/i)).toBeInTheDocument();
    });
  });

  it("validates description is required", async () => {
    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    const urlInput = screen.getByLabelText(/target url/i);
    fireEvent.change(urlInput, { target: { value: "https://example.com" } });

    const generateButton = screen.getByRole("button", {
      name: /generate template/i,
    });
    fireEvent.click(generateButton);

    await waitFor(() => {
      expect(screen.getByText(/description is required/i)).toBeInTheDocument();
    });
  });

  it("calls aiTemplateGenerate API with correct params", async () => {
    vi.mocked(api.aiTemplateGenerate).mockResolvedValue({
      data: mockGeneratedResponse,
      error: undefined,
      request: new Request(
        "http://localhost:8741/v1/extract/ai-template-generate",
      ),
      response: new Response(),
    });

    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    const urlInput = screen.getByLabelText(/target url/i);
    fireEvent.change(urlInput, {
      target: { value: "https://example.com/products" },
    });

    const descriptionInput = screen.getByLabelText(/description/i);
    fireEvent.change(descriptionInput, {
      target: { value: "Extract product information" },
    });

    const fieldsInput = screen.getByLabelText(/sample fields/i);
    fireEvent.change(fieldsInput, {
      target: { value: "title, price, rating" },
    });

    const generateButton = screen.getByRole("button", {
      name: /generate template/i,
    });
    fireEvent.click(generateButton);

    await waitFor(() => {
      expect(api.aiTemplateGenerate).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
        body: {
          url: "https://example.com/products",
          description: "Extract product information",
          sample_fields: ["title", "price", "rating"],
          headless: false,
        },
      });
    });
  });

  it("displays generated template after successful generation", async () => {
    vi.mocked(api.aiTemplateGenerate).mockResolvedValue({
      data: mockGeneratedResponse,
      error: undefined,
      request: new Request(
        "http://localhost:8741/v1/extract/ai-template-generate",
      ),
      response: new Response(),
    });

    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    const urlInput = screen.getByLabelText(/target url/i);
    fireEvent.change(urlInput, {
      target: { value: "https://example.com/products" },
    });

    const descriptionInput = screen.getByLabelText(/description/i);
    fireEvent.change(descriptionInput, {
      target: { value: "Extract product information" },
    });

    const generateButton = screen.getByRole("button", {
      name: /generate template/i,
    });
    fireEvent.click(generateButton);

    await waitFor(() => {
      expect(screen.getByText(/generated template/i)).toBeInTheDocument();
    });

    expect(screen.getByText(/ai explanation/i)).toBeInTheDocument();
    expect(
      screen.getByText(mockGeneratedResponse.explanation),
    ).toBeInTheDocument();
    expect(screen.getByText("title")).toBeInTheDocument();
    expect(screen.getByText("price")).toBeInTheDocument();
  });

  it("handles missing AI config error", async () => {
    vi.mocked(api.aiTemplateGenerate).mockResolvedValue({
      data: undefined,
      error: {
        error: "AI extraction is not configured",
      } as ErrorResponse,
      request: new Request(
        "http://localhost:8741/v1/extract/ai-template-generate",
      ),
      response: new Response(),
    });

    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    const urlInput = screen.getByLabelText(/target url/i);
    fireEvent.change(urlInput, { target: { value: "https://example.com" } });

    const descriptionInput = screen.getByLabelText(/description/i);
    fireEvent.change(descriptionInput, { target: { value: "Extract data" } });

    const generateButton = screen.getByRole("button", {
      name: /generate template/i,
    });
    fireEvent.click(generateButton);

    await waitFor(() => {
      expect(
        screen.getByText(/ai extraction is not configured/i),
      ).toBeInTheDocument();
    });
  });

  it("handles API failures gracefully", async () => {
    vi.mocked(api.aiTemplateGenerate).mockRejectedValue(
      new Error("Network error"),
    );

    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    const urlInput = screen.getByLabelText(/target url/i);
    fireEvent.change(urlInput, { target: { value: "https://example.com" } });

    const descriptionInput = screen.getByLabelText(/description/i);
    fireEvent.change(descriptionInput, { target: { value: "Extract data" } });

    const generateButton = screen.getByRole("button", {
      name: /generate template/i,
    });
    fireEvent.click(generateButton);

    await waitFor(() => {
      expect(screen.getByText(/network error/i)).toBeInTheDocument();
    });
  });

  it("calls createTemplate API on save", async () => {
    vi.mocked(api.aiTemplateGenerate).mockResolvedValue({
      data: mockGeneratedResponse,
      error: undefined,
      request: new Request(
        "http://localhost:8741/v1/extract/ai-template-generate",
      ),
      response: new Response(),
    });

    vi.mocked(api.createTemplate).mockResolvedValue({
      data: undefined,
      error: undefined as unknown as ErrorResponse,
      request: new Request("http://localhost:8741/v1/templates"),
      response: new Response(),
    });

    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    // Generate template first
    const urlInput = screen.getByLabelText(/target url/i);
    fireEvent.change(urlInput, { target: { value: "https://example.com" } });

    const descriptionInput = screen.getByLabelText(/description/i);
    fireEvent.change(descriptionInput, { target: { value: "Extract data" } });

    const generateButton = screen.getByRole("button", {
      name: /generate template/i,
    });
    fireEvent.click(generateButton);

    await waitFor(() => {
      expect(screen.getByText(/generated template/i)).toBeInTheDocument();
    });

    // Enter template name
    const nameInput = screen.getByLabelText(/template name/i);
    fireEvent.change(nameInput, { target: { value: "my-custom-template" } });

    // Save template
    const saveButton = screen.getByRole("button", { name: /save template/i });
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(api.createTemplate).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
        body: {
          ...mockGeneratedResponse.template,
          name: "my-custom-template",
          selectors: mockGeneratedResponse.template.selectors,
        },
      });
    });

    expect(mockOnTemplateSaved).toHaveBeenCalled();
    expect(mockOnClose).toHaveBeenCalled();
  });

  it("validates template name before save", async () => {
    vi.mocked(api.aiTemplateGenerate).mockResolvedValue({
      data: mockGeneratedResponse,
      error: undefined,
      request: new Request(
        "http://localhost:8741/v1/extract/ai-template-generate",
      ),
      response: new Response(),
    });

    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    // Generate template
    const urlInput = screen.getByLabelText(/target url/i);
    fireEvent.change(urlInput, { target: { value: "https://example.com" } });

    const descriptionInput = screen.getByLabelText(/description/i);
    fireEvent.change(descriptionInput, { target: { value: "Extract data" } });

    const generateButton = screen.getByRole("button", {
      name: /generate template/i,
    });
    fireEvent.click(generateButton);

    await waitFor(() => {
      expect(screen.getByText(/generated template/i)).toBeInTheDocument();
    });

    // Clear the template name to test validation
    const nameInput = screen.getByLabelText(/template name/i);
    fireEvent.change(nameInput, { target: { value: "" } });

    // Try to save without name
    const saveButton = screen.getByRole("button", { name: /save template/i });
    expect(saveButton).toBeDisabled();
  });

  it("closes modal on cancel", () => {
    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    const cancelButton = screen.getByRole("button", { name: /cancel/i });
    fireEvent.click(cancelButton);

    expect(mockOnClose).toHaveBeenCalled();
  });

  it("allows toggling headless option", () => {
    render(
      <AITemplateGenerator
        isOpen={true}
        onClose={mockOnClose}
        onTemplateSaved={mockOnTemplateSaved}
      />,
    );

    const headlessCheckbox = screen.getByRole("checkbox");
    expect(headlessCheckbox).not.toBeChecked();

    fireEvent.click(headlessCheckbox);
    expect(headlessCheckbox).toBeChecked();
  });
});
