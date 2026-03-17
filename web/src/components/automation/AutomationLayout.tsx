/**
 * Purpose: Host the automation hub's active section framing and mount-preserving content panels.
 * Responsibilities: Keep section-local UI state alive across section switches while showing only the active section visually.
 * Scope: Automation route layout only; individual automation workflows remain owned by their existing containers.
 * Usage: Render from the app shell with the active section and a section renderer callback.
 * Invariants/Assumptions: Only one section is visible at a time, but previously visited sections stay mounted to preserve local state.
 */

import { useEffect, useState, type ReactNode } from "react";
import type { AutomationSection } from "./automationSections";

interface AutomationLayoutProps {
  activeSection: AutomationSection;
  renderSection: (section: AutomationSection) => ReactNode;
}

export function AutomationLayout({
  activeSection,
  renderSection,
}: AutomationLayoutProps) {
  const [visitedSections, setVisitedSections] = useState<AutomationSection[]>([
    activeSection,
  ]);

  useEffect(() => {
    setVisitedSections((current) =>
      current.includes(activeSection) ? current : [...current, activeSection],
    );
  }, [activeSection]);

  const sectionsToRender = visitedSections.includes(activeSection)
    ? visitedSections
    : [...visitedSections, activeSection];

  return (
    <div className="automation-hub">
      <div className="automation-hub__content">
        {sectionsToRender.map((section) => {
          const isActive = section === activeSection;

          return (
            <section
              key={section}
              className="automation-hub__panel"
              aria-labelledby={`automation-section-title-${section}`}
              hidden={!isActive}
            >
              {renderSection(section)}
            </section>
          );
        })}
      </div>
    </div>
  );
}
