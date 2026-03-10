/**
 * Research Form Component
 *
 * Form for submitting research jobs. Handles research query, source URLs,
 * crawl parameters (max depth/pages), headless/playwright options, authentication,
 * and extraction template configuration. Builds ResearchRequest objects using shared
 * utilities and submits them via callback.
 *
 * @module ResearchForm
 */
import { useState, useCallback, forwardRef, useImperativeHandle } from "react";
import { AuthConfig } from "./AuthConfig";
import { PipelineOptions } from "./PipelineOptions";
import {
  buildResearchRequest,
  buildSharedRequestConfig,
  parseUrlList,
} from "../lib/form-utils";
import type { FormController, ProfileOption } from "../hooks/useFormState";
import type { PresetConfig } from "../types/presets";
import { WebhookConfig } from "./WebhookConfig";
import { DeviceSelector } from "./DeviceSelector";
import { NetworkInterceptConfig } from "./NetworkInterceptConfig";
import type { DeviceEmulation } from "../api";

export interface ResearchFormRef {
  /** Submit the form programmatically */
  submit: () => Promise<void>;
  /** Get the current query value */
  getQuery: () => string;
  /** Set the query value */
  setQuery: (query: string) => void;
  /** Get the current configuration as a preset */
  getConfig: () => PresetConfig;
}

interface ResearchFormProps {
  form: FormController;
  profiles: ProfileOption[];
  onSubmit: (request: import("../api").ResearchRequest) => Promise<void>;
  loading: boolean;
}

export const ResearchForm = forwardRef<ResearchFormRef, ResearchFormProps>(
  function ResearchForm(
    { form, profiles, onSubmit, loading }: ResearchFormProps,
    ref,
  ) {
    const {
      maxDepth,
      setMaxDepth,
      maxPages,
      setMaxPages,
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

    const [researchQuery, setResearchQuery] = useState("");
    const [researchUrls, setResearchUrls] = useState("");
    const [device, setDevice] = useState<DeviceEmulation | null>(null);

    const handleSubmit = useCallback(async () => {
      if (!researchQuery || !researchUrls) {
        alert("Research query and URLs are required.");
        return;
      }
      const shared = buildSharedRequestConfig(form);
      const request = buildResearchRequest(
        researchQuery,
        parseUrlList(researchUrls),
        maxDepth,
        maxPages,
        headless,
        usePlaywright,
        timeoutSeconds,
        shared.authProfile,
        shared.auth,
        shared.extract,
        shared.preProcessors,
        shared.postProcessors,
        shared.transformers,
        shared.webhook,
        device || undefined,
        shared.networkIntercept,
      );
      await onSubmit(request);
    }, [
      researchQuery,
      researchUrls,
      maxDepth,
      maxPages,
      headless,
      usePlaywright,
      timeoutSeconds,
      form,
      device,
      onSubmit,
    ]);

    // Build config from current form state
    const getConfig = useCallback(
      (): PresetConfig => ({
        query: researchQuery,
        urls: researchUrls,
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
        maxDepth,
        maxPages,
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
        researchQuery,
        researchUrls,
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
        maxDepth,
        maxPages,
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
      getQuery: () => researchQuery,
      setQuery: (query: string) => setResearchQuery(query),
      getConfig,
    }));

    return (
      <div className="panel">
        <h2>Deep Research</h2>
        <label htmlFor="research-query">Research query</label>
        <input
          id="research-query"
          value={researchQuery}
          onChange={(event) => setResearchQuery(event.target.value)}
          placeholder="pricing model, security posture, roadmap..."
        />
        <label htmlFor="research-urls" style={{ marginTop: 12 }}>
          Source URLs (comma-separated)
        </label>
        <textarea
          id="research-urls"
          rows={3}
          value={researchUrls}
          onChange={(event) => setResearchUrls(event.target.value)}
          placeholder="https://example.com, https://example.com/docs"
        />
        <div className="row" style={{ marginTop: 12 }}>
          <label>
            Max depth
            <input
              type="number"
              min={0}
              value={maxDepth}
              onChange={(event) => setMaxDepth(Number(event.target.value))}
            />
          </label>
          <label>
            Max pages
            <input
              type="number"
              min={1}
              value={maxPages}
              onChange={(event) => setMaxPages(Number(event.target.value))}
            />
          </label>
        </div>
        <div className="row" style={{ marginTop: 12 }}>
          <label>
            <input
              type="checkbox"
              checked={headless}
              onChange={(event) => setHeadless(event.target.checked)}
            />{" "}
            Headless
          </label>
          <label>
            <input
              type="checkbox"
              checked={usePlaywright}
              disabled={!headless}
              onChange={(event) => setUsePlaywright(event.target.checked)}
            />{" "}
            Playwright
          </label>
          <label>
            Timeout (s)
            <input
              type="number"
              min={5}
              value={timeoutSeconds}
              onChange={(event) =>
                setTimeoutSeconds(Number(event.target.value))
              }
            />
          </label>
        </div>
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
          inputPrefix="research"
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
          inputPrefix="research"
        />
        <WebhookConfig
          webhookUrl={webhookUrl}
          setWebhookUrl={setWebhookUrl}
          webhookEvents={webhookEvents}
          setWebhookEvents={setWebhookEvents}
          webhookSecret={webhookSecret}
          setWebhookSecret={setWebhookSecret}
          inputPrefix="research"
        />
        <div style={{ marginTop: 16, display: "flex", gap: 12 }}>
          <button type="button" disabled={loading} onClick={handleSubmit}>
            Run Research
          </button>
          <button
            type="button"
            className="secondary"
            onClick={() => {
              setResearchQuery("");
              setResearchUrls("");
            }}
          >
            Clear
          </button>
        </div>
      </div>
    );
  },
);

ResearchForm.displayName = "ResearchForm";
