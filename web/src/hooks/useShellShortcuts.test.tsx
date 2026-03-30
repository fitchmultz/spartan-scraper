/**
 * Purpose: Verify shell shortcut orchestration stays aligned with the top-level web shell.
 * Responsibilities: Assert keyboard navigation dispatches, assistant launchers open the expected contexts, and route-help/onboarding callbacks keep canonical route targets.
 * Scope: Hook-level shell shortcut behavior only.
 * Usage: Run via Vitest as part of the web test suite.
 * Invariants/Assumptions: The hook is exercised in jsdom with a mocked navigation surface and a minimal job-submission ref.
 */

import { renderHook, act } from "@testing-library/react";
import type { RefObject } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

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
import type { UseShellShortcutsOptions } from "./useShellShortcuts";
import { useShellShortcuts } from "./useShellShortcuts";

const DEFAULT_SHORTCUTS: UseShellShortcutsOptions["shortcuts"] = {
  commandPalette: "mod+k",
  submitForm: "mod+enter",
  search: "/",
  help: "?",
  escape: "escape",
  navigateJobs: "g j",
  navigateResults: "g r",
  navigateForms: "g f",
};

function createSubmissionRef({
  currentConfig = {},
  scrapeUrl = "https://scrape.example.com",
  crawlUrl = "https://crawl.example.com",
}: {
  currentConfig?: PresetConfig;
  scrapeUrl?: string;
  crawlUrl?: string;
} = {}): RefObject<JobSubmissionContainerRef | null> {
  return {
    current: {
      getCurrentConfig: () => currentConfig,
      getScrapeUrl: () => scrapeUrl,
      getCrawlUrl: () => crawlUrl,
    },
  } as RefObject<JobSubmissionContainerRef | null>;
}

function buildHarness(overrides: Partial<UseShellShortcutsOptions> = {}) {
  const navigate = vi.fn();
  const persistJobsViewState = vi.fn();
  const openAssistant = vi.fn();
  const openCommandPalette = vi.fn();
  const openHelp = vi.fn();
  const resetOnboarding = vi.fn();

  const options: UseShellShortcutsOptions = {
    navigate,
    persistJobsViewState,
    routeKind: "jobs",
    selectedJobId: "job-selected",
    jobs: [
      { id: "job-selected", status: "running" },
      { id: "job-succeeded", status: "succeeded" },
    ] as JobEntry[],
    activeTab: "research" as JobType,
    extractTemplate: "article-template",
    jobSubmissionRef: createSubmissionRef(),
    openAssistant,
    shortcuts: DEFAULT_SHORTCUTS,
    isMac: false,
    openCommandPalette,
    openHelp,
    resetOnboarding,
    ...overrides,
  };

  return {
    options,
    navigate,
    persistJobsViewState,
    openAssistant,
    openCommandPalette,
    openHelp,
    resetOnboarding,
  };
}

describe("useShellShortcuts", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("routes keyboard navigation events to the current shell views", () => {
    const { options, navigate, persistJobsViewState } = buildHarness({
      selectedJobId: "job-123",
      routeKind: "jobs",
    });

    renderHook(() => useShellShortcuts(options));

    act(() => {
      window.dispatchEvent(
        new CustomEvent("keyboard-navigate", {
          detail: { destination: "navigateResults" },
        }),
      );
    });

    expect(persistJobsViewState).toHaveBeenCalledTimes(1);
    expect(navigate).toHaveBeenCalledWith("/jobs/job-123");

    act(() => {
      window.dispatchEvent(
        new CustomEvent("keyboard-navigate", {
          detail: { destination: "navigateForms" },
        }),
      );
    });

    expect(navigate).toHaveBeenCalledWith("/jobs/new");

    act(() => {
      window.dispatchEvent(
        new CustomEvent("keyboard-navigate", {
          detail: { destination: "navigateJobs" },
        }),
      );
    });

    expect(navigate).toHaveBeenCalledWith("/jobs");
  });

  it("opens the job assistant with the current submission snapshot", () => {
    const currentConfig: PresetConfig = {
      query: "research notes",
      extractTemplate: "article-template",
    };
    const { options, navigate, openAssistant } = buildHarness({
      activeTab: "research",
      extractTemplate: "article-template",
      jobSubmissionRef: createSubmissionRef({
        currentConfig,
        scrapeUrl: "https://scrape.example.com",
        crawlUrl: "https://crawl.example.com",
      }),
    });

    const { result } = renderHook(() => useShellShortcuts(options));

    act(() => {
      result.current.openJobAssistant();
    });

    expect(navigate).toHaveBeenCalledWith("/jobs/new");
    expect(openAssistant).toHaveBeenCalledWith({
      surface: "job-submission",
      jobType: "research",
      url: undefined,
      query: "research notes",
      templateName: "article-template",
      formSnapshot: currentConfig,
    });
  });

  it("opens the template assistant with the current route URL", () => {
    const { options, navigate, openAssistant } = buildHarness({
      activeTab: "crawl",
      jobSubmissionRef: createSubmissionRef({
        crawlUrl: "https://crawl.example.com",
      }),
    });

    const { result } = renderHook(() => useShellShortcuts(options));

    act(() => {
      result.current.openTemplateAssistant();
    });

    expect(navigate).toHaveBeenCalledWith("/templates");
    expect(openAssistant).toHaveBeenCalledWith({
      surface: "templates",
      templateName: undefined,
      templateSnapshot: undefined,
      selectedUrl: "https://crawl.example.com",
    });
  });

  it("wires route-help callbacks and onboarding route changes to canonical paths", () => {
    const { options, navigate, openCommandPalette, openHelp, resetOnboarding } =
      buildHarness({
        routeKind: "job-detail",
        selectedJobId: null,
        jobs: [
          { id: "job-succeeded", status: "succeeded" },
          { id: "job-failed", status: "failed" },
        ] as JobEntry[],
      });

    const { result } = renderHook(() => useShellShortcuts(options));

    act(() => {
      result.current.routeHelpProps.onOpenCommandPalette();
      result.current.routeHelpProps.onOpenShortcuts();
      result.current.routeHelpProps.onRestartTour();
      result.current.routeHelpProps.onAction("create-job");
      result.current.routeHelpProps.onAction("open-jobs");
      result.current.handleTourRouteChange("job-detail");
      result.current.handleTourRouteChange("automation");
      result.current.handleTourRouteChange("settings");
      result.current.handleTourRouteChange("templates");
      result.current.handleTourRouteChange("new-job");
    });

    expect(openCommandPalette).toHaveBeenCalledTimes(1);
    expect(openHelp).toHaveBeenCalledTimes(1);
    expect(resetOnboarding).toHaveBeenCalledTimes(1);
    expect(navigate).toHaveBeenCalledWith("/jobs/new");
    expect(navigate).toHaveBeenCalledWith("/jobs");
    expect(navigate).toHaveBeenCalledWith("/jobs/job-succeeded");
    expect(navigate).toHaveBeenCalledWith(
      getAutomationPath(DEFAULT_AUTOMATION_SECTION),
    );
    expect(navigate).toHaveBeenCalledWith(
      getSettingsPath(DEFAULT_SETTINGS_SECTION),
    );
    expect(navigate).toHaveBeenCalledWith("/templates");
  });
});
