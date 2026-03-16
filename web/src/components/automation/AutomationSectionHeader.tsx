/**
 * Purpose: Provide focused framing for the active automation section.
 * Responsibilities: Show the current section title, supporting copy, and quick-scan highlights above the active automation workspace.
 * Scope: Section-level introductory framing within the automation hub only.
 * Usage: Render once per active automation section above the mounted container content.
 * Invariants/Assumptions: Section metadata comes from the shared automation section registry and stays aligned with canonical section names.
 */

import {
  AUTOMATION_SECTION_META,
  type AutomationSection,
} from "./automationSections";

interface AutomationSectionHeaderProps {
  section: AutomationSection;
}

export function AutomationSectionHeader({
  section,
}: AutomationSectionHeaderProps) {
  const meta = AUTOMATION_SECTION_META[section];

  return (
    <section className="panel automation-section-header">
      <div className="automation-section-header__eyebrow">
        Automation Section
      </div>
      <div className="automation-section-header__copy">
        <h3 id={`automation-section-title-${section}`}>{meta.label}</h3>
        <p>{meta.description}</p>
      </div>
      <div className="automation-section-header__highlights">
        {meta.highlights.map((highlight) => (
          <span
            key={highlight}
            className="automation-section-header__highlight"
          >
            {highlight}
          </span>
        ))}
      </div>
    </section>
  );
}
