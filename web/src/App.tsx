/**
 * Spartan Scraper Web UI - Main Application Component
 *
 * This is the primary React component for the Spartan Scraper web interface.
 * It provides a single-page application for:
 *
 * - Submitting scrape, crawl, and research jobs
 * - Configuring authentication, headers, cookies, and query parameters
 * - Managing extraction templates and validation
 * - Viewing real-time job status and manager state
 * - Browsing and analyzing job results
 *
 * The component has been refactored to use container components for each
 * major functional area, reducing its size and improving maintainability.
 *
 * @module App
 */

import {
  useCallback,
  useEffect,
  useRef,
  Suspense,
  lazy,
  useState,
} from "react";
import {
  deleteV1JobsById,
  postV1Crawl,
  postV1Research,
  postV1Scrape,
  type ScrapeRequest,
  type CrawlRequest,
  type ResearchRequest,
} from "./api";
import { Hero } from "./components/Hero";
import { JobList } from "./components/JobList";
import { InfoSections } from "./components/InfoSections";
import { CommandPalette } from "./components/CommandPalette";
import { KeyboardShortcutsHelp } from "./components/KeyboardShortcutsHelp";
import { WelcomeModal } from "./components/WelcomeModal";
import { OnboardingFlow } from "./components/OnboardingFlow";
import { WatchContainer } from "./components/watches/WatchContainer";
import { ExportScheduleContainer } from "./components/export-schedules/ExportScheduleContainer";
import { FeedContainer } from "./components/feeds/FeedContainer";
import { WebhookDeliveryContainer } from "./components/webhooks/WebhookDeliveryContainer";
import { RetentionStatusPanel } from "./components/RetentionStatusPanel";
import { DedupExplorer } from "./components/DedupExplorer";
import { TemplatePerformance } from "./components/TemplatePerformance";
import { TemplateABTestManager } from "./components/TemplateABTestManager";
import { AITemplateGenerator } from "./components/AITemplateGenerator";
import { ChainContainer } from "./components/chains/ChainContainer";
import { BatchContainer } from "./components/batches/BatchContainer";
import { PresetContainer } from "./components/presets/PresetContainer";
import {
  JobSubmissionContainer,
  type JobSubmissionContainerRef,
} from "./components/jobs/JobSubmissionContainer";
import { ResultsContainer } from "./components/results/ResultsContainer";
import { useKeyboard } from "./hooks/useKeyboard";
import { useAppData } from "./hooks/useAppData";
import { useFormState } from "./hooks/useFormState";
import { useResultsState } from "./hooks/useResultsState";
import { useTheme } from "./hooks/useTheme";
import { usePresets } from "./hooks/usePresets";
import { useOnboarding } from "./hooks/useOnboarding";
import {
  submitScrapeJob,
  submitCrawlJob,
  submitResearchJob,
} from "./lib/job-actions";
import { getApiBaseUrl } from "./lib/api-config";
import type { JobPreset, JobType } from "./types/presets";

// Lazy load heavy components to reduce initial bundle size
const MetricsDashboard = lazy(() =>
  import("./components/MetricsDashboard").then((mod) => ({
    default: mod.MetricsDashboard,
  })),
);

export function App() {
  const appData = useAppData();
  const formState = useFormState();
  const resultsState = useResultsState();
  const { theme, resolvedTheme, setTheme, toggleTheme } = useTheme();
  const { presets, savePreset } = usePresets();

  // Onboarding state management
  const {
    shouldShowWelcome,
    isTourActive,
    currentStep,
    startOnboarding,
    skipOnboarding,
    resetOnboarding,
    goToStep,
    finishOnboarding,
  } = useOnboarding();

  // Keyboard shortcuts and command palette
  const {
    isCommandPaletteOpen,
    isHelpOpen,
    closeCommandPalette,
    closeHelp,
    shortcuts,
    isMac,
  } = useKeyboard();

  // Active tab state for preset filtering
  const [activeTab, setActiveTab] = useState<JobType>("scrape");

  // AI Template Generator modal state
  const [isAIGeneratorOpen, setIsAIGeneratorOpen] = useState(false);

  // Ref for JobSubmissionContainer to access form methods
  const jobSubmissionRef = useRef<JobSubmissionContainerRef>(null);

  const {
    jobs,
    profiles,
    schedules,
    templates,
    crawlStates,
    managerStatus,
    metrics,
    jobsTotal,
    jobsPage,
    crawlStatesTotal,
    crawlStatesPage,
    error,
    loading,
    connectionState,
    refreshJobs,
    refreshTemplates,
    setJobsPage,
    setCrawlStatesPage,
  } = appData;

  const { selectedJobId, loadResults } = resultsState;

  // Handle navigation from keyboard shortcuts
  const handleNavigate = useCallback((view: "jobs" | "results" | "forms") => {
    const elementId =
      view === "jobs" ? "jobs" : view === "results" ? "results" : "forms";
    const element = document.getElementById(elementId);
    if (element) {
      element.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }, []);

  // Handle form submission from command palette
  const handleSubmitForm = useCallback(
    async (formType: "scrape" | "crawl" | "research") => {
      if (formType === "scrape") {
        await jobSubmissionRef.current?.submitScrape();
      } else if (formType === "crawl") {
        await jobSubmissionRef.current?.submitCrawl();
      } else if (formType === "research") {
        await jobSubmissionRef.current?.submitResearch();
      }
    },
    [],
  );

  // Listen for keyboard navigation events
  useEffect(() => {
    const handleKeyboardNavigate = (event: CustomEvent) => {
      const { destination } = event.detail;
      if (destination === "navigateJobs") handleNavigate("jobs");
      if (destination === "navigateResults") handleNavigate("results");
      if (destination === "navigateForms") handleNavigate("forms");
    };

    window.addEventListener(
      "keyboard-navigate",
      handleKeyboardNavigate as EventListener,
    );
    return () => {
      window.removeEventListener(
        "keyboard-navigate",
        handleKeyboardNavigate as EventListener,
      );
    };
  }, [handleNavigate]);

  const handleSubmitScrape = useCallback(
    async (request: ScrapeRequest) => {
      await submitScrapeJob(postV1Scrape, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });
    },
    [refreshJobs],
  );

  const handleSubmitCrawl = useCallback(
    async (request: CrawlRequest) => {
      await submitCrawlJob(postV1Crawl, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });
    },
    [refreshJobs],
  );

  const handleSubmitResearch = useCallback(
    async (request: ResearchRequest) => {
      await submitResearchJob(postV1Research, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });
    },
    [refreshJobs],
  );

  const cancelJob = useCallback(
    async (jobId: string) => {
      try {
        const { error: apiError } = await deleteV1JobsById({
          baseUrl: getApiBaseUrl(),
          path: { id: jobId },
        });
        if (apiError) {
          console.error(String(apiError));
          return;
        }
        await refreshJobs();
      } catch (err) {
        console.error(String(err));
      }
    },
    [refreshJobs],
  );

  const deleteJob = useCallback(
    async (jobId: string) => {
      if (!confirm("Are you sure you want to permanently delete this job?")) {
        return;
      }
      try {
        const { error: apiError } = await deleteV1JobsById({
          baseUrl: getApiBaseUrl(),
          path: { id: jobId },
          query: { force: true },
        });
        if (apiError) {
          console.error(String(apiError));
          return;
        }
        await refreshJobs();
        if (selectedJobId === jobId) {
          loadResults("");
        }
      } catch (err) {
        console.error(String(err));
      }
    },
    [refreshJobs, selectedJobId, loadResults],
  );

  // Find active (running) job for command palette
  const activeJob = jobs.find((job) => job.status === "running");

  // Handle preset selection - coordinate with form refs for URL/query setting
  const handleSelectPreset = useCallback((preset: JobPreset) => {
    // Set URL/query if provided in preset (form refs are in JobSubmissionContainer)
    if (preset.config.url) {
      if (preset.jobType === "scrape") {
        jobSubmissionRef.current?.setScrapeUrl(preset.config.url);
      } else if (preset.jobType === "crawl") {
        jobSubmissionRef.current?.setCrawlUrl(preset.config.url);
      }
    }
    if (preset.config.query) {
      jobSubmissionRef.current?.setResearchQuery(preset.config.query);
    }
  }, []);

  // Get current config from active form - simplified for container approach
  const getCurrentConfig = useCallback(() => {
    // Return current form state as preset config
    return {
      headless: formState.headless,
      usePlaywright: formState.usePlaywright,
      timeoutSeconds: formState.timeoutSeconds,
      authProfile: formState.authProfile,
      authBasic: formState.authBasic,
      headersRaw: formState.headersRaw,
      cookiesRaw: formState.cookiesRaw,
      queryRaw: formState.queryRaw,
      loginUrl: formState.loginUrl,
      loginUserSelector: formState.loginUserSelector,
      loginPassSelector: formState.loginPassSelector,
      loginSubmitSelector: formState.loginSubmitSelector,
      loginUser: formState.loginUser,
      loginPass: formState.loginPass,
      extractTemplate: formState.extractTemplate,
      extractValidate: formState.extractValidate,
      preProcessors: formState.preProcessors,
      postProcessors: formState.postProcessors,
      transformers: formState.transformers,
      incremental: formState.incremental,
      maxDepth: formState.maxDepth,
      maxPages: formState.maxPages,
      webhookUrl: formState.webhookUrl,
      webhookEvents: formState.webhookEvents,
      webhookSecret: formState.webhookSecret,
      interceptEnabled: formState.interceptEnabled,
      interceptURLPatterns: formState.interceptURLPatterns,
      interceptResourceTypes: formState.interceptResourceTypes,
      interceptCaptureRequestBody: formState.interceptCaptureRequestBody,
      interceptCaptureResponseBody: formState.interceptCaptureResponseBody,
      interceptMaxBodySize: formState.interceptMaxBodySize,
    };
  }, [formState]);

  // Get current URL for preset matching
  const getCurrentUrl = useCallback(() => {
    switch (activeTab) {
      case "scrape":
        return jobSubmissionRef.current?.getScrapeUrl() ?? "";
      case "crawl":
        return jobSubmissionRef.current?.getCrawlUrl() ?? "";
      default:
        return "";
    }
  }, [activeTab]);

  return (
    <div className="app">
      <Hero
        loading={loading}
        managerStatus={managerStatus}
        jobsCount={jobs.length}
        headless={formState.headless}
        usePlaywright={formState.usePlaywright}
        theme={theme}
        resolvedTheme={resolvedTheme}
        onThemeChange={setTheme}
        onThemeToggle={toggleTheme}
      />

      <CommandPalette
        isOpen={isCommandPaletteOpen}
        onClose={closeCommandPalette}
        jobs={jobs}
        onNavigate={handleNavigate}
        onSubmitForm={handleSubmitForm}
        onCancelJob={cancelJob}
        activeJobId={activeJob?.id}
        isMac={isMac}
        presets={presets}
        onSelectPreset={handleSelectPreset}
        onRestartTour={resetOnboarding}
      />

      <PresetContainer
        presets={presets}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        applyPreset={formState.applyPreset}
        savePreset={savePreset}
        getCurrentConfig={getCurrentConfig}
        getCurrentUrl={getCurrentUrl}
        onSelectPreset={handleSelectPreset}
      />

      <KeyboardShortcutsHelp
        isOpen={isHelpOpen}
        onClose={closeHelp}
        shortcuts={shortcuts}
        isMac={isMac}
      />

      {/* Onboarding Components */}
      <WelcomeModal
        isOpen={shouldShowWelcome}
        onStartTour={startOnboarding}
        onSkip={skipOnboarding}
      />

      <OnboardingFlow
        isRunning={isTourActive}
        currentStep={currentStep}
        onComplete={finishOnboarding}
        onSkip={skipOnboarding}
        onStepChange={goToStep}
      />

      <Suspense
        fallback={<div className="loading-placeholder">Loading metrics...</div>}
      >
        <MetricsDashboard metrics={metrics} connectionState={connectionState} />
      </Suspense>

      <JobSubmissionContainer
        ref={jobSubmissionRef}
        formState={formState}
        onSubmitScrape={handleSubmitScrape}
        onSubmitCrawl={handleSubmitCrawl}
        onSubmitResearch={handleSubmitResearch}
        loading={loading}
        profiles={profiles}
      />

      <BatchContainer
        headless={formState.headless}
        setHeadless={formState.setHeadless}
        usePlaywright={formState.usePlaywright}
        setUsePlaywright={formState.setUsePlaywright}
        timeoutSeconds={formState.timeoutSeconds}
        setTimeoutSeconds={formState.setTimeoutSeconds}
        authProfile={formState.authProfile}
        setAuthProfile={formState.setAuthProfile}
        authBasic={formState.authBasic}
        setAuthBasic={formState.setAuthBasic}
        headersRaw={formState.headersRaw}
        setHeadersRaw={formState.setHeadersRaw}
        cookiesRaw={formState.cookiesRaw}
        setCookiesRaw={formState.setCookiesRaw}
        queryRaw={formState.queryRaw}
        setQueryRaw={formState.setQueryRaw}
        loginUrl={formState.loginUrl}
        setLoginUrl={formState.setLoginUrl}
        loginUserSelector={formState.loginUserSelector}
        setLoginUserSelector={formState.setLoginUserSelector}
        loginPassSelector={formState.loginPassSelector}
        setLoginPassSelector={formState.setLoginPassSelector}
        loginSubmitSelector={formState.loginSubmitSelector}
        setLoginSubmitSelector={formState.setLoginSubmitSelector}
        loginUser={formState.loginUser}
        setLoginUser={formState.setLoginUser}
        loginPass={formState.loginPass}
        setLoginPass={formState.setLoginPass}
        extractTemplate={formState.extractTemplate}
        setExtractTemplate={formState.setExtractTemplate}
        extractValidate={formState.extractValidate}
        setExtractValidate={formState.setExtractValidate}
        preProcessors={formState.preProcessors}
        setPreProcessors={formState.setPreProcessors}
        postProcessors={formState.postProcessors}
        setPostProcessors={formState.setPostProcessors}
        transformers={formState.transformers}
        setTransformers={formState.setTransformers}
        incremental={formState.incremental}
        setIncremental={formState.setIncremental}
        maxDepth={formState.maxDepth}
        setMaxDepth={formState.setMaxDepth}
        maxPages={formState.maxPages}
        setMaxPages={formState.setMaxPages}
        webhookUrl={formState.webhookUrl}
        setWebhookUrl={formState.setWebhookUrl}
        webhookEvents={formState.webhookEvents}
        setWebhookEvents={formState.setWebhookEvents}
        webhookSecret={formState.webhookSecret}
        setWebhookSecret={formState.setWebhookSecret}
        profiles={profiles}
        loading={loading}
      />

      <ChainContainer onChainSubmit={refreshJobs} />

      <section id="jobs">
        <JobList
          jobs={jobs}
          error={error}
          onViewResults={loadResults}
          onCancel={cancelJob}
          onDelete={deleteJob}
          onRefresh={refreshJobs}
          currentPage={jobsPage}
          totalJobs={jobsTotal}
          jobsPerPage={100}
          onPageChange={setJobsPage}
          connectionState={connectionState}
        />
      </section>

      <ResultsContainer resultsState={resultsState} jobs={jobs} />

      <WatchContainer />

      <ExportScheduleContainer />

      <FeedContainer />

      <WebhookDeliveryContainer />

      <RetentionStatusPanel />

      <section id="dedup">
        <DedupExplorer />
      </section>

      <section id="templates-insights" className="templates-insights-section">
        <div className="templates-insights__header">
          <h2>Templates Insights</h2>
          <button
            type="button"
            className="btn btn--small btn--secondary"
            onClick={() => setIsAIGeneratorOpen(true)}
          >
            <span className="mr-1">✨</span> Generate with AI
          </button>
        </div>
        <div className="templates-insights__content">
          {templates.length > 0 ? (
            <>
              <div className="template-performance-panel">
                <h3>Template Performance</h3>
                <TemplatePerformance templateName={templates[0]} />
              </div>
              <div className="template-ab-tests-panel">
                <h3>A/B Testing</h3>
                <TemplateABTestManager />
              </div>
            </>
          ) : (
            <div className="templates-insights__empty">
              <p>
                No templates available. Create templates to see performance
                metrics and run A/B tests.
              </p>
            </div>
          )}
        </div>
      </section>

      <AITemplateGenerator
        isOpen={isAIGeneratorOpen}
        onClose={() => setIsAIGeneratorOpen(false)}
        onTemplateSaved={() => {
          setIsAIGeneratorOpen(false);
          void refreshTemplates();
        }}
      />

      <InfoSections
        profiles={profiles}
        schedules={schedules}
        templates={templates}
        crawlStates={crawlStates}
        crawlStatesPage={crawlStatesPage}
        crawlStatesTotal={crawlStatesTotal}
        crawlStatesPerPage={100}
        onCrawlStatesPageChange={setCrawlStatesPage}
        onTemplatesChanged={() => {
          void refreshTemplates();
        }}
      />

      <div className="footer">
        Spartan Scraper — build once, deploy everywhere.
      </div>
    </div>
  );
}
