/**
 * Spartan Scraper Web UI - Route-aware application shell.
 *
 * Purpose:
 * - Provide the route-based application shell for the local-first operator workbench.
 *
 * Responsibilities:
 * - Route between jobs, templates, automation, and settings views.
 * - Wire shared data hooks, job submission, and result loading.
 * - Keep route-specific surfaces above the fold instead of repeating a shared
 *   marketing-style stack on every route.
 *
 * Scope:
 * - Application shell and cross-page coordination only.
 *
 * Usage:
 * - Rendered once from `main.tsx` as the root React application.
 *
 * Invariants/Assumptions:
 * - Supported routes are `/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`,
 *   `/automation`, `/automation/:section`, and `/settings`.
 * - Results detail routes load job results directly from the canonical REST API.
 * - Deleted feature areas stay out of navigation and render tree entirely.
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import {
  deleteV1JobsById,
  postV1Crawl,
  postV1Research,
  postV1Scrape,
  type CrawlRequest,
  type ResearchRequest,
  type ScrapeRequest,
} from "./api";
import { InfoSections } from "./components/InfoSections";
import { ActionEmptyState } from "./components/ActionEmptyState";
import { SettingsOverviewPanel } from "./components/SettingsOverviewPanel";
import { CommandPalette } from "./components/CommandPalette";
import { KeyboardShortcutsHelp } from "./components/KeyboardShortcutsHelp";
import { OnboardingFlow } from "./components/OnboardingFlow";
import { OnboardingNudge } from "./components/OnboardingNudge";
import { RouteHelpPanel } from "./components/RouteHelpPanel";
import { SystemStatusPanel } from "./components/SystemStatusPanel";
import { AutomationLayout } from "./components/automation/AutomationLayout";
import { AutomationSubnav } from "./components/automation/AutomationSubnav";
import {
  DEFAULT_AUTOMATION_SECTION,
  getAutomationPath,
  getAutomationSectionFromHash,
  getAutomationSectionFromPath,
  type AutomationSection,
} from "./components/automation/automationSections";
import { WatchContainer } from "./components/watches/WatchContainer";
import { ExportScheduleContainer } from "./components/export-schedules/ExportScheduleContainer";
import { WebhookDeliveryContainer } from "./components/webhooks/WebhookDeliveryContainer";
import { RetentionStatusPanel } from "./components/RetentionStatusPanel";
import { ProxyPoolStatusPanel } from "./components/ProxyPoolStatusPanel";
import { ChainContainer } from "./components/chains/ChainContainer";
import { BatchContainer } from "./components/batches/BatchContainer";
import { AIAssistantProvider, useAIAssistant } from "./components/ai-assistant";
import { useToast } from "./components/toast";
import { TemplateManager } from "./components/templates/TemplateManager";
import { PresetContainer } from "./components/presets/PresetContainer";
import {
  JobSubmissionContainer,
  type JobSubmissionContainerRef,
} from "./components/jobs/JobSubmissionContainer";
import { JobMonitoringDashboard } from "./components/jobs/JobMonitoringDashboard";
import { ResultsContainer } from "./components/results/ResultsContainer";
import { RenderProfileEditor } from "./components/render-profiles";
import { PipelineJSEditor } from "./components/pipeline-js/PipelineJSEditor";
import {
  AppTopBar,
  RouteHeader,
  RouteSignals,
  type RouteSignal,
} from "./components/shell/ShellPrimitives";
import { ThemeToggle } from "./components/ThemeToggle";
import { ShortcutHint } from "./components/ShortcutHint";
import { TutorialTooltip } from "./components/TutorialTooltip";
import {
  SETTINGS_SECTION_META,
  SETTINGS_SECTION_ORDER,
  SettingsSubnav,
  type SettingsSectionId,
} from "./components/settings/SettingsSubnav";
import { useKeyboard } from "./hooks/useKeyboard";
import { useAppData } from "./hooks/useAppData";
import { useFormState } from "./hooks/useFormState";
import { useResultsState } from "./hooks/useResultsState";
import { useTheme } from "./hooks/useTheme";
import { usePresets } from "./hooks/usePresets";
import { useOnboarding } from "./hooks/useOnboarding";
import {
  submitCrawlJob,
  submitResearchJob,
  submitScrapeJob,
} from "./lib/job-actions";
import { getApiBaseUrl } from "./lib/api-config";
import { getApiErrorMessage } from "./lib/api-errors";
import { saveJobsViewState } from "./lib/job-monitoring";
import type { OnboardingRouteKey, RouteHelpAction } from "./lib/onboarding";
import { shouldShowSettingsOverviewPanel } from "./lib/settings-overview";
import type { JobPreset, JobType } from "./types/presets";
import type {
  ExportSchedulePromotionSeed,
  PromotionDestination,
  PromotionSeed,
  TemplatePromotionSeed,
  WatchPromotionSeed,
} from "./types/promotion";
import {
  buildExportSchedulePromotionSeed,
  buildTemplatePromotionSeed,
  buildWatchPromotionSeed,
} from "./lib/promotion";

export type RouteKind =
  | "jobs"
  | "new-job"
  | "job-detail"
  | "templates"
  | "automation"
  | "settings";

export interface AppRoute {
  kind: RouteKind;
  path: string;
  jobId?: string;
  automationSection?: AutomationSection;
}

interface NavItem {
  kind: Exclude<RouteKind, "job-detail">;
  label: string;
  path: string;
  description: string;
}

interface RouteMeta {
  title: string;
  description?: string;
}

interface AppNavigationState {
  promotionSeed?: PromotionSeed | null;
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
    path: "/settings",
    description:
      "Saved auth, reusable runtime tools, and optional maintenance controls.",
  },
] as const satisfies readonly NavItem[];

const SETTINGS_SECTION_VIEWPORT_ANCHOR_PX = 180;

function getSettingsSectionInView(
  sections: Array<{
    id: SettingsSectionId;
    element: HTMLElement;
  }>,
): SettingsSectionId {
  const sectionRects = sections.map((section) => ({
    ...section,
    rect: section.element.getBoundingClientRect(),
  }));

  const hasMeasuredLayout = sectionRects.some(
    ({ rect }) => rect.top !== 0 || rect.bottom !== 0 || rect.height !== 0,
  );

  if (!hasMeasuredLayout) {
    return SETTINGS_SECTION_ORDER[0];
  }

  let activeSection = SETTINGS_SECTION_ORDER[0];

  for (const section of sectionRects) {
    if (section.rect.top <= SETTINGS_SECTION_VIEWPORT_ANCHOR_PX) {
      activeSection = section.id;
      continue;
    }

    break;
  }

  return activeSection;
}

export function normalizePath(pathname: string): string {
  if (!pathname || pathname === "/") {
    return "/jobs";
  }

  const trimmed = pathname.replace(/\/+$/, "");
  return trimmed === "" ? "/jobs" : trimmed;
}

export function parseRoute(pathname: string): AppRoute {
  const path = normalizePath(pathname);
  if (path === "/jobs") {
    return { kind: "jobs", path };
  }
  if (path === "/jobs/new") {
    return { kind: "new-job", path };
  }
  if (path.startsWith("/jobs/")) {
    const jobId = decodeURIComponent(path.slice("/jobs/".length));
    return { kind: "job-detail", path, jobId };
  }
  if (path === "/templates") {
    return { kind: "templates", path };
  }
  if (path === "/automation" || path.startsWith("/automation/")) {
    return {
      kind: "automation",
      path,
      automationSection:
        getAutomationSectionFromPath(path) ?? DEFAULT_AUTOMATION_SECTION,
    };
  }
  if (path === "/settings") {
    return { kind: "settings", path };
  }
  return { kind: "jobs", path: "/jobs" };
}

function formatShortJobId(id: string): string {
  if (id.length <= 14) {
    return id;
  }

  return `${id.slice(0, 8)}…${id.slice(-4)}`;
}

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
  const [pathname, setPathname] = useState(() =>
    normalizePath(window.location.pathname),
  );
  const [navigationState, setNavigationState] = useState<AppNavigationState>(
    () => (window.history.state as AppNavigationState | null) ?? {},
  );
  const [activeTab, setActiveTab] = useState<JobType>("scrape");
  const [pendingPreset, setPendingPreset] = useState<JobPreset | null>(null);
  const [pendingSubmission, setPendingSubmission] = useState<JobType | null>(
    null,
  );
  const jobSubmissionRef = useRef<JobSubmissionContainerRef>(null);
  const previousRouteKeyRef = useRef<OnboardingRouteKey | null>(null);

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
    detailJob: routeDetailJob,
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
    markRouteVisited,
    visitedRoutes,
  } = useOnboarding({ hasStartedWork: jobsTotal > 0 });

  const { selectedJobId, loadResults } = resultsState;

  const route = useMemo(() => parseRoute(pathname), [pathname]);
  const routeKey = route.kind as OnboardingRouteKey;
  const showGlobalFirstRunPrompt =
    shouldShowFirstRunHint && route.kind === "jobs";
  const [routeHelpDefaultExpanded, setRouteHelpDefaultExpanded] =
    useState(false);
  const [renderProfileCount, setRenderProfileCount] = useState<number | null>(
    null,
  );
  const [pipelineScriptCount, setPipelineScriptCount] = useState<number | null>(
    null,
  );
  const [activeSettingsSection, setActiveSettingsSection] =
    useState<SettingsSectionId>("authoring");

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

  const navigate = useCallback(
    (path: string, state?: AppNavigationState | null) => {
      const nextPath = normalizePath(path);
      const nextState = state ?? {};
      if (
        nextPath === pathname &&
        JSON.stringify(window.history.state ?? {}) === JSON.stringify(nextState)
      ) {
        return;
      }
      window.history.pushState(nextState, "", nextPath);
      setPathname(nextPath);
      setNavigationState(nextState);
    },
    [pathname],
  );

  useEffect(() => {
    const handlePopState = () => {
      setPathname(normalizePath(window.location.pathname));
      setNavigationState(
        (window.history.state as AppNavigationState | null) ?? {},
      );
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  useEffect(() => {
    if (route.kind !== "automation") {
      return;
    }

    const nextSection =
      pathname === "/automation"
        ? (getAutomationSectionFromHash(window.location.hash) ??
          route.automationSection ??
          DEFAULT_AUTOMATION_SECTION)
        : (route.automationSection ?? DEFAULT_AUTOMATION_SECTION);
    const canonicalPath = getAutomationPath(nextSection);

    if (pathname === canonicalPath) {
      return;
    }

    window.history.replaceState(window.history.state ?? {}, "", canonicalPath);
    setPathname(canonicalPath);
  }, [pathname, route.automationSection, route.kind]);

  useEffect(() => {
    if (route.kind === "job-detail" && route.jobId) {
      void loadResults(route.jobId);
      void refreshJobDetail(route.jobId);
      return;
    }

    clearJobDetail();
  }, [clearJobDetail, loadResults, refreshJobDetail, route]);

  useEffect(() => {
    if (previousRouteKeyRef.current === routeKey) {
      return;
    }

    const firstVisit = !visitedRoutes.includes(routeKey);
    setRouteHelpDefaultExpanded(firstVisit);
    if (firstVisit) {
      markRouteVisited(routeKey);
    }
    previousRouteKeyRef.current = routeKey;
  }, [markRouteVisited, routeKey, visitedRoutes]);

  const handleNavigate = useCallback(
    (view: "jobs" | "results" | "forms") => {
      if (view === "forms") {
        navigate("/jobs/new");
        return;
      }

      if (view === "results" && selectedJobId) {
        if (route.kind === "jobs") {
          persistJobsViewState();
        }
        navigate(`/jobs/${selectedJobId}`);
        return;
      }

      navigate("/jobs");
    },
    [navigate, persistJobsViewState, route.kind, selectedJobId],
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
      const toastId = toast.show({
        tone: "loading",
        title: "Submitting scrape job",
        description:
          "Queueing your scrape request and refreshing the Jobs view.",
      });

      const result = await submitScrapeJob(postV1Scrape, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });

      if (result.status === "error") {
        toast.update(toastId, {
          tone: "error",
          title: "Scrape job failed",
          description: result.message,
        });
        return;
      }

      jobSubmissionRef.current?.clearDraft("scrape");
      toast.update(toastId, {
        tone: "success",
        title: "Scrape job queued",
        description: "The new run is now visible from Jobs.",
      });
      navigate("/jobs");
    },
    [navigate, refreshJobs, toast],
  );

  const handleSubmitCrawl = useCallback(
    async (request: CrawlRequest) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Submitting crawl job",
        description:
          "Queueing your crawl request and refreshing the Jobs view.",
      });

      const result = await submitCrawlJob(postV1Crawl, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });

      if (result.status === "error") {
        toast.update(toastId, {
          tone: "error",
          title: "Crawl job failed",
          description: result.message,
        });
        return;
      }

      jobSubmissionRef.current?.clearDraft("crawl");
      toast.update(toastId, {
        tone: "success",
        title: "Crawl job queued",
        description: "The crawl is now visible from Jobs.",
      });
      navigate("/jobs");
    },
    [navigate, refreshJobs, toast],
  );

  const handleSubmitResearch = useCallback(
    async (request: ResearchRequest) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Submitting research job",
        description:
          "Queueing your research request and refreshing the Jobs view.",
      });

      const result = await submitResearchJob(postV1Research, {
        request,
        setLoading: () => {},
        setError: () => {},
        refreshJobs,
        getApiBaseUrl,
      });

      if (result.status === "error") {
        toast.update(toastId, {
          tone: "error",
          title: "Research job failed",
          description: result.message,
        });
        return;
      }

      jobSubmissionRef.current?.clearDraft("research");
      toast.update(toastId, {
        tone: "success",
        title: "Research job queued",
        description: "The research run is now visible from Jobs.",
      });
      navigate("/jobs");
    },
    [navigate, refreshJobs, toast],
  );

  const cancelJob = useCallback(
    async (jobId: string) => {
      const toastId = toast.show({
        tone: "loading",
        title: `Canceling job ${formatShortJobId(jobId)}`,
        description: "Requesting a graceful stop for the active run.",
      });

      try {
        const { error: apiError } = await deleteV1JobsById({
          baseUrl: getApiBaseUrl(),
          path: { id: jobId },
        });
        if (apiError) {
          toast.update(toastId, {
            tone: "error",
            title: "Failed to cancel job",
            description: getApiErrorMessage(
              apiError,
              "Unable to stop the selected job.",
            ),
          });
          return;
        }
        await refreshJobs();
        toast.update(toastId, {
          tone: "success",
          title: "Job canceled",
          description: `Job ${formatShortJobId(jobId)} is no longer running.`,
        });
      } catch (err) {
        toast.update(toastId, {
          tone: "error",
          title: "Failed to cancel job",
          description: getApiErrorMessage(
            err,
            "Unable to stop the selected job.",
          ),
        });
      }
    },
    [refreshJobs, toast],
  );

  const deleteJob = useCallback(
    async (jobId: string) => {
      const confirmed = await toast.confirm({
        title: "Delete this job permanently?",
        description:
          "This removes the saved run and its local artifacts. This action cannot be undone.",
        confirmLabel: "Delete job",
        cancelLabel: "Keep job",
        tone: "error",
      });
      if (!confirmed) {
        return;
      }

      const toastId = toast.show({
        tone: "loading",
        title: `Deleting job ${formatShortJobId(jobId)}`,
        description: "Removing the saved run from local storage.",
      });

      try {
        const { error: apiError } = await deleteV1JobsById({
          baseUrl: getApiBaseUrl(),
          path: { id: jobId },
          query: { force: true },
        });
        if (apiError) {
          toast.update(toastId, {
            tone: "error",
            title: "Failed to delete job",
            description: getApiErrorMessage(
              apiError,
              "Unable to delete the selected job.",
            ),
          });
          return;
        }
        await refreshJobs();
        if (selectedJobId === jobId) {
          navigate("/jobs");
        }
        toast.update(toastId, {
          tone: "success",
          title: "Job deleted",
          description: `Job ${formatShortJobId(jobId)} has been removed.`,
        });
      } catch (err) {
        toast.update(toastId, {
          tone: "error",
          title: "Failed to delete job",
          description: getApiErrorMessage(
            err,
            "Unable to delete the selected job.",
          ),
        });
      }
    },
    [navigate, refreshJobs, selectedJobId, toast],
  );

  const handleViewResults = useCallback(
    (jobId: string, _format: string, _page: number) => {
      if (route.kind === "jobs") {
        persistJobsViewState();
      }

      navigate(`/jobs/${jobId}`);
    },
    [navigate, persistJobsViewState, route.kind],
  );

  const activeJob = jobs.find((job) => job.status === "running");

  const handleSelectPreset = useCallback(
    (preset: JobPreset) => {
      navigate("/jobs/new");
      setActiveTab(preset.jobType);
      setPendingPreset(preset);
    },
    [navigate],
  );

  useEffect(() => {
    if (!pendingPreset || route.kind !== "new-job") {
      return;
    }
    if (pendingPreset.jobType !== activeTab) {
      return;
    }

    jobSubmissionRef.current?.applyPreset(
      pendingPreset.config,
      pendingPreset.jobType,
    );
    setPendingPreset(null);
  }, [activeTab, pendingPreset, route.kind]);

  const getCurrentConfig = useCallback(() => {
    return jobSubmissionRef.current?.getCurrentConfig() ?? {};
  }, []);

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

  const openJobAssistant = useCallback(() => {
    const currentConfig = getCurrentConfig();

    navigate("/jobs/new");
    aiAssistant.open({
      surface: "job-submission",
      jobType: activeTab,
      url:
        activeTab === "scrape"
          ? jobSubmissionRef.current?.getScrapeUrl()
          : activeTab === "crawl"
            ? jobSubmissionRef.current?.getCrawlUrl()
            : undefined,
      query:
        activeTab === "research"
          ? (currentConfig.query as string | undefined)
          : undefined,
      templateName: formState.extractTemplate || undefined,
      formSnapshot: currentConfig as Record<string, unknown>,
    });
  }, [
    activeTab,
    aiAssistant,
    formState.extractTemplate,
    getCurrentConfig,
    navigate,
  ]);

  const openTemplateAssistant = useCallback(() => {
    navigate("/templates");
    aiAssistant.open({
      surface: "templates",
      templateName: undefined,
      templateSnapshot: undefined,
      selectedUrl: getCurrentUrl() || undefined,
    });
  }, [aiAssistant, getCurrentUrl, navigate]);

  const handleSubmitForm = useCallback(
    async (formType: "scrape" | "crawl" | "research") => {
      navigate("/jobs/new");
      setActiveTab(formType);
      setPendingSubmission(formType);
    },
    [navigate],
  );

  useEffect(() => {
    if (!pendingSubmission || route.kind !== "new-job") {
      return;
    }
    if (pendingSubmission !== activeTab) {
      return;
    }

    const submit = async () => {
      if (pendingSubmission === "scrape") {
        await jobSubmissionRef.current?.submitScrape();
      } else if (pendingSubmission === "crawl") {
        await jobSubmissionRef.current?.submitCrawl();
      } else {
        await jobSubmissionRef.current?.submitResearch();
      }
      setPendingSubmission(null);
    };

    void submit();
  }, [activeTab, pendingSubmission, route.kind]);

  const routePromotionSeed = navigationState.promotionSeed ?? null;

  const detailJob =
    route.kind === "job-detail"
      ? routeDetailJob && routeDetailJob.id === route.jobId
        ? routeDetailJob
        : (jobs.find((job) => job.id === route.jobId) ?? null)
      : null;

  const clearPromotionSeed = useCallback(() => {
    if (!navigationState.promotionSeed) {
      return;
    }

    const nextState: AppNavigationState = {};
    window.history.replaceState(nextState, "", window.location.pathname);
    setNavigationState(nextState);
  }, [navigationState.promotionSeed]);

  const navigateToSourceJob = useCallback(
    (jobId: string) => {
      navigate(`/jobs/${jobId}`);
    },
    [navigate],
  );

  const handlePromoteJob = useCallback(
    (
      destination: PromotionDestination,
      options?: {
        preferredExportFormat?: "json" | "jsonl" | "md" | "csv" | "xlsx";
      },
    ) => {
      if (!detailJob || detailJob.status !== "succeeded") {
        return;
      }

      let promotionSeed: PromotionSeed;
      let path = "/templates";

      if (destination === "template") {
        promotionSeed = buildTemplatePromotionSeed(detailJob);
        path = "/templates";
      } else if (destination === "watch") {
        const watchSeed = buildWatchPromotionSeed(detailJob);
        if (!watchSeed.eligible) {
          return;
        }
        promotionSeed = watchSeed;
        path = getAutomationPath("watches");
      } else {
        promotionSeed = buildExportSchedulePromotionSeed(
          detailJob,
          options?.preferredExportFormat,
        );
        path = getAutomationPath("exports");
      }

      navigate(path, { promotionSeed });
    },
    [detailJob, navigate],
  );

  const templatePromotionSeed =
    route.kind === "templates" && routePromotionSeed?.kind === "template"
      ? (routePromotionSeed as TemplatePromotionSeed)
      : null;

  const activeAutomationSection =
    route.kind === "automation"
      ? (route.automationSection ?? DEFAULT_AUTOMATION_SECTION)
      : DEFAULT_AUTOMATION_SECTION;

  const watchPromotionSeed =
    route.kind === "automation" &&
    activeAutomationSection === "watches" &&
    routePromotionSeed?.kind === "watch"
      ? (routePromotionSeed as WatchPromotionSeed)
      : null;
  const exportPromotionSeed =
    route.kind === "automation" &&
    activeAutomationSection === "exports" &&
    routePromotionSeed?.kind === "export-schedule"
      ? (routePromotionSeed as ExportSchedulePromotionSeed)
      : null;

  const handleTourRouteChange = useCallback(
    (targetRoute: OnboardingRouteKey) => {
      switch (targetRoute) {
        case "jobs":
          navigate("/jobs");
          return;
        case "new-job":
          navigate("/jobs/new");
          return;
        case "job-detail": {
          const targetJobId =
            selectedJobId ??
            jobs.find((job) => job.status === "succeeded")?.id ??
            jobs[0]?.id;
          if (targetJobId) {
            navigate(`/jobs/${targetJobId}`);
            return;
          }
          navigate("/jobs");
          return;
        }
        case "templates":
          navigate("/templates");
          return;
        case "automation":
          navigate("/automation/batches");
          return;
        case "settings":
          navigate("/settings");
          return;
      }
    },
    [jobs, navigate, selectedJobId],
  );

  const renderAutomationSection = useCallback(
    (section: AutomationSection): ReactNode => {
      switch (section) {
        case "batches":
          return (
            <BatchContainer
              formState={formState}
              profiles={profiles}
              loading={loading}
            />
          );
        case "chains":
          return <ChainContainer onChainSubmit={refreshJobs} />;
        case "watches":
          return (
            <WatchContainer
              promotionSeed={watchPromotionSeed}
              onClearPromotionSeed={clearPromotionSeed}
              onOpenSourceJob={navigateToSourceJob}
            />
          );
        case "exports":
          return (
            <ExportScheduleContainer
              aiStatus={health?.components?.ai ?? null}
              promotionSeed={exportPromotionSeed}
              onClearPromotionSeed={clearPromotionSeed}
              onOpenSourceJob={navigateToSourceJob}
            />
          );
        case "webhooks":
          return <WebhookDeliveryContainer />;
      }
    },
    [
      clearPromotionSeed,
      exportPromotionSeed,
      formState,
      health?.components?.ai,
      loading,
      navigateToSourceJob,
      profiles,
      refreshJobs,
      watchPromotionSeed,
    ],
  );

  const activeRouteForNav = route.kind === "job-detail" ? "jobs" : route.kind;

  const routeMeta = useMemo<RouteMeta>(() => {
    switch (route.kind) {
      case "jobs":
        return {
          title: "Jobs",
        };
      case "job-detail":
        return {
          title: route.jobId
            ? `Job ${formatShortJobId(route.jobId)}`
            : "Results",
        };
      case "new-job":
        return {
          title: "Create Job",
        };
      case "templates":
        return {
          title: "Templates",
        };
      case "automation":
        return {
          title: "Automation",
        };
      case "settings":
        return {
          title: "Settings",
        };
    }
  }, [route.jobId, route.kind]);

  const showSettingsOverview = useMemo(
    () =>
      shouldShowSettingsOverviewPanel({
        isSettingsRoute: route.kind === "settings",
        setupRequired,
        jobsTotal,
        profilesCount: profiles.length,
        schedulesCount: schedules.length,
        crawlStatesTotal,
        renderProfileCount,
        pipelineScriptCount,
        proxyStatus: health?.components?.proxy_pool?.status,
        retentionStatus: health?.components?.retention?.status,
      }),
    [
      crawlStatesTotal,
      health?.components?.proxy_pool?.status,
      health?.components?.retention?.status,
      jobsTotal,
      pipelineScriptCount,
      profiles.length,
      renderProfileCount,
      route.kind,
      schedules.length,
      setupRequired,
    ],
  );

  const scrollToSettingsSection = useCallback((section: SettingsSectionId) => {
    setActiveSettingsSection(section);

    if (typeof document === "undefined") {
      return;
    }

    document
      .getElementById(SETTINGS_SECTION_META[section].elementId)
      ?.scrollIntoView({ behavior: "smooth", block: "start" });
  }, []);

  const syncActiveSettingsSection = useCallback(() => {
    if (typeof document === "undefined") {
      return;
    }

    const sections = SETTINGS_SECTION_ORDER.map((section) => ({
      id: section,
      element: document.getElementById(
        SETTINGS_SECTION_META[section].elementId,
      ) as HTMLElement | null,
    })).filter(
      (section): section is { id: SettingsSectionId; element: HTMLElement } =>
        section.element !== null,
    );

    if (sections.length === 0) {
      return;
    }

    const nextSection = getSettingsSectionInView(sections);
    setActiveSettingsSection((currentSection) =>
      currentSection === nextSection ? currentSection : nextSection,
    );
  }, []);

  useEffect(() => {
    if (route.kind !== "settings" || typeof window === "undefined") {
      return;
    }

    let animationFrameId = 0;
    const syncOnNextFrame = () => {
      cancelAnimationFrame(animationFrameId);
      animationFrameId = requestAnimationFrame(() => {
        syncActiveSettingsSection();
      });
    };

    setActiveSettingsSection("authoring");
    syncOnNextFrame();
    window.addEventListener("scroll", syncOnNextFrame, { passive: true });
    window.addEventListener("resize", syncOnNextFrame);

    return () => {
      cancelAnimationFrame(animationFrameId);
      window.removeEventListener("scroll", syncOnNextFrame);
      window.removeEventListener("resize", syncOnNextFrame);
    };
  }, [route.kind, syncActiveSettingsSection]);

  useEffect(() => {
    if (route.kind !== "settings" || typeof window === "undefined") {
      return;
    }

    let animationFrameId = 0;
    let remainingSyncPasses = showSettingsOverview ? 2 : 1;

    const syncOnNextFrame = () => {
      animationFrameId = requestAnimationFrame(() => {
        syncActiveSettingsSection();
        remainingSyncPasses -= 1;
        if (remainingSyncPasses > 0) {
          syncOnNextFrame();
        }
      });
    };

    syncOnNextFrame();

    return () => {
      cancelAnimationFrame(animationFrameId);
    };
  }, [route.kind, showSettingsOverview, syncActiveSettingsSection]);

  const jobDetailSignals = useMemo<RouteSignal[]>(() => {
    if (route.kind !== "job-detail") {
      return [];
    }

    return [
      {
        label: "Status",
        value: detailJob?.status ?? "unknown",
      },
      {
        label: "Results",
        value: resultsState.totalResults,
      },
      {
        label: "Format",
        value: resultsState.resultFormat.toUpperCase(),
      },
      {
        label: "Connection",
        value: connectionState,
      },
    ];
  }, [
    connectionState,
    detailJob?.status,
    resultsState.resultFormat,
    resultsState.totalResults,
    route.kind,
  ]);

  const handleRouteHelpAction = useCallback(
    (actionId: RouteHelpAction["id"]) => {
      switch (actionId) {
        case "create-job":
          navigate("/jobs/new");
          return;
        case "open-jobs":
          navigate("/jobs");
          return;
      }
    },
    [navigate],
  );

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

  const routeHelpPanel = (
    <RouteHelpPanel
      routeKey={routeKey}
      shortcuts={shortcuts}
      isMac={isMac}
      defaultExpanded={routeHelpDefaultExpanded}
      onOpenCommandPalette={openCommandPalette}
      onOpenShortcuts={openHelp}
      onRestartTour={resetOnboarding}
      onAction={handleRouteHelpAction}
    />
  );

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

      {setupRequired ? (
        <div className="route-stack">
          <RouteHeader
            title="Setup required"
            description="Spartan is running in guided recovery mode so the issue is visible in-product instead of only in terminal output."
          />

          <ActionEmptyState
            eyebrow="Guided recovery"
            title={health?.setup?.title ?? "Setup required"}
            description={
              health?.setup?.message ??
              "Resolve the setup issue, then restart the server."
            }
          />
        </div>
      ) : null}

      {!setupRequired && route.kind === "jobs" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
          />

          <section id="jobs" data-tour="jobs-dashboard">
            <JobMonitoringDashboard
              jobs={jobs}
              failedJobs={failedJobs}
              error={error}
              loading={loading}
              statusFilter={jobStatusFilter}
              onStatusFilterChange={setJobStatusFilter}
              onViewResults={handleViewResults}
              onCancel={cancelJob}
              onDelete={deleteJob}
              onRefresh={refreshJobs}
              onCreateJob={() => navigate("/jobs/new")}
              currentPage={jobsPage}
              totalJobs={jobsTotal}
              jobsPerPage={100}
              onPageChange={setJobsPage}
              connectionState={connectionState}
              managerStatus={managerStatus}
            />
          </section>

          {routeHelpPanel}
        </div>
      )}

      {!setupRequired && route.kind === "job-detail" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
            actions={
              <button
                type="button"
                className="secondary"
                onClick={() => navigate("/jobs")}
              >
                Back to Jobs
              </button>
            }
          />

          {detailJobLoading && !detailJob ? (
            <section className="panel">
              <div className="loading-placeholder">Loading job details…</div>
            </section>
          ) : null}

          {detailJobError && !detailJob ? (
            <ActionEmptyState
              eyebrow="Job detail"
              title="Unable to load this saved job"
              description={detailJobError}
              actions={[
                {
                  label: "Back to jobs",
                  onClick: () => navigate("/jobs"),
                },
              ]}
            />
          ) : null}

          {!detailJobError ? (
            <>
              <ResultsContainer
                resultsState={resultsState}
                jobs={jobs}
                currentJob={detailJob}
                aiStatus={health?.components?.ai ?? null}
                onPromote={handlePromoteJob}
              />

              <RouteSignals
                ariaLabel="Result context"
                items={jobDetailSignals}
              />

              {routeHelpPanel}
            </>
          ) : null}
        </div>
      )}

      {!setupRequired && route.kind === "new-job" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
          />

          <div className="route-grid route-grid--new-job">
            <div className="route-primary route-stack" data-tour="job-wizard">
              <JobSubmissionContainer
                ref={jobSubmissionRef}
                activeTab={activeTab}
                setActiveTab={setActiveTab}
                formState={formState}
                aiStatus={health?.components?.ai ?? null}
                onSubmitScrape={handleSubmitScrape}
                onSubmitCrawl={handleSubmitCrawl}
                onSubmitResearch={handleSubmitResearch}
                loading={loading}
                profiles={profiles}
              />
            </div>
            <aside className="route-sidebar">
              <PresetContainer
                presets={presets}
                activeTab={activeTab}
                setActiveTab={setActiveTab}
                savePreset={savePreset}
                getCurrentConfig={getCurrentConfig}
                getCurrentUrl={getCurrentUrl}
                onSelectPreset={handleSelectPreset}
                onOpenAssistant={openJobAssistant}
                onOpenTemplateAssistant={openTemplateAssistant}
              />
            </aside>
          </div>

          {jobsTotal === 0 && jobStatusFilter === "" ? (
            <ActionEmptyState
              eyebrow="First run"
              title="Start with a single page scrape"
              description="Paste a URL into the form below, keep the defaults, and submit one successful run before moving on to templates or automation."
            />
          ) : null}

          {routeHelpPanel}
        </div>
      )}

      {!setupRequired && route.kind === "templates" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
          />

          <div data-tour="templates-workspace">
            <TemplateManager
              templateNames={templates}
              aiStatus={health?.components?.ai ?? null}
              promotionSeed={templatePromotionSeed}
              onClearPromotionSeed={clearPromotionSeed}
              onOpenSourceJob={navigateToSourceJob}
              onTemplatesChanged={() => {
                void refreshTemplates();
              }}
            />
          </div>

          {routeHelpPanel}
        </div>
      )}

      {!setupRequired && route.kind === "automation" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
            subnav={
              <div data-tour="automation-subnav">
                <AutomationSubnav
                  activeSection={activeAutomationSection}
                  onSectionChange={(section) =>
                    navigate(getAutomationPath(section))
                  }
                />
              </div>
            }
          />
          <section data-tour="automation-hub">
            <AutomationLayout
              activeSection={activeAutomationSection}
              renderSection={renderAutomationSection}
            />
          </section>

          {routeHelpPanel}
        </div>
      )}

      {!setupRequired && route.kind === "settings" && (
        <div className="route-stack">
          <RouteHeader
            title={routeMeta.title}
            description={routeMeta.description}
            subnav={
              <SettingsSubnav
                activeSection={activeSettingsSection}
                onSectionChange={scrollToSettingsSection}
              />
            }
          />

          <div data-tour="settings-workspace" className="settings-route">
            <section
              id={SETTINGS_SECTION_META.authoring.elementId}
              className="settings-route__section"
              aria-labelledby="settings-route-authoring-title"
            >
              <div className="settings-route__section-header">
                <div className="settings-route__section-eyebrow">
                  {SETTINGS_SECTION_META.authoring.label}
                </div>
                <h2 id="settings-route-authoring-title">
                  {SETTINGS_SECTION_META.authoring.title}
                </h2>
                <p>{SETTINGS_SECTION_META.authoring.description}</p>
              </div>

              <div className="settings-route__section-stack">
                <section className="panel">
                  <RenderProfileEditor
                    aiStatus={health?.components?.ai ?? null}
                    onInventoryChange={setRenderProfileCount}
                  />
                </section>

                <section className="panel">
                  <PipelineJSEditor
                    aiStatus={health?.components?.ai ?? null}
                    onInventoryChange={setPipelineScriptCount}
                  />
                </section>
              </div>
            </section>

            {showSettingsOverview ? (
              <SettingsOverviewPanel
                onCreateJob={() => navigate("/jobs/new")}
                onOpenJobs={() => navigate("/jobs")}
              />
            ) : null}

            <section
              id={SETTINGS_SECTION_META.inventory.elementId}
              className="settings-route__section"
              aria-labelledby="settings-route-inventory-title"
            >
              <div className="settings-route__section-header">
                <div className="settings-route__section-eyebrow">
                  {SETTINGS_SECTION_META.inventory.label}
                </div>
                <h2 id="settings-route-inventory-title">
                  {SETTINGS_SECTION_META.inventory.title}
                </h2>
                <p>{SETTINGS_SECTION_META.inventory.description}</p>
              </div>

              <InfoSections
                profiles={profiles}
                schedules={schedules}
                crawlStates={crawlStates}
                crawlStatesPage={crawlStatesPage}
                crawlStatesTotal={crawlStatesTotal}
                crawlStatesPerPage={100}
                onCrawlStatesPageChange={setCrawlStatesPage}
                onCreateJob={() => navigate("/jobs/new")}
                onOpenAutomation={() => navigate("/automation/batches")}
                onOpenJobs={() => navigate("/jobs")}
              />
            </section>

            <section
              id={SETTINGS_SECTION_META.operations.elementId}
              className="settings-route__section"
              aria-labelledby="settings-route-operations-title"
            >
              <div className="settings-route__section-header">
                <div className="settings-route__section-eyebrow">
                  {SETTINGS_SECTION_META.operations.label}
                </div>
                <h2 id="settings-route-operations-title">
                  {SETTINGS_SECTION_META.operations.title}
                </h2>
                <p>{SETTINGS_SECTION_META.operations.description}</p>
              </div>

              <div className="settings-route__section-stack">
                <ProxyPoolStatusPanel
                  health={health}
                  onNavigate={navigate}
                  onRefreshHealth={refreshHealth}
                />
                <RetentionStatusPanel
                  health={health}
                  onNavigate={navigate}
                  onRefreshHealth={refreshHealth}
                  onCreateJob={() => navigate("/jobs/new")}
                  onOpenAutomation={() => navigate("/automation/batches")}
                />
              </div>
            </section>
          </div>

          {routeHelpPanel}
        </div>
      )}

      <div className="footer">Spartan Scraper 1.0 local-first workbench.</div>
    </div>
  );
}
