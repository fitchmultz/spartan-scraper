/**
 * Purpose: Render the shared collapsible AI assistant shell used by route-specific assistant adapters.
 * Responsibilities: Show the assistant header, persisted collapse/width controls, route context summary, and a consistent body container for route-owned AI tools.
 * Scope: Shared AI shell presentation only; route adapters own API requests, validation, and explicit apply actions.
 * Usage: Wrap route-specific AI content inside `AIAssistantPanel` from `/jobs/new`, `/templates`, and `/jobs/:id` adapters.
 * Invariants/Assumptions: The panel reads context from `AIAssistantProvider`, collapsed state hides route content without unmounting the route surface, and route adapters provide all mutating actions explicitly.
 */

import { useMemo, type ReactNode } from "react";
import type { AssistantContext } from "./AIAssistantProvider";
import { useAIAssistant } from "./useAIAssistant";

interface AIAssistantPanelProps {
  title: string;
  routeLabel: string;
  suggestedActions?: ReactNode;
  children: ReactNode;
}

interface ContextEntry {
  label: string;
  value: string;
}

function truncateValue(value: string, maxLength = 96): string {
  if (value.length <= maxLength) {
    return value;
  }

  return `${value.slice(0, maxLength - 1)}…`;
}

function countSelectors(snapshot: Record<string, unknown> | undefined): number {
  const selectors = snapshot?.selectors;
  return Array.isArray(selectors) ? selectors.length : 0;
}

function buildContextEntries(context: AssistantContext | null): ContextEntry[] {
  if (!context) {
    return [];
  }

  if (context.surface === "job-submission") {
    const entries: ContextEntry[] = [
      {
        label: "Workflow",
        value: context.jobType,
      },
    ];

    if (context.url?.trim()) {
      entries.push({
        label: "Target",
        value: truncateValue(context.url.trim()),
      });
    }

    if (context.query?.trim()) {
      entries.push({
        label: "Query",
        value: truncateValue(context.query.trim()),
      });
    }

    if (context.templateName?.trim()) {
      entries.push({
        label: "Template",
        value: context.templateName.trim(),
      });
    }

    return entries;
  }

  if (context.surface === "templates") {
    const entries: ContextEntry[] = [
      {
        label: "Template",
        value: context.templateName?.trim() || "Unsaved workspace",
      },
      {
        label: "Selectors",
        value: String(countSelectors(context.templateSnapshot)),
      },
    ];

    if (context.selectedUrl?.trim()) {
      entries.push({
        label: "Preview URL",
        value: truncateValue(context.selectedUrl.trim()),
      });
    }

    return entries;
  }

  return [
    {
      label: "Job",
      value: context.jobId,
    },
    {
      label: "Format",
      value: context.resultFormat.toUpperCase(),
    },
    {
      label: "Selected item",
      value: String(context.selectedResultIndex + 1),
    },
    ...(context.resultSummary
      ? [
          {
            label: "Summary",
            value: truncateValue(context.resultSummary),
          },
        ]
      : []),
  ];
}

export function AIAssistantPanel({
  title,
  routeLabel,
  suggestedActions,
  children,
}: AIAssistantPanelProps) {
  const { close, context, isOpen, setWidth, toggle, width } = useAIAssistant();

  const contextEntries = useMemo(() => buildContextEntries(context), [context]);

  return (
    <aside
      className={`panel ai-assistant-panel ${isOpen ? "" : "is-collapsed"}`}
      style={isOpen ? { width } : undefined}
      aria-label={`${title} panel`}
    >
      {isOpen ? (
        <>
          <div className="ai-assistant-panel__header">
            <div>
              <div className="results-viewer__section-label">
                Integrated AI assistant
              </div>
              <h3>{title}</h3>
              <p>
                Route <code>{routeLabel}</code>
              </p>
            </div>

            <div className="ai-assistant-panel__header-actions">
              <button
                type="button"
                className="secondary ai-assistant-panel__toggle"
                onClick={toggle}
              >
                Collapse
              </button>
              <button
                type="button"
                className="secondary ai-assistant-panel__toggle"
                onClick={close}
              >
                Hide
              </button>
              <label className="ai-assistant-panel__width-control">
                <span>Width {width}px</span>
                <input
                  type="range"
                  min="340"
                  max="460"
                  step="10"
                  value={width}
                  onChange={(event) =>
                    setWidth(Number.parseInt(event.target.value, 10))
                  }
                />
              </label>
            </div>
          </div>

          {contextEntries.length > 0 ? (
            <section className="ai-assistant-panel__section">
              <div className="results-viewer__section-label">
                Current context
              </div>
              <ul className="ai-assistant-panel__context-list">
                {contextEntries.map((entry) => (
                  <li key={entry.label}>
                    <strong>{entry.label}:</strong> {entry.value}
                  </li>
                ))}
              </ul>
            </section>
          ) : null}

          {suggestedActions ? (
            <section className="ai-assistant-panel__section">
              <div className="results-viewer__section-label">
                Suggested actions
              </div>
              <div className="ai-assistant-panel__actions">
                {suggestedActions}
              </div>
            </section>
          ) : null}

          <div className="ai-assistant-panel__body">{children}</div>
        </>
      ) : (
        <div className="ai-assistant-panel__collapsed">
          <button
            type="button"
            className="secondary ai-assistant-panel__toggle"
            onClick={toggle}
          >
            Open AI
          </button>
          <span className="ai-assistant-panel__collapsed-label">
            {routeLabel}
          </span>
        </div>
      )}
    </aside>
  );
}
