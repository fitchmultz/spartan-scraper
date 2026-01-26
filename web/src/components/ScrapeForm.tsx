/**
 * Scrape Form Component
 *
 * Form for submitting single-page scrape jobs. Handles URL input, headless/playwright
 * options, timeout configuration, authentication settings, and extraction template selection.
 * Builds ScrapeRequest objects using shared utilities and submits them via callback.
 *
 * @module ScrapeForm
 */
import { useMemo, useState } from "react";
import { AuthConfig } from "./AuthConfig";
import { PipelineOptions } from "./PipelineOptions";
import {
  parseHeaders,
  parseCookies,
  parseQueryParams,
  buildAuth,
  buildScrapeRequest,
} from "../lib/form-utils";

interface ScrapeFormProps {
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
  onSubmit: (request: import("../api").ScrapeRequest) => Promise<void>;
  loading: boolean;
}

export function ScrapeForm({
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
}: ScrapeFormProps) {
  const [scrapeUrl, setScrapeUrl] = useState("");

  const headerMap = useMemo(() => parseHeaders(headersRaw), [headersRaw]);
  const cookieList = useMemo(() => parseCookies(cookiesRaw), [cookiesRaw]);
  const queryMap = useMemo(() => parseQueryParams(queryRaw), [queryRaw]);

  const handleSubmit = async () => {
    if (!scrapeUrl) {
      alert("Scrape URL is required.");
      return;
    }
    const request = buildScrapeRequest(
      scrapeUrl,
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
  };

  return (
    <div className="panel">
      <h2>Scrape a Page</h2>
      <label htmlFor="scrape-url">Target URL</label>
      <input
        id="scrape-url"
        value={scrapeUrl}
        onChange={(event) => setScrapeUrl(event.target.value)}
        placeholder="https://example.com"
      />
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
            onChange={(event) => setTimeoutSeconds(Number(event.target.value))}
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
        inputPrefix="scrape"
      />
      <div style={{ marginTop: 16, display: "flex", gap: 12 }}>
        <button type="button" disabled={loading} onClick={handleSubmit}>
          Deploy Scrape
        </button>
        <button
          type="button"
          className="secondary"
          onClick={() => setScrapeUrl("")}
        >
          Clear
        </button>
      </div>
    </div>
  );
}
