/**
 * Purpose: Render the guided wizard basics step for selecting job type and entering the minimum required target information.
 * Responsibilities: Present job-type choices, collect scrape/crawl/research entry fields, and surface blocking basics-step validation errors.
 * Scope: Guided job wizard basics step only.
 * Usage: Render from `JobSubmissionContainer` when the wizard is on the `basics` step.
 * Invariants/Assumptions: Job type is always one of scrape/crawl/research, research keeps source URLs as a required input, and max depth/max pages are controlled by the shared form controller.
 */

import type { JobType } from "../../../types/presets";

const JOB_TYPE_OPTIONS: Array<{
  jobType: JobType;
  title: string;
  summary: string;
}> = [
  {
    jobType: "scrape",
    title: "Scrape",
    summary:
      "Extract one target page with focused runtime and extraction controls.",
  },
  {
    jobType: "crawl",
    title: "Crawl",
    summary:
      "Traverse a bounded section of a site with explicit depth and page limits.",
  },
  {
    jobType: "research",
    title: "Research",
    summary:
      "Synthesize findings across a guided source set with AI-assisted analysis.",
  },
];

interface BasicsStepProps {
  activeTab: JobType;
  setActiveTab: (tab: JobType) => void;
  scrapeUrl: string;
  setScrapeUrl: (value: string) => void;
  crawlUrl: string;
  setCrawlUrl: (value: string) => void;
  researchQuery: string;
  setResearchQuery: (value: string) => void;
  researchUrls: string;
  setResearchUrls: (value: string) => void;
  maxDepth: number;
  setMaxDepth: (value: number) => void;
  maxPages: number;
  setMaxPages: (value: number) => void;
  errors: string[];
}

export function BasicsStep({
  activeTab,
  setActiveTab,
  scrapeUrl,
  setScrapeUrl,
  crawlUrl,
  setCrawlUrl,
  researchQuery,
  setResearchQuery,
  researchUrls,
  setResearchUrls,
  maxDepth,
  setMaxDepth,
  maxPages,
  setMaxPages,
  errors,
}: BasicsStepProps) {
  return (
    <section className="panel job-wizard__panel job-wizard__panel--basics">
      <div className="job-wizard__panel-header">
        <div className="job-workflow__eyebrow">Basics</div>
        <h2>Pick the workflow and define the target</h2>
        <p>
          Start with the smallest amount of information Spartan needs to run the
          job. You can tune runtime and extraction details in the next steps.
        </p>
      </div>

      {errors.length > 0 ? (
        <div className="job-wizard__error-summary" role="alert">
          <strong>Fix these before continuing:</strong>
          <ul>
            {errors.map((error) => (
              <li key={error}>{error}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="job-wizard__type-picker" data-tour="job-type-selection">
        {JOB_TYPE_OPTIONS.map((option) => {
          const isActive = activeTab === option.jobType;
          return (
            <button
              key={option.jobType}
              type="button"
              className={isActive ? "is-active" : undefined}
              onClick={() => setActiveTab(option.jobType)}
            >
              <strong>{option.title}</strong>
              <span>{option.summary}</span>
            </button>
          );
        })}
      </div>

      {activeTab === "scrape" ? (
        <div className="job-wizard__field-stack">
          <label htmlFor="wizard-scrape-url">Target URL</label>
          <input
            id="wizard-scrape-url"
            value={scrapeUrl}
            onChange={(event) => setScrapeUrl(event.target.value)}
            placeholder="https://example.com"
          />
        </div>
      ) : null}

      {activeTab === "crawl" ? (
        <div className="job-wizard__field-stack">
          <label htmlFor="wizard-crawl-url">Start URL</label>
          <input
            id="wizard-crawl-url"
            value={crawlUrl}
            onChange={(event) => setCrawlUrl(event.target.value)}
            placeholder="https://example.com"
          />

          <div className="row">
            <label>
              Max depth
              <input
                type="number"
                min={1}
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
        </div>
      ) : null}

      {activeTab === "research" ? (
        <div className="job-wizard__field-stack">
          <label htmlFor="wizard-research-query">Research query</label>
          <input
            id="wizard-research-query"
            value={researchQuery}
            onChange={(event) => setResearchQuery(event.target.value)}
            placeholder="pricing model, security posture, roadmap..."
          />

          <label htmlFor="wizard-research-urls">
            Source URLs (one per line or comma-separated)
          </label>
          <textarea
            id="wizard-research-urls"
            rows={4}
            value={researchUrls}
            onChange={(event) => setResearchUrls(event.target.value)}
            placeholder="https://example.com\nhttps://example.com/docs"
          />

          <div className="row">
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
        </div>
      ) : null}
    </section>
  );
}
