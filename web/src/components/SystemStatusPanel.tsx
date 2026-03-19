/**
 * Purpose: Render in-product setup and runtime diagnostics sourced from `/healthz`.
 * Responsibilities: Highlight setup-mode guidance, degraded components, and operator-facing notices with actionable follow-up steps.
 * Scope: Diagnostic panel presentation only.
 * Usage: Mount near the top of `App.tsx` and pass the latest health response plus navigation helpers.
 * Invariants/Assumptions: Healthy states should stay out of the way, while setup/degraded states must remain obvious and actionable.
 */

import type { HealthResponse, RuntimeNotice } from "../api";
import { CapabilityActionList } from "./CapabilityActionList";

interface SystemStatusPanelProps {
  health: HealthResponse | null;
  onNavigate: (path: string) => void;
  onRefresh: () => Promise<unknown> | undefined;
}

function isVisibleNotice(notice: RuntimeNotice) {
  return notice.severity === "warning" || notice.severity === "error";
}

export function SystemStatusPanel({
  health,
  onNavigate,
  onRefresh,
}: SystemStatusPanelProps) {
  if (!health) {
    return null;
  }

  const notices = (health.notices ?? []).filter(
    (notice) =>
      isVisibleNotice(notice) &&
      (!health.setup?.required || notice.scope !== "setup"),
  );
  const degradedComponents = Object.entries(health.components ?? {}).filter(
    ([, component]) =>
      component.status === "degraded" ||
      component.status === "error" ||
      (!health.setup?.required && component.status === "setup_required"),
  );

  if (
    !health.setup?.required &&
    degradedComponents.length === 0 &&
    notices.length === 0
  ) {
    return null;
  }

  return (
    <section className="panel system-status" aria-live="polite">
      <div className="system-status__header">
        <div>
          <div className="system-status__eyebrow">System status</div>
          <h2>
            {health.setup?.required
              ? health.setup.title
              : "Spartan found setup or runtime issues"}
          </h2>
          <p>
            {health.setup?.required
              ? health.setup.message
              : "Core workflows stay available where possible. Use the guidance below for components that truly need recovery; optional capabilities you left off by choice stay out of this panel."}
          </p>
        </div>

        <button
          type="button"
          className="secondary"
          onClick={() => void onRefresh()}
        >
          Refresh status
        </button>
      </div>

      {health.setup?.required ? (
        <div className="system-status__setup">
          <div className="system-status__meta">
            <span>Data directory: {health.setup.dataDir}</span>
            {health.setup.schemaVersion ? (
              <span>Schema: {health.setup.schemaVersion}</span>
            ) : null}
          </div>

          <CapabilityActionList
            actions={health.setup.actions ?? []}
            onNavigate={onNavigate}
            onRefresh={onRefresh}
          />
        </div>
      ) : null}

      {degradedComponents.length > 0 ? (
        <div className="system-status__section">
          <h3>Degraded components</h3>
          <ul className="system-status__list">
            {degradedComponents.map(([name, component]) => (
              <li key={name}>
                <strong>{name.replaceAll("_", " ")}</strong>
                <span>{component.message || component.status}</span>
                <CapabilityActionList
                  actions={component.actions ?? []}
                  onNavigate={onNavigate}
                  onRefresh={onRefresh}
                />
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {notices.length > 0 ? (
        <div className="system-status__section">
          <h3>Notices</h3>
          <ul className="system-status__list">
            {notices.map((notice) => (
              <li key={notice.id}>
                <strong>{notice.title}</strong>
                <span>{notice.message}</span>
                <CapabilityActionList
                  actions={notice.actions ?? []}
                  onNavigate={onNavigate}
                  onRefresh={onRefresh}
                />
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}
