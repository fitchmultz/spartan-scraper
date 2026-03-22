/**
 * Purpose: Render the Settings route's in-page section navigation.
 * Responsibilities: Present the major Settings section groups, mark the active quick-jump target, and notify the parent route when operators switch sections.
 * Scope: Settings route navigation chrome only; section layout and content stay in the route container.
 * Usage: Render inside the shared RouteHeader sub-navigation slot for the `/settings` route.
 * Invariants/Assumptions: Each item maps to a stable in-page section within `/settings`, and section changes are handled by the parent shell.
 */

export const SETTINGS_SECTION_META = {
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
} as const;

export type SettingsSectionId = keyof typeof SETTINGS_SECTION_META;

const SETTINGS_SECTION_ORDER: SettingsSectionId[] = [
  "authoring",
  "inventory",
  "operations",
];

interface SettingsSubnavProps {
  activeSection: SettingsSectionId;
  onSectionChange: (section: SettingsSectionId) => void;
}

export function SettingsSubnav({
  activeSection,
  onSectionChange,
}: SettingsSubnavProps) {
  return (
    <nav className="settings-subnav" aria-label="Settings sections">
      {SETTINGS_SECTION_ORDER.map((section) => {
        const isActive = section === activeSection;
        const meta = SETTINGS_SECTION_META[section];

        return (
          <button
            key={section}
            type="button"
            className={`settings-subnav__link${isActive ? " is-active" : ""}`}
            onClick={() => onSectionChange(section)}
            aria-current={isActive ? "page" : undefined}
            title={meta.description}
          >
            {meta.label}
          </button>
        );
      })}
    </nav>
  );
}
