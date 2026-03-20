/**
 * Purpose: Render a cohesive first-run orientation panel for the Settings route.
 * Responsibilities: Explain which Settings capabilities are optional, which appear later, and what first actions operators should take instead of over-configuring early.
 * Scope: Settings-route overview guidance only; section-specific inventories and editors stay in their own panels.
 * Usage: Render near the top of the Settings workspace when the installation is still in a true first-run state.
 * Invariants/Assumptions: The panel should stay calm, action-oriented, and focused on helping operators defer configuration until a workflow proves it is needed.
 */

import { ActionEmptyState } from "./ActionEmptyState";

interface SettingsOverviewPanelProps {
  onCreateJob: () => void;
  onOpenJobs: () => void;
}

const SETTINGS_CAPABILITIES = [
  {
    title: "Auth Profiles",
    description:
      "Save login headers, cookies, and inherited auth only after a target proves it needs authenticated access.",
  },
  {
    title: "Schedules",
    description:
      "Open a succeeded job you trust, then promote it into recurring automation once the output is stable enough to reuse.",
  },
  {
    title: "Crawl States",
    description:
      "Incremental crawl history appears automatically after crawl runs have something worth resuming.",
  },
  {
    title: "Render Profiles",
    description:
      "Pin repeatable fetch and browser behavior for hosts that need a non-default runtime strategy.",
  },
  {
    title: "Pipeline JavaScript",
    description:
      "Add host-specific JavaScript only when a site needs repeatable DOM preparation or wait logic.",
  },
  {
    title: "Proxy Pool",
    description:
      "Leave pooled routing off unless your scraping workflow genuinely needs multiple proxies and rotation.",
  },
  {
    title: "Retention",
    description:
      "Enable cleanup policies later when local job history grows enough that storage pressure matters.",
  },
] as const;

export function SettingsOverviewPanel({
  onCreateJob,
  onOpenJobs,
}: SettingsOverviewPanelProps) {
  return (
    <section className="panel">
      <ActionEmptyState
        eyebrow="Settings overview"
        title="Most Settings controls can wait until a workflow proves it needs them"
        description="Start by getting one real job working end to end. Come back here when you need saved auth, reusable runtime overrides, optional proxy pooling, or cleanup policy—everything else works out of the box."
        actions={[
          { label: "Create job", onClick: onCreateJob },
          {
            label: "Review jobs",
            onClick: onOpenJobs,
            tone: "secondary",
          },
        ]}
      >
        <div className="settings-overview__grid">
          {SETTINGS_CAPABILITIES.map((capability) => (
            <div key={capability.title} className="settings-overview__item">
              <div className="settings-overview__item-title">
                {capability.title}
              </div>
              <p>{capability.description}</p>
            </div>
          ))}
        </div>
      </ActionEmptyState>
    </section>
  );
}
