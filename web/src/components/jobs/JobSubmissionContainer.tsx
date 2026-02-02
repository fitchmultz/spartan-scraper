/**
 * JobSubmissionContainer - Container component for job submission functionality
 *
 * This component encapsulates all job submission-related state and operations:
 * - Managing active tab state for job forms
 * - Handling individual job submissions (scrape, crawl, research)
 * - Managing form refs for programmatic submission
 * - Rendering ScrapeForm, CrawlForm, ResearchForm with Suspense
 *
 * It does NOT handle:
 * - Batch operations
 * - Watch or chain management
 * - Results viewing
 *
 * @module JobSubmissionContainer
 */

import { useRef, Suspense, lazy, forwardRef, useImperativeHandle } from "react";
import {
  type ScrapeRequest,
  type CrawlRequest,
  type ResearchRequest,
} from "../../api";
import type { Profile } from "../../hooks/useAppData";
import type { FormState, FormActions } from "../../hooks/useFormState";

const ScrapeForm = lazy(() =>
  import("../../components/ScrapeForm").then((mod) => ({
    default: mod.ScrapeForm,
  })),
);

const CrawlForm = lazy(() =>
  import("../../components/CrawlForm").then((mod) => ({
    default: mod.CrawlForm,
  })),
);

const ResearchForm = lazy(() =>
  import("../../components/ResearchForm").then((mod) => ({
    default: mod.ResearchForm,
  })),
);

type ScrapeFormRef = import("../../components/ScrapeForm").ScrapeFormRef;
type CrawlFormRef = import("../../components/CrawlForm").CrawlFormRef;
type ResearchFormRef = import("../../components/ResearchForm").ResearchFormRef;

export interface JobSubmissionContainerRef {
  submitScrape: () => Promise<void>;
  submitCrawl: () => Promise<void>;
  submitResearch: () => Promise<void>;
  setScrapeUrl: (url: string) => void;
  setCrawlUrl: (url: string) => void;
  setResearchQuery: (query: string) => void;
  getScrapeUrl: () => string;
  getCrawlUrl: () => string;
}

interface JobSubmissionContainerProps {
  // Form state (from useFormState)
  formState: FormState & FormActions;
  // Submission handlers
  onSubmitScrape: (request: ScrapeRequest) => void;
  onSubmitCrawl: (request: CrawlRequest) => void;
  onSubmitResearch: (request: ResearchRequest) => void;

  // Loading state
  loading: boolean;
  // Profiles for auth
  profiles: Profile[];
}

export const JobSubmissionContainer = forwardRef<
  JobSubmissionContainerRef,
  JobSubmissionContainerProps
>(function JobSubmissionContainer(
  {
    formState,
    onSubmitScrape,
    onSubmitCrawl,
    onSubmitResearch,
    loading,
    profiles,
  },
  ref,
) {
  const scrapeFormRef = useRef<ScrapeFormRef>(null);
  const crawlFormRef = useRef<CrawlFormRef>(null);
  const researchFormRef = useRef<ResearchFormRef>(null);

  // Expose form methods to parent via imperative handle
  useImperativeHandle(ref, () => ({
    submitScrape: async () => {
      await scrapeFormRef.current?.submit();
    },
    submitCrawl: async () => {
      await crawlFormRef.current?.submit();
    },
    submitResearch: async () => {
      await researchFormRef.current?.submit();
    },
    setScrapeUrl: (url: string) => {
      scrapeFormRef.current?.setUrl(url);
    },
    setCrawlUrl: (url: string) => {
      crawlFormRef.current?.setUrl(url);
    },
    setResearchQuery: (query: string) => {
      researchFormRef.current?.setQuery(query);
    },
    getScrapeUrl: () => {
      return scrapeFormRef.current?.getUrl() ?? "";
    },
    getCrawlUrl: () => {
      return crawlFormRef.current?.getUrl() ?? "";
    },
  }));

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
    maxDepth,
    setMaxDepth,
    maxPages,
    setMaxPages,
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
  } = formState;

  return (
    <section id="forms" className="grid" data-tour="form-types">
      <Suspense
        fallback={<div className="loading-placeholder">Loading forms...</div>}
      >
        <ScrapeForm
          ref={scrapeFormRef}
          headless={headless}
          setHeadless={setHeadless}
          usePlaywright={usePlaywright}
          setUsePlaywright={setUsePlaywright}
          timeoutSeconds={timeoutSeconds}
          setTimeoutSeconds={setTimeoutSeconds}
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
          webhookUrl={webhookUrl}
          setWebhookUrl={setWebhookUrl}
          webhookEvents={webhookEvents}
          setWebhookEvents={setWebhookEvents}
          webhookSecret={webhookSecret}
          setWebhookSecret={setWebhookSecret}
          profiles={profiles}
          onSubmit={async (req) => {
            await onSubmitScrape(req);
          }}
          loading={loading}
          interceptEnabled={interceptEnabled}
          setInterceptEnabled={setInterceptEnabled}
          interceptURLPatterns={interceptURLPatterns}
          setInterceptURLPatterns={setInterceptURLPatterns}
          interceptResourceTypes={interceptResourceTypes}
          setInterceptResourceTypes={setInterceptResourceTypes}
          interceptCaptureRequestBody={interceptCaptureRequestBody}
          setInterceptCaptureRequestBody={setInterceptCaptureRequestBody}
          interceptCaptureResponseBody={interceptCaptureResponseBody}
          setInterceptCaptureResponseBody={setInterceptCaptureResponseBody}
          interceptMaxBodySize={interceptMaxBodySize}
          setInterceptMaxBodySize={setInterceptMaxBodySize}
        />

        <CrawlForm
          ref={crawlFormRef}
          headless={headless}
          setHeadless={setHeadless}
          usePlaywright={usePlaywright}
          setUsePlaywright={setUsePlaywright}
          timeoutSeconds={timeoutSeconds}
          setTimeoutSeconds={setTimeoutSeconds}
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
          webhookUrl={webhookUrl}
          setWebhookUrl={setWebhookUrl}
          webhookEvents={webhookEvents}
          setWebhookEvents={setWebhookEvents}
          webhookSecret={webhookSecret}
          setWebhookSecret={setWebhookSecret}
          profiles={profiles}
          onSubmit={async (req) => {
            await onSubmitCrawl(req);
          }}
          loading={loading}
          interceptEnabled={interceptEnabled}
          setInterceptEnabled={setInterceptEnabled}
          interceptURLPatterns={interceptURLPatterns}
          setInterceptURLPatterns={setInterceptURLPatterns}
          interceptResourceTypes={interceptResourceTypes}
          setInterceptResourceTypes={setInterceptResourceTypes}
          interceptCaptureRequestBody={interceptCaptureRequestBody}
          setInterceptCaptureRequestBody={setInterceptCaptureRequestBody}
          interceptCaptureResponseBody={interceptCaptureResponseBody}
          setInterceptCaptureResponseBody={setInterceptCaptureResponseBody}
          interceptMaxBodySize={interceptMaxBodySize}
          setInterceptMaxBodySize={setInterceptMaxBodySize}
        />

        <ResearchForm
          ref={researchFormRef}
          maxDepth={maxDepth}
          setMaxDepth={setMaxDepth}
          maxPages={maxPages}
          setMaxPages={setMaxPages}
          headless={headless}
          setHeadless={setHeadless}
          usePlaywright={usePlaywright}
          setUsePlaywright={setUsePlaywright}
          timeoutSeconds={timeoutSeconds}
          setTimeoutSeconds={setTimeoutSeconds}
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
          webhookUrl={webhookUrl}
          setWebhookUrl={setWebhookUrl}
          webhookEvents={webhookEvents}
          setWebhookEvents={setWebhookEvents}
          webhookSecret={webhookSecret}
          setWebhookSecret={setWebhookSecret}
          profiles={profiles}
          onSubmit={async (req) => {
            await onSubmitResearch(req);
          }}
          loading={loading}
          interceptEnabled={interceptEnabled}
          setInterceptEnabled={setInterceptEnabled}
          interceptURLPatterns={interceptURLPatterns}
          setInterceptURLPatterns={setInterceptURLPatterns}
          interceptResourceTypes={interceptResourceTypes}
          setInterceptResourceTypes={setInterceptResourceTypes}
          interceptCaptureRequestBody={interceptCaptureRequestBody}
          setInterceptCaptureRequestBody={setInterceptCaptureRequestBody}
          interceptCaptureResponseBody={interceptCaptureResponseBody}
          setInterceptCaptureResponseBody={setInterceptCaptureResponseBody}
          interceptMaxBodySize={interceptMaxBodySize}
          setInterceptMaxBodySize={setInterceptMaxBodySize}
        />
      </Suspense>
    </section>
  );
});
