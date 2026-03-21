/**
 * Purpose: Render a consistent comparison panel for AI authoring attempts.
 * Responsibilities: Show attempt labels, request metadata, optional diagnostics, resolved goals, explanations, and caller-supplied artifact previews.
 * Scope: Shared Web UI presentation for AI automation generator and debugger results.
 * Usage: Mount for latest and previous candidates inside AI authoring modals.
 * Invariants/Assumptions: The caller owns artifact-specific preview markup and passes only request-scoped metadata from the latest or previous attempt.
 */

import type { ReactNode } from "react";
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
  muted = false,
  children,
}: AIAuthoringAttemptPanelProps) {
  return (
    <section
      aria-label={label}
      className={`space-y-4 rounded-lg border p-4 ${
        muted
          ? "border-slate-800 bg-slate-950/40 opacity-80"
          : "border-slate-700 bg-slate-900/60"
      }`}
    >
      <div className="text-sm font-medium text-slate-100">{label}</div>

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
    </section>
  );
}
