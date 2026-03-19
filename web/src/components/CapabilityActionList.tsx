/**
 * Purpose: Render shared guided recovery actions for capability-aware product surfaces.
 * Responsibilities: Support copyable commands/env vars, route/doc links, external links, and inline one-click diagnostic checks with follow-up results.
 * Scope: Action-list presentation and diagnostic execution only; parent panels own layout, health data, and higher-level guidance copy.
 * Usage: Mount anywhere a `RecommendedAction[]` should render with consistent web behavior.
 * Invariants/Assumptions: One-click actions stay read-only, copy actions remain operator-friendly, and nested diagnostic results must not recurse infinitely.
 */

import { useCallback, useState } from "react";
import {
  postV1DiagnosticsAiCheck,
  postV1DiagnosticsBrowserCheck,
  postV1DiagnosticsProxyPoolCheck,
  type DiagnosticActionResponse,
  type RecommendedAction,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import { getApiErrorMessage } from "../lib/api-errors";

interface CapabilityActionListProps {
  actions: RecommendedAction[];
  onNavigate: (path: string) => void;
  onRefresh: () => Promise<unknown> | undefined;
}

interface ActionRunState {
  status: "idle" | "running" | "success" | "info" | "warning" | "error";
  result?: DiagnosticActionResponse;
  message?: string;
}

function actionKey(action: RecommendedAction) {
  return `${action.kind}:${action.label}:${action.value ?? ""}`;
}

function isExternalHref(value: string) {
  return /^https?:\/\//i.test(value);
}

function classifyDiagnosticRunStatus(
  responseStatus: DiagnosticActionResponse["status"],
): Exclude<ActionRunState["status"], "idle" | "running"> {
  switch (responseStatus) {
    case "ok":
      return "success";
    case "disabled":
      return "info";
    case "degraded":
      return "warning";
    default:
      return "error";
  }
}

function diagnosticResultClassName(
  responseStatus: DiagnosticActionResponse["status"],
) {
  switch (classifyDiagnosticRunStatus(responseStatus)) {
    case "success":
      return "system-status__action-result--ok";
    case "info":
      return "system-status__action-result--info";
    default:
      return "system-status__action-result--warning";
  }
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

export function CapabilityActionList({
  actions,
  onNavigate,
  onRefresh,
}: CapabilityActionListProps) {
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
            status: classifyDiagnosticRunStatus(payload.status),
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

    if (
      (action.kind === "command" ||
        action.kind === "copy" ||
        action.kind === "env") &&
      actionValue
    ) {
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
              className={`system-status__action-result ${diagnosticResultClassName(
                runState.result.status,
              )}`}
            >
              {runState.result.title ? (
                <strong>{runState.result.title}</strong>
              ) : null}
              <span>{runState.result.message}</span>
              {runState.result.actions?.length ? (
                <CapabilityActionList
                  actions={runState.result.actions}
                  onNavigate={onNavigate}
                  onRefresh={onRefresh}
                />
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

  if (actions.length === 0) {
    return null;
  }

  return (
    <div className="system-status__actions system-status__actions--stacked">
      {actions.map((action) => renderAction(action))}
    </div>
  );
}
