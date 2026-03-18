/**
 * Purpose: Render in-product setup and runtime diagnostics sourced from `/healthz`.
 * Responsibilities: Highlight setup-mode guidance, degraded components, and operator-facing notices with actionable follow-up steps.
 * Scope: Diagnostic panel presentation only.
 * Usage: Mount near the top of `App.tsx` and pass the latest health response plus navigation helpers.
 * Invariants/Assumptions: Healthy states should stay out of the way, while setup/degraded states must remain obvious and actionable.
 */

import { useCallback, useState } from "react";
import {
  postV1DiagnosticsAiCheck,
  postV1DiagnosticsBrowserCheck,
  postV1DiagnosticsProxyPoolCheck,
  type DiagnosticActionResponse,
  type HealthResponse,
  type RecommendedAction,
  type RuntimeNotice,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import { getApiErrorMessage } from "../lib/api-errors";

interface SystemStatusPanelProps {
  health: HealthResponse | null;
  onNavigate: (path: string) => void;
  onRefresh: () => Promise<unknown> | undefined;
}

interface ActionRunState {
  status: "idle" | "running" | "success" | "error";
  result?: DiagnosticActionResponse;
  message?: string;
}

function actionKey(action: RecommendedAction) {
  return `${action.kind}:${action.label}:${action.value ?? ""}`;
}

function isExternalHref(value: string) {
  return /^https?:\/\//i.test(value);
}

async function copyText(text: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      // Fall back to the legacy execCommand path when clipboard permissions are denied.
    }
  }

  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "true");
  textarea.style.position = "absolute";
  textarea.style.left = "-9999px";
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand("copy");
  document.body.removeChild(textarea);
}

async function executeDiagnosticAction(
  actionValue: string,
): Promise<DiagnosticActionResponse> {
  const baseUrl = getApiBaseUrl();

  switch (actionValue) {
    case "/v1/diagnostics/browser-check": {
      const response = await postV1DiagnosticsBrowserCheck({ baseUrl });
      if (response.data) {
        return response.data;
      }
      throw response.error ?? new Error("Browser diagnostic failed");
    }
    case "/v1/diagnostics/ai-check": {
      const response = await postV1DiagnosticsAiCheck({ baseUrl });
      if (response.data) {
        return response.data;
      }
      throw response.error ?? new Error("AI diagnostic failed");
    }
    case "/v1/diagnostics/proxy-pool-check": {
      const response = await postV1DiagnosticsProxyPoolCheck({ baseUrl });
      if (response.data) {
        return response.data;
      }
      throw response.error ?? new Error("Proxy-pool diagnostic failed");
    }
    default:
      throw new Error(`Unsupported diagnostic action: ${actionValue}`);
  }
}

function isVisibleNotice(notice: RuntimeNotice) {
  return notice.severity === "warning" || notice.severity === "error";
}

export function SystemStatusPanel({
  health,
  onNavigate,
  onRefresh,
}: SystemStatusPanelProps) {
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [actionRuns, setActionRuns] = useState<Record<string, ActionRunState>>(
    {},
  );

  const clearCopiedLater = useCallback((key: string) => {
    window.setTimeout(() => {
      setCopiedKey((current) => (current === key ? null : current));
    }, 1800);
  }, []);

  const handleCopy = useCallback(
    async (action: RecommendedAction) => {
      if (!action.value) {
        return;
      }
      const key = actionKey(action);
      await copyText(action.value);
      setCopiedKey(key);
      clearCopiedLater(key);
    },
    [clearCopiedLater],
  );

  const runOneClick = useCallback(
    async (action: RecommendedAction) => {
      if (!action.value) {
        return;
      }

      const key = actionKey(action);
      setActionRuns((current) => ({
        ...current,
        [key]: { status: "running" },
      }));

      try {
        const payload = await executeDiagnosticAction(action.value);
        setActionRuns((current) => ({
          ...current,
          [key]: {
            status: payload.status === "ok" ? "success" : "error",
            result: payload,
          },
        }));
        await Promise.resolve(onRefresh());
      } catch (error) {
        setActionRuns((current) => ({
          ...current,
          [key]: {
            status: "error",
            message: getApiErrorMessage(error, "Diagnostic check failed"),
          },
        }));
      }
    },
    [onRefresh],
  );

  const renderAction = (action: RecommendedAction, nested = false) => {
    const key = actionKey(action);
    const actionValue = action.value;
    const runState = actionRuns[key];

    if (action.kind === "route" && actionValue) {
      return (
        <button
          key={key}
          type="button"
          className="secondary"
          onClick={() => onNavigate(actionValue)}
        >
          {action.label}
        </button>
      );
    }

    if ((action.kind === "command" || action.kind === "copy") && actionValue) {
      return (
        <div key={key} className="system-status__action-card">
          <strong>{action.label}</strong>
          <div className="system-status__code-row">
            <code>{actionValue}</code>
            <button
              type="button"
              className="secondary"
              aria-label={`Copy ${action.label}`}
              onClick={() => {
                void handleCopy(action);
              }}
            >
              {copiedKey === key ? "Copied!" : "Copy"}
            </button>
          </div>
        </div>
      );
    }

    if (action.kind === "doc" && actionValue) {
      if (isExternalHref(actionValue)) {
        return (
          <a
            key={key}
            className="secondary system-status__link"
            href={actionValue}
            target="_blank"
            rel="noreferrer"
          >
            {action.label}
          </a>
        );
      }

      return (
        <button
          key={key}
          type="button"
          className="secondary"
          onClick={() => onNavigate(actionValue)}
        >
          {action.label}
        </button>
      );
    }

    if (action.kind === "external-link" && actionValue) {
      return (
        <a
          key={key}
          className="secondary system-status__link"
          href={actionValue}
          target="_blank"
          rel="noreferrer"
        >
          {action.label}
        </a>
      );
    }

    if (action.kind === "one-click" && actionValue) {
      return (
        <div key={key} className="system-status__action-card">
          <div className="system-status__action-inline">
            <strong>{action.label}</strong>
            <button
              type="button"
              className="secondary"
              aria-label={action.label}
              disabled={runState?.status === "running"}
              onClick={() => {
                void runOneClick(action);
              }}
            >
              {runState?.status === "running" ? "Running…" : "Run check"}
            </button>
          </div>

          {!nested && runState?.message ? (
            <div className="system-status__action-result system-status__action-result--warning">
              <span>{runState.message}</span>
            </div>
          ) : null}

          {!nested && runState?.result ? (
            <div
              className={`system-status__action-result ${
                runState.result.status === "ok"
                  ? "system-status__action-result--ok"
                  : "system-status__action-result--warning"
              }`}
            >
              {runState.result.title ? (
                <strong>{runState.result.title}</strong>
              ) : null}
              <span>{runState.result.message}</span>
              {runState.result.actions?.length ? (
                <div className="system-status__actions system-status__actions--stacked">
                  {runState.result.actions.map((resultAction) =>
                    renderAction(resultAction, true),
                  )}
                </div>
              ) : null}
            </div>
          ) : null}
        </div>
      );
    }

    return (
      <div key={key} className="system-status__hint">
        <strong>{action.label}</strong>
        {action.value ? <span>{action.value}</span> : null}
      </div>
    );
  };

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
              : "Core workflows stay available where possible. Use the guidance below to recover intentionally."}
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

          <div className="system-status__actions system-status__actions--stacked">
            {(health.setup.actions ?? []).map((action) => renderAction(action))}
          </div>
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
                {component.actions?.length ? (
                  <div className="system-status__actions system-status__actions--stacked">
                    {component.actions.map((action) => renderAction(action))}
                  </div>
                ) : null}
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
                {notice.actions?.length ? (
                  <div className="system-status__actions system-status__actions--stacked">
                    {notice.actions.map((action) => renderAction(action))}
                  </div>
                ) : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}
