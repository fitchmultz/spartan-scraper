/**
 * Purpose: Define the shared contract for the batch authoring workspace.
 * Responsibilities: Centralize batch form prop types, tab identifiers, and submission-notice shapes used by the batch container, form, and supporting helpers.
 * Scope: Type definitions only; request assembly, validation, and rendering stay in the form and helper modules.
 * Usage: Import from batch authoring components that need a stable type contract without depending on the full `BatchForm` implementation file.
 * Invariants/Assumptions: Batch tabs remain limited to scrape/crawl/research, submission notices always include the accepted batch id plus submitted URLs, and request callbacks stay async.
 */

import type {
  BatchCrawlRequest,
  BatchResearchRequest,
  BatchScrapeRequest,
} from "../../api";
import type { FormController, ProfileOption } from "../../hooks/useFormState";

export type BatchFormTab = "scrape" | "crawl" | "research";

export interface BatchSubmissionNotice {
  batchId: string;
  kind: BatchFormTab;
  submittedUrls: string[];
}

export interface BatchFormProps {
  activeTab: BatchFormTab;
  setActiveTab: (tab: BatchFormTab) => void;
  form: FormController;
  profiles: ProfileOption[];
  urlsInput: string;
  setUrlsInput: (value: string) => void;
  submissionNotice: BatchSubmissionNotice | null;
  onViewSubmittedBatch: () => void;
  maxDepth: number;
  setMaxDepth: (value: number) => void;
  maxPages: number;
  setMaxPages: (value: number) => void;
  query: string;
  setQuery: (value: string) => void;
  onSubmitScrape: (request: BatchScrapeRequest) => Promise<void>;
  onSubmitCrawl: (request: BatchCrawlRequest) => Promise<void>;
  onSubmitResearch: (request: BatchResearchRequest) => Promise<void>;
  loading: boolean;
}
