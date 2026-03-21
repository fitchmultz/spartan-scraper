/**
 * Purpose: Render the exact AI goal Spartan resolved for automation authoring flows.
 * Responsibilities: Show the final goal text, distinguish explicit guidance from system-derived defaults, and stay visually consistent across AI result surfaces.
 * Scope: Shared result metadata card for render-profile and pipeline-JS generation/debugging modals.
 * Usage: Mount inside an AI result panel before explanation text.
 * Invariants/Assumptions: Goal text must match the API response verbatim and source is either `explicit` or `derived`.
 */

import type { ResolvedGoal } from "../api";

interface AIResolvedGoalCardProps {
  resolvedGoal?: ResolvedGoal | null;
}

function getResolvedGoalSourceLabel(source?: string): string {
  return source === "explicit" ? "Explicit" : "System-derived";
}

export function AIResolvedGoalCard({ resolvedGoal }: AIResolvedGoalCardProps) {
  if (!resolvedGoal?.text) {
    return null;
  }

  const explicit = resolvedGoal.source === "explicit";

  return (
    <section
      aria-label="Resolved goal"
      className="rounded-md border border-sky-700/40 bg-sky-950/30 p-4"
    >
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-sm font-medium text-slate-100">Resolved goal</h3>
        <span
          className={
            explicit
              ? "inline-flex rounded-full bg-emerald-500/15 px-2 py-1 text-[11px] font-medium text-emerald-300"
              : "inline-flex rounded-full bg-amber-500/15 px-2 py-1 text-[11px] font-medium text-amber-300"
          }
        >
          {getResolvedGoalSourceLabel(resolvedGoal.source)}
        </span>
      </div>
      <p className="whitespace-pre-wrap text-sm text-slate-200">
        {resolvedGoal.text}
      </p>
    </section>
  );
}
