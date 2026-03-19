/**
 * Purpose: Verify the job-submission assistant runs AI extraction previews and exposes explicit apply actions back into the shared form controller.
 * Responsibilities: Mock preview API responses, mount the assistant inside the provider, and assert field-name application behavior.
 * Scope: Component coverage for `JobSubmissionAssistantSection` only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: The assistant remains route-aware, preview requests stay explicit, and returned fields are only applied after an operator click.
 */

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import * as api from "../../../api";
import type { FormController } from "../../../hooks/useFormState";
import { AIAssistantProvider } from "../AIAssistantProvider";
import { JobSubmissionAssistantSection } from "../JobSubmissionAssistantSection";

vi.mock("../../../api", () => ({
  aiExtractPreview: vi.fn(),
}));

vi.mock("../../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

function createFormController(): FormController {
  return {
    headless: false,
    usePlaywright: false,
    extractTemplate: "article",
    aiExtractMode: "natural_language",
    aiExtractPrompt: "Extract article title",
    aiExtractSchema: "",
    aiExtractFields: "title",
    setAIExtractEnabled: vi.fn(),
    setAIExtractMode: vi.fn(),
    setAIExtractPrompt: vi.fn(),
    setAIExtractSchema: vi.fn(),
    setAIExtractFields: vi.fn(),
  } as unknown as FormController;
}

describe("JobSubmissionAssistantSection", () => {
  const localState = {
    scrape: { url: "https://example.com", device: null },
    crawl: {
      url: "",
      sitemapURL: "",
      sitemapOnly: false,
      includePatterns: "",
      excludePatterns: "",
      device: null,
    },
    research: { query: "", urls: "", device: null },
  };

  it("runs preview and applies returned field names", async () => {
    vi.mocked(api.aiExtractPreview).mockResolvedValue({
      data: {
        fields: {
          title: { values: ["Example"], source: "llm" },
          author: { values: ["Spartan"], source: "llm" },
        },
        confidence: 0.92,
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/ai/extract-preview"),
      response: new Response(),
    });

    const form = createFormController();

    render(
      <AIAssistantProvider>
        <JobSubmissionAssistantSection
          activeTab="scrape"
          form={form}
          localState={localState}
        />
      </AIAssistantProvider>,
    );

    fireEvent.click(screen.getByText(/run preview/i));

    await waitFor(() => {
      expect(api.aiExtractPreview).toHaveBeenCalled();
    });

    fireEvent.click(screen.getByText(/use returned field names/i));

    expect(form.setAIExtractFields).toHaveBeenCalledWith("title, author");
  });

  it("shows the unavailable notice and disables preview when ai is off", () => {
    const form = createFormController();

    render(
      <AIAssistantProvider>
        <JobSubmissionAssistantSection
          activeTab="scrape"
          form={form}
          localState={localState}
          aiStatus={{
            status: "disabled",
            message: "AI helpers are disabled.",
          }}
        />
      </AIAssistantProvider>,
    );

    expect(
      screen.getAllByText(/AI helpers are disabled\./i).length,
    ).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: /run preview/i })).toBeDisabled();
  });
});
