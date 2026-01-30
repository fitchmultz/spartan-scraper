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

import { useCallback, useRef, useEffect, useState } from "react";
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
import { ResultsExplorer } from "./components/ResultsExplorer";
import { InfoSections } from "./components/InfoSections";
import { ScrapeForm, type ScrapeFormRef } from "./components/ScrapeForm";
import { CrawlForm, type CrawlFormRef } from "./components/CrawlForm";
import { ResearchForm, type ResearchFormRef } from "./components/ResearchForm";
import { CommandPalette } from "./components/CommandPalette";
import { KeyboardShortcutsHelp } from "./components/KeyboardShortcutsHelp";
import { QuickStartPanel } from "./components/QuickStartPanel";
import { SavePresetDialog } from "./components/SavePresetDialog";
import { useKeyboard } from "./hooks/useKeyboard";
import { useAppData } from "./hooks/useAppData";
import { useFormState } from "./hooks/useFormState";
import { useResultsState } from "./hooks/useResultsState";
import { useTheme } from "./hooks/useTheme";
import { usePresets } from "./hooks/usePresets";
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
      />

      <QuickStartPanel
        presets={presets}
        activeJobType={activeTab}
        onSelectPreset={handleSelectPreset}
        onSavePreset={handleSavePreset}
        currentUrl={getCurrentUrl()}
      />

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

      <section id="forms" className="grid">
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
          profiles={profiles}
          onSubmit={handleSubmitScrape}
          loading={loading}
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
          profiles={profiles}
          onSubmit={handleSubmitCrawl}
          loading={loading}
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
          profiles={profiles}
          onSubmit={handleSubmitResearch}
          loading={loading}
        />
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
