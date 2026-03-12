/**
 * Scrape Form Component
 *
 * Form for submitting single-page scrape jobs. Handles URL input, headless/playwright
 * options, timeout configuration, authentication settings, and extraction template selection.
 * Builds ScrapeRequest objects using shared utilities and submits them via callback.
 *
 * @module ScrapeForm
 */
import {
  useState,
  useCallback,
  forwardRef,
  useImperativeHandle,
  type FormEvent,
} from "react";
import { AuthConfig } from "./AuthConfig";
import { PipelineOptions } from "./PipelineOptions";
import { AIExtractSection } from "./AIExtractSection";
import {
  buildScrapeRequest,
  buildSharedRequestConfig,
} from "../lib/form-utils";
import type { FormController, ProfileOption } from "../hooks/useFormState";
import type { PresetConfig } from "../types/presets";
import { WebhookConfig } from "./WebhookConfig";
import { BrowserExecutionControls } from "./BrowserExecutionControls";
import { DeviceSelector } from "./DeviceSelector";
import { NetworkInterceptConfig } from "./NetworkInterceptConfig";
import { JobFormAdvancedSection, JobFormIntro } from "./jobs/JobFormSections";
import type { DeviceEmulation } from "../api";

export interface ScrapeFormRef {
  /** Submit the form programmatically */
  submit: () => Promise<void>;
  /** Get the current URL value */
  getUrl: () => string;
  /** Set the URL value */
  setUrl: (url: string) => void;
  /** Get the current configuration as a preset */
  getConfig: () => PresetConfig;
}

interface ScrapeFormProps {
  form: FormController;
  profiles: ProfileOption[];
  onSubmit: (request: import("../api").ScrapeRequest) => Promise<void>;
  loading: boolean;
}

export const ScrapeForm = forwardRef<ScrapeFormRef, ScrapeFormProps>(
  function ScrapeForm(
    { form, profiles, onSubmit, loading }: ScrapeFormProps,
    ref,
  ) {
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
    } = form;

    const [scrapeUrl, setScrapeUrl] = useState("");
    const [device, setDevice] = useState<DeviceEmulation | null>(null);

    // AI extraction state
    const [aiEnabled, setAiEnabled] = useState(false);
    const [aiMode, setAiMode] = useState<"natural_language" | "schema_guided">(
      "natural_language",
    );
    const [aiPrompt, setAiPrompt] = useState("");
    const [aiFields, setAiFields] = useState("");

    const handleSubmit = useCallback(async () => {
      if (!scrapeUrl) {
        alert("Scrape URL is required.");
        return;
      }
      const shared = buildSharedRequestConfig(form);
      // Build AI extraction options if enabled
      const aiExtractOptions = aiEnabled
        ? {
            enabled: true,
            mode: aiMode,
            prompt: aiPrompt || undefined,
            fields: aiFields
              .split(",")
              .map((f) => f.trim())
              .filter(Boolean),
          }
        : undefined;

      const request = buildScrapeRequest(
        scrapeUrl,
        headless,
        usePlaywright,
        timeoutSeconds,
        shared.authProfile,
        shared.auth,
        shared.extract,
        shared.preProcessors,
        shared.postProcessors,
        shared.transformers,
        incremental,
        shared.webhook,
        device || undefined,
        shared.networkIntercept,
        aiExtractOptions,
      );
      await onSubmit(request);
    }, [
      scrapeUrl,
      headless,
      usePlaywright,
      timeoutSeconds,
      form,
      aiEnabled,
      aiMode,
      aiPrompt,
      aiFields,
      incremental,
      device,
      onSubmit,
    ]);

    // Build config from current form state
    const getConfig = useCallback(
      (): PresetConfig => ({
        url: scrapeUrl,
        headless,
        usePlaywright,
        timeoutSeconds,
        authProfile,
        authBasic,
        headersRaw,
        cookiesRaw,
        queryRaw,
        loginUrl,
        loginUserSelector,
        loginPassSelector,
        loginSubmitSelector,
        loginUser,
        loginPass,
        extractTemplate,
        extractValidate,
        preProcessors,
        postProcessors,
        transformers,
        incremental,
        webhookUrl,
        webhookEvents,
        webhookSecret,
        device: device || undefined,
        interceptEnabled,
        interceptURLPatterns,
        interceptResourceTypes,
        interceptCaptureRequestBody,
        interceptCaptureResponseBody,
        interceptMaxBodySize,
      }),
      [
        scrapeUrl,
        headless,
        usePlaywright,
        timeoutSeconds,
        authProfile,
        authBasic,
        headersRaw,
        cookiesRaw,
        queryRaw,
        loginUrl,
        loginUserSelector,
        loginPassSelector,
        loginSubmitSelector,
        loginUser,
        loginPass,
        extractTemplate,
        extractValidate,
        preProcessors,
        postProcessors,
        transformers,
        incremental,
        webhookUrl,
        webhookEvents,
        webhookSecret,
        device,
        interceptEnabled,
        interceptURLPatterns,
        interceptResourceTypes,
        interceptCaptureRequestBody,
        interceptCaptureResponseBody,
        interceptMaxBodySize,
      ],
    );

    // Expose imperative handle for external submission
    useImperativeHandle(ref, () => ({
      submit: handleSubmit,
      getUrl: () => scrapeUrl,
      setUrl: (url: string) => setScrapeUrl(url),
      getConfig,
    }));

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
                onClick={() => setScrapeUrl("")}
              >
                Clear
              </button>
            </>
          }
        >
          <label htmlFor="scrape-url">Target URL</label>
          <input
            id="scrape-url"
            value={scrapeUrl}
            onChange={(event) => setScrapeUrl(event.target.value)}
            placeholder="https://example.com"
          />
          <BrowserExecutionControls
            headless={headless}
            setHeadless={setHeadless}
            usePlaywright={usePlaywright}
            setUsePlaywright={setUsePlaywright}
            timeoutSeconds={timeoutSeconds}
            setTimeoutSeconds={setTimeoutSeconds}
          />
        </JobFormIntro>

        <JobFormAdvancedSection
          title="Browser and capture controls"
          description="Device emulation, network interception, and browser-only diagnostics."
        >
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
            disabled={!headless}
            inputPrefix="scrape"
          />
        </JobFormAdvancedSection>

        <JobFormAdvancedSection
          title="Authentication and request shaping"
          description="Profiles, cookies, login automation, and request overrides."
        >
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
        </JobFormAdvancedSection>

        <JobFormAdvancedSection
          title="Extraction and processing"
          description="Templates, validation, transformers, and AI extraction helpers."
        >
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
            inputPrefix="scrape"
          />
          <AIExtractSection
            enabled={aiEnabled}
            setEnabled={setAiEnabled}
            mode={aiMode}
            setMode={setAiMode}
            prompt={aiPrompt}
            setPrompt={setAiPrompt}
            fields={aiFields}
            setFields={setAiFields}
          />
        </JobFormAdvancedSection>

        <JobFormAdvancedSection
          title="Delivery hooks"
          description="Optional webhook notifications for external systems."
        >
          <WebhookConfig
            webhookUrl={webhookUrl}
            setWebhookUrl={setWebhookUrl}
            webhookEvents={webhookEvents}
            setWebhookEvents={setWebhookEvents}
            webhookSecret={webhookSecret}
            setWebhookSecret={setWebhookSecret}
            inputPrefix="scrape"
          />
        </JobFormAdvancedSection>
      </form>
    );
  },
);

ScrapeForm.displayName = "ScrapeForm";
