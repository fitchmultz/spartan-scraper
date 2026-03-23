/**
 * Purpose: Render the Settings route's explicit section navigation.
 * Responsibilities: Present the major Settings sections, mark the active section, and notify the parent route when operators switch sections.
 * Scope: Settings route navigation chrome only; content rendering and route decisions live elsewhere.
 * Usage: Render inside the shared RouteHeader sub-navigation slot for the `/settings/:section` route.
 * Invariants/Assumptions: Every item maps to a canonical Settings section and section changes are handled by the parent shell.
 */

import {
  SETTINGS_SECTION_META,
  SETTINGS_SECTION_ORDER,
  type SettingsSectionId,
} from "./settingsSections";

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
