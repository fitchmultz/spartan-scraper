/**
 * Purpose: Render the expert research authoring surface and expose imperative submission/config helpers for guided and command-palette flows.
 * Responsibilities: Keep research-local query/source fields controlled, build research requests from shared form state, and optionally mount headlessly for wizard submission.
 * Scope: Research job authoring only.
 * Usage: Render from `JobSubmissionContainer` with shared form state plus controlled research-local inputs.
 * Invariants/Assumptions: Research jobs require both a query and source URLs, shared crawl/runtime fields live in `useFormState`, and `surface="headless"` must still provide a working imperative ref.
 */

import {
  useCallback,
  forwardRef,
  useImperativeHandle,
  type FormEvent,
} from "react";
import { AuthConfig } from "./AuthConfig";
import { PipelineOptions } from "./PipelineOptions";
import {
  buildAIExtractOptions,
  buildResearchAgenticOptions,
  buildResearchRequest,
  buildSharedRequestConfig,
  parseUrlList,
} from "../lib/form-utils";
import { buildPresetConfig, type JobDraftLocalState } from "../lib/job-drafts";
import type { FormController, ProfileOption } from "../hooks/useFormState";
import type { PresetConfig } from "../types/presets";
import { WebhookConfig } from "./WebhookConfig";
import { BrowserExecutionControls } from "./BrowserExecutionControls";
import { DeviceSelector } from "./DeviceSelector";
import { ScreenshotConfig } from "./ScreenshotConfig";
import { NetworkInterceptConfig } from "./NetworkInterceptConfig";
import { AIExtractSection } from "./AIExtractSection";
import { ResearchAgenticSection } from "./ResearchAgenticSection";
import { JobFormAdvancedSection, JobFormIntro } from "./jobs/JobFormSections";
import { useToast } from "./toast";
import type {
  AiExtractOptions,
  DeviceEmulation,
  ResearchAgenticConfig,
} from "../api";

export interface ResearchFormRef {
  submit: () => Promise<void>;
  getQuery: () => string;
  setQuery: (query: string) => void;
  getConfig: () => PresetConfig;
}

interface ResearchFormProps {
  form: FormController;
  profiles: ProfileOption[];
  onSubmit: (request: import("../api").ResearchRequest) => Promise<void>;
  loading: boolean;
  query: string;
  setQuery: (value: string) => void;
  urls: string;
  setUrls: (value: string) => void;
  device: DeviceEmulation | null;
  setDevice: (value: DeviceEmulation | null) => void;
  surface?: "full" | "headless";
}

export const ResearchForm = forwardRef<ResearchFormRef, ResearchFormProps>(
  function ResearchForm(
    {
      form,
      profiles,
      onSubmit,
      loading,
      query,
      setQuery,
      urls,
      setUrls,
      device,
      setDevice,
      surface = "full",
    },
    ref,
  ) {
    const toast = useToast();

    const handleSubmit = useCallback(async () => {
      if (!query.trim() || !urls.trim()) {
        toast.show({
          tone: "warning",
          title: "Research query and URLs required",
          description:
            "Add both a research prompt and at least one source URL before submitting.",
        });
        return;
      }

      let shared: ReturnType<typeof buildSharedRequestConfig>;
      let aiExtractOptions: AiExtractOptions | undefined;
      let agenticOptions: ResearchAgenticConfig | undefined;
      try {
        shared = buildSharedRequestConfig(form);
        aiExtractOptions = buildAIExtractOptions(
          form.aiExtractEnabled,
          form.aiExtractMode,
          form.aiExtractPrompt,
          form.aiExtractSchema,
          form.aiExtractFields,
        );
        agenticOptions = buildResearchAgenticOptions(
          form.agenticResearchEnabled,
          form.agenticResearchInstructions,
          form.agenticResearchMaxRounds,
          form.agenticResearchMaxFollowUpUrls,
        );
      } catch (error) {
        toast.show({
          tone: "error",
          title: "Research configuration is invalid",
          description: error instanceof Error ? error.message : String(error),
        });
        return;
      }

      const request = buildResearchRequest(
        query.trim(),
        parseUrlList(urls),
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
        shared.webhook,
        shared.screenshot,
        device || undefined,
        shared.networkIntercept,
        aiExtractOptions,
        agenticOptions,
      );
      await onSubmit(request);
    }, [device, form, onSubmit, query, toast, urls]);

    const getConfig = useCallback((): PresetConfig => {
      const draftState: JobDraftLocalState = {
        scrape: {
          url: "",
          device: null,
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
          query,
          urls,
          device,
        },
      };

      return buildPresetConfig("research", form, draftState);
    }, [device, form, query, urls]);

    useImperativeHandle(
      ref,
      () => ({
        submit: handleSubmit,
        getQuery: () => query,
        setQuery,
        getConfig,
      }),
      [getConfig, handleSubmit, query, setQuery],
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
          title="Deep Research"
          description="Frame the question, point Spartan at a source set, and keep the synthesis path visible right from the first viewport."
          actions={
            <>
              <button type="submit" disabled={loading}>
                Run Research
              </button>
              <button
                type="button"
                className="secondary"
                onClick={() => {
                  setQuery("");
                  setUrls("");
                }}
              >
                Clear
              </button>
            </>
          }
        >
          <label htmlFor="research-query">Research query</label>
          <input
            id="research-query"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="pricing model, security posture, roadmap..."
          />
          <label htmlFor="research-urls" style={{ marginTop: 12 }}>
            Source URLs (one per line or comma-separated)
          </label>
          <textarea
            id="research-urls"
            rows={3}
            value={urls}
            onChange={(event) => setUrls(event.target.value)}
            placeholder="https://example.com\nhttps://example.com/docs"
          />
          <div className="row" style={{ marginTop: 12 }}>
            <label>
              Max depth
              <input
                type="number"
                min={0}
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
            inputPrefix="research"
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
            inputPrefix="research"
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
          description="Templates, processors, and optional webhook notifications."
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
            inputPrefix="research"
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
          <ResearchAgenticSection
            enabled={form.agenticResearchEnabled}
            setEnabled={form.setAgenticResearchEnabled}
            instructions={form.agenticResearchInstructions}
            setInstructions={form.setAgenticResearchInstructions}
            maxRounds={form.agenticResearchMaxRounds}
            setMaxRounds={form.setAgenticResearchMaxRounds}
            maxFollowUpUrls={form.agenticResearchMaxFollowUpUrls}
            setMaxFollowUpUrls={form.setAgenticResearchMaxFollowUpUrls}
          />
          <WebhookConfig
            webhookUrl={form.webhookUrl}
            setWebhookUrl={form.setWebhookUrl}
            webhookEvents={form.webhookEvents}
            setWebhookEvents={form.setWebhookEvents}
            webhookSecret={form.webhookSecret}
            setWebhookSecret={form.setWebhookSecret}
            inputPrefix="research"
          />
        </JobFormAdvancedSection>
      </form>
    );
  },
);

ResearchForm.displayName = "ResearchForm";
