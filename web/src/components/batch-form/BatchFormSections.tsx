/**
 * Purpose: Render the reusable visual sections that make up the batch authoring form.
 * Responsibilities: Present submission feedback, tab controls, URL/file inputs, tab-specific fields, shared execution options, and footer actions without owning submission side effects.
 * Scope: Batch form presentation only; validation, request construction, and async submit orchestration stay in `useBatchFormSubmission`.
 * Usage: Compose these sections from `BatchForm` to keep the main container focused on state wiring.
 * Invariants/Assumptions: UI copy, labels, and input ids remain stable for existing tests and operator workflows.
 */

import type { ChangeEvent, RefObject } from "react";

import type { DeviceEmulation } from "../../api";
import type { FormController, ProfileOption } from "../../hooks/useFormState";
import { summarizeBatchUrls, MAX_BATCH_SIZE } from "../../lib/batch-urls";
import { AIExtractSection } from "../AIExtractSection";
import { AuthConfig } from "../AuthConfig";
import { BrowserExecutionControls } from "../BrowserExecutionControls";
import { DeviceSelector } from "../DeviceSelector";
import { NetworkInterceptConfig } from "../NetworkInterceptConfig";
import { PipelineOptions } from "../PipelineOptions";
import { ResearchAgenticSection } from "../ResearchAgenticSection";
import { ScreenshotConfig } from "../ScreenshotConfig";
import { WebhookConfig } from "../WebhookConfig";
import type { BatchFormTab, BatchSubmissionNotice } from "./types";

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

export function BatchSubmissionNoticePanel({
  submissionNotice,
  onViewSubmittedBatch,
  submissionNoticeRef,
}: {
  submissionNotice: BatchSubmissionNotice | null;
  onViewSubmittedBatch: () => void;
  submissionNoticeRef: RefObject<HTMLOutputElement | null>;
}) {
  const submissionSummary = submissionNotice
    ? summarizeBatchUrls(submissionNotice.submittedUrls)
    : null;

  if (!submissionNotice || !submissionSummary) {
    return null;
  }

  return (
    <output
      ref={submissionNoticeRef}
      className="batch-form__confirmation"
      aria-live="polite"
      tabIndex={-1}
    >
      <div className="batch-form__confirmation-copy">
        <div className="batch-form__confirmation-eyebrow">Batch submitted</div>
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
  );
}

export function BatchTabSelector({
  activeTab,
  setActiveTab,
}: {
  activeTab: BatchFormTab;
  setActiveTab: (tab: BatchFormTab) => void;
}) {
  return (
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
  );
}

export function BatchUrlInputSection({
  parsedUrlCount,
  urlsInput,
  onUrlsInputChange,
  isValidBatchSize,
  urlError,
  fileInputRef,
  onFileUpload,
  fileError,
}: {
  parsedUrlCount: number;
  urlsInput: string;
  onUrlsInputChange: (value: string) => void;
  isValidBatchSize: boolean;
  urlError: string | null;
  fileInputRef: RefObject<HTMLInputElement | null>;
  onFileUpload: (event: ChangeEvent<HTMLInputElement>) => void;
  fileError: string | null;
}) {
  return (
    <>
      <label htmlFor="batch-urls">
        URLs ({parsedUrlCount}/{MAX_BATCH_SIZE})
        <small style={{ marginLeft: 8, color: "var(--text-muted)" }}>
          One per line or comma-separated
        </small>
      </label>
      <textarea
        id="batch-urls"
        rows={5}
        value={urlsInput}
        onChange={(event) => onUrlsInputChange(event.target.value)}
        placeholder="https://example.com&#10;https://example.org&#10;https://site.com"
        style={{
          borderColor:
            urlError || !isValidBatchSize ? "var(--error)" : undefined,
        }}
      />
      {urlError ? (
        <small style={{ color: "var(--error)" }}>{urlError}</small>
      ) : null}
      {!isValidBatchSize ? (
        <small style={{ color: "var(--error)" }}>
          Maximum {MAX_BATCH_SIZE} URLs allowed
        </small>
      ) : null}

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
            onChange={onFileUpload}
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
        {fileError ? (
          <small style={{ color: "var(--error)", display: "block" }}>
            {fileError}
          </small>
        ) : null}
      </div>
    </>
  );
}

export function BatchTabSpecificFields({
  activeTab,
  maxDepth,
  setMaxDepth,
  maxPages,
  setMaxPages,
  query,
  setQuery,
}: {
  activeTab: BatchFormTab;
  maxDepth: number;
  setMaxDepth: (value: number) => void;
  maxPages: number;
  setMaxPages: (value: number) => void;
  query: string;
  setQuery: (value: string) => void;
}) {
  if (activeTab === "crawl") {
    return (
      <div className="row" style={{ marginTop: 16 }}>
        <label>
          Max depth per job
          <input
            type="number"
            min={1}
            value={maxDepth}
            onChange={(event) => setMaxDepth(Number(event.target.value))}
          />
        </label>
        <label>
          Max pages per job
          <input
            type="number"
            min={1}
            value={maxPages}
            onChange={(event) => setMaxPages(Number(event.target.value))}
          />
        </label>
      </div>
    );
  }

  if (activeTab !== "research") {
    return null;
  }

  return (
    <>
      <div className="row" style={{ marginTop: 16 }}>
        <label>
          Max depth per job
          <input
            type="number"
            min={1}
            value={maxDepth}
            onChange={(event) => setMaxDepth(Number(event.target.value))}
          />
        </label>
        <label>
          Max pages per job
          <input
            type="number"
            min={1}
            value={maxPages}
            onChange={(event) => setMaxPages(Number(event.target.value))}
          />
        </label>
      </div>
      <label htmlFor="batch-query" style={{ marginTop: 12 }}>
        Research Query (shared across all URLs)
      </label>
      <input
        id="batch-query"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder="Enter research query..."
      />
    </>
  );
}

export function BatchCommonOptions({
  activeTab,
  form,
  profiles,
  device,
  onDeviceChange,
}: {
  activeTab: BatchFormTab;
  form: FormController;
  profiles: ProfileOption[];
  device: DeviceEmulation | null;
  onDeviceChange: (device: DeviceEmulation | null) => void;
}) {
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

  return (
    <>
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
        onChange={onDeviceChange}
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
    </>
  );
}

export function BatchSubmitActions({
  activeTab,
  parsedUrlCount,
  isValidBatchSize,
  loading,
  onClear,
}: {
  activeTab: BatchFormTab;
  parsedUrlCount: number;
  isValidBatchSize: boolean;
  loading: boolean;
  onClear: () => void;
}) {
  return (
    <div style={{ marginTop: 16, display: "flex", gap: 12 }}>
      <button
        type="submit"
        disabled={loading || parsedUrlCount === 0 || !isValidBatchSize}
      >
        Submit Batch {activeTab}
        {parsedUrlCount > 0 &&
          ` (${parsedUrlCount} URL${parsedUrlCount === 1 ? "" : "s"})`}
      </button>
      <button type="button" className="secondary" onClick={onClear}>
        Clear
      </button>
    </div>
  );
}
