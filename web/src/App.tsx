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
 * @module App
 */

import {
  useCallback,
  useRef,
  useEffect,
  useState,
  Suspense,
  lazy,
} from "react";
import {
  deleteV1JobsById,
  postV1Crawl,
  postV1Research,
  postV1Scrape,
  listWatches,
  createWatch,
  updateWatch,
  deleteWatch,
  checkWatch,
  listChains,
  createChain,
  deleteChain,
  submitChain,
  type ScrapeRequest,
  type CrawlRequest,
  type ResearchRequest,
  type BatchScrapeRequest,
  type BatchCrawlRequest,
  type BatchResearchRequest,
  type Watch,
  type WatchInput,
  type WatchCheckResult,
  type JobChain,
  type ChainCreateRequest,
} from "./api";
import { Hero } from "./components/Hero";
import { JobList } from "./components/JobList";
import { InfoSections } from "./components/InfoSections";
import { BatchList } from "./components/BatchList";
import { ChainList } from "./components/ChainList";

// Lazy load heavy components to reduce initial bundle size
const MetricsDashboard = lazy(() =>
  import("./components/MetricsDashboard").then((mod) => ({
    default: mod.MetricsDashboard,
  })),
);
const ResultsExplorer = lazy(() =>
  import("./components/ResultsExplorer").then((mod) => ({
    default: mod.ResultsExplorer,
  })),
);
const WatchManager = lazy(() =>
  import("./components/WatchManager").then((mod) => ({
    default: mod.WatchManager,
  })),
);
const ChainBuilder = lazy(() =>
  import("./components/ChainBuilder").then((mod) => ({
    default: mod.ChainBuilder,
  })),
);

// Lazy load form components (large components only needed when user interacts with forms)
const ScrapeForm = lazy(() =>
  import("./components/ScrapeForm").then((mod) => ({
    default: mod.ScrapeForm,
  })),
);
const CrawlForm = lazy(() =>
  import("./components/CrawlForm").then((mod) => ({
    default: mod.CrawlForm,
  })),
);
const ResearchForm = lazy(() =>
  import("./components/ResearchForm").then((mod) => ({
    default: mod.ResearchForm,
  })),
);
const BatchForm = lazy(() =>
  import("./components/BatchForm").then((mod) => ({
    default: mod.BatchForm,
  })),
);

// Re-export types from lazy-loaded components
type ScrapeFormRef = import("./components/ScrapeForm").ScrapeFormRef;
type CrawlFormRef = import("./components/CrawlForm").CrawlFormRef;
type ResearchFormRef = import("./components/ResearchForm").ResearchFormRef;
import { useBatches } from "./hooks/useBatches";
import { CommandPalette } from "./components/CommandPalette";
import { KeyboardShortcutsHelp } from "./components/KeyboardShortcutsHelp";
import { QuickStartPanel } from "./components/QuickStartPanel";
import { SavePresetDialog } from "./components/SavePresetDialog";
import { WelcomeModal } from "./components/WelcomeModal";
import { OnboardingFlow } from "./components/OnboardingFlow";
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

  // Form refs for programmatic submission
  const scrapeFormRef = useRef<ScrapeFormRef>(null);
  const crawlFormRef = useRef<CrawlFormRef>(null);
  const researchFormRef = useRef<ResearchFormRef>(null);

  // Active tab state for preset filtering
  const [activeTab, setActiveTab] = useState<JobType>("scrape");

  // Batch form state
  const [batchTab, setBatchTab] = useState<"scrape" | "crawl" | "research">(
    "scrape",
  );
  const [batchUrls, setBatchUrls] = useState("");
  const [batchQuery, setBatchQuery] = useState("");

  // Watch state
  const [watches, setWatches] = useState<Watch[]>([]);
  const [watchesLoading, setWatchesLoading] = useState(false);

  // Chain state
  const [chains, setChains] = useState<JobChain[]>([]);
  const [chainsLoading, setChainsLoading] = useState(false);
  const [showChainBuilder, setShowChainBuilder] = useState(false);

  // Load watches
  const refreshWatches = useCallback(async () => {
    setWatchesLoading(true);
    try {
      const { data, error } = await listWatches({ baseUrl: getApiBaseUrl() });
      if (error) {
        console.error("Failed to load watches:", error);
        return;
      }
      setWatches(data?.watches || []);
    } catch (err) {
      console.error("Error loading watches:", err);
    } finally {
      setWatchesLoading(false);
    }
  }, []);

  // Watch CRUD handlers
  const handleCreateWatch = useCallback(
    async (input: WatchInput) => {
      const { error } = await createWatch({
        baseUrl: getApiBaseUrl(),
        body: input,
      });
      if (error) throw error;
      await refreshWatches();
    },
    [refreshWatches],
  );

  const handleUpdateWatch = useCallback(
    async (id: string, input: WatchInput) => {
      const { error } = await updateWatch({
        baseUrl: getApiBaseUrl(),
        path: { id },
        body: input,
      });
      if (error) throw error;
      await refreshWatches();
    },
    [refreshWatches],
  );

  const handleDeleteWatch = useCallback(
    async (id: string) => {
      const { error } = await deleteWatch({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) throw error;
      await refreshWatches();
    },
    [refreshWatches],
  );

  const handleCheckWatch = useCallback(
    async (id: string): Promise<WatchCheckResult | undefined> => {
      const { data, error } = await checkWatch({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) throw error;
      await refreshWatches();
      return data;
    },
    [refreshWatches],
  );

  // Load watches on mount
  useEffect(() => {
    refreshWatches();
  }, [refreshWatches]);

  // Chain CRUD handlers
  const refreshChains = useCallback(async () => {
    setChainsLoading(true);
    try {
      const { data, error } = await listChains({ baseUrl: getApiBaseUrl() });
      if (error) throw error;
      setChains(data?.chains || []);
    } catch (err) {
      console.error("Failed to load chains:", err);
    } finally {
      setChainsLoading(false);
    }
  }, []);

  const handleCreateChain = useCallback(
    async (request: ChainCreateRequest) => {
      const { error } = await createChain({
        baseUrl: getApiBaseUrl(),
        body: request,
      });
      if (error) throw error;
      await refreshChains();
      setShowChainBuilder(false);
    },
    [refreshChains],
  );

  const handleDeleteChain = useCallback(
    async (id: string) => {
      const { error } = await deleteChain({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) throw error;
      await refreshChains();
    },
    [refreshChains],
  );

  // Load chains on mount
  useEffect(() => {
    refreshChains();
  }, [refreshChains]);

  // Batch data management
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

  // Save preset dialog state
  const [isSaveDialogOpen, setIsSaveDialogOpen] = useState(false);

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
      if (formType === "scrape" && scrapeFormRef.current) {
        await scrapeFormRef.current.submit();
      } else if (formType === "crawl" && crawlFormRef.current) {
        await crawlFormRef.current.submit();
      } else if (formType === "research" && researchFormRef.current) {
        await researchFormRef.current.submit();
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
    setJobsPage,
    setCrawlStatesPage,
  } = appData;

  // Chain submit handler (defined after refreshJobs is available)
  const handleSubmitChain = useCallback(
    async (id: string, overrides?: Record<string, unknown>) => {
      // Convert overrides to the expected format
      const formattedOverrides: { [key: string]: { [key: string]: unknown } } =
        {};
      if (overrides) {
        for (const [key, value] of Object.entries(overrides)) {
          if (typeof value === "object" && value !== null) {
            formattedOverrides[key] = value as { [key: string]: unknown };
          }
        }
      }
      const { error } = await submitChain({
        baseUrl: getApiBaseUrl(),
        path: { id },
        body: { overrides: formattedOverrides },
      });
      if (error) throw error;
      // Refresh jobs to show newly created jobs
      await refreshJobs();
    },
    [refreshJobs],
  );

  const {
    headless,
    usePlaywright,
    timeoutSeconds,
    authProfile,
    authBasic,
    headersRaw,
    cookiesRaw,
    queryRaw,
    loginUrl,
    loginUserSelector,
    loginPassSelector,
    loginSubmitSelector,
    loginUser,
    loginPass,
    extractTemplate,
    extractValidate,
    preProcessors,
    postProcessors,
    transformers,
    incremental,
    maxDepth,
    maxPages,
    webhookUrl,
    webhookEvents,
    webhookSecret,
    interceptEnabled,
    interceptURLPatterns,
    interceptResourceTypes,
    interceptCaptureRequestBody,
    interceptCaptureResponseBody,
    interceptMaxBodySize,
    setHeadless,
    setUsePlaywright,
    setTimeoutSeconds,
    setAuthProfile,
    setAuthBasic,
    setHeadersRaw,
    setCookiesRaw,
    setQueryRaw,
    setLoginUrl,
    setLoginUserSelector,
    setLoginPassSelector,
    setLoginSubmitSelector,
    setLoginUser,
    setLoginPass,
    setExtractTemplate,
    setExtractValidate,
    setPreProcessors,
    setPostProcessors,
    setTransformers,
    setIncremental,
    setMaxDepth,
    setMaxPages,
    setWebhookUrl,
    setWebhookEvents,
    setWebhookSecret,
    setInterceptEnabled,
    setInterceptURLPatterns,
    setInterceptResourceTypes,
    setInterceptCaptureRequestBody,
    setInterceptCaptureResponseBody,
    setInterceptMaxBodySize,
    applyPreset,
  } = formState;

  const {
    selectedJobId,
    resultItems,
    selectedResultIndex,
    resultSummary,
    resultConfidence,
    resultEvidence,
    resultClusters,
    resultCitations,
    rawResult,
    resultFormat,
    currentPage,
    totalResults,
    loadResults,
    setSelectedResultIndex,
    setCurrentPage,
  } = resultsState;

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

  // Batch submission handlers
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
          resultsState.loadResults("");
        }
      } catch (err) {
        console.error(String(err));
      }
    },
    [refreshJobs, selectedJobId, resultsState],
  );

  // Find active (running) job for command palette
  const activeJob = jobs.find((job) => job.status === "running");

  // Handle preset selection
  const handleSelectPreset = useCallback(
    (preset: JobPreset) => {
      // Switch to the appropriate tab
      setActiveTab(preset.jobType);

      // Apply preset config to form state
      applyPreset(preset.config);

      // Set URL/query if provided in preset
      if (preset.config.url) {
        if (preset.jobType === "scrape" && scrapeFormRef.current) {
          scrapeFormRef.current.setUrl(preset.config.url);
        } else if (preset.jobType === "crawl" && crawlFormRef.current) {
          crawlFormRef.current.setUrl(preset.config.url);
        }
      }
      if (preset.config.query && researchFormRef.current) {
        researchFormRef.current.setQuery(preset.config.query);
      }
      if (preset.config.urls && researchFormRef.current) {
        // Note: ResearchForm doesn't have setUrls, we'd need to add it
        // For now, just log it
        console.log("Would set URLs:", preset.config.urls);
      }

      // Scroll to forms section
      const formsSection = document.getElementById("forms");
      if (formsSection) {
        formsSection.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    },
    [applyPreset],
  );

  // Handle save preset
  const handleSavePreset = useCallback(() => {
    setIsSaveDialogOpen(true);
  }, []);

  // Get current config from active form
  const getCurrentConfig = useCallback(() => {
    switch (activeTab) {
      case "scrape":
        return scrapeFormRef.current?.getConfig() ?? {};
      case "crawl":
        return crawlFormRef.current?.getConfig() ?? {};
      case "research":
        return researchFormRef.current?.getConfig() ?? {};
      default:
        return {};
    }
  }, [activeTab]);

  // Handle save preset confirmation
  const handleConfirmSavePreset = useCallback(
    (name: string, description: string) => {
      const config = getCurrentConfig();
      savePreset(name, description, activeTab, config);
      setIsSaveDialogOpen(false);
    },
    [activeTab, getCurrentConfig, savePreset],
  );

  // Get current URL for preset matching
  const getCurrentUrl = useCallback(() => {
    switch (activeTab) {
      case "scrape":
        return scrapeFormRef.current?.getUrl() ?? "";
      case "crawl":
        return crawlFormRef.current?.getUrl() ?? "";
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
        headless={headless}
        usePlaywright={usePlaywright}
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
        _onDeleteJob={deleteJob}
        activeJobId={activeJob?.id}
        isMac={isMac}
        presets={presets}
        onSelectPreset={handleSelectPreset}
        onRestartTour={resetOnboarding}
      />

      <div data-tour="quickstart">
        <QuickStartPanel
          presets={presets}
          activeJobType={activeTab}
          onSelectPreset={handleSelectPreset}
          onSavePreset={handleSavePreset}
          currentUrl={getCurrentUrl()}
        />
      </div>

      <SavePresetDialog
        isOpen={isSaveDialogOpen}
        onClose={() => setIsSaveDialogOpen(false)}
        jobType={activeTab}
        currentConfig={getCurrentConfig()}
        onSave={handleConfirmSavePreset}
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

      <Suspense
        fallback={<div className="loading-placeholder">Loading forms...</div>}
      >
        <section id="forms" className="grid" data-tour="form-types">
          <ScrapeForm
            ref={scrapeFormRef}
            headless={headless}
            setHeadless={setHeadless}
            usePlaywright={usePlaywright}
            setUsePlaywright={setUsePlaywright}
            timeoutSeconds={timeoutSeconds}
            setTimeoutSeconds={setTimeoutSeconds}
            authProfile={authProfile}
            setAuthProfile={setAuthProfile}
            authBasic={authBasic}
            setAuthBasic={setAuthBasic}
            headersRaw={headersRaw}
            setHeadersRaw={setHeadersRaw}
            cookiesRaw={cookiesRaw}
            setCookiesRaw={setCookiesRaw}
            queryRaw={queryRaw}
            setQueryRaw={setQueryRaw}
            loginUrl={loginUrl}
            setLoginUrl={setLoginUrl}
            loginUserSelector={loginUserSelector}
            setLoginUserSelector={setLoginUserSelector}
            loginPassSelector={loginPassSelector}
            setLoginPassSelector={setLoginPassSelector}
            loginSubmitSelector={loginSubmitSelector}
            setLoginSubmitSelector={setLoginSubmitSelector}
            loginUser={loginUser}
            setLoginUser={setLoginUser}
            loginPass={loginPass}
            setLoginPass={setLoginPass}
            extractTemplate={extractTemplate}
            setExtractTemplate={setExtractTemplate}
            extractValidate={extractValidate}
            setExtractValidate={setExtractValidate}
            preProcessors={preProcessors}
            setPreProcessors={setPreProcessors}
            postProcessors={postProcessors}
            setPostProcessors={setPostProcessors}
            transformers={transformers}
            setTransformers={setTransformers}
            incremental={incremental}
            setIncremental={setIncremental}
            webhookUrl={webhookUrl}
            setWebhookUrl={setWebhookUrl}
            webhookEvents={webhookEvents}
            setWebhookEvents={setWebhookEvents}
            webhookSecret={webhookSecret}
            setWebhookSecret={setWebhookSecret}
            profiles={profiles}
            onSubmit={handleSubmitScrape}
            loading={loading}
            interceptEnabled={interceptEnabled}
            setInterceptEnabled={setInterceptEnabled}
            interceptURLPatterns={interceptURLPatterns}
            setInterceptURLPatterns={setInterceptURLPatterns}
            interceptResourceTypes={interceptResourceTypes}
            setInterceptResourceTypes={setInterceptResourceTypes}
            interceptCaptureRequestBody={interceptCaptureRequestBody}
            setInterceptCaptureRequestBody={setInterceptCaptureRequestBody}
            interceptCaptureResponseBody={interceptCaptureResponseBody}
            setInterceptCaptureResponseBody={setInterceptCaptureResponseBody}
            interceptMaxBodySize={interceptMaxBodySize}
            setInterceptMaxBodySize={setInterceptMaxBodySize}
          />

          <CrawlForm
            ref={crawlFormRef}
            headless={headless}
            setHeadless={setHeadless}
            usePlaywright={usePlaywright}
            setUsePlaywright={setUsePlaywright}
            timeoutSeconds={timeoutSeconds}
            setTimeoutSeconds={setTimeoutSeconds}
            authProfile={authProfile}
            setAuthProfile={setAuthProfile}
            authBasic={authBasic}
            setAuthBasic={setAuthBasic}
            headersRaw={headersRaw}
            setHeadersRaw={setHeadersRaw}
            cookiesRaw={cookiesRaw}
            setCookiesRaw={setCookiesRaw}
            queryRaw={queryRaw}
            setQueryRaw={setQueryRaw}
            loginUrl={loginUrl}
            setLoginUrl={setLoginUrl}
            loginUserSelector={loginUserSelector}
            setLoginUserSelector={setLoginUserSelector}
            loginPassSelector={loginPassSelector}
            setLoginPassSelector={setLoginPassSelector}
            loginSubmitSelector={loginSubmitSelector}
            setLoginSubmitSelector={setLoginSubmitSelector}
            loginUser={loginUser}
            setLoginUser={setLoginUser}
            loginPass={loginPass}
            setLoginPass={setLoginPass}
            extractTemplate={extractTemplate}
            setExtractTemplate={setExtractTemplate}
            extractValidate={extractValidate}
            setExtractValidate={setExtractValidate}
            preProcessors={preProcessors}
            setPreProcessors={setPreProcessors}
            postProcessors={postProcessors}
            setPostProcessors={setPostProcessors}
            transformers={transformers}
            setTransformers={setTransformers}
            incremental={incremental}
            setIncremental={setIncremental}
            webhookUrl={webhookUrl}
            setWebhookUrl={setWebhookUrl}
            webhookEvents={webhookEvents}
            setWebhookEvents={setWebhookEvents}
            webhookSecret={webhookSecret}
            setWebhookSecret={setWebhookSecret}
            profiles={profiles}
            onSubmit={handleSubmitCrawl}
            loading={loading}
            interceptEnabled={interceptEnabled}
            setInterceptEnabled={setInterceptEnabled}
            interceptURLPatterns={interceptURLPatterns}
            setInterceptURLPatterns={setInterceptURLPatterns}
            interceptResourceTypes={interceptResourceTypes}
            setInterceptResourceTypes={setInterceptResourceTypes}
            interceptCaptureRequestBody={interceptCaptureRequestBody}
            setInterceptCaptureRequestBody={setInterceptCaptureRequestBody}
            interceptCaptureResponseBody={interceptCaptureResponseBody}
            setInterceptCaptureResponseBody={setInterceptCaptureResponseBody}
            interceptMaxBodySize={interceptMaxBodySize}
            setInterceptMaxBodySize={setInterceptMaxBodySize}
          />

          <ResearchForm
            ref={researchFormRef}
            maxDepth={maxDepth}
            setMaxDepth={setMaxDepth}
            maxPages={maxPages}
            setMaxPages={setMaxPages}
            headless={headless}
            setHeadless={setHeadless}
            usePlaywright={usePlaywright}
            setUsePlaywright={setUsePlaywright}
            timeoutSeconds={timeoutSeconds}
            setTimeoutSeconds={setTimeoutSeconds}
            authProfile={authProfile}
            setAuthProfile={setAuthProfile}
            authBasic={authBasic}
            setAuthBasic={setAuthBasic}
            headersRaw={headersRaw}
            setHeadersRaw={setHeadersRaw}
            cookiesRaw={cookiesRaw}
            setCookiesRaw={setCookiesRaw}
            queryRaw={queryRaw}
            setQueryRaw={setQueryRaw}
            loginUrl={loginUrl}
            setLoginUrl={setLoginUrl}
            loginUserSelector={loginUserSelector}
            setLoginUserSelector={setLoginUserSelector}
            loginPassSelector={loginPassSelector}
            setLoginPassSelector={setLoginPassSelector}
            loginSubmitSelector={loginSubmitSelector}
            setLoginSubmitSelector={setLoginSubmitSelector}
            loginUser={loginUser}
            setLoginUser={setLoginUser}
            loginPass={loginPass}
            setLoginPass={setLoginPass}
            extractTemplate={extractTemplate}
            setExtractTemplate={setExtractTemplate}
            extractValidate={extractValidate}
            setExtractValidate={setExtractValidate}
            preProcessors={preProcessors}
            setPreProcessors={setPreProcessors}
            postProcessors={postProcessors}
            setPostProcessors={setPostProcessors}
            transformers={transformers}
            setTransformers={setTransformers}
            webhookUrl={webhookUrl}
            setWebhookUrl={setWebhookUrl}
            webhookEvents={webhookEvents}
            setWebhookEvents={setWebhookEvents}
            webhookSecret={webhookSecret}
            setWebhookSecret={setWebhookSecret}
            profiles={profiles}
            onSubmit={handleSubmitResearch}
            loading={loading}
            interceptEnabled={interceptEnabled}
            setInterceptEnabled={setInterceptEnabled}
            interceptURLPatterns={interceptURLPatterns}
            setInterceptURLPatterns={setInterceptURLPatterns}
            interceptResourceTypes={interceptResourceTypes}
            setInterceptResourceTypes={setInterceptResourceTypes}
            interceptCaptureRequestBody={interceptCaptureRequestBody}
            setInterceptCaptureRequestBody={setInterceptCaptureRequestBody}
            interceptCaptureResponseBody={interceptCaptureResponseBody}
            setInterceptCaptureResponseBody={setInterceptCaptureResponseBody}
            interceptMaxBodySize={interceptMaxBodySize}
            setInterceptMaxBodySize={setInterceptMaxBodySize}
          />

          <BatchForm
            activeTab={batchTab}
            setActiveTab={setBatchTab}
            headless={headless}
            setHeadless={setHeadless}
            usePlaywright={usePlaywright}
            setUsePlaywright={setUsePlaywright}
            timeoutSeconds={timeoutSeconds}
            setTimeoutSeconds={setTimeoutSeconds}
            authProfile={authProfile}
            setAuthProfile={setAuthProfile}
            authBasic={authBasic}
            setAuthBasic={setAuthBasic}
            headersRaw={headersRaw}
            setHeadersRaw={setHeadersRaw}
            cookiesRaw={cookiesRaw}
            setCookiesRaw={setCookiesRaw}
            queryRaw={queryRaw}
            setQueryRaw={setQueryRaw}
            loginUrl={loginUrl}
            setLoginUrl={setLoginUrl}
            loginUserSelector={loginUserSelector}
            setLoginUserSelector={setLoginUserSelector}
            loginPassSelector={loginPassSelector}
            setLoginPassSelector={setLoginPassSelector}
            loginSubmitSelector={loginSubmitSelector}
            setLoginSubmitSelector={setLoginSubmitSelector}
            loginUser={loginUser}
            setLoginUser={setLoginUser}
            loginPass={loginPass}
            setLoginPass={setLoginPass}
            extractTemplate={extractTemplate}
            setExtractTemplate={setExtractTemplate}
            extractValidate={extractValidate}
            setExtractValidate={setExtractValidate}
            preProcessors={preProcessors}
            setPreProcessors={setPreProcessors}
            postProcessors={postProcessors}
            setPostProcessors={setPostProcessors}
            transformers={transformers}
            setTransformers={setTransformers}
            incremental={incremental}
            setIncremental={setIncremental}
            webhookUrl={webhookUrl}
            setWebhookUrl={setWebhookUrl}
            webhookEvents={webhookEvents}
            setWebhookEvents={setWebhookEvents}
            webhookSecret={webhookSecret}
            setWebhookSecret={setWebhookSecret}
            profiles={profiles}
            urlsInput={batchUrls}
            setUrlsInput={setBatchUrls}
            maxDepth={maxDepth}
            setMaxDepth={setMaxDepth}
            maxPages={maxPages}
            setMaxPages={setMaxPages}
            query={batchQuery}
            setQuery={setBatchQuery}
            onSubmitScrape={handleSubmitBatchScrape}
            onSubmitCrawl={handleSubmitBatchCrawl}
            onSubmitResearch={handleSubmitBatchResearch}
            loading={loading}
          />
        </section>
      </Suspense>

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

      <section id="chains">
        {showChainBuilder ? (
          <Suspense
            fallback={
              <div className="loading-placeholder">
                Loading chain builder...
              </div>
            }
          >
            <ChainBuilder
              onCreate={handleCreateChain}
              onCancel={() => setShowChainBuilder(false)}
            />
          </Suspense>
        ) : (
          <ChainList
            chains={chains}
            onRefresh={refreshChains}
            onDelete={handleDeleteChain}
            onSubmit={handleSubmitChain}
            loading={chainsLoading}
            onCreateClick={() => setShowChainBuilder(true)}
          />
        )}
      </section>

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

      <section id="results">
        <Suspense
          fallback={
            <div className="loading-placeholder">
              Loading results explorer...
            </div>
          }
        >
          <ResultsExplorer
            jobId={selectedJobId}
            resultItems={resultItems}
            selectedResultIndex={selectedResultIndex}
            setSelectedResultIndex={setSelectedResultIndex}
            resultSummary={resultSummary}
            resultConfidence={resultConfidence}
            resultEvidence={resultEvidence}
            resultClusters={resultClusters}
            resultCitations={resultCitations}
            rawResult={rawResult}
            resultFormat={resultFormat}
            currentPage={currentPage}
            totalResults={totalResults}
            resultsPerPage={100}
            onLoadPage={setCurrentPage}
            availableJobs={jobs}
          />
        </Suspense>
      </section>

      <section id="watches">
        <Suspense
          fallback={
            <div className="loading-placeholder">Loading watch manager...</div>
          }
        >
          <WatchManager
            watches={watches}
            onRefresh={refreshWatches}
            onCreate={handleCreateWatch}
            onUpdate={handleUpdateWatch}
            onDelete={handleDeleteWatch}
            onCheck={handleCheckWatch}
            loading={watchesLoading}
          />
        </Suspense>
      </section>

      <InfoSections
        profiles={profiles}
        schedules={schedules}
        templates={templates}
        crawlStates={crawlStates}
        crawlStatesPage={crawlStatesPage}
        crawlStatesTotal={crawlStatesTotal}
        crawlStatesPerPage={100}
        onCrawlStatesPageChange={setCrawlStatesPage}
      />

      <div className="footer">
        Spartan Scraper — build once, deploy everywhere.
      </div>
    </div>
  );
}
