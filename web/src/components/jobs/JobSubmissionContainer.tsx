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
import type { ScrapeRequest, CrawlRequest, ResearchRequest } from "../../api";
import type { Profile } from "../../hooks/useAppData";
import type { FormController, ProfileOption } from "../../hooks/useFormState";

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
  formState: FormController;
  onSubmitScrape: (request: ScrapeRequest) => void;
  onSubmitCrawl: (request: CrawlRequest) => void;
  onSubmitResearch: (request: ResearchRequest) => void;
  loading: boolean;
  profiles: Profile[] | ProfileOption[];
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

  return (
    <section id="forms" className="grid" data-tour="form-types">
      <Suspense
        fallback={<div className="loading-placeholder">Loading forms...</div>}
      >
        <ScrapeForm
          ref={scrapeFormRef}
          form={formState}
          profiles={profiles}
          onSubmit={async (req) => {
            await onSubmitScrape(req);
          }}
          loading={loading}
        />

        <CrawlForm
          ref={crawlFormRef}
          form={formState}
          profiles={profiles}
          onSubmit={async (req) => {
            await onSubmitCrawl(req);
          }}
          loading={loading}
        />

        <ResearchForm
          ref={researchFormRef}
          form={formState}
          profiles={profiles}
          onSubmit={async (req) => {
            await onSubmitResearch(req);
          }}
          loading={loading}
        />
      </Suspense>
    </section>
  );
});
