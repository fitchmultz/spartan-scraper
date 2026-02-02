/**
 * BatchContainer - Container component for batch job management
 *
 * This component encapsulates all batch-related state and operations:
 * - Managing batch form state (tab, URLs, query)
 * - Handling batch submissions (scrape, crawl, research)
 * - Displaying batch list with status
 * - Canceling batches
 *
 * It does NOT handle:
 * - Individual job submission
 * - Watch or chain management
 * - Results viewing
 *
 * @module BatchContainer
 */

import { useState, useCallback } from "react";
import type {
  BatchScrapeRequest,
  BatchCrawlRequest,
  BatchResearchRequest,
} from "../../api";
import type { Profile } from "../../hooks/useAppData";
import { useBatches } from "../../hooks/useBatches";
import { BatchList } from "../../components/BatchList";
import { BatchForm } from "../../components/BatchForm";

interface BatchContainerProps {
  // Form state props (shared with job forms)
  headless: boolean;
  setHeadless: (v: boolean) => void;
  usePlaywright: boolean;
  setUsePlaywright: (v: boolean) => void;
  timeoutSeconds: number;
  setTimeoutSeconds: (v: number) => void;
  authProfile: string;
  setAuthProfile: (v: string) => void;
  authBasic: string;
  setAuthBasic: (v: string) => void;
  headersRaw: string;
  setHeadersRaw: (v: string) => void;
  cookiesRaw: string;
  setCookiesRaw: (v: string) => void;
  queryRaw: string;
  setQueryRaw: (v: string) => void;
  loginUrl: string;
  setLoginUrl: (v: string) => void;
  loginUserSelector: string;
  setLoginUserSelector: (v: string) => void;
  loginPassSelector: string;
  setLoginPassSelector: (v: string) => void;
  loginSubmitSelector: string;
  setLoginSubmitSelector: (v: string) => void;
  loginUser: string;
  setLoginUser: (v: string) => void;
  loginPass: string;
  setLoginPass: (v: string) => void;
  extractTemplate: string;
  setExtractTemplate: (v: string) => void;
  extractValidate: boolean;
  setExtractValidate: (v: boolean) => void;
  preProcessors: string;
  setPreProcessors: (v: string) => void;
  postProcessors: string;
  setPostProcessors: (v: string) => void;
  transformers: string;
  setTransformers: (v: string) => void;
  incremental: boolean;
  setIncremental: (v: boolean) => void;
  maxDepth: number;
  setMaxDepth: (v: number) => void;
  maxPages: number;
  setMaxPages: (v: number) => void;
  webhookUrl: string;
  setWebhookUrl: (v: string) => void;
  webhookEvents: string[];
  setWebhookEvents: (v: string[]) => void;
  webhookSecret: string;
  setWebhookSecret: (v: string) => void;
  profiles: Profile[];
  loading: boolean;
}

export function BatchContainer(props: BatchContainerProps) {
  const [batchTab, setBatchTab] = useState<"scrape" | "crawl" | "research">(
    "scrape",
  );
  const [batchUrls, setBatchUrls] = useState("");
  const [batchQuery, setBatchQuery] = useState("");

  const {
    batches,
    batchJobs,
    loading: batchesLoading,
    refreshBatches,
    cancelBatch,
    submitBatchScrape,
    submitBatchCrawl,
    submitBatchResearch,
  } = useBatches();

  const handleSubmitBatchScrape = useCallback(
    async (request: BatchScrapeRequest) => {
      try {
        await submitBatchScrape(request);
        setBatchUrls("");
      } catch (err) {
        console.error("Failed to submit batch scrape:", err);
        alert(`Failed to submit batch: ${String(err)}`);
      }
    },
    [submitBatchScrape],
  );

  const handleSubmitBatchCrawl = useCallback(
    async (request: BatchCrawlRequest) => {
      try {
        await submitBatchCrawl(request);
        setBatchUrls("");
      } catch (err) {
        console.error("Failed to submit batch crawl:", err);
        alert(`Failed to submit batch: ${String(err)}`);
      }
    },
    [submitBatchCrawl],
  );

  const handleSubmitBatchResearch = useCallback(
    async (request: BatchResearchRequest) => {
      try {
        await submitBatchResearch(request);
        setBatchUrls("");
        setBatchQuery("");
      } catch (err) {
        console.error("Failed to submit batch research:", err);
        alert(`Failed to submit batch: ${String(err)}`);
      }
    },
    [submitBatchResearch],
  );

  const handleCancelBatch = useCallback(
    async (batchId: string) => {
      try {
        await cancelBatch(batchId);
      } catch (err) {
        console.error("Failed to cancel batch:", err);
        alert(`Failed to cancel batch: ${String(err)}`);
      }
    },
    [cancelBatch],
  );

  return (
    <>
      <section id="batch-forms" className="grid" data-tour="batch-forms">
        <BatchForm
          activeTab={batchTab}
          setActiveTab={setBatchTab}
          headless={props.headless}
          setHeadless={props.setHeadless}
          usePlaywright={props.usePlaywright}
          setUsePlaywright={props.setUsePlaywright}
          timeoutSeconds={props.timeoutSeconds}
          setTimeoutSeconds={props.setTimeoutSeconds}
          authProfile={props.authProfile}
          setAuthProfile={props.setAuthProfile}
          authBasic={props.authBasic}
          setAuthBasic={props.setAuthBasic}
          headersRaw={props.headersRaw}
          setHeadersRaw={props.setHeadersRaw}
          cookiesRaw={props.cookiesRaw}
          setCookiesRaw={props.setCookiesRaw}
          queryRaw={props.queryRaw}
          setQueryRaw={props.setQueryRaw}
          loginUrl={props.loginUrl}
          setLoginUrl={props.setLoginUrl}
          loginUserSelector={props.loginUserSelector}
          setLoginUserSelector={props.setLoginUserSelector}
          loginPassSelector={props.loginPassSelector}
          setLoginPassSelector={props.setLoginPassSelector}
          loginSubmitSelector={props.loginSubmitSelector}
          setLoginSubmitSelector={props.setLoginSubmitSelector}
          loginUser={props.loginUser}
          setLoginUser={props.setLoginUser}
          loginPass={props.loginPass}
          setLoginPass={props.setLoginPass}
          extractTemplate={props.extractTemplate}
          setExtractTemplate={props.setExtractTemplate}
          extractValidate={props.extractValidate}
          setExtractValidate={props.setExtractValidate}
          preProcessors={props.preProcessors}
          setPreProcessors={props.setPreProcessors}
          postProcessors={props.postProcessors}
          setPostProcessors={props.setPostProcessors}
          transformers={props.transformers}
          setTransformers={props.setTransformers}
          incremental={props.incremental}
          setIncremental={props.setIncremental}
          webhookUrl={props.webhookUrl}
          setWebhookUrl={props.setWebhookUrl}
          webhookEvents={props.webhookEvents}
          setWebhookEvents={props.setWebhookEvents}
          webhookSecret={props.webhookSecret}
          setWebhookSecret={props.setWebhookSecret}
          profiles={props.profiles}
          urlsInput={batchUrls}
          setUrlsInput={setBatchUrls}
          maxDepth={props.maxDepth}
          setMaxDepth={props.setMaxDepth}
          maxPages={props.maxPages}
          setMaxPages={props.setMaxPages}
          query={batchQuery}
          setQuery={setBatchQuery}
          onSubmitScrape={handleSubmitBatchScrape}
          onSubmitCrawl={handleSubmitBatchCrawl}
          onSubmitResearch={handleSubmitBatchResearch}
          loading={props.loading}
        />
      </section>

      <section id="batches">
        <BatchList
          batches={batches}
          jobs={batchJobs}
          onViewStatus={refreshBatches}
          onCancel={handleCancelBatch}
          onRefresh={refreshBatches}
          loading={batchesLoading}
        />
      </section>
    </>
  );
}
