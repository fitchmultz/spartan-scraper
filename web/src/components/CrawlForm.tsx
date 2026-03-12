/**
 * Crawl Form Component
 *
 * Form for submitting website crawl jobs. Handles root URL, max depth, max pages,
 * headless/playwright options, authentication, and extraction template configuration.
 * Builds CrawlRequest objects using shared utilities and submits them via callback.
 *
 * @module CrawlForm
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
import {
  buildCrawlRequest,
  buildSharedRequestConfig,
  parsePatternList,
} from "../lib/form-utils";
import type { FormController, ProfileOption } from "../hooks/useFormState";
import type { PresetConfig } from "../types/presets";
import { WebhookConfig } from "./WebhookConfig";
import { DeviceSelector } from "./DeviceSelector";
import { NetworkInterceptConfig } from "./NetworkInterceptConfig";
import type { DeviceEmulation } from "../api";

export interface CrawlFormRef {
  /** Submit the form programmatically */
  submit: () => Promise<void>;
  /** Get the current URL value */
  getUrl: () => string;
  /** Set the URL value */
  setUrl: (url: string) => void;
  /** Get the current configuration as a preset */
  getConfig: () => PresetConfig;
}

interface CrawlFormProps {
  form: FormController;
  profiles: ProfileOption[];
  onSubmit: (request: import("../api").CrawlRequest) => Promise<void>;
  loading: boolean;
}

export const CrawlForm = forwardRef<CrawlFormRef, CrawlFormProps>(
  function CrawlForm(
    { form, profiles, onSubmit, loading }: CrawlFormProps,
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

    const [crawlUrl, setCrawlUrl] = useState("");
    const [maxDepth, setMaxDepth] = useState(2);
    const [maxPages, setMaxPages] = useState(200);
    const [sitemapURL, setSitemapURL] = useState("");
    const [sitemapOnly, setSitemapOnly] = useState(false);
    const [includePatterns, setIncludePatterns] = useState("");
    const [excludePatterns, setExcludePatterns] = useState("");
    const [device, setDevice] = useState<DeviceEmulation | null>(null);

    const handleSubmit = useCallback(async () => {
      if (!crawlUrl) {
        alert("Crawl URL is required.");
        return;
      }
      const shared = buildSharedRequestConfig(form);
      const request = buildCrawlRequest(
        crawlUrl,
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
        incremental,
        sitemapURL,
        sitemapOnly,
        shared.webhook,
        parsePatternList(includePatterns),
        parsePatternList(excludePatterns),
        device || undefined,
        shared.networkIntercept,
      );
      await onSubmit(request);
    }, [
      crawlUrl,
      maxDepth,
      maxPages,
      headless,
      usePlaywright,
      timeoutSeconds,
      form,
      incremental,
      sitemapURL,
      sitemapOnly,
      includePatterns,
      excludePatterns,
      device,
      onSubmit,
    ]);

    // Build config from current form state
    const getConfig = useCallback(
      (): PresetConfig => ({
        url: crawlUrl,
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
        maxDepth,
        maxPages,
        sitemapURL,
        sitemapOnly,
        webhookUrl,
        webhookEvents,
        webhookSecret,
        includePatterns,
        excludePatterns,
        device: device || undefined,
        interceptEnabled,
        interceptURLPatterns,
        interceptResourceTypes,
        interceptCaptureRequestBody,
        interceptCaptureResponseBody,
        interceptMaxBodySize,
      }),
      [
        crawlUrl,
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
        maxDepth,
        maxPages,
        sitemapURL,
        sitemapOnly,
        webhookUrl,
        webhookEvents,
        webhookSecret,
        includePatterns,
        excludePatterns,
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
      getUrl: () => crawlUrl,
      setUrl: (url: string) => setCrawlUrl(url),
      getConfig,
    }));

    const handleFormSubmit = (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      void handleSubmit();
    };

    return (
      <form className="panel" onSubmit={handleFormSubmit}>
        <h2>Crawl a Site</h2>
        <label htmlFor="crawl-url">Root URL</label>
        <input
          id="crawl-url"
          value={crawlUrl}
          onChange={(event) => setCrawlUrl(event.target.value)}
          placeholder="https://example.com"
        />
        <label htmlFor="sitemap-url" style={{ marginTop: 12 }}>
          Sitemap URL (optional)
        </label>
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
        <div className="row" style={{ marginTop: 12 }}>
          <label>
            Max depth
            <input
              type="number"
              min={1}
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
          inputPrefix="crawl"
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
          inputPrefix="crawl"
        />
        <WebhookConfig
          webhookUrl={webhookUrl}
          setWebhookUrl={setWebhookUrl}
          webhookEvents={webhookEvents}
          setWebhookEvents={setWebhookEvents}
          webhookSecret={webhookSecret}
          setWebhookSecret={setWebhookSecret}
          inputPrefix="crawl"
        />
        <div style={{ marginTop: 16, display: "flex", gap: 12 }}>
          <button type="submit" disabled={loading}>
            Launch Crawl
          </button>
          <button
            type="button"
            className="secondary"
            onClick={() => setCrawlUrl("")}
          >
            Clear
          </button>
        </div>
      </form>
    );
  },
);

CrawlForm.displayName = "CrawlForm";
