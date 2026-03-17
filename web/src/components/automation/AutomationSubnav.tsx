/**
 * Purpose: Render the automation hub's explicit in-route section navigation.
 * Responsibilities: Present all automation sections, mark the active section, and notify the shell when operators switch sections.
 * Scope: Automation route navigation chrome only; content rendering and routing decisions live elsewhere.
 * Usage: Render inside the shared RouteHeader sub-navigation slot for the `/automation` route.
 * Invariants/Assumptions: Every item maps to a canonical automation section and section changes are handled by the parent shell.
 */

import {
  AUTOMATION_SECTIONS,
  AUTOMATION_SECTION_META,
  type AutomationSection,
} from "./automationSections";

interface AutomationSubnavProps {
  activeSection: AutomationSection;
  onSectionChange: (section: AutomationSection) => void;
}

export function AutomationSubnav({
  activeSection,
  onSectionChange,
}: AutomationSubnavProps) {
  return (
    <nav
      className="automation-subnav"
      aria-label="Automation sections"
      data-tour="automation-subnav"
    >
      {AUTOMATION_SECTIONS.map((section) => {
        const isActive = section === activeSection;
        const meta = AUTOMATION_SECTION_META[section];

        return (
          <button
            key={section}
            type="button"
            className={`automation-subnav__link${isActive ? " is-active" : ""}`}
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
