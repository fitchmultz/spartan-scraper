/**
 * Batch Form Component
 *
 * Form for submitting batch jobs (scrape, crawl, research).
 * Supports URL list input (textarea), file upload (CSV/JSON),
 * and shared configuration options.
 *
 * @module BatchForm
 */
import { useMemo, useState, useCallback, useRef, type FormEvent } from "react";
import { AuthConfig } from "./AuthConfig";
import { PipelineOptions } from "./PipelineOptions";
import { buildSharedRequestConfig } from "../lib/form-utils";
import type { FormController, ProfileOption } from "../hooks/useFormState";
import { WebhookConfig } from "./WebhookConfig";
import { DeviceSelector } from "./DeviceSelector";
import type { DeviceEmulation } from "../api";
import type {
  BatchScrapeRequest,
  BatchCrawlRequest,
  BatchResearchRequest,
  BatchJobRequest,
} from "../api";

interface BatchFormProps {
  // Job type selector
  activeTab: "scrape" | "crawl" | "research";
  setActiveTab: (tab: "scrape" | "crawl" | "research") => void;

  form: FormController;
  profiles: ProfileOption[];

  // Batch-specific
  urlsInput: string;
  setUrlsInput: (value: string) => void;

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

const MAX_BATCH_SIZE = 100;

export function BatchForm({
  activeTab,
  setActiveTab,
  form,
  profiles,
  urlsInput,
  setUrlsInput,
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
  } = form;

  const [device, setDevice] = useState<DeviceEmulation | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [urlError, setUrlError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Parse URLs from input (one per line or comma-separated)
  const parsedUrls = useMemo(() => {
    if (!urlsInput.trim()) return [];
    return urlsInput
      .split(/[\n,]/)
      .map((u) => u.trim())
      .filter(Boolean);
  }, [urlsInput]);

  // Validate URL count
  const isValidBatchSize = parsedUrls.length <= MAX_BATCH_SIZE;

  // Build job requests from URLs
  const buildJobRequests = useCallback((): BatchJobRequest[] => {
    return parsedUrls.map((url) => ({ url }));
  }, [parsedUrls]);

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

    const jobs = buildJobRequests();
    const shared = buildSharedRequestConfig(form);
    const request: BatchScrapeRequest = {
      jobs,
      headless,
      playwright: headless ? usePlaywright : false,
      timeoutSeconds,
      authProfile: shared.authProfile,
      auth: shared.auth,
      extract: shared.extract,
      pipeline: shared.pipeline,
      incremental: incremental || undefined,
      webhook: shared.webhook,
      device: device || undefined,
    };

    await onSubmitScrape(request);
  }, [
    validateUrls,
    buildJobRequests,
    headless,
    usePlaywright,
    timeoutSeconds,
    form,
    incremental,
    device,
    onSubmitScrape,
  ]);

  // Handle crawl submit
  const handleSubmitCrawl = useCallback(async () => {
    if (!validateUrls()) return;

    const jobs = buildJobRequests();
    const shared = buildSharedRequestConfig(form);
    const request: BatchCrawlRequest = {
      jobs,
      maxDepth,
      maxPages,
      headless,
      playwright: headless ? usePlaywright : false,
      timeoutSeconds,
      authProfile: shared.authProfile,
      auth: shared.auth,
      extract: shared.extract,
      pipeline: shared.pipeline,
      incremental: incremental || undefined,
      webhook: shared.webhook,
      device: device || undefined,
    };

    await onSubmitCrawl(request);
  }, [
    validateUrls,
    buildJobRequests,
    maxDepth,
    maxPages,
    headless,
    usePlaywright,
    timeoutSeconds,
    form,
    incremental,
    device,
    onSubmitCrawl,
  ]);

  // Handle research submit
  const handleSubmitResearch = useCallback(async () => {
    if (!validateUrls()) return;
    if (!query.trim()) {
      alert("Research query is required");
      return;
    }

    const jobs = buildJobRequests();
    const shared = buildSharedRequestConfig(form);
    const request: BatchResearchRequest = {
      jobs,
      query,
      maxDepth,
      maxPages,
      headless,
      playwright: headless ? usePlaywright : false,
      timeoutSeconds,
      authProfile: shared.authProfile,
      auth: shared.auth,
      extract: shared.extract,
      pipeline: shared.pipeline,
      webhook: shared.webhook,
      device: device || undefined,
    };

    await onSubmitResearch(request);
  }, [
    validateUrls,
    buildJobRequests,
    query,
    maxDepth,
    maxPages,
    headless,
    usePlaywright,
    timeoutSeconds,
    form,
    device,
    onSubmitResearch,
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
      <div className="row" style={{ marginTop: 16 }}>
        <label>
          <input
            type="checkbox"
            checked={headless}
            onChange={(e) => setHeadless(e.target.checked)}
          />{" "}
          Headless
        </label>
        <label>
          <input
            type="checkbox"
            checked={usePlaywright}
            disabled={!headless}
            onChange={(e) => setUsePlaywright(e.target.checked)}
          />{" "}
          Playwright
        </label>
        <label>
          Timeout (s)
          <input
            type="number"
            min={5}
            value={timeoutSeconds}
            onChange={(e) => setTimeoutSeconds(Number(e.target.value))}
          />
        </label>
      </div>

      <DeviceSelector
        device={device}
        onChange={setDevice}
        disabled={!headless}
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
