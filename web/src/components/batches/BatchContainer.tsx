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
import { BatchForm } from "../../components/BatchForm";

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
          form={props.formState}
          profiles={props.profiles}
          urlsInput={batchUrls}
          setUrlsInput={setBatchUrls}
          maxDepth={props.formState.maxDepth}
          setMaxDepth={props.formState.setMaxDepth}
          maxPages={props.formState.maxPages}
          setMaxPages={props.formState.setMaxPages}
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
