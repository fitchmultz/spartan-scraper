/**
 * Purpose: Host the guided job-creation wizard and expert-mode forms for scrape, crawl, and research runs.
 * Responsibilities: Orchestrate guided steps, preserve per-job drafts, expose imperative helpers for presets and command-palette submission, and lazy-render the active expert form.
 * Scope: Single-job submission only; batch flows, automation workflows, and results exploration stay outside this component.
 * Usage: Render from the New Job route with shared form state, submission callbacks, and profile options.
 * Invariants/Assumptions: Only one job workflow is active at a time, guided and expert modes share the same underlying draft values, and form refs remain the source of imperative submit helpers.
 */

import {
  Suspense,
  forwardRef,
  lazy,
  useCallback,
  useImperativeHandle,
  useRef,
} from "react";
import type {
  ComponentStatus,
  CrawlRequest,
  DeviceEmulation,
  ResearchRequest,
  ScrapeRequest,
} from "../../api";
import type { Profile } from "../../hooks/useAppData";
import type { FormController, ProfileOption } from "../../hooks/useFormState";
import type { JobType, PresetConfig } from "../../types/presets";
import { JobSubmissionAssistantSection } from "../ai-assistant";
import { WizardActions } from "./WizardActions";
import { WizardStepper } from "./WizardStepper";
import { useJobWizard } from "./useJobWizard";
import { BasicsStep } from "./steps/BasicsStep";
import { RuntimeStep } from "./steps/RuntimeStep";
import { ExtractionStep } from "./steps/ExtractionStep";
import { ReviewStep } from "./steps/ReviewStep";

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
  getCurrentConfig: () => PresetConfig;
  applyPreset: (config: PresetConfig, jobType?: JobType) => void;
  clearDraft: (jobType?: JobType) => void;
}

interface JobSubmissionContainerProps {
  activeTab: JobType;
  setActiveTab: (tab: JobType) => void;
  formState: FormController;
  aiStatus?: ComponentStatus | null;
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
    aiStatus = null,
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
  const profileOptions = profiles as ProfileOption[];

  const wizard = useJobWizard({
    activeTab,
    formState,
  });

  const submitActiveJob = useCallback(async () => {
    const blockers = wizard.validateCurrentStep();
    if (wizard.activeStep !== "review" || blockers.length > 0) {
      return;
    }

    if (activeTab === "scrape") {
      await scrapeFormRef.current?.submit();
      return;
    }
    if (activeTab === "crawl") {
      await crawlFormRef.current?.submit();
      return;
    }
    await researchFormRef.current?.submit();
  }, [activeTab, wizard]);

  useImperativeHandle(
    ref,
    () => ({
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
        wizard.updateLocalState("scrape", { url });
      },
      setCrawlUrl: (url: string) => {
        wizard.updateLocalState("crawl", { url });
      },
      setResearchQuery: (query: string) => {
        wizard.updateLocalState("research", { query });
      },
      getScrapeUrl: () => wizard.localState.scrape.url,
      getCrawlUrl: () => wizard.localState.crawl.url,
      getCurrentConfig: () => wizard.getCurrentConfig(activeTab),
      applyPreset: (config, jobType = activeTab) => {
        wizard.applyPreset(config, jobType);
      },
      clearDraft: (jobType = activeTab) => {
        wizard.clearDraft(jobType);
      },
    }),
    [activeTab, wizard],
  );

  const activeRuntimeDevice =
    activeTab === "scrape"
      ? wizard.localState.scrape.device
      : activeTab === "crawl"
        ? wizard.localState.crawl.device
        : wizard.localState.research.device;

  const setActiveRuntimeDevice = useCallback(
    (value: DeviceEmulation | null) => {
      if (activeTab === "scrape") {
        wizard.updateLocalState("scrape", { device: value ?? null });
        return;
      }
      if (activeTab === "crawl") {
        wizard.updateLocalState("crawl", { device: value ?? null });
        return;
      }
      wizard.updateLocalState("research", { device: value ?? null });
    },
    [activeTab, wizard],
  );

  return (
    <section
      id="forms"
      className="job-workflow job-wizard"
      data-tour="form-types"
    >
      <div className="ai-assistant-surface">
        <div className="ai-assistant-surface__main">
          <div className="panel job-workflow__header job-wizard__header">
            <div className="job-workflow__header-copy">
              <div className="job-workflow__eyebrow">Workflow</div>
              <h2>Create a job with a guided workflow</h2>
              <p>
                Move step by step, keep presets in the sidebar, and switch to
                Expert mode at any time without losing entered values.
              </p>
            </div>

            <div className="job-wizard__header-controls">
              <label
                className="job-wizard__mode-toggle"
                data-tour="expert-mode"
              >
                <input
                  type="checkbox"
                  checked={wizard.expertMode}
                  onChange={(event) =>
                    wizard.setExpertMode(event.target.checked)
                  }
                />
                <span>{wizard.expertMode ? "Expert mode" : "Guided mode"}</span>
              </label>
              <span className="job-wizard__draft-status">
                {wizard.draftSavedAt ? "Draft saved" : "Draft active"}
              </span>
            </div>
          </div>

          {!wizard.expertMode ? (
            <>
              <WizardStepper
                activeStep={wizard.activeStep}
                completedSteps={wizard.completedSteps}
                onStepChange={wizard.setActiveStep}
              />

              {wizard.activeStep === "basics" ? (
                <BasicsStep
                  activeTab={activeTab}
                  setActiveTab={setActiveTab}
                  scrapeUrl={wizard.localState.scrape.url}
                  setScrapeUrl={(value) =>
                    wizard.updateLocalState("scrape", { url: value })
                  }
                  crawlUrl={wizard.localState.crawl.url}
                  setCrawlUrl={(value) =>
                    wizard.updateLocalState("crawl", { url: value })
                  }
                  researchQuery={wizard.localState.research.query}
                  setResearchQuery={(value) =>
                    wizard.updateLocalState("research", { query: value })
                  }
                  researchUrls={wizard.localState.research.urls}
                  setResearchUrls={(value) =>
                    wizard.updateLocalState("research", { urls: value })
                  }
                  maxDepth={formState.maxDepth}
                  setMaxDepth={formState.setMaxDepth}
                  maxPages={formState.maxPages}
                  setMaxPages={formState.setMaxPages}
                  errors={wizard.validationErrors.basics ?? []}
                />
              ) : null}

              {wizard.activeStep === "runtime" ? (
                <RuntimeStep
                  form={formState}
                  profiles={profileOptions}
                  device={activeRuntimeDevice}
                  setDevice={setActiveRuntimeDevice}
                  errors={wizard.validationErrors.runtime ?? []}
                  inputPrefix={activeTab}
                />
              ) : null}

              {wizard.activeStep === "extraction" ? (
                <ExtractionStep
                  activeTab={activeTab}
                  form={formState}
                  errors={wizard.validationErrors.extraction ?? []}
                />
              ) : null}

              {wizard.activeStep === "review" ? (
                <ReviewStep
                  activeTab={activeTab}
                  config={wizard.getCurrentConfig(activeTab)}
                  warnings={wizard.reviewWarnings}
                />
              ) : null}

              <WizardActions
                activeStep={wizard.activeStep}
                activeTab={activeTab}
                loading={loading}
                draftSavedAt={wizard.draftSavedAt}
                onBack={wizard.goBack}
                onNext={wizard.goNext}
                onSubmit={() => {
                  void submitActiveJob();
                }}
                onResetDraft={() => wizard.clearDraft(activeTab)}
              />
            </>
          ) : null}

          <Suspense
            fallback={
              <div className="loading-placeholder">Loading job form...</div>
            }
          >
            {activeTab === "scrape" ? (
              <ScrapeForm
                ref={scrapeFormRef}
                surface={wizard.expertMode ? "full" : "headless"}
                form={formState}
                profiles={profileOptions}
                url={wizard.localState.scrape.url}
                setUrl={(value) =>
                  wizard.updateLocalState("scrape", { url: value })
                }
                device={wizard.localState.scrape.device}
                setDevice={(value) =>
                  wizard.updateLocalState("scrape", { device: value })
                }
                onSubmit={async (req) => {
                  await onSubmitScrape(req);
                }}
                loading={loading}
              />
            ) : null}

            {activeTab === "crawl" ? (
              <CrawlForm
                ref={crawlFormRef}
                surface={wizard.expertMode ? "full" : "headless"}
                form={formState}
                profiles={profileOptions}
                url={wizard.localState.crawl.url}
                setUrl={(value) =>
                  wizard.updateLocalState("crawl", { url: value })
                }
                sitemapURL={wizard.localState.crawl.sitemapURL}
                setSitemapURL={(value) =>
                  wizard.updateLocalState("crawl", { sitemapURL: value })
                }
                sitemapOnly={wizard.localState.crawl.sitemapOnly}
                setSitemapOnly={(value) =>
                  wizard.updateLocalState("crawl", { sitemapOnly: value })
                }
                includePatterns={wizard.localState.crawl.includePatterns}
                setIncludePatterns={(value) =>
                  wizard.updateLocalState("crawl", { includePatterns: value })
                }
                excludePatterns={wizard.localState.crawl.excludePatterns}
                setExcludePatterns={(value) =>
                  wizard.updateLocalState("crawl", { excludePatterns: value })
                }
                device={wizard.localState.crawl.device}
                setDevice={(value) =>
                  wizard.updateLocalState("crawl", { device: value })
                }
                onSubmit={async (req) => {
                  await onSubmitCrawl(req);
                }}
                loading={loading}
              />
            ) : null}

            {activeTab === "research" ? (
              <ResearchForm
                ref={researchFormRef}
                surface={wizard.expertMode ? "full" : "headless"}
                form={formState}
                profiles={profileOptions}
                query={wizard.localState.research.query}
                setQuery={(value) =>
                  wizard.updateLocalState("research", { query: value })
                }
                urls={wizard.localState.research.urls}
                setUrls={(value) =>
                  wizard.updateLocalState("research", { urls: value })
                }
                device={wizard.localState.research.device}
                setDevice={(value) =>
                  wizard.updateLocalState("research", { device: value })
                }
                onSubmit={async (req) => {
                  await onSubmitResearch(req);
                }}
                loading={loading}
              />
            ) : null}
          </Suspense>
        </div>

        <JobSubmissionAssistantSection
          activeTab={activeTab}
          form={formState}
          localState={wizard.localState}
          aiStatus={aiStatus}
        />
      </div>
    </section>
  );
});
