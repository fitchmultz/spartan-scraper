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
  useMemo,
  useState,
  useCallback,
  forwardRef,
  useImperativeHandle,
} from "react";
import { AuthConfig } from "./AuthConfig";
import { PipelineOptions } from "./PipelineOptions";
import {
  parseHeaders,
  parseCookies,
  parseQueryParams,
  buildAuth,
  buildCrawlRequest,
} from "../lib/form-utils";
import type { PresetConfig } from "../types/presets";

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
  headless: boolean;
  setHeadless: (value: boolean) => void;
  usePlaywright: boolean;
  setUsePlaywright: (value: boolean) => void;
  timeoutSeconds: number;
  setTimeoutSeconds: (value: number) => void;
  authProfile: string;
  setAuthProfile: (value: string) => void;
  authBasic: string;
  setAuthBasic: (value: string) => void;
  headersRaw: string;
  setHeadersRaw: (value: string) => void;
  cookiesRaw: string;
  setCookiesRaw: (value: string) => void;
  queryRaw: string;
  setQueryRaw: (value: string) => void;
  loginUrl: string;
  setLoginUrl: (value: string) => void;
  loginUserSelector: string;
  setLoginUserSelector: (value: string) => void;
  loginPassSelector: string;
  setLoginPassSelector: (value: string) => void;
  loginSubmitSelector: string;
  setLoginSubmitSelector: (value: string) => void;
  loginUser: string;
  setLoginUser: (value: string) => void;
  loginPass: string;
  setLoginPass: (value: string) => void;
  extractTemplate: string;
  setExtractTemplate: (value: string) => void;
  extractValidate: boolean;
  setExtractValidate: (value: boolean) => void;
  preProcessors: string;
  setPreProcessors: (value: string) => void;
  postProcessors: string;
  setPostProcessors: (value: string) => void;
  transformers: string;
  setTransformers: (value: string) => void;
  incremental: boolean;
  setIncremental: (value: boolean) => void;
  profiles: Array<{ name: string; parents: string[] }>;
  onSubmit: (request: import("../api").CrawlRequest) => Promise<void>;
  loading: boolean;
}

export const CrawlForm = forwardRef<CrawlFormRef, CrawlFormProps>(
  function CrawlForm(
    {
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
      profiles,
      onSubmit,
      loading,
    }: CrawlFormProps,
    ref,
  ) {
    const [crawlUrl, setCrawlUrl] = useState("");
    const [maxDepth, setMaxDepth] = useState(2);
    const [maxPages, setMaxPages] = useState(200);

    const headerMap = useMemo(() => parseHeaders(headersRaw), [headersRaw]);
    const cookieList = useMemo(() => parseCookies(cookiesRaw), [cookiesRaw]);
    const queryMap = useMemo(() => parseQueryParams(queryRaw), [queryRaw]);

    const handleSubmit = useCallback(async () => {
      if (!crawlUrl) {
        alert("Crawl URL is required.");
        return;
      }
      const request = buildCrawlRequest(
        crawlUrl,
        maxDepth,
        maxPages,
        headless,
        usePlaywright,
        timeoutSeconds,
        authProfile || undefined,
        buildAuth(
          authBasic,
          headerMap,
          cookieList,
          queryMap,
          loginUrl,
          loginUserSelector,
          loginPassSelector,
          loginSubmitSelector,
          loginUser,
          loginPass,
        ),
        {
          template: extractTemplate || undefined,
          validate: extractValidate,
        },
        preProcessors,
        postProcessors,
        transformers,
        incremental,
      );
      await onSubmit(request);
    }, [
      crawlUrl,
      maxDepth,
      maxPages,
      headless,
      usePlaywright,
      timeoutSeconds,
      authProfile,
      authBasic,
      headerMap,
      cookieList,
      queryMap,
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
      ],
    );

    // Expose imperative handle for external submission
    useImperativeHandle(ref, () => ({
      submit: handleSubmit,
      getUrl: () => crawlUrl,
      setUrl: (url: string) => setCrawlUrl(url),
      getConfig,
    }));

    return (
      <div className="panel">
        <h2>Crawl a Site</h2>
        <label htmlFor="crawl-url">Root URL</label>
        <input
          id="crawl-url"
          value={crawlUrl}
          onChange={(event) => setCrawlUrl(event.target.value)}
          placeholder="https://example.com"
        />
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
        <div style={{ marginTop: 16, display: "flex", gap: 12 }}>
          <button type="button" disabled={loading} onClick={handleSubmit}>
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
      </div>
    );
  },
);

CrawlForm.displayName = "CrawlForm";
