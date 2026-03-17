/**
 * Purpose: Render the operator-facing batch submission workspace for scrape, crawl, and research jobs.
 * Responsibilities: Parse URL input and uploads, validate batch-specific requirements, build typed API requests from shared form state, and preserve in-context submission notices.
 * Scope: Batch authoring only; server mutation side effects and batch list rendering stay outside this component.
 * Usage: Render from `BatchContainer` with the shared `FormController`, controlled batch-local fields, and submit callbacks.
 * Invariants/Assumptions: URLs are validated before request assembly, research batches require a non-empty query, and transient validation problems surface through the global toast system instead of browser alerts.
 */
import {
  useMemo,
  useState,
  useCallback,
  useRef,
  useEffect,
  type FormEvent,
} from "react";
import { AuthConfig } from "./AuthConfig";
import { PipelineOptions } from "./PipelineOptions";
import { AIExtractSection } from "./AIExtractSection";
import { ResearchAgenticSection } from "./ResearchAgenticSection";
import {
  buildAIExtractOptions,
  buildResearchAgenticOptions,
  buildSharedRequestConfig,
} from "../lib/form-utils";
import type { FormController, ProfileOption } from "../hooks/useFormState";
import { WebhookConfig } from "./WebhookConfig";
import { BrowserExecutionControls } from "./BrowserExecutionControls";
import { DeviceSelector } from "./DeviceSelector";
import { ScreenshotConfig } from "./ScreenshotConfig";
import { NetworkInterceptConfig } from "./NetworkInterceptConfig";
import type { DeviceEmulation } from "../api";
import type {
  BatchScrapeRequest,
  BatchCrawlRequest,
  BatchResearchRequest,
} from "../api";
import {
  buildBatchCrawlRequest,
  buildBatchResearchRequest,
  buildBatchScrapeRequest,
} from "../lib/batch-utils";
import {
  MAX_BATCH_SIZE,
  parseBatchUrls,
  summarizeBatchUrls,
} from "../lib/batch-urls";
import { useToast } from "./toast";

interface BatchFormProps {
  // Job type selector
  activeTab: "scrape" | "crawl" | "research";
  setActiveTab: (tab: "scrape" | "crawl" | "research") => void;

  form: FormController;
  profiles: ProfileOption[];

  // Batch-specific
  urlsInput: string;
  setUrlsInput: (value: string) => void;
  submissionNotice: BatchSubmissionNotice | null;
  onViewSubmittedBatch: () => void;

  // Crawl-specific
  maxDepth: number;
  setMaxDepth: (value: number) => void;
  maxPages: number;
  setMaxPages: (value: number) => void;

  // Research-specific
  query: string;
  setQuery: (value: string) => void;

  // Submit callbacks
  onSubmitScrape: (request: BatchScrapeRequest) => Promise<void>;
  onSubmitCrawl: (request: BatchCrawlRequest) => Promise<void>;
  onSubmitResearch: (request: BatchResearchRequest) => Promise<void>;
  loading: boolean;
}

export interface BatchSubmissionNotice {
  batchId: string;
  kind: "scrape" | "crawl" | "research";
  submittedUrls: string[];
}

function formatSubmittedUrl(url: string): string {
  try {
    const parsed = new URL(url);
    const suffix = `${parsed.pathname}${parsed.search}${parsed.hash}`.replace(
      /\/$/,
      "",
    );
    return `${parsed.host}${suffix}` || parsed.host;
  } catch {
    return url;
  }
}

export function BatchForm({
  activeTab,
  setActiveTab,
  form,
  profiles,
  urlsInput,
  setUrlsInput,
  submissionNotice,
  onViewSubmittedBatch,
  maxDepth,
  setMaxDepth,
  maxPages,
  setMaxPages,
  query,
  setQuery,
  onSubmitScrape,
  onSubmitCrawl,
  onSubmitResearch,
  loading,
}: BatchFormProps) {
  const toast = useToast();

  const {
    headless,
    setHeadless,
    usePlaywright,
    setUsePlaywright,
    timeoutSeconds,
    setTimeoutSeconds,
    authProfile,
    setAuthProfile,
    authBasic,
    setAuthBasic,
    headersRaw,
    setHeadersRaw,
    cookiesRaw,
    setCookiesRaw,
    queryRaw,
    setQueryRaw,
    proxyUrl,
    setProxyUrl,
    proxyUsername,
    setProxyUsername,
    proxyPassword,
    setProxyPassword,
    proxyRegion,
    setProxyRegion,
    proxyRequiredTags,
    setProxyRequiredTags,
    proxyExcludeProxyIds,
    setProxyExcludeProxyIds,
    loginUrl,
    setLoginUrl,
    loginUserSelector,
    setLoginUserSelector,
    loginPassSelector,
    setLoginPassSelector,
    loginSubmitSelector,
    setLoginSubmitSelector,
    loginUser,
    setLoginUser,
    loginPass,
    setLoginPass,
    extractTemplate,
    setExtractTemplate,
    extractValidate,
    setExtractValidate,
    aiExtractEnabled,
    setAIExtractEnabled,
    aiExtractMode,
    setAIExtractMode,
    aiExtractPrompt,
    setAIExtractPrompt,
    aiExtractSchema,
    setAIExtractSchema,
    aiExtractFields,
    setAIExtractFields,
    agenticResearchEnabled,
    setAgenticResearchEnabled,
    agenticResearchInstructions,
    setAgenticResearchInstructions,
    agenticResearchMaxRounds,
    setAgenticResearchMaxRounds,
    agenticResearchMaxFollowUpUrls,
    setAgenticResearchMaxFollowUpUrls,
    preProcessors,
    setPreProcessors,
    postProcessors,
    setPostProcessors,
    transformers,
    setTransformers,
    incremental,
    setIncremental,
    webhookUrl,
    setWebhookUrl,
    webhookEvents,
    setWebhookEvents,
    webhookSecret,
    setWebhookSecret,
    screenshotEnabled,
    setScreenshotEnabled,
    screenshotFullPage,
    setScreenshotFullPage,
    screenshotFormat,
    setScreenshotFormat,
    screenshotQuality,
    setScreenshotQuality,
    screenshotWidth,
    setScreenshotWidth,
    screenshotHeight,
    setScreenshotHeight,
    interceptEnabled,
    setInterceptEnabled,
    interceptURLPatterns,
    setInterceptURLPatterns,
    interceptResourceTypes,
    setInterceptResourceTypes,
    interceptCaptureRequestBody,
    setInterceptCaptureRequestBody,
    interceptCaptureResponseBody,
    setInterceptCaptureResponseBody,
    interceptMaxBodySize,
    setInterceptMaxBodySize,
    interceptMaxEntries,
    setInterceptMaxEntries,
  } = form;

  const [device, setDevice] = useState<DeviceEmulation | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [urlError, setUrlError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const submissionNoticeRef = useRef<HTMLOutputElement>(null);

  // Parse URLs from input (one per line or comma-separated)
  const parsedUrls = useMemo(() => parseBatchUrls(urlsInput), [urlsInput]);
  const submissionSummary = useMemo(
    () =>
      submissionNotice
        ? summarizeBatchUrls(submissionNotice.submittedUrls)
        : null,
    [submissionNotice],
  );

  useEffect(() => {
    if (submissionNotice) {
      submissionNoticeRef.current?.focus();
    }
  }, [submissionNotice]);

  // Validate URL count
  const isValidBatchSize = parsedUrls.length <= MAX_BATCH_SIZE;

  const resolveAIExtractOptions = useCallback(() => {
    return buildAIExtractOptions(
      aiExtractEnabled,
      aiExtractMode,
      aiExtractPrompt,
      aiExtractSchema,
      aiExtractFields,
    );
  }, [
    aiExtractEnabled,
    aiExtractMode,
    aiExtractPrompt,
    aiExtractSchema,
    aiExtractFields,
  ]);

  const resolveAgenticOptions = useCallback(() => {
    return buildResearchAgenticOptions(
      agenticResearchEnabled,
      agenticResearchInstructions,
      agenticResearchMaxRounds,
      agenticResearchMaxFollowUpUrls,
    );
  }, [
    agenticResearchEnabled,
    agenticResearchInstructions,
    agenticResearchMaxRounds,
    agenticResearchMaxFollowUpUrls,
  ]);

  // Handle file upload for CSV/JSON
  const handleFileUpload = useCallback(
    async (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (!file) return;

      setFileError(null);

      try {
        const text = await file.text();
        let urls: string[] = [];

        if (file.name.endsWith(".json")) {
          const jobs = JSON.parse(text) as Array<{ url: string }>;
          if (!Array.isArray(jobs)) {
            throw new Error("JSON file must contain an array");
          }
          urls = jobs.map((j) => j.url).filter(Boolean);
        } else if (file.name.endsWith(".csv")) {
          // Simple CSV parsing - assumes first column is URL
          const lines = text.split("\n").filter(Boolean);
          const hasHeader =
            lines.length > 0 &&
            (lines[0].toLowerCase().includes("url") ||
              !lines[0].startsWith("http"));
          const startIdx = hasHeader ? 1 : 0;
          urls = lines
            .slice(startIdx)
            .map((line) => {
              const cols = line.split(",");
              return cols[0]?.trim();
            })
            .filter((u): u is string => Boolean(u) && u.startsWith("http"));
        } else {
          throw new Error("File must be .json or .csv");
        }

        if (urls.length === 0) {
          throw new Error("No valid URLs found in file");
        }

        if (urls.length > MAX_BATCH_SIZE) {
          throw new Error(
            `File contains ${urls.length} URLs, but maximum is ${MAX_BATCH_SIZE}`,
          );
        }

        setUrlsInput(urls.join("\n"));
      } catch (err) {
        setFileError(
          err instanceof Error ? err.message : "Failed to parse file",
        );
      }

      // Reset file input
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    },
    [setUrlsInput],
  );

  // Validate URLs
  const validateUrls = useCallback(() => {
    if (parsedUrls.length === 0) {
      setUrlError("At least one URL is required");
      return false;
    }
    if (parsedUrls.length > MAX_BATCH_SIZE) {
      setUrlError(`Maximum ${MAX_BATCH_SIZE} URLs allowed`);
      return false;
    }
    // Basic URL validation
    const invalid = parsedUrls.filter((u) => {
      try {
        new URL(u);
        return false;
      } catch {
        return true;
      }
    });
    if (invalid.length > 0) {
      setUrlError(`Invalid URLs: ${invalid.slice(0, 3).join(", ")}`);
      return false;
    }
    setUrlError(null);
    return true;
  }, [parsedUrls]);

  // Handle scrape submit
  const handleSubmitScrape = useCallback(async () => {
    if (!validateUrls()) return;

    let shared: ReturnType<typeof buildSharedRequestConfig>;
    let aiExtractOptions: ReturnType<typeof buildAIExtractOptions>;
    try {
      shared = buildSharedRequestConfig(form);
      aiExtractOptions = resolveAIExtractOptions();
    } catch (error) {
      toast.show({
        tone: "error",
        title: "Batch scrape configuration is invalid",
        description: error instanceof Error ? error.message : String(error),
      });
      return;
    }

    const request = buildBatchScrapeRequest(
      parsedUrls,
      headless,
      usePlaywright,
      timeoutSeconds,
      shared.authProfile,
      shared.auth,
      shared.extract,
      shared.pipeline,
      incremental,
      shared.webhook,
      shared.screenshot,
      device || undefined,
      shared.networkIntercept,
      aiExtractOptions,
    );

    await onSubmitScrape(request);
  }, [
    validateUrls,
    form,
    parsedUrls,
    headless,
    usePlaywright,
    timeoutSeconds,
    incremental,
    device,
    resolveAIExtractOptions,
    onSubmitScrape,
    toast,
  ]);

  // Handle crawl submit
  const handleSubmitCrawl = useCallback(async () => {
    if (!validateUrls()) return;

    let shared: ReturnType<typeof buildSharedRequestConfig>;
    let aiExtractOptions: ReturnType<typeof buildAIExtractOptions>;
    try {
      shared = buildSharedRequestConfig(form);
      aiExtractOptions = resolveAIExtractOptions();
    } catch (error) {
      toast.show({
        tone: "error",
        title: "Batch crawl configuration is invalid",
        description: error instanceof Error ? error.message : String(error),
      });
      return;
    }

    const request = buildBatchCrawlRequest(
      parsedUrls,
      maxDepth,
      maxPages,
      headless,
      usePlaywright,
      timeoutSeconds,
      shared.authProfile,
      shared.auth,
      shared.extract,
      shared.pipeline,
      incremental,
      shared.webhook,
      shared.screenshot,
      device || undefined,
      shared.networkIntercept,
      aiExtractOptions,
    );

    await onSubmitCrawl(request);
  }, [
    validateUrls,
    form,
    parsedUrls,
    maxDepth,
    maxPages,
    headless,
    usePlaywright,
    timeoutSeconds,
    incremental,
    device,
    resolveAIExtractOptions,
    onSubmitCrawl,
    toast,
  ]);

  // Handle research submit
  const handleSubmitResearch = useCallback(async () => {
    if (!validateUrls()) return;
    if (!query.trim()) {
      toast.show({
        tone: "warning",
        title: "Research query required",
        description: "Add the question you want this batch of URLs to answer.",
      });
      return;
    }

    let shared: ReturnType<typeof buildSharedRequestConfig>;
    let aiExtractOptions: ReturnType<typeof buildAIExtractOptions>;
    const agenticOptions = resolveAgenticOptions();
    try {
      shared = buildSharedRequestConfig(form);
      aiExtractOptions = resolveAIExtractOptions();
    } catch (error) {
      toast.show({
        tone: "error",
        title: "Batch research configuration is invalid",
        description: error instanceof Error ? error.message : String(error),
      });
      return;
    }

    const request = buildBatchResearchRequest(
      parsedUrls,
      query,
      maxDepth,
      maxPages,
      headless,
      usePlaywright,
      timeoutSeconds,
      shared.authProfile,
      shared.auth,
      shared.extract,
      shared.pipeline,
      shared.webhook,
      shared.screenshot,
      device || undefined,
      shared.networkIntercept,
      aiExtractOptions,
      agenticOptions,
    );

    await onSubmitResearch(request);
  }, [
    validateUrls,
    form,
    parsedUrls,
    query,
    maxDepth,
    maxPages,
    headless,
    usePlaywright,
    timeoutSeconds,
    device,
    resolveAIExtractOptions,
    resolveAgenticOptions,
    onSubmitResearch,
    toast,
  ]);

  // Handle submit based on active tab
  const handleSubmit = useCallback(() => {
    switch (activeTab) {
      case "scrape":
        return handleSubmitScrape();
      case "crawl":
        return handleSubmitCrawl();
      case "research":
        return handleSubmitResearch();
    }
  }, [activeTab, handleSubmitScrape, handleSubmitCrawl, handleSubmitResearch]);

  const handleFormSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    void handleSubmit();
  };

  return (
    <form className="panel" onSubmit={handleFormSubmit}>
      <h2>Batch Jobs</h2>

      {submissionNotice && submissionSummary ? (
        <output
          ref={submissionNoticeRef}
          className="batch-form__confirmation"
          aria-live="polite"
          tabIndex={-1}
        >
          <div className="batch-form__confirmation-copy">
            <div className="batch-form__confirmation-eyebrow">
              Batch submitted
            </div>
            <h3>
              Queued {submissionNotice.submittedUrls.length} URL
              {submissionNotice.submittedUrls.length === 1 ? "" : "s"} for{" "}
              {submissionNotice.kind}
            </h3>
            <p>
              Batch <code>{submissionNotice.batchId.slice(0, 8)}...</code> was
              accepted and is now tracked in Batch Jobs below. Use the list to
              inspect live progress and open results as each URL completes, even
              after refreshing this browser session.
            </p>
            <div className="batch-form__confirmation-preview">
              {submissionSummary.visible.map((url) => (
                <span key={url} className="signal-pill" title={url}>
                  {formatSubmittedUrl(url)}
                </span>
              ))}
              {submissionSummary.remaining > 0 ? (
                <span className="signal-pill">
                  +{submissionSummary.remaining} more
                </span>
              ) : null}
            </div>
          </div>
          <div className="batch-form__confirmation-actions">
            <button type="button" onClick={onViewSubmittedBatch}>
              View Batch Progress
            </button>
          </div>
        </output>
      ) : null}

      {/* Tab selector */}
      <div
        className="batch-tabs"
        style={{
          display: "flex",
          gap: 8,
          marginBottom: 16,
          borderBottom: "1px solid var(--border)",
          paddingBottom: 8,
        }}
      >
        {(["scrape", "crawl", "research"] as const).map((tab) => (
          <button
            key={tab}
            type="button"
            className={activeTab === tab ? "active" : "secondary"}
            onClick={() => setActiveTab(tab)}
            style={{
              textTransform: "capitalize",
              fontWeight: activeTab === tab ? 600 : 400,
            }}
          >
            Batch {tab}
          </button>
        ))}
      </div>

      {/* URL Input */}
      <label htmlFor="batch-urls">
        URLs ({parsedUrls.length}/{MAX_BATCH_SIZE})
        <small style={{ marginLeft: 8, color: "var(--text-muted)" }}>
          One per line or comma-separated
        </small>
      </label>
      <textarea
        id="batch-urls"
        rows={5}
        value={urlsInput}
        onChange={(e) => {
          setUrlsInput(e.target.value);
          setUrlError(null);
        }}
        placeholder="https://example.com&#10;https://example.org&#10;https://site.com"
        style={{
          borderColor:
            urlError || !isValidBatchSize ? "var(--error)" : undefined,
        }}
      />
      {urlError && <small style={{ color: "var(--error)" }}>{urlError}</small>}
      {!isValidBatchSize && (
        <small style={{ color: "var(--error)" }}>
          Maximum {MAX_BATCH_SIZE} URLs allowed
        </small>
      )}

      {/* File upload */}
      <div style={{ marginTop: 12 }}>
        <label
          htmlFor="batch-file"
          style={{ display: "flex", alignItems: "center", gap: 8 }}
        >
          <input
            ref={fileInputRef}
            id="batch-file"
            type="file"
            accept=".csv,.json"
            onChange={handleFileUpload}
            style={{ display: "none" }}
          />
          <button
            type="button"
            className="secondary"
            onClick={() => fileInputRef.current?.click()}
          >
            Upload CSV/JSON
          </button>
          <small style={{ color: "var(--text-muted)" }}>
            JSON: [{`{url: "..."}`}] or CSV: url,column2,...
          </small>
        </label>
        {fileError && (
          <small style={{ color: "var(--error)", display: "block" }}>
            {fileError}
          </small>
        )}
      </div>

      {/* Tab-specific fields */}
      {activeTab === "crawl" && (
        <div className="row" style={{ marginTop: 16 }}>
          <label>
            Max depth per job
            <input
              type="number"
              min={1}
              value={maxDepth}
              onChange={(e) => setMaxDepth(Number(e.target.value))}
            />
          </label>
          <label>
            Max pages per job
            <input
              type="number"
              min={1}
              value={maxPages}
              onChange={(e) => setMaxPages(Number(e.target.value))}
            />
          </label>
        </div>
      )}

      {activeTab === "research" && (
        <>
          <div className="row" style={{ marginTop: 16 }}>
            <label>
              Max depth per job
              <input
                type="number"
                min={1}
                value={maxDepth}
                onChange={(e) => setMaxDepth(Number(e.target.value))}
              />
            </label>
            <label>
              Max pages per job
              <input
                type="number"
                min={1}
                value={maxPages}
                onChange={(e) => setMaxPages(Number(e.target.value))}
              />
            </label>
          </div>
          <label htmlFor="batch-query" style={{ marginTop: 12 }}>
            Research Query (shared across all URLs)
          </label>
          <input
            id="batch-query"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Enter research query..."
          />
        </>
      )}

      {/* Common options */}
      <BrowserExecutionControls
        headless={headless}
        setHeadless={setHeadless}
        usePlaywright={usePlaywright}
        setUsePlaywright={setUsePlaywright}
        timeoutSeconds={timeoutSeconds}
        setTimeoutSeconds={setTimeoutSeconds}
      />

      <ScreenshotConfig
        enabled={screenshotEnabled}
        setEnabled={setScreenshotEnabled}
        fullPage={screenshotFullPage}
        setFullPage={setScreenshotFullPage}
        format={screenshotFormat}
        setFormat={setScreenshotFormat}
        quality={screenshotQuality}
        setQuality={setScreenshotQuality}
        width={screenshotWidth}
        setWidth={setScreenshotWidth}
        height={screenshotHeight}
        setHeight={setScreenshotHeight}
        disabled={!headless}
        inputPrefix="batch"
      />

      <DeviceSelector
        device={device}
        onChange={setDevice}
        disabled={!headless}
      />

      <NetworkInterceptConfig
        enabled={interceptEnabled}
        setEnabled={setInterceptEnabled}
        urlPatterns={interceptURLPatterns}
        setURLPatterns={setInterceptURLPatterns}
        resourceTypes={interceptResourceTypes}
        setResourceTypes={setInterceptResourceTypes}
        captureRequestBody={interceptCaptureRequestBody}
        setCaptureRequestBody={setInterceptCaptureRequestBody}
        captureResponseBody={interceptCaptureResponseBody}
        setCaptureResponseBody={setInterceptCaptureResponseBody}
        maxBodySize={interceptMaxBodySize}
        setMaxBodySize={setInterceptMaxBodySize}
        maxEntries={interceptMaxEntries}
        setMaxEntries={setInterceptMaxEntries}
        disabled={!headless}
        inputPrefix="batch"
      />

      <AuthConfig
        authProfile={authProfile}
        setAuthProfile={setAuthProfile}
        authBasic={authBasic}
        setAuthBasic={setAuthBasic}
        headersRaw={headersRaw}
        setHeadersRaw={setHeadersRaw}
        cookiesRaw={cookiesRaw}
        setCookiesRaw={setCookiesRaw}
        queryRaw={queryRaw}
        setQueryRaw={setQueryRaw}
        proxyUrl={proxyUrl}
        setProxyUrl={setProxyUrl}
        proxyUsername={proxyUsername}
        setProxyUsername={setProxyUsername}
        proxyPassword={proxyPassword}
        setProxyPassword={setProxyPassword}
        proxyRegion={proxyRegion}
        setProxyRegion={setProxyRegion}
        proxyRequiredTags={proxyRequiredTags}
        setProxyRequiredTags={setProxyRequiredTags}
        proxyExcludeProxyIds={proxyExcludeProxyIds}
        setProxyExcludeProxyIds={setProxyExcludeProxyIds}
        loginUrl={loginUrl}
        setLoginUrl={setLoginUrl}
        loginUserSelector={loginUserSelector}
        setLoginUserSelector={setLoginUserSelector}
        loginPassSelector={loginPassSelector}
        setLoginPassSelector={setLoginPassSelector}
        loginSubmitSelector={loginSubmitSelector}
        setLoginSubmitSelector={setLoginSubmitSelector}
        loginUser={loginUser}
        setLoginUser={setLoginUser}
        loginPass={loginPass}
        setLoginPass={setLoginPass}
        profiles={profiles}
      />

      <PipelineOptions
        extractTemplate={extractTemplate}
        setExtractTemplate={setExtractTemplate}
        extractValidate={extractValidate}
        setExtractValidate={setExtractValidate}
        preProcessors={preProcessors}
        setPreProcessors={setPreProcessors}
        postProcessors={postProcessors}
        setPostProcessors={setPostProcessors}
        transformers={transformers}
        setTransformers={setTransformers}
        incremental={incremental}
        setIncremental={setIncremental}
        inputPrefix="batch"
      />

      <AIExtractSection
        enabled={aiExtractEnabled}
        setEnabled={setAIExtractEnabled}
        mode={aiExtractMode}
        setMode={setAIExtractMode}
        prompt={aiExtractPrompt}
        setPrompt={setAIExtractPrompt}
        schemaText={aiExtractSchema}
        setSchemaText={setAIExtractSchema}
        fields={aiExtractFields}
        setFields={setAIExtractFields}
      />

      {activeTab === "research" ? (
        <ResearchAgenticSection
          enabled={agenticResearchEnabled}
          setEnabled={setAgenticResearchEnabled}
          instructions={agenticResearchInstructions}
          setInstructions={setAgenticResearchInstructions}
          maxRounds={agenticResearchMaxRounds}
          setMaxRounds={setAgenticResearchMaxRounds}
          maxFollowUpUrls={agenticResearchMaxFollowUpUrls}
          setMaxFollowUpUrls={setAgenticResearchMaxFollowUpUrls}
        />
      ) : null}

      <WebhookConfig
        webhookUrl={webhookUrl}
        setWebhookUrl={setWebhookUrl}
        webhookEvents={webhookEvents}
        setWebhookEvents={setWebhookEvents}
        webhookSecret={webhookSecret}
        setWebhookSecret={setWebhookSecret}
        inputPrefix="batch"
      />

      {/* Submit */}
      <div style={{ marginTop: 16, display: "flex", gap: 12 }}>
        <button
          type="submit"
          disabled={loading || parsedUrls.length === 0 || !isValidBatchSize}
        >
          Submit Batch {activeTab}
          {parsedUrls.length > 0 && ` (${parsedUrls.length} URLs)`}
        </button>
        <button
          type="button"
          className="secondary"
          onClick={() => {
            setUrlsInput("");
            setQuery("");
            setUrlError(null);
            setFileError(null);
          }}
        >
          Clear
        </button>
      </div>
    </form>
  );
}
