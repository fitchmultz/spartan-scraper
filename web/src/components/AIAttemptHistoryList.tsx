/**
 * Purpose: Render full AI attempt history with explicit selection, baseline, and restore actions.
 * Responsibilities: Show every attempt in the current modal session and provide a consistent control surface across all AI authoring modals.
 * Scope: Shared presentation for AI generator/debugger history only.
 * Usage: Mount above the candidate panels and wire it to `useAIAttemptHistory`.
 * Invariants/Assumptions: Attempts render newest-first, the selected attempt drives save behavior, and only older attempts can become the active baseline.
 */

import type { AIAttempt } from "../hooks/useAIAttemptHistory";

interface AIAttemptHistoryListProps<TArtifact> {
  attempts: AIAttempt<TArtifact>[];
  activeAttemptId: string | null;
  baselineAttemptId: string | null;
  onSelectAttempt: (attemptId: string) => void;
  onSelectBaseline: (attemptId: string) => void;
  onRestoreGuidance: (attempt: AIAttempt<TArtifact>) => void;
}

export function AIAttemptHistoryList<TArtifact>({
  attempts,
  activeAttemptId,
  baselineAttemptId,
  onSelectAttempt,
  onSelectBaseline,
  onRestoreGuidance,
}: AIAttemptHistoryListProps<TArtifact>) {
  const activeAttempt =
    attempts.find((attempt) => attempt.id === activeAttemptId) ?? null;
  const latestAttempt = attempts.at(-1) ?? null;

  if (attempts.length === 0) {
    return null;
  }

  return (
    <section
      aria-label="Attempt history"
      className="space-y-3 rounded-lg border border-slate-700 bg-slate-900/60 p-4"
    >
      <div className="text-sm font-medium text-slate-100">Attempt history</div>

      <ol className="space-y-2">
        {[...attempts].reverse().map((attempt) => {
          const isSelected = attempt.id === activeAttemptId;
          const isBaseline = attempt.id === baselineAttemptId;
          const isLatest = attempt.id === latestAttempt?.id;
          const baselineAllowed =
            !!activeAttempt &&
            attempt.id !== activeAttempt.id &&
            attempt.ordinal < activeAttempt.ordinal;

          return (
            <li key={attempt.id}>
              <section
                aria-label={`Attempt ${attempt.ordinal}`}
                className={`rounded-md border p-3 ${
                  isSelected
                    ? "border-sky-500/50 bg-sky-950/20"
                    : "border-slate-700 bg-slate-950/40"
                }`}
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-medium text-slate-100">
                        Attempt {attempt.ordinal}
                      </span>
                      {isLatest ? (
                        <span className="rounded-full bg-purple-500/15 px-2 py-1 text-[11px] text-purple-200">
                          Latest
                        </span>
                      ) : null}
                      {isSelected ? (
                        <span className="rounded-full bg-sky-500/15 px-2 py-1 text-[11px] text-sky-200">
                          Selected
                        </span>
                      ) : null}
                      {isBaseline ? (
                        <span className="rounded-full bg-amber-500/15 px-2 py-1 text-[11px] text-amber-200">
                          Baseline
                        </span>
                      ) : null}
                    </div>

                    <div className="text-xs text-slate-400">
                      {attempt.routeId || "Unknown route"}
                      {attempt.provider ? ` · ${attempt.provider}` : ""}
                      {attempt.model ? ` / ${attempt.model}` : ""}
                    </div>

                    {attempt.resolvedGoal?.text ? (
                      <p className="line-clamp-2 text-xs text-slate-300">
                        {attempt.resolvedGoal.text}
                      </p>
                    ) : null}
                  </div>

                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      className="button-secondary"
                      aria-label={`Select attempt ${attempt.ordinal}`}
                      onClick={() => onSelectAttempt(attempt.id)}
                      disabled={isSelected}
                    >
                      {isSelected ? "Selected" : "Select candidate"}
                    </button>

                    <button
                      type="button"
                      className="button-secondary"
                      aria-label={`Use attempt ${attempt.ordinal} as baseline`}
                      onClick={() => onSelectBaseline(attempt.id)}
                      disabled={!baselineAllowed}
                    >
                      {isBaseline ? "Using as baseline" : "Use as baseline"}
                    </button>

                    <button
                      type="button"
                      className="button-secondary"
                      aria-label={`Restore guidance from attempt ${attempt.ordinal}`}
                      onClick={() => onRestoreGuidance(attempt)}
                      disabled={!attempt.guidanceText}
                    >
                      Restore guidance
                    </button>
                  </div>
                </div>
              </section>
            </li>
          );
        })}
      </ol>
    </section>
  );
}
