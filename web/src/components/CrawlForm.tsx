/**
 * Purpose: Render the expert crawl authoring surface and expose imperative submission/config helpers for guided and command-palette flows.
 * Responsibilities: Keep crawl-local scope fields controlled, build crawl requests from shared form state, and optionally mount headlessly for wizard submission.
 * Scope: Site crawl job authoring only.
 * Usage: Render from `JobSubmissionContainer` with shared form state plus controlled crawl-local inputs.
 * Invariants/Assumptions: The root URL is required before submit, max depth and max pages are owned by `useFormState`, and `surface="headless"` must still provide a working imperative ref.
 */

import {
  useCallback,
  forwardRef,
  useImperativeHandle,
  type FormEvent,
} from "react";
import { AuthConfig } from "./AuthConfig";
import { PipelineOptions } from "./PipelineOptions";
import { AIExtractSection } from "./AIExtractSection";
import {
  buildAIExtractOptions,
  buildCrawlRequest,
  buildSharedRequestConfig,
  getHttpUrlValidationState,
  parsePatternList,
} from "../lib/form-utils";
import { buildPresetConfig, type JobDraftLocalState } from "../lib/job-drafts";
import type { FormController, ProfileOption } from "../hooks/useFormState";
import type { PresetConfig } from "../types/presets";
import { WebhookConfig } from "./WebhookConfig";
import { BrowserExecutionControls } from "./BrowserExecutionControls";
import { DeviceSelector } from "./DeviceSelector";
import { ScreenshotConfig } from "./ScreenshotConfig";
import { NetworkInterceptConfig } from "./NetworkInterceptConfig";
import { JobFormAdvancedSection, JobFormIntro } from "./jobs/JobFormSections";
import { useToast } from "./toast";
import type { AiExtractOptions, DeviceEmulation } from "../api";

export interface CrawlFormRef {
  submit: () => Promise<void>;
  getUrl: () => string;
  setUrl: (url: string) => void;
  getConfig: () => PresetConfig;
}

interface CrawlFormProps {
  form: FormController;
  profiles: ProfileOption[];
  onSubmit: (request: import("../api").CrawlRequest) => Promise<void>;
  loading: boolean;
  url: string;
  setUrl: (value: string) => void;
  sitemapURL: string;
  setSitemapURL: (value: string) => void;
  sitemapOnly: boolean;
  setSitemapOnly: (value: boolean) => void;
  includePatterns: string;
  setIncludePatterns: (value: string) => void;
  excludePatterns: string;
  setExcludePatterns: (value: string) => void;
  device: DeviceEmulation | null;
  setDevice: (value: DeviceEmulation | null) => void;
  surface?: "full" | "headless";
}

export const CrawlForm = forwardRef<CrawlFormRef, CrawlFormProps>(
  function CrawlForm(
    {
      form,
      profiles,
      onSubmit,
      loading,
      url,
      setUrl,
      sitemapURL,
      setSitemapURL,
      sitemapOnly,
      setSitemapOnly,
      includePatterns,
      setIncludePatterns,
      excludePatterns,
      setExcludePatterns,
      device,
      setDevice,
      surface = "full",
    },
    ref,
  ) {
    const toast = useToast();

    const handleSubmit = useCallback(async () => {
      const urlState = getHttpUrlValidationState(url);
      if (urlState === "missing") {
        toast.show({
          tone: "warning",
          title: "Crawl URL required",
          description: "Add a root URL before launching the crawl.",
        });
        return;
      }
      if (urlState === "invalid") {
        toast.show({
          tone: "warning",
          title: "Crawl URL is invalid",
          description:
            "Use a full http:// or https:// root URL with a host before launching the crawl.",
        });
        return;
      }

      let shared: ReturnType<typeof buildSharedRequestConfig>;
      let aiExtractOptions: AiExtractOptions | undefined;
      try {
        shared = buildSharedRequestConfig(form);
        aiExtractOptions = buildAIExtractOptions(
          form.aiExtractEnabled,
          form.aiExtractMode,
          form.aiExtractPrompt,
          form.aiExtractSchema,
          form.aiExtractFields,
        );
      } catch (error) {
        toast.show({
          tone: "error",
          title: "Crawl configuration is invalid",
          description: error instanceof Error ? error.message : String(error),
        });
        return;
      }

      const request = buildCrawlRequest(
        url.trim(),
        form.maxDepth,
        form.maxPages,
        form.headless,
        form.usePlaywright,
        form.timeoutSeconds,
        shared.authProfile,
        shared.auth,
        shared.extract,
        shared.preProcessors,
        shared.postProcessors,
        shared.transformers,
        form.incremental,
        sitemapURL,
        sitemapOnly,
        shared.webhook,
        parsePatternList(includePatterns),
        parsePatternList(excludePatterns),
        shared.screenshot,
        device || undefined,
        shared.networkIntercept,
        aiExtractOptions,
      );
      await onSubmit(request);
    }, [
      device,
      excludePatterns,
      form,
      includePatterns,
      onSubmit,
      sitemapOnly,
      sitemapURL,
      toast,
      url,
    ]);

    const getConfig = useCallback((): PresetConfig => {
      const draftState: JobDraftLocalState = {
        scrape: {
          url: "",
          device: null,
        },
        crawl: {
          url,
          sitemapURL,
          sitemapOnly,
          includePatterns,
          excludePatterns,
          device,
        },
        research: {
          query: "",
          urls: "",
          device: null,
        },
      };

      return buildPresetConfig("crawl", form, draftState);
    }, [
      device,
      excludePatterns,
      form,
      includePatterns,
      sitemapOnly,
      sitemapURL,
      url,
    ]);

    useImperativeHandle(
      ref,
      () => ({
        submit: handleSubmit,
        getUrl: () => url,
        setUrl,
        getConfig,
      }),
      [getConfig, handleSubmit, setUrl, url],
    );

    if (surface === "headless") {
      return null;
    }

    const handleFormSubmit = (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      void handleSubmit();
    };

    return (
      <form className="job-workflow-form" onSubmit={handleFormSubmit}>
        <JobFormIntro
          title="Crawl a Site"
          description="Set the root URL, define the crawl boundaries, and launch the sweep without paging through every advanced block first."
          actions={
            <>
              <button type="submit" disabled={loading}>
                Launch Crawl
              </button>
              <button
                type="button"
                className="secondary"
                onClick={() => setUrl("")}
              >
                Clear
              </button>
            </>
          }
        >
          <label htmlFor="crawl-url">Root URL</label>
          <input
            id="crawl-url"
            value={url}
            onChange={(event) => setUrl(event.target.value)}
            placeholder="https://example.com"
          />
          <div className="row" style={{ marginTop: 12 }}>
            <label>
              Max depth
              <input
                type="number"
                min={1}
                value={form.maxDepth}
                onChange={(event) =>
                  form.setMaxDepth(Number(event.target.value))
                }
              />
            </label>
            <label>
              Max pages
              <input
                type="number"
                min={1}
                value={form.maxPages}
                onChange={(event) =>
                  form.setMaxPages(Number(event.target.value))
                }
              />
            </label>
            <label>
              Timeout (s)
              <input
                type="number"
                min={5}
                value={form.timeoutSeconds}
                onChange={(event) =>
                  form.setTimeoutSeconds(Number(event.target.value))
                }
              />
            </label>
          </div>
          <BrowserExecutionControls
            headless={form.headless}
            setHeadless={form.setHeadless}
            usePlaywright={form.usePlaywright}
            setUsePlaywright={form.setUsePlaywright}
            timeoutSeconds={form.timeoutSeconds}
            setTimeoutSeconds={form.setTimeoutSeconds}
          />
        </JobFormIntro>

        <JobFormAdvancedSection
          title="Scope and discovery rules"
          description="Sitemaps and include or exclude patterns that refine crawl coverage."
        >
          <label htmlFor="sitemap-url">Sitemap URL (optional)</label>
          <input
            id="sitemap-url"
            value={sitemapURL}
            onChange={(event) => setSitemapURL(event.target.value)}
            placeholder="https://example.com/sitemap.xml"
          />
          <label
            style={{
              marginTop: 8,
              display: "flex",
              alignItems: "center",
              gap: 8,
            }}
          >
            <input
              type="checkbox"
              checked={sitemapOnly}
              disabled={!sitemapURL}
              onChange={(event) => setSitemapOnly(event.target.checked)}
            />
            Sitemap only (don't crawl root URL)
          </label>
          <label htmlFor="include-patterns" style={{ marginTop: 12 }}>
            Include Patterns (optional)
          </label>
          <input
            id="include-patterns"
            value={includePatterns}
            onChange={(event) => setIncludePatterns(event.target.value)}
            placeholder="/blog/**, /products/*"
          />
          <small>
            Comma-separated glob patterns. Only matching URLs will be crawled.
          </small>
          <label htmlFor="exclude-patterns" style={{ marginTop: 12 }}>
            Exclude Patterns (optional)
          </label>
          <input
            id="exclude-patterns"
            value={excludePatterns}
            onChange={(event) => setExcludePatterns(event.target.value)}
            placeholder="/admin/*, /api/**"
          />
          <small>
            Comma-separated glob patterns. Matching URLs will be skipped.
          </small>
        </JobFormAdvancedSection>

        <JobFormAdvancedSection
          title="Browser and capture controls"
          description="Screenshot capture, device emulation, network interception, and browser-only diagnostics."
        >
          <ScreenshotConfig
            enabled={form.screenshotEnabled}
            setEnabled={form.setScreenshotEnabled}
            fullPage={form.screenshotFullPage}
            setFullPage={form.setScreenshotFullPage}
            format={form.screenshotFormat}
            setFormat={form.setScreenshotFormat}
            quality={form.screenshotQuality}
            setQuality={form.setScreenshotQuality}
            width={form.screenshotWidth}
            setWidth={form.setScreenshotWidth}
            height={form.screenshotHeight}
            setHeight={form.setScreenshotHeight}
            disabled={!form.headless}
            inputPrefix="crawl"
          />
          <DeviceSelector
            device={device}
            onChange={setDevice}
            disabled={!form.headless}
          />
          <NetworkInterceptConfig
            enabled={form.interceptEnabled}
            setEnabled={form.setInterceptEnabled}
            urlPatterns={form.interceptURLPatterns}
            setURLPatterns={form.setInterceptURLPatterns}
            resourceTypes={form.interceptResourceTypes}
            setResourceTypes={form.setInterceptResourceTypes}
            captureRequestBody={form.interceptCaptureRequestBody}
            setCaptureRequestBody={form.setInterceptCaptureRequestBody}
            captureResponseBody={form.interceptCaptureResponseBody}
            setCaptureResponseBody={form.setInterceptCaptureResponseBody}
            maxBodySize={form.interceptMaxBodySize}
            setMaxBodySize={form.setInterceptMaxBodySize}
            maxEntries={form.interceptMaxEntries}
            setMaxEntries={form.setInterceptMaxEntries}
            disabled={!form.headless}
            inputPrefix="crawl"
          />
        </JobFormAdvancedSection>

        <JobFormAdvancedSection
          title="Authentication and request shaping"
          description="Profiles, cookies, login automation, and request overrides."
        >
          <AuthConfig
            authProfile={form.authProfile}
            setAuthProfile={form.setAuthProfile}
            authBasic={form.authBasic}
            setAuthBasic={form.setAuthBasic}
            headersRaw={form.headersRaw}
            setHeadersRaw={form.setHeadersRaw}
            cookiesRaw={form.cookiesRaw}
            setCookiesRaw={form.setCookiesRaw}
            queryRaw={form.queryRaw}
            setQueryRaw={form.setQueryRaw}
            proxyUrl={form.proxyUrl}
            setProxyUrl={form.setProxyUrl}
            proxyUsername={form.proxyUsername}
            setProxyUsername={form.setProxyUsername}
            proxyPassword={form.proxyPassword}
            setProxyPassword={form.setProxyPassword}
            proxyRegion={form.proxyRegion}
            setProxyRegion={form.setProxyRegion}
            proxyRequiredTags={form.proxyRequiredTags}
            setProxyRequiredTags={form.setProxyRequiredTags}
            proxyExcludeProxyIds={form.proxyExcludeProxyIds}
            setProxyExcludeProxyIds={form.setProxyExcludeProxyIds}
            loginUrl={form.loginUrl}
            setLoginUrl={form.setLoginUrl}
            loginUserSelector={form.loginUserSelector}
            setLoginUserSelector={form.setLoginUserSelector}
            loginPassSelector={form.loginPassSelector}
            setLoginPassSelector={form.setLoginPassSelector}
            loginSubmitSelector={form.loginSubmitSelector}
            setLoginSubmitSelector={form.setLoginSubmitSelector}
            loginUser={form.loginUser}
            setLoginUser={form.setLoginUser}
            loginPass={form.loginPass}
            setLoginPass={form.setLoginPass}
            profiles={profiles}
          />
        </JobFormAdvancedSection>

        <JobFormAdvancedSection
          title="Extraction and delivery"
          description="Templates, processors, AI extraction, incremental runs, and webhook notifications."
        >
          <PipelineOptions
            extractTemplate={form.extractTemplate}
            setExtractTemplate={form.setExtractTemplate}
            extractValidate={form.extractValidate}
            setExtractValidate={form.setExtractValidate}
            preProcessors={form.preProcessors}
            setPreProcessors={form.setPreProcessors}
            postProcessors={form.postProcessors}
            setPostProcessors={form.setPostProcessors}
            transformers={form.transformers}
            setTransformers={form.setTransformers}
            incremental={form.incremental}
            setIncremental={form.setIncremental}
            inputPrefix="crawl"
          />
          <AIExtractSection
            enabled={form.aiExtractEnabled}
            setEnabled={form.setAIExtractEnabled}
            mode={form.aiExtractMode}
            setMode={form.setAIExtractMode}
            prompt={form.aiExtractPrompt}
            setPrompt={form.setAIExtractPrompt}
            schemaText={form.aiExtractSchema}
            setSchemaText={form.setAIExtractSchema}
            fields={form.aiExtractFields}
            setFields={form.setAIExtractFields}
          />
          <WebhookConfig
            webhookUrl={form.webhookUrl}
            setWebhookUrl={form.setWebhookUrl}
            webhookEvents={form.webhookEvents}
            setWebhookEvents={form.setWebhookEvents}
            webhookSecret={form.webhookSecret}
            setWebhookSecret={form.setWebhookSecret}
            inputPrefix="crawl"
          />
        </JobFormAdvancedSection>
      </form>
    );
  },
);

CrawlForm.displayName = "CrawlForm";
