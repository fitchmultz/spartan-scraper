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
import type { FormController } from "../../hooks/useFormState";
import { useBatches } from "../../hooks/useBatches";
import { BatchList } from "../../components/BatchList";
import {
  BatchForm,
  type BatchSubmissionNotice,
} from "../../components/BatchForm";

interface BatchContainerProps {
  formState: FormController;
  profiles: Profile[];
  loading: boolean;
}

export function BatchContainer(props: BatchContainerProps) {
  const [batchTab, setBatchTab] = useState<"scrape" | "crawl" | "research">(
    "scrape",
  );
  const [batchUrls, setBatchUrls] = useState("");
  const [batchQuery, setBatchQuery] = useState("");
  const [submissionNotice, setSubmissionNotice] =
    useState<BatchSubmissionNotice | null>(null);
  const [highlightedBatchId, setHighlightedBatchId] = useState<string | null>(
    null,
  );

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

  const clearSubmissionFeedback = useCallback(() => {
    setSubmissionNotice(null);
    setHighlightedBatchId(null);
  }, []);

  const handleSetBatchTab = useCallback(
    (tab: "scrape" | "crawl" | "research") => {
      clearSubmissionFeedback();
      setBatchTab(tab);
    },
    [clearSubmissionFeedback],
  );

  const handleSetBatchUrls = useCallback(
    (value: string) => {
      clearSubmissionFeedback();
      setBatchUrls(value);
    },
    [clearSubmissionFeedback],
  );

  const handleSetBatchQuery = useCallback(
    (value: string) => {
      clearSubmissionFeedback();
      setBatchQuery(value);
    },
    [clearSubmissionFeedback],
  );

  const showSubmissionFeedback = useCallback(
    (
      kind: BatchSubmissionNotice["kind"],
      batchId: string,
      submittedUrls: string[],
    ) => {
      setSubmissionNotice({
        batchId,
        kind,
        submittedUrls,
      });
      setHighlightedBatchId(batchId);
    },
    [],
  );

  const handleViewSubmittedBatch = useCallback(() => {
    document.getElementById("batches")?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
  }, []);

  const handleSubmitBatchScrape = useCallback(
    async (request: BatchScrapeRequest) => {
      try {
        const createdBatch = await submitBatchScrape(request);
        showSubmissionFeedback(
          createdBatch.kind,
          createdBatch.id,
          request.jobs.map((job) => job.url),
        );
        setBatchUrls("");
      } catch (err) {
        console.error("Failed to submit batch scrape:", err);
        alert(`Failed to submit batch: ${String(err)}`);
      }
    },
    [showSubmissionFeedback, submitBatchScrape],
  );

  const handleSubmitBatchCrawl = useCallback(
    async (request: BatchCrawlRequest) => {
      try {
        const createdBatch = await submitBatchCrawl(request);
        showSubmissionFeedback(
          createdBatch.kind,
          createdBatch.id,
          request.jobs.map((job) => job.url),
        );
        setBatchUrls("");
      } catch (err) {
        console.error("Failed to submit batch crawl:", err);
        alert(`Failed to submit batch: ${String(err)}`);
      }
    },
    [showSubmissionFeedback, submitBatchCrawl],
  );

  const handleSubmitBatchResearch = useCallback(
    async (request: BatchResearchRequest) => {
      try {
        const createdBatch = await submitBatchResearch(request);
        showSubmissionFeedback(
          createdBatch.kind,
          createdBatch.id,
          request.jobs.map((job) => job.url),
        );
        setBatchUrls("");
        setBatchQuery("");
      } catch (err) {
        console.error("Failed to submit batch research:", err);
        alert(`Failed to submit batch: ${String(err)}`);
      }
    },
    [showSubmissionFeedback, submitBatchResearch],
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
          setActiveTab={handleSetBatchTab}
          form={props.formState}
          profiles={props.profiles}
          urlsInput={batchUrls}
          setUrlsInput={handleSetBatchUrls}
          submissionNotice={submissionNotice}
          onViewSubmittedBatch={handleViewSubmittedBatch}
          maxDepth={props.formState.maxDepth}
          setMaxDepth={props.formState.setMaxDepth}
          maxPages={props.formState.maxPages}
          setMaxPages={props.formState.setMaxPages}
          query={batchQuery}
          setQuery={handleSetBatchQuery}
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
          highlightedBatchId={highlightedBatchId}
          onViewStatus={refreshBatches}
          onCancel={handleCancelBatch}
          onRefresh={refreshBatches}
          loading={batchesLoading}
        />
      </section>
    </>
  );
}
