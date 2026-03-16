/**
 * Purpose: Define the canonical automation hub sections, copy, and deep-link helpers.
 * Responsibilities: Export the section union, default section, per-section metadata, and path/hash parsing utilities.
 * Scope: Automation route information architecture only; rendering lives in automation UI components.
 * Usage: Imported by the app shell, automation sub-navigation, and tests that need stable section semantics.
 * Invariants/Assumptions: `/automation/:section` is the canonical deep-link model and legacy hash anchors map onto one of the supported sections.
 */

export const AUTOMATION_SECTIONS = [
  "batches",
  "chains",
  "watches",
  "exports",
  "webhooks",
] as const;

export type AutomationSection = (typeof AUTOMATION_SECTIONS)[number];

interface AutomationSectionMeta {
  label: string;
  description: string;
  highlights: readonly string[];
}

export const DEFAULT_AUTOMATION_SECTION: AutomationSection = "batches";

export const AUTOMATION_SECTION_META: Record<
  AutomationSection,
  AutomationSectionMeta
> = {
  batches: {
    label: "Batches",
    description:
      "Create grouped scrape, crawl, and research submissions while keeping recent runs and batch drill-down close at hand.",
    highlights: [
      "Create new batch",
      "Switch batch type",
      "Review recent submissions",
    ],
  },
  chains: {
    label: "Chains",
    description:
      "Build and run reusable multi-step automations from a focused chain workspace instead of a below-the-fold afterthought.",
    highlights: [
      "Build chain",
      "Review latest chain status",
      "Submit chain runs",
    ],
  },
  watches: {
    label: "Watches",
    description:
      "Track watched targets, inspect recent change outcomes, and run manual checks without leaving the automation hub.",
    highlights: ["Add watch", "Check watch status", "Edit monitoring rules"],
  },
  exports: {
    label: "Exports",
    description:
      "Manage export schedules, toggle them quickly, and inspect schedule history from one dedicated export surface.",
    highlights: [
      "Create schedule",
      "Enable or disable quickly",
      "Inspect export history",
    ],
  },
  webhooks: {
    label: "Webhooks",
    description:
      "Inspect delivery attempts, filter failures, and drill into webhook records with delivery-native context.",
    highlights: [
      "Filter deliveries",
      "Inspect failures",
      "Open delivery detail",
    ],
  },
};

const LEGACY_HASH_SECTION_MAP: Record<string, AutomationSection> = {
  "#batch-forms": "batches",
  "#batches": "batches",
  "#chains": "chains",
  "#watches": "watches",
  "#export-schedules": "exports",
  "#webhook-deliveries": "webhooks",
};

export function isAutomationSection(value: string): value is AutomationSection {
  return (AUTOMATION_SECTIONS as readonly string[]).includes(value);
}

export function getAutomationPath(section: AutomationSection): string {
  return `/automation/${section}`;
}

export function getAutomationSectionFromPath(
  path: string,
): AutomationSection | null {
  if (path === "/automation") {
    return DEFAULT_AUTOMATION_SECTION;
  }
  if (!path.startsWith("/automation/")) {
    return null;
  }

  const segment = path.slice("/automation/".length).split("/")[0] ?? "";
  return isAutomationSection(segment) ? segment : null;
}

export function getAutomationSectionFromHash(
  hash: string,
): AutomationSection | null {
  return LEGACY_HASH_SECTION_MAP[hash] ?? null;
}
