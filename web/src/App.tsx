/**
 * Purpose: Provide the route-based application shell for the local-first operator workbench.
 * Responsibilities: Own global navigation chrome, coordinate shared data hooks, and delegate major route workflows to route-local containers.
 * Scope: Application shell and cross-route coordination only.
 * Usage: Rendered once from `main.tsx` as the root React application.
 * Invariants/Assumptions: Supported routes are `/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, `/automation/:section`, `/settings`, and `/settings/:section`, and route-local containers own route framing once selected.
 */

import { useCallback } from "react";

import { CommandPalette } from "./components/CommandPalette";
import { KeyboardShortcutsHelp } from "./components/KeyboardShortcutsHelp";
import { OnboardingFlow } from "./components/OnboardingFlow";
import { OnboardingNudge } from "./components/OnboardingNudge";
import { AIAssistantProvider, useAIAssistant } from "./components/ai-assistant";
import {
  AutomationRoute,
  JobDetailRoute,
  JobsRoute,
  NewJobRoute,
  SettingsRoute,
  SetupRequiredRoute,
  TemplatesRoute,
} from "./components/routes/AppRoutes";
import { DEFAULT_AUTOMATION_SECTION } from "./components/automation/automationSections";
import { AppTopBar } from "./components/shell/ShellPrimitives";
import {
  DEFAULT_SETTINGS_SECTION,
  getSettingsPath,
} from "./components/settings/settingsSections";
import { ShortcutHint } from "./components/ShortcutHint";
import { SystemStatusPanel } from "./components/SystemStatusPanel";
import { ThemeToggle } from "./components/ThemeToggle";
import { useToast } from "./components/toast";
import { TutorialTooltip } from "./components/TutorialTooltip";
import { useAppData } from "./hooks/useAppData";
import { useFormState } from "./hooks/useFormState";
import { useKeyboard } from "./hooks/useKeyboard";
import { useOnboarding } from "./hooks/useOnboarding";
import { usePresets } from "./hooks/usePresets";
import { useResultsState } from "./hooks/useResultsState";
import { useTheme } from "./hooks/useTheme";
import { type RouteKind, useAppShellRouting } from "./hooks/useAppShellRouting";
import { useJobSubmissionActions } from "./hooks/useJobSubmissionActions";
import { useShellShortcuts } from "./hooks/useShellShortcuts";
import { getApiBaseUrl } from "./lib/api-config";
import { saveJobsViewState } from "./lib/job-monitoring";
import type { OnboardingRouteKey } from "./lib/onboarding";

interface NavItem {
  kind: Exclude<RouteKind, "job-detail">;
  label: string;
  path: string;
  description: string;
}

const NAV_ITEMS = [
  {
    kind: "jobs",
    label: "Jobs",
    path: "/jobs",
    description: "Recent jobs, live queue state, and result drill-down.",
  },
  {
    kind: "new-job",
    label: "New Job",
    path: "/jobs/new",
    description: "Submit scrape, crawl, or research work with saved presets.",
  },
  {
    kind: "templates",
    label: "Templates",
    path: "/templates",
    description:
      "Manage extraction templates, with optional AI-assisted generation when that capability is enabled.",
  },
  {
    kind: "automation",
    label: "Automation",
    path: "/automation/batches",
    description:
      "Batches, chains, watches, export schedules, and webhook delivery history.",
  },
  {
    kind: "settings",
    label: "Settings",
    path: getSettingsPath(DEFAULT_SETTINGS_SECTION),
    description:
      "Saved auth, reusable runtime tools, and optional maintenance controls.",
  },
] as const satisfies readonly NavItem[];

function ErrorBanner({ message }: { message: string | null }) {
  if (!message) {
    return null;
  }

  return (
    <section className="panel">
      <div className="error">{message}</div>
    </section>
  );
}

export function App() {
  return (
    <AIAssistantProvider>
      <AppShell />
    </AIAssistantProvider>
  );
}

function AppShell() {
  const aiAssistant = useAIAssistant();
  const toast = useToast();
  const appData = useAppData();
  const formState = useFormState();
  const resultsState = useResultsState();
  const { theme, resolvedTheme, setTheme, toggleTheme } = useTheme();
  const { presets, savePreset } = usePresets();
  const { route, navigate, routePromotionSeed, clearPromotionSeed } =
    useAppShellRouting();
  const {
    isCommandPaletteOpen,
    isHelpOpen,
    openCommandPalette,
    closeCommandPalette,
    openHelp,
    closeHelp,
    shortcuts,
    isMac,
  } = useKeyboard();

  const {
    jobs,
    failedJobs,
    jobStatusFilter,
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
    health,
    setupRequired,
    detailJob,
    detailJobLoading,
    detailJobError,
    refreshHealth,
    refreshJobs,
    refreshTemplates,
    refreshJobDetail,
    clearJobDetail,
    setJobsPage,
    setCrawlStatesPage,
    setJobStatusFilter,
  } = appData;

  const {
    shouldShowFirstRunHint,
    isTourActive,
    currentStep,
    startOnboarding,
    skipOnboarding,
    resetOnboarding,
    goToStep,
    finishOnboarding,
    dismissFirstRunHint,
  } = useOnboarding({ hasStartedWork: jobsTotal > 0 });

  const { selectedJobId } = resultsState;
  const routeKey = route.kind as OnboardingRouteKey;
  const showGlobalFirstRunPrompt =
    shouldShowFirstRunHint && route.kind === "jobs";

  const {
    activeTab,
    setActiveTab,
    jobSubmissionRef,
    handleSelectPreset,
    handleSubmitForm,
    handleSubmitScrape,
    handleSubmitCrawl,
    handleSubmitResearch,
    cancelJob,
    deleteJob,
  } = useJobSubmissionActions({
    navigate,
    routeKind: route.kind,
    refreshJobs,
    selectedJobId,
    toast,
    getApiBaseUrl,
  });

  const persistJobsViewState = useCallback(() => {
    if (typeof window === "undefined") {
      return;
    }

    saveJobsViewState({
      statusFilter: jobStatusFilter,
      currentPage: jobsPage,
      scrollY: window.scrollY,
    });
  }, [jobStatusFilter, jobsPage]);

  const {
    openJobAssistant,
    openTemplateAssistant,
    handleTourRouteChange,
    routeHelpProps,
  } = useShellShortcuts({
    navigate,
    persistJobsViewState,
    routeKind: route.kind,
    selectedJobId,
    jobs,
    activeTab,
    extractTemplate: formState.extractTemplate,
    jobSubmissionRef,
    openAssistant: aiAssistant.open,
    shortcuts,
    isMac,
    openCommandPalette,
    openHelp,
    resetOnboarding,
  });

  const handleViewResults = useCallback(
    (jobId: string, _format: string, _page: number) => {
      if (route.kind === "jobs") {
        persistJobsViewState();
      }

      navigate(`/jobs/${jobId}`);
    },
    [navigate, persistJobsViewState, route.kind],
  );

  const handlePaletteNavigate = useCallback(
    (path: string) => {
      if (route.kind === "jobs" && path.startsWith("/jobs/")) {
        persistJobsViewState();
      }
      navigate(path);
    },
    [navigate, persistJobsViewState, route.kind],
  );

  const activeJob = jobs.find((job) => job.status === "running");
  const shellUtilities = (
    <>
      <button
        type="button"
        className="secondary app-toolbar-shortcut"
        data-tour="command-palette"
        onClick={openCommandPalette}
      >
        Command Palette
        <ShortcutHint shortcut={shortcuts.commandPalette} isMac={isMac} />
      </button>
      <button
        type="button"
        className="secondary app-toolbar-shortcut"
        data-tour="keyboard-help"
        onClick={openHelp}
      >
        Shortcuts
        <ShortcutHint shortcut={shortcuts.help} isMac={isMac} />
      </button>
      <ThemeToggle
        theme={theme}
        resolvedTheme={resolvedTheme}
        onThemeChange={setTheme}
        onToggle={toggleTheme}
      />
    </>
  );

  const activeRouteForNav = route.kind === "job-detail" ? "jobs" : route.kind;

  return (
    <div className={`app app--${route.kind}`}>
      <AppTopBar
        activeRoute={activeRouteForNav}
        navItems={NAV_ITEMS}
        onNavigate={navigate}
        globalAction={
          !setupRequired && route.kind !== "new-job" ? (
            <button type="button" onClick={() => navigate("/jobs/new")}>
              Create Job
            </button>
          ) : null
        }
        utilities={shellUtilities}
      />

      {!setupRequired ? (
        <OnboardingNudge
          isVisible={showGlobalFirstRunPrompt}
          isMac={isMac}
          onStartTour={startOnboarding}
          onOpenHelp={openHelp}
          onDismiss={dismissFirstRunHint}
          onCreateJob={() => navigate("/jobs/new")}
          health={health}
          hasTemplates={templates.length > 0}
        />
      ) : null}

      <TutorialTooltip
        target='[data-tour="command-palette"]'
        title="Jump anywhere fast"
        content="Use the command palette to navigate routes, submit work, select presets, and restart onboarding."
        position="bottom"
        showBeacon={showGlobalFirstRunPrompt}
        showDelay={500}
      />

      <TutorialTooltip
        target='[data-tour="keyboard-help"]'
        title="Shortcut help is visible now"
        content="Open this anytime to see global shortcuts and a route-specific section for what matters on the current screen."
        position="bottom"
        showBeacon={showGlobalFirstRunPrompt}
        showDelay={500}
      />

      {isCommandPaletteOpen ? (
        <CommandPalette
          isOpen={isCommandPaletteOpen}
          onClose={closeCommandPalette}
          jobs={jobs}
          onNavigateToPath={handlePaletteNavigate}
          onSubmitForm={handleSubmitForm}
          onCancelJob={cancelJob}
          activeJobId={activeJob?.id}
          isMac={isMac}
          presets={presets}
          onSelectPreset={handleSelectPreset}
          onRestartTour={resetOnboarding}
        />
      ) : null}

      <KeyboardShortcutsHelp
        isOpen={isHelpOpen}
        onClose={closeHelp}
        shortcuts={shortcuts}
        isMac={isMac}
        routeKind={routeKey}
      />

      <OnboardingFlow
        isRunning={isTourActive}
        currentStep={currentStep}
        currentRoute={routeKey}
        onComplete={finishOnboarding}
        onSkip={skipOnboarding}
        onStepChange={goToStep}
        onRouteChange={handleTourRouteChange}
      />

      <ErrorBanner message={error} />

      <SystemStatusPanel
        health={health}
        onNavigate={navigate}
        onRefresh={refreshHealth}
      />

      {setupRequired ? <SetupRequiredRoute health={health} /> : null}

      {!setupRequired && route.kind === "jobs" ? (
        <JobsRoute
          jobs={jobs}
          failedJobs={failedJobs}
          error={error}
          loading={loading}
          statusFilter={jobStatusFilter}
          currentPage={jobsPage}
          totalJobs={jobsTotal}
          connectionState={connectionState}
          managerStatus={managerStatus}
          routeHelp={routeHelpProps}
          onStatusFilterChange={setJobStatusFilter}
          onViewResults={handleViewResults}
          onCancel={cancelJob}
          onDelete={deleteJob}
          onRefresh={refreshJobs}
          onCreateJob={() => navigate("/jobs/new")}
          onPageChange={setJobsPage}
        />
      ) : null}

      {!setupRequired && route.kind === "job-detail" && route.jobId ? (
        <JobDetailRoute
          jobId={route.jobId}
          jobs={jobs}
          routeDetailJob={detailJob}
          detailJobLoading={detailJobLoading}
          detailJobError={detailJobError}
          resultsState={resultsState}
          connectionState={connectionState}
          aiStatus={health?.components?.ai ?? null}
          routeHelp={routeHelpProps}
          refreshJobDetail={refreshJobDetail}
          clearJobDetail={clearJobDetail}
          navigate={navigate}
        />
      ) : null}

      {!setupRequired && route.kind === "new-job" ? (
        <NewJobRoute
          activeTab={activeTab}
          formState={formState}
          loading={loading}
          profiles={profiles}
          presets={presets}
          jobsTotal={jobsTotal}
          jobStatusFilter={jobStatusFilter}
          aiStatus={health?.components?.ai ?? null}
          routeHelp={routeHelpProps}
          jobSubmissionRef={jobSubmissionRef}
          savePreset={savePreset}
          setActiveTab={setActiveTab}
          onSubmitScrape={handleSubmitScrape}
          onSubmitCrawl={handleSubmitCrawl}
          onSubmitResearch={handleSubmitResearch}
          onSelectPreset={handleSelectPreset}
          onOpenAssistant={openJobAssistant}
          onOpenTemplateAssistant={openTemplateAssistant}
        />
      ) : null}

      {!setupRequired && route.kind === "templates" ? (
        <TemplatesRoute
          templateNames={templates}
          promotionSeed={routePromotionSeed}
          aiStatus={health?.components?.ai ?? null}
          routeHelp={routeHelpProps}
          onClearPromotionSeed={clearPromotionSeed}
          onOpenSourceJob={(jobId) => navigate(`/jobs/${jobId}`)}
          onTemplatesChanged={() => {
            void refreshTemplates();
          }}
        />
      ) : null}

      {!setupRequired && route.kind === "automation" ? (
        <AutomationRoute
          section={route.automationSection ?? DEFAULT_AUTOMATION_SECTION}
          promotionSeed={routePromotionSeed}
          formState={formState}
          profiles={profiles}
          loading={loading}
          aiStatus={health?.components?.ai ?? null}
          routeHelp={routeHelpProps}
          onClearPromotionSeed={clearPromotionSeed}
          navigate={navigate}
          onRefreshJobs={refreshJobs}
        />
      ) : null}

      {!setupRequired && route.kind === "settings" ? (
        <SettingsRoute
          section={route.settingsSection ?? DEFAULT_SETTINGS_SECTION}
          path={route.path}
          health={health}
          profiles={profiles}
          schedules={schedules}
          crawlStates={crawlStates}
          crawlStatesPage={crawlStatesPage}
          crawlStatesTotal={crawlStatesTotal}
          jobsTotal={jobsTotal}
          routeHelp={routeHelpProps}
          onNavigate={navigate}
          onRefreshHealth={refreshHealth}
          onCrawlStatesPageChange={setCrawlStatesPage}
        />
      ) : null}
    </div>
  );
}
