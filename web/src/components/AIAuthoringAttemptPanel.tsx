/**
 * Purpose: Render a consistent comparison panel for AI authoring attempts.
 * Responsibilities: Show attempt labels, request metadata, diagnostics, resolved goals, explanations, operator-controlled response JSON, and caller-supplied artifact previews.
 * Scope: Shared Web UI presentation for AI automation generator and debugger results.
 * Usage: Mount for selected and baseline candidates inside AI authoring modals.
 * Invariants/Assumptions: The caller owns artifact-specific preview markup, and raw-response inspection must remain operator-controlled.
 */

import { useState, type ReactNode } from "react";
import { AIResolvedGoalCard } from "./AIResolvedGoalCard";
import type { ResolvedGoal } from "../api";

interface AIAuthoringAttemptPanelProps {
  label: string;
  routeId?: string;
  provider?: string;
  model?: string;
  visualContextUsed?: boolean;
  recheckStatus?: number;
  recheckEngine?: string;
  recheckError?: string;
  issues?: string[];
  resolvedGoal?: ResolvedGoal | null;
  explanation?: string;
  rawResponse?: unknown;
  muted?: boolean;
  children?: ReactNode;
}

export function AIAuthoringAttemptPanel({
  label,
  routeId,
  provider,
  model,
  visualContextUsed = false,
  recheckStatus,
  recheckEngine,
  recheckError,
  issues = [],
  resolvedGoal,
  explanation,
  rawResponse,
  muted = false,
  children,
}: AIAuthoringAttemptPanelProps) {
  const [showRawResponse, setShowRawResponse] = useState(false);

  return (
    <section
      aria-label={label}
      className={`space-y-4 rounded-lg border p-4 ${
        muted
          ? "border-slate-800 bg-slate-950/40 opacity-80"
          : "border-slate-700 bg-slate-900/60"
      }`}
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="text-sm font-medium text-slate-100">{label}</div>
        {rawResponse ? (
          <button
            type="button"
            className="button-secondary"
            onClick={() => setShowRawResponse((current) => !current)}
          >
            {showRawResponse ? "Hide response JSON" : "Show response JSON"}
          </button>
        ) : null}
      </div>

      <div className="space-y-2 text-sm text-slate-300">
        {recheckStatus ? (
          <div>
            <span className="font-medium text-slate-100">Recheck:</span> HTTP{" "}
            {recheckStatus}
            {recheckEngine ? ` via ${recheckEngine}` : ""}
          </div>
        ) : null}
        {recheckError ? <div>{recheckError}</div> : null}
        <div className="flex flex-wrap items-center gap-2 text-xs text-slate-300">
          {routeId ? <span>Route: {routeId}</span> : null}
          {provider ? <span>Provider: {provider}</span> : null}
          {model ? <span>Model: {model}</span> : null}
          {visualContextUsed ? <span>Visual context used</span> : null}
        </div>
      </div>

      {issues.length > 0 ? (
        <div>
          <h3 className="mb-2 text-sm font-medium text-slate-100">
            Detected issues
          </h3>
          <ul className="list-disc space-y-1 pl-5 text-sm text-slate-300">
            {issues.map((issue) => (
              <li key={issue}>{issue}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <AIResolvedGoalCard resolvedGoal={resolvedGoal} />

      {explanation ? (
        <p className="text-sm text-slate-200">{explanation}</p>
      ) : null}

      {children}

      {showRawResponse ? (
        <pre className="ai-candidate-raw-json">
          {JSON.stringify(rawResponse, null, 2)}
        </pre>
      ) : null}
    </section>
  );
}
