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
  activeTab: "scrape" | "crawl" | "research";
  setActiveTab: (tab: "scrape" | "crawl" | "research") => void;
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
    activeTab,
    setActiveTab,
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
    <section id="forms" className="job-workflow" data-tour="form-types">
      <div className="panel job-workflow__header">
        <div className="job-workflow__header-copy">
          <div className="job-workflow__eyebrow">New Job Workflow</div>
          <h2>Run one job path at a time</h2>
          <p>
            Keep the active workflow and submit action visible first, then pull
            in presets or advanced controls only when you need them.
          </p>
        </div>
        <div className="job-workflow__header-controls">
          <div className="job-workflow__status">
            <span>{loading ? "Submitting..." : "Ready to submit"}</span>
            <span>{activeTab}</span>
          </div>
          <div
            className="job-workflow__tabs"
            role="tablist"
            aria-label="Job type"
          >
            <button
              type="button"
              className={activeTab === "scrape" ? "active" : "secondary"}
              onClick={() => setActiveTab("scrape")}
            >
              Scrape
            </button>
            <button
              type="button"
              className={activeTab === "crawl" ? "active" : "secondary"}
              onClick={() => setActiveTab("crawl")}
            >
              Crawl
            </button>
            <button
              type="button"
              className={activeTab === "research" ? "active" : "secondary"}
              onClick={() => setActiveTab("research")}
            >
              Research
            </button>
          </div>
        </div>
      </div>
      <Suspense
        fallback={<div className="loading-placeholder">Loading forms...</div>}
      >
        {activeTab === "scrape" ? (
          <ScrapeForm
            ref={scrapeFormRef}
            form={formState}
            profiles={profiles}
            onSubmit={async (req) => {
              await onSubmitScrape(req);
            }}
            loading={loading}
          />
        ) : null}

        {activeTab === "crawl" ? (
          <CrawlForm
            ref={crawlFormRef}
            form={formState}
            profiles={profiles}
            onSubmit={async (req) => {
              await onSubmitCrawl(req);
            }}
            loading={loading}
          />
        ) : null}

        {activeTab === "research" ? (
          <ResearchForm
            ref={researchFormRef}
            form={formState}
            profiles={profiles}
            onSubmit={async (req) => {
              await onSubmitResearch(req);
            }}
            loading={loading}
          />
        ) : null}
      </Suspense>
    </section>
  );
});
