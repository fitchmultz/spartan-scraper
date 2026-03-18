/**
 * Purpose: Centralize progressive onboarding copy, route help content, and guided-tour targets for the web app.
 * Responsibilities: Define the supported onboarding route keys, route-specific help panels, guided-tour step metadata, and the canonical onboarding step count.
 * Scope: Shared onboarding configuration only; state persistence and UI rendering live in hooks and components.
 * Usage: Import from onboarding hooks, the Joyride flow, route help surfaces, and shortcut-help views.
 * Invariants/Assumptions: Route keys stay aligned with the top-level web shell, tour targets point at visible UI anchors, and route help remains available even after operators skip the full tour.
 */

import type { ShortcutConfig } from "../hooks/useKeyboard";

export type OnboardingRouteKey =
  | "jobs"
  | "new-job"
  | "job-detail"
  | "templates"
  | "automation"
  | "settings";

export interface RouteHelpShortcutConfig {
  label: string;
  shortcut: keyof ShortcutConfig;
}

export interface RouteHelpAction {
  id:
    | "create-job"
    | "open-templates"
    | "open-automation"
    | "open-settings"
    | "start-tour";
  label: string;
}

export interface RouteHelpContent {
  title: string;
  summary: string;
  whatYouCanDo: readonly string[];
  shortcuts: readonly RouteHelpShortcutConfig[];
  nextActions: readonly RouteHelpAction[];
}

export interface OnboardingTourStepConfig {
  id: string;
  route: OnboardingRouteKey;
  target: string;
  title: string;
  body: string;
  placement?: "top" | "bottom" | "left" | "right" | "center" | "auto";
  disableBeacon?: boolean;
  bullets?: readonly string[];
}

export const ROUTE_HELP_CONTENT: Record<OnboardingRouteKey, RouteHelpContent> =
  {
    jobs: {
      title: "What can I do here?",
      summary:
        "Monitor active work, scan failures, and jump into saved results without leaving the main operations view.",
      whatYouCanDo: [
        "Check queue health and recent failures at a glance.",
        "Open the latest result context without re-running work.",
        "Move straight into job creation or route-level actions from the shell.",
      ],
      shortcuts: [
        { label: "Open command palette", shortcut: "commandPalette" },
        { label: "Open keyboard help", shortcut: "help" },
        { label: "Go to New Job", shortcut: "navigateForms" },
      ],
      nextActions: [
        { id: "create-job", label: "Create first job" },
        { id: "open-templates", label: "Browse templates" },
      ],
    },
    "new-job": {
      title: "What can I do here?",
      summary:
        "Create scrape, crawl, or research work with the guided wizard, presets, and expert controls in one route.",
      whatYouCanDo: [
        "Switch between guided and expert authoring without losing draft values.",
        "Reuse presets and open AI helpers from the quick-start rail.",
        "Submit the active draft directly from keyboard or shell actions.",
      ],
      shortcuts: [
        { label: "Open command palette", shortcut: "commandPalette" },
        { label: "Open keyboard help", shortcut: "help" },
        { label: "Go to Jobs", shortcut: "navigateJobs" },
      ],
      nextActions: [
        { id: "open-templates", label: "Open templates" },
        { id: "start-tour", label: "Restart tour" },
      ],
    },
    "job-detail": {
      title: "What can I do here?",
      summary:
        "Inspect saved output, understand job context, and branch into export or deeper analysis only when you need it.",
      whatYouCanDo: [
        "Read the canonical result first instead of juggling multiple modes immediately.",
        "Review status, output format, and result count before exporting or comparing.",
        "Return to the jobs dashboard fast when you need to pivot back to live work.",
      ],
      shortcuts: [
        { label: "Open command palette", shortcut: "commandPalette" },
        { label: "Open keyboard help", shortcut: "help" },
        { label: "Go to Jobs", shortcut: "navigateJobs" },
      ],
      nextActions: [{ id: "create-job", label: "Queue another job" }],
    },
    templates: {
      title: "What can I do here?",
      summary:
        "Author, preview, duplicate, and refine extraction templates in a persistent workspace instead of modal fragments.",
      whatYouCanDo: [
        "Manage the template library and open one working editor at a time.",
        "Use previews, the visual selector builder, and AI assistance in one flow.",
        "Duplicate built-in templates into editable drafts without losing context.",
      ],
      shortcuts: [
        { label: "Open command palette", shortcut: "commandPalette" },
        { label: "Open keyboard help", shortcut: "help" },
        { label: "Go to Jobs", shortcut: "navigateJobs" },
      ],
      nextActions: [
        { id: "create-job", label: "Create first job" },
        { id: "open-automation", label: "Open automation" },
      ],
    },
    automation: {
      title: "What can I do here?",
      summary:
        "Switch between batches, chains, watches, export schedules, and webhook deliveries from one focused automation hub.",
      whatYouCanDo: [
        "Move between automation domains with stable sub-navigation instead of scrolling a stacked mega-page.",
        "Submit and inspect automations in context without losing section-local state.",
        "Use route help and the command palette when you need the next action fast.",
      ],
      shortcuts: [
        { label: "Open command palette", shortcut: "commandPalette" },
        { label: "Open keyboard help", shortcut: "help" },
        { label: "Go to Jobs", shortcut: "navigateJobs" },
      ],
      nextActions: [
        { id: "create-job", label: "Create single job" },
        { id: "open-settings", label: "Open settings" },
      ],
    },
    settings: {
      title: "What can I do here?",
      summary:
        "Configure profiles, schedules, pipeline tools, and runtime maintenance surfaces from the control-center route.",
      whatYouCanDo: [
        "Review platform configuration without leaving the main shell.",
        "Move from settings to templates or jobs through visible shell controls.",
        "Use the help panel to understand what belongs in this route.",
      ],
      shortcuts: [
        { label: "Open command palette", shortcut: "commandPalette" },
        { label: "Open keyboard help", shortcut: "help" },
        { label: "Go to Jobs", shortcut: "navigateJobs" },
      ],
      nextActions: [
        { id: "create-job", label: "Create first job" },
        { id: "open-automation", label: "Open automation" },
      ],
    },
  };

export const ONBOARDING_TOUR_STEPS: readonly OnboardingTourStepConfig[] = [
  {
    id: "jobs-overview",
    route: "jobs",
    target: '[data-tour="jobs-dashboard"], body',
    title: "Jobs is the operations center",
    body: "Start here to monitor active work, scan failures, and jump into saved results quickly.",
    placement: "bottom",
    disableBeacon: true,
  },
  {
    id: "command-palette",
    route: "jobs",
    target: '[data-tour="command-palette"], body',
    title: "The command palette is always visible now",
    body: "Use the toolbar button or its shortcut anytime for navigation, job actions, presets, and restarting onboarding.",
    placement: "bottom",
  },
  {
    id: "new-job-quickstart",
    route: "new-job",
    target: '[data-tour="quickstart"], body',
    title: "Quick Start keeps job creation fast",
    body: "Use presets, workflow switching, and AI entry points without leaving the creation route.",
    placement: "bottom",
  },
  {
    id: "new-job-wizard",
    route: "new-job",
    target: '[data-tour="wizard-steps"], body',
    title: "The guided wizard is the default path",
    body: "Move through Basics, Runtime, Extraction, and Review without losing access to expert controls.",
    placement: "bottom",
    bullets: [
      "Choose the right job type first.",
      "Complete required inputs step by step.",
      "Submit confidently from the review stage.",
    ],
  },
  {
    id: "results-reader",
    route: "job-detail",
    target: '[data-tour="job-results"], body',
    title: "Results live on their own route",
    body: "Open a completed run to inspect output, understand context, and branch into export or deeper analysis.",
    placement: "bottom",
  },
  {
    id: "templates-workspace",
    route: "templates",
    target: '[data-tour="templates-workspace"], body',
    title: "Templates are a real workspace now",
    body: "Manage extraction logic, previews, and AI-assisted authoring from one route instead of bouncing through modal-first flows.",
    placement: "bottom",
  },
  {
    id: "automation-hub",
    route: "automation",
    target: '[data-tour="automation-hub"], body',
    title: "Automation is sectioned, not stacked",
    body: "Use sub-navigation to switch between automation domains instead of scrolling through unrelated tools.",
    placement: "bottom",
  },
  {
    id: "automation-subnav",
    route: "automation",
    target: '[data-tour="automation-subnav"], body',
    title: "Change automation sections from here",
    body: "Batches, chains, watches, exports, and webhooks each have a stable deep link and focused workspace.",
    placement: "bottom",
  },
  {
    id: "settings-workspace",
    route: "settings",
    target: '[data-tour="settings-workspace"], body',
    title: "Settings centralizes configuration surfaces",
    body: "Use this route for profiles, schedules, render and pipeline tools, and runtime maintenance settings.",
    placement: "bottom",
  },
  {
    id: "route-help",
    route: "settings",
    target: '[data-tour="route-help"], body',
    title: "Route help stays in the UI",
    body: 'Each major route now exposes a visible "What can I do here?" panel, plus route-relevant shortcuts and actions.',
    placement: "bottom",
  },
];

export const ONBOARDING_TOTAL_STEPS = ONBOARDING_TOUR_STEPS.length;
