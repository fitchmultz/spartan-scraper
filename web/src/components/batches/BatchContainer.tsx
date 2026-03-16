/**
 * Purpose: Coordinate the Web batch route's form state, submission actions, detail refreshes, and paginated list rendering.
 * Responsibilities: Bind the batch form to hook actions, preserve submission notices, and wire list/detail/pagination callbacks into presentational components.
 * Scope: Batch route container behavior only; presentational layout lives in BatchForm/BatchList and data access lives in useBatches.
 * Usage: Render from the main app shell wherever the batch operator workflow should appear.
 * Invariants/Assumptions: Submission notices only highlight batches visible in the current page and detail loading is triggered on demand for a selected batch.
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

  const {
    batches,
    batchJobs,
    lastSubmittedBatch,
    loading: batchesLoading,
    total,
    limit,
    offset,
    refreshBatch,
    refreshBatches,
    cancelBatch,
    submitBatchScrape,
    submitBatchCrawl,
    submitBatchResearch,
    clearLastSubmittedBatch,
  } = useBatches();

  const submissionNotice: BatchSubmissionNotice | null = lastSubmittedBatch
    ? {
        batchId: lastSubmittedBatch.batchId,
        kind: lastSubmittedBatch.kind,
        submittedUrls: lastSubmittedBatch.submittedUrls,
      }
    : null;

  const highlightedBatchId = batches.some(
    (batch) => batch.id === submissionNotice?.batchId,
  )
    ? (submissionNotice?.batchId ?? null)
    : null;

  const clearSubmissionFeedback = useCallback(() => {
    clearLastSubmittedBatch();
  }, [clearLastSubmittedBatch]);

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

  const showSubmissionFeedback = useCallback(async () => {
    await refreshBatches(0);
    document.getElementById("batch-forms")?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
  }, [refreshBatches]);

  const handleViewSubmittedBatch = useCallback(() => {
    const targetId = submissionNotice
      ? `batch-${submissionNotice.batchId}`
      : "batches";

    document.getElementById(targetId)?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
  }, [submissionNotice]);

  const handleSubmitBatchScrape = useCallback(
    async (request: BatchScrapeRequest) => {
      try {
        await submitBatchScrape(request);
        setBatchUrls("");
        await showSubmissionFeedback();
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
        await submitBatchCrawl(request);
        setBatchUrls("");
        await showSubmissionFeedback();
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
        await submitBatchResearch(request);
        setBatchUrls("");
        setBatchQuery("");
        await showSubmissionFeedback();
      } catch (err) {
        console.error("Failed to submit batch research:", err);
        alert(`Failed to submit batch: ${String(err)}`);
      }
    },
    [showSubmissionFeedback, submitBatchResearch],
  );

  const handleViewBatchStatus = useCallback(
    async (batchId: string) => {
      try {
        await refreshBatch(batchId);
      } catch (err) {
        console.error("Failed to load batch details:", err);
        alert(`Failed to load batch details: ${String(err)}`);
      }
    },
    [refreshBatch],
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
          total={total}
          limit={limit}
          offset={offset}
          highlightedBatchId={highlightedBatchId}
          onViewStatus={handleViewBatchStatus}
          onCancel={handleCancelBatch}
          onRefresh={() => void refreshBatches()}
          onPageChange={(nextOffset) => void refreshBatches(nextOffset)}
          loading={batchesLoading}
        />
      </section>
    </>
  );
}
