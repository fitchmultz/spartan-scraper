/**
 * Purpose: Render the expert single-page scrape surface and expose imperative submission/config helpers for guided and command-palette flows.
 * Responsibilities: Keep scrape-specific inputs controlled, build scrape requests from shared form state, and optionally mount in headless mode for non-visual wizard submission.
 * Scope: Single-page scrape job authoring only.
 * Usage: Render from `JobSubmissionContainer` with shared form state plus controlled scrape-local fields.
 * Invariants/Assumptions: Shared runtime and extraction options live in `useFormState`, the target URL is required before submit, and `surface="headless"` must still provide a working imperative ref.
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
  buildScrapeRequest,
  buildSharedRequestConfig,
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
import type { AiExtractOptions, DeviceEmulation } from "../api";

export interface ScrapeFormRef {
  submit: () => Promise<void>;
  getUrl: () => string;
  setUrl: (url: string) => void;
  getConfig: () => PresetConfig;
}

interface ScrapeFormProps {
  form: FormController;
  profiles: ProfileOption[];
  onSubmit: (request: import("../api").ScrapeRequest) => Promise<void>;
  loading: boolean;
  url: string;
  setUrl: (value: string) => void;
  device: DeviceEmulation | null;
  setDevice: (value: DeviceEmulation | null) => void;
  surface?: "full" | "headless";
}

export const ScrapeForm = forwardRef<ScrapeFormRef, ScrapeFormProps>(
  function ScrapeForm(
    {
      form,
      profiles,
      onSubmit,
      loading,
      url,
      setUrl,
      device,
      setDevice,
      surface = "full",
    },
    ref,
  ) {
    const handleSubmit = useCallback(async () => {
      if (!url.trim()) {
        alert("Scrape URL is required.");
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
        alert(error instanceof Error ? error.message : String(error));
        return;
      }

      const request = buildScrapeRequest(
        url.trim(),
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
        shared.webhook,
        shared.screenshot,
        device || undefined,
        shared.networkIntercept,
        aiExtractOptions,
      );
      await onSubmit(request);
    }, [device, form, onSubmit, url]);

    const getConfig = useCallback((): PresetConfig => {
      const draftState: JobDraftLocalState = {
        scrape: {
          url,
          device,
        },
        crawl: {
          url: "",
          sitemapURL: "",
          sitemapOnly: false,
          includePatterns: "",
          excludePatterns: "",
          device: null,
        },
        research: {
          query: "",
          urls: "",
          device: null,
        },
      };

      return buildPresetConfig("scrape", form, draftState);
    }, [device, form, url]);

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
          title="Scrape a Page"
          description="Drop in one URL, keep the execution controls nearby, and launch without wading through the full advanced stack."
          actions={
            <>
              <button type="submit" disabled={loading}>
                Deploy Scrape
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
          <label htmlFor="scrape-url">Target URL</label>
          <input
            id="scrape-url"
            value={url}
            onChange={(event) => setUrl(event.target.value)}
            placeholder="https://example.com"
          />
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
            inputPrefix="scrape"
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
            inputPrefix="scrape"
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
          title="Extraction and processing"
          description="Templates, validation, transformers, and AI extraction helpers."
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
            inputPrefix="scrape"
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
        </JobFormAdvancedSection>

        <JobFormAdvancedSection
          title="Delivery hooks"
          description="Optional webhook notifications for external systems."
        >
          <WebhookConfig
            webhookUrl={form.webhookUrl}
            setWebhookUrl={form.setWebhookUrl}
            webhookEvents={form.webhookEvents}
            setWebhookEvents={form.setWebhookEvents}
            webhookSecret={form.webhookSecret}
            setWebhookSecret={form.setWebhookSecret}
            inputPrefix="scrape"
          />
        </JobFormAdvancedSection>
      </form>
    );
  },
);

ScrapeForm.displayName = "ScrapeForm";
