import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { FormController } from "../hooks/useFormState";
import { BatchForm } from "./BatchForm";

function createFormController(): FormController {
  return {
    headless: false,
    setHeadless: vi.fn(),
    usePlaywright: false,
    setUsePlaywright: vi.fn(),
    timeoutSeconds: 30,
    setTimeoutSeconds: vi.fn(),
    authProfile: "",
    setAuthProfile: vi.fn(),
    authBasic: "",
    setAuthBasic: vi.fn(),
    headersRaw: "",
    setHeadersRaw: vi.fn(),
    cookiesRaw: "",
    setCookiesRaw: vi.fn(),
    queryRaw: "",
    setQueryRaw: vi.fn(),
    proxyUrl: "",
    setProxyUrl: vi.fn(),
    proxyUsername: "",
    setProxyUsername: vi.fn(),
    proxyPassword: "",
    setProxyPassword: vi.fn(),
    proxyRegion: "",
    setProxyRegion: vi.fn(),
    proxyRequiredTags: "",
    setProxyRequiredTags: vi.fn(),
    proxyExcludeProxyIds: "",
    setProxyExcludeProxyIds: vi.fn(),
    loginUrl: "",
    setLoginUrl: vi.fn(),
    loginUserSelector: "",
    setLoginUserSelector: vi.fn(),
    loginPassSelector: "",
    setLoginPassSelector: vi.fn(),
    loginSubmitSelector: "",
    setLoginSubmitSelector: vi.fn(),
    loginUser: "",
    setLoginUser: vi.fn(),
    loginPass: "",
    setLoginPass: vi.fn(),
    extractTemplate: "",
    setExtractTemplate: vi.fn(),
    extractValidate: false,
    setExtractValidate: vi.fn(),
    aiExtractEnabled: false,
    setAIExtractEnabled: vi.fn(),
    aiExtractMode: "natural_language",
    setAIExtractMode: vi.fn(),
    aiExtractPrompt: "",
    setAIExtractPrompt: vi.fn(),
    aiExtractSchema: '{\n  "title": "Example product",\n  "price": "$19.99"\n}',
    setAIExtractSchema: vi.fn(),
    aiExtractFields: "",
    setAIExtractFields: vi.fn(),
    agenticResearchEnabled: false,
    setAgenticResearchEnabled: vi.fn(),
    agenticResearchInstructions: "",
    setAgenticResearchInstructions: vi.fn(),
    agenticResearchMaxRounds: 1,
    setAgenticResearchMaxRounds: vi.fn(),
    agenticResearchMaxFollowUpUrls: 3,
    setAgenticResearchMaxFollowUpUrls: vi.fn(),
    preProcessors: "",
    setPreProcessors: vi.fn(),
    postProcessors: "",
    setPostProcessors: vi.fn(),
    transformers: "",
    setTransformers: vi.fn(),
    incremental: false,
    setIncremental: vi.fn(),
    maxDepth: 2,
    setMaxDepth: vi.fn(),
    maxPages: 200,
    setMaxPages: vi.fn(),
    webhookUrl: "",
    setWebhookUrl: vi.fn(),
    webhookEvents: ["completed"],
    setWebhookEvents: vi.fn(),
    webhookSecret: "",
    setWebhookSecret: vi.fn(),
    interceptEnabled: false,
    setInterceptEnabled: vi.fn(),
    interceptURLPatterns: "",
    setInterceptURLPatterns: vi.fn(),
    interceptResourceTypes: ["xhr", "fetch"],
    setInterceptResourceTypes: vi.fn(),
    interceptCaptureRequestBody: true,
    setInterceptCaptureRequestBody: vi.fn(),
    interceptCaptureResponseBody: true,
    setInterceptCaptureResponseBody: vi.fn(),
    interceptMaxBodySize: 1048576,
    setInterceptMaxBodySize: vi.fn(),
    applyPreset: vi.fn(),
  };
}

function renderBatchForm(urlsInput = "") {
  const setUrlsInput = vi.fn();

  render(
    <BatchForm
      activeTab="scrape"
      setActiveTab={vi.fn()}
      form={createFormController()}
      profiles={[]}
      urlsInput={urlsInput}
      setUrlsInput={setUrlsInput}
      submissionNotice={null}
      onViewSubmittedBatch={vi.fn()}
      maxDepth={2}
      setMaxDepth={vi.fn()}
      maxPages={200}
      setMaxPages={vi.fn()}
      query=""
      setQuery={vi.fn()}
      onSubmitScrape={vi.fn()}
      onSubmitCrawl={vi.fn()}
      onSubmitResearch={vi.fn()}
      loading={false}
    />,
  );

  return { setUrlsInput };
}

describe("BatchForm", () => {
  it("shows the parsed URL count for comma-separated and newline-separated input", () => {
    renderBatchForm(
      "https://example.com, https://example.org\nhttps://example.net",
    );

    expect(
      screen.getByRole("textbox", {
        name: /URLs \(3\/100\)One per line or comma-separated/,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Submit Batch scrape (3 URLs)" }),
    ).toBeEnabled();
  });

  it("forwards textarea changes so the container can recalculate the count", () => {
    const { setUrlsInput } = renderBatchForm();

    fireEvent.change(screen.getByRole("textbox", { name: /URLs \(0\/100\)/ }), {
      target: {
        value: "https://example.com, https://example.org",
      },
    });

    expect(setUrlsInput).toHaveBeenCalledWith(
      "https://example.com, https://example.org",
    );
  });

  it("renders an in-context confirmation with the submitted URLs", () => {
    render(
      <BatchForm
        activeTab="scrape"
        setActiveTab={vi.fn()}
        form={createFormController()}
        profiles={[]}
        urlsInput=""
        setUrlsInput={vi.fn()}
        submissionNotice={{
          batchId: "batch-12345678",
          kind: "scrape",
          submittedUrls: ["https://example.com", "https://example.org"],
        }}
        onViewSubmittedBatch={vi.fn()}
        maxDepth={2}
        setMaxDepth={vi.fn()}
        maxPages={200}
        setMaxPages={vi.fn()}
        query=""
        setQuery={vi.fn()}
        onSubmitScrape={vi.fn()}
        onSubmitCrawl={vi.fn()}
        onSubmitResearch={vi.fn()}
        loading={false}
      />,
    );

    expect(screen.getByRole("status")).toHaveTextContent(
      "Queued 2 URLs for scrape",
    );
    expect(screen.getByText("example.com")).toBeInTheDocument();
    expect(screen.getByText("example.org")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "View Batch Progress" }),
    ).toBeInTheDocument();
  });
});
