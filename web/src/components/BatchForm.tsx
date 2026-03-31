/**
 * Purpose: Render the operator-facing batch submission workspace for scrape, crawl, and research jobs.
 * Responsibilities: Wire batch-specific authoring state into the extracted UI sections and submission hook, keep device state local to the form, and preserve in-context submission feedback.
 * Scope: Batch authoring only; request construction details live in `batch-form/useBatchFormSubmission` and surrounding route data orchestration stays outside this component.
 * Usage: Render from `BatchContainer` with the shared `FormController`, controlled batch-local fields, and submit callbacks.
 * Invariants/Assumptions: URLs are validated before request assembly, research batches require a non-empty query, and transient validation problems surface through the global toast system instead of browser alerts.
 */

import { useEffect, useRef, useState } from "react";

import type { DeviceEmulation } from "../api";
import {
  BatchCommonOptions,
  BatchSubmissionNoticePanel,
  BatchSubmitActions,
  BatchTabSelector,
  BatchTabSpecificFields,
  BatchUrlInputSection,
} from "./batch-form/BatchFormSections";
import type { BatchFormProps } from "./batch-form/types";
import { useBatchFormSubmission } from "./batch-form/useBatchFormSubmission";

export type {
  BatchFormProps,
  BatchFormTab,
  BatchSubmissionNotice,
} from "./batch-form/types";

export function BatchForm({
  activeTab,
  setActiveTab,
  form,
  profiles,
  urlsInput,
  setUrlsInput,
  submissionNotice,
  onViewSubmittedBatch,
  maxDepth,
  setMaxDepth,
  maxPages,
  setMaxPages,
  query,
  setQuery,
  onSubmitScrape,
  onSubmitCrawl,
  onSubmitResearch,
  loading,
}: BatchFormProps) {
  const [device, setDevice] = useState<DeviceEmulation | null>(null);
  const submissionNoticeRef = useRef<HTMLOutputElement>(null);
  const {
    fileError,
    fileInputRef,
    handleClear,
    handleFileUpload,
    handleFormSubmit,
    handleUrlsInputChange,
    isValidBatchSize,
    parsedUrls,
    urlError,
  } = useBatchFormSubmission({
    activeTab,
    form,
    urlsInput,
    setUrlsInput,
    maxDepth,
    maxPages,
    query,
    setQuery,
    device,
    onSubmitScrape,
    onSubmitCrawl,
    onSubmitResearch,
  });

  useEffect(() => {
    if (submissionNotice) {
      submissionNoticeRef.current?.focus();
    }
  }, [submissionNotice]);

  return (
    <form className="panel" onSubmit={handleFormSubmit}>
      <h2>Batch Jobs</h2>

      <BatchSubmissionNoticePanel
        submissionNotice={submissionNotice}
        onViewSubmittedBatch={onViewSubmittedBatch}
        submissionNoticeRef={submissionNoticeRef}
      />

      <BatchTabSelector activeTab={activeTab} setActiveTab={setActiveTab} />

      <BatchUrlInputSection
        parsedUrlCount={parsedUrls.length}
        urlsInput={urlsInput}
        onUrlsInputChange={handleUrlsInputChange}
        isValidBatchSize={isValidBatchSize}
        urlError={urlError}
        fileInputRef={fileInputRef}
        onFileUpload={handleFileUpload}
        fileError={fileError}
      />

      <BatchTabSpecificFields
        activeTab={activeTab}
        maxDepth={maxDepth}
        setMaxDepth={setMaxDepth}
        maxPages={maxPages}
        setMaxPages={setMaxPages}
        query={query}
        setQuery={setQuery}
      />

      <BatchCommonOptions
        activeTab={activeTab}
        form={form}
        profiles={profiles}
        device={device}
        onDeviceChange={setDevice}
      />

      <BatchSubmitActions
        activeTab={activeTab}
        parsedUrlCount={parsedUrls.length}
        isValidBatchSize={isValidBatchSize}
        loading={loading}
        onClear={handleClear}
      />
    </form>
  );
}
