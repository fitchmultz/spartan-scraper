/**
 * Purpose: Define the canonical Settings sections, copy, and deep-link helpers.
 * Responsibilities: Export the Settings section union, default section, per-section metadata, and path parsing utilities.
 * Scope: Settings route information architecture only; rendering lives in the app shell and Settings components.
 * Usage: Imported by the app shell, Settings sub-navigation, and tests that need stable section semantics.
 * Invariants/Assumptions: `/settings/:section` is the canonical deep-link shape and `/settings` resolves to the default section.
 */

export const SETTINGS_SECTIONS = [
  "authoring",
  "inventory",
  "operations",
] as const;

export type SettingsSectionId = (typeof SETTINGS_SECTIONS)[number];

interface SettingsSectionMeta {
  label: string;
  title: string;
  description: string;
  elementId: string;
}

export const DEFAULT_SETTINGS_SECTION: SettingsSectionId = "authoring";

export const SETTINGS_SECTION_META: Record<
  SettingsSectionId,
  SettingsSectionMeta
> = {
  authoring: {
    label: "Authoring tools",
    title: "Authoring tools",
    description:
      "Keep render profiles and pipeline JavaScript together as reusable runtime authoring tools.",
    elementId: "settings-authoring-tools",
  },
  inventory: {
    label: "Saved state",
    title: "Saved state and history",
    description:
      "Review reusable auth, recurring schedules, and crawl-state history without mixing them into authoring tools.",
    elementId: "settings-saved-state",
  },
  operations: {
    label: "Operations",
    title: "Operational controls",
    description:
      "Keep optional proxy routing and retention maintenance together, separate from day-to-day authoring.",
    elementId: "settings-operational-controls",
  },
};

export const SETTINGS_SECTION_ORDER = SETTINGS_SECTIONS;

export function isSettingsSection(value: string): value is SettingsSectionId {
  return (SETTINGS_SECTIONS as readonly string[]).includes(value);
}

export function getSettingsPath(section: SettingsSectionId): string {
  return `/settings/${section}`;
}

export function getSettingsSectionFromPath(
  path: string,
): SettingsSectionId | null {
  if (path === "/settings") {
    return DEFAULT_SETTINGS_SECTION;
  }
  if (!path.startsWith("/settings/")) {
    return null;
  }

  const segment = path.slice("/settings/".length).split("/")[0] ?? "";
  return isSettingsSection(segment) ? segment : null;
}
