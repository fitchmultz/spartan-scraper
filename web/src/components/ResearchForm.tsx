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
  buildResearchRequest,
  parseUrlList,
} from "../lib/form-utils";

export interface ResearchFormRef {
  /** Submit the form programmatically */
  submit: () => Promise<void>;
  /** Get the current query value */
  getQuery: () => string;
  /** Set the query value */
  setQuery: (query: string) => void;
}

interface ResearchFormProps {
  maxDepth: number;
  setMaxDepth: (value: number) => void;
  maxPages: number;
  setMaxPages: (value: number) => void;
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
  profiles: Array<{ name: string; parents: string[] }>;
  onSubmit: (request: import("../api").ResearchRequest) => Promise<void>;
  loading: boolean;
}

export const ResearchForm = forwardRef<ResearchFormRef, ResearchFormProps>(
  function ResearchForm(
    {
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
      profiles,
      onSubmit,
      loading,
    }: ResearchFormProps,
    ref,
  ) {
    const [researchQuery, setResearchQuery] = useState("");
    const [researchUrls, setResearchUrls] = useState("");

    const headerMap = useMemo(() => parseHeaders(headersRaw), [headersRaw]);
    const cookieList = useMemo(() => parseCookies(cookiesRaw), [cookiesRaw]);
    const queryMap = useMemo(() => parseQueryParams(queryRaw), [queryRaw]);

    const handleSubmit = useCallback(async () => {
      if (!researchQuery || !researchUrls) {
        alert("Research query and URLs are required.");
        return;
      }
      const request = buildResearchRequest(
        researchQuery,
        parseUrlList(researchUrls),
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
      onSubmit,
    ]);

    // Expose imperative handle for external submission
    useImperativeHandle(ref, () => ({
      submit: handleSubmit,
      getQuery: () => researchQuery,
      setQuery: (query: string) => setResearchQuery(query),
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
