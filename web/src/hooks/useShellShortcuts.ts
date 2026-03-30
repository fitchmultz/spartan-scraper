/**
 * Purpose: Coordinate shell-level keyboard navigation, assistant launchers, and route-help actions for the app shell.
 * Responsibilities: Translate shared keyboard events into route changes, open the AI assistant on the current job or template context, and keep route-help/onboarding callbacks aligned with shell state.
 * Scope: App-shell shortcut orchestration only; route rendering and data loading remain in `App.tsx` and the route containers.
 * Usage: Call from `App.tsx` with the current shell state, navigation helpers, and global shortcut controls.
 * Invariants/Assumptions: The shell runs in a browser context, route keys stay aligned with the onboarding config, and keyboard navigation events only originate from the shared keyboard hook.
 */

import {
  useCallback,
  useEffect,
  useEffectEvent,
  useMemo,
  type RefObject,
} from "react";

import type { AssistantContext } from "../components/ai-assistant/AIAssistantProvider";
import {
  DEFAULT_AUTOMATION_SECTION,
  getAutomationPath,
} from "../components/automation/automationSections";
import type { JobSubmissionContainerRef } from "../components/jobs/JobSubmissionContainer";
import {
  DEFAULT_SETTINGS_SECTION,
  getSettingsPath,
} from "../components/settings/settingsSections";
import type { JobEntry } from "../types";
import type { JobType, PresetConfig } from "../types/presets";
import type { RouteHelpAction, OnboardingRouteKey } from "../lib/onboarding";
import type { ShortcutConfig } from "./useKeyboard";
import type { RouteKind } from "./useAppShellRouting";

type KeyboardNavigateDestination =
  | "navigateJobs"
  | "navigateResults"
  | "navigateForms";

export interface ShellRouteHelpProps {
  shortcuts: ShortcutConfig;
  isMac: boolean;
  onOpenCommandPalette: () => void;
  onOpenShortcuts: () => void;
  onRestartTour: () => void;
  onAction: (actionId: RouteHelpAction["id"]) => void;
}

export interface UseShellShortcutsOptions {
  navigate: (path: string) => void;
  persistJobsViewState: () => void;
  routeKind: RouteKind;
  selectedJobId: string | null;
  jobs: JobEntry[];
  activeTab: JobType;
  extractTemplate: string;
  jobSubmissionRef: RefObject<JobSubmissionContainerRef | null>;
  openAssistant: (context: AssistantContext) => void;
  shortcuts: ShortcutConfig;
  isMac: boolean;
  openCommandPalette: () => void;
  openHelp: () => void;
  resetOnboarding: () => void;
}

export interface UseShellShortcutsReturn {
  openJobAssistant: () => void;
  openTemplateAssistant: () => void;
  handleTourRouteChange: (targetRoute: OnboardingRouteKey) => void;
  routeHelpProps: ShellRouteHelpProps;
}

export function useShellShortcuts(
  options: UseShellShortcutsOptions,
): UseShellShortcutsReturn {
  const {
    navigate,
    persistJobsViewState,
    routeKind,
    selectedJobId,
    jobs,
    activeTab,
    extractTemplate,
    jobSubmissionRef,
    openAssistant,
    shortcuts,
    isMac,
    openCommandPalette,
    openHelp,
    resetOnboarding,
  } = options;

  const openJobAssistant = useCallback(() => {
    const currentConfig = (jobSubmissionRef.current?.getCurrentConfig() ??
      {}) as PresetConfig;
    const currentUrl =
      activeTab === "scrape"
        ? jobSubmissionRef.current?.getScrapeUrl()
        : activeTab === "crawl"
          ? jobSubmissionRef.current?.getCrawlUrl()
          : undefined;

    navigate("/jobs/new");
    openAssistant({
      surface: "job-submission",
      jobType: activeTab,
      url: currentUrl,
      query:
        activeTab === "research"
          ? (currentConfig.query as string | undefined)
          : undefined,
      templateName: extractTemplate || undefined,
      formSnapshot: currentConfig as Record<string, unknown>,
    });
  }, [activeTab, extractTemplate, jobSubmissionRef, navigate, openAssistant]);

  const openTemplateAssistant = useCallback(() => {
    const currentUrl =
      activeTab === "scrape"
        ? jobSubmissionRef.current?.getScrapeUrl()
        : activeTab === "crawl"
          ? jobSubmissionRef.current?.getCrawlUrl()
          : undefined;

    navigate("/templates");
    openAssistant({
      surface: "templates",
      templateName: undefined,
      templateSnapshot: undefined,
      selectedUrl: currentUrl || undefined,
    });
  }, [activeTab, jobSubmissionRef, navigate, openAssistant]);

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
          navigate(getAutomationPath(DEFAULT_AUTOMATION_SECTION));
          return;
        case "settings":
          navigate(getSettingsPath(DEFAULT_SETTINGS_SECTION));
          return;
      }
    },
    [jobs, navigate, selectedJobId],
  );

  const handleKeyboardNavigate = useEffectEvent((event: Event) => {
    const customEvent = event as CustomEvent<{
      destination?: KeyboardNavigateDestination;
    }>;
    const destination = customEvent.detail?.destination;

    if (!destination) {
      return;
    }

    if (destination === "navigateJobs") {
      navigate("/jobs");
      return;
    }

    if (destination === "navigateResults") {
      if (routeKind === "jobs") {
        persistJobsViewState();
      }

      if (selectedJobId) {
        navigate(`/jobs/${selectedJobId}`);
        return;
      }

      navigate("/jobs");
      return;
    }

    if (destination === "navigateForms") {
      navigate("/jobs/new");
    }
  });

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    window.addEventListener("keyboard-navigate", handleKeyboardNavigate);
    return () => {
      window.removeEventListener("keyboard-navigate", handleKeyboardNavigate);
    };
  }, []);

  const routeHelpProps = useMemo(
    () => ({
      shortcuts,
      isMac,
      onOpenCommandPalette: openCommandPalette,
      onOpenShortcuts: openHelp,
      onRestartTour: resetOnboarding,
      onAction: handleRouteHelpAction,
    }),
    [
      handleRouteHelpAction,
      isMac,
      openCommandPalette,
      openHelp,
      resetOnboarding,
      shortcuts,
    ],
  );

  return {
    openJobAssistant,
    openTemplateAssistant,
    handleTourRouteChange,
    routeHelpProps,
  };
}
