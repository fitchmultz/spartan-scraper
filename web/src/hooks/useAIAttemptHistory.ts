/**
 * Purpose: Manage full per-session AI authoring attempt history for generator and debugger modals.
 * Responsibilities: Store ordered attempts, track the selected attempt and comparison baseline, and expose deterministic append/select/reset behavior.
 * Scope: Web-only session state for AI authoring flows.
 * Usage: Call from AI generation/debugging modals and append a normalized attempt after each successful AI response.
 * Invariants/Assumptions: History resets when the modal closes, new attempts become selected automatically, and baselines must always be older than the selected attempt.
 */

import { useMemo, useReducer } from "react";
import type { ResolvedGoal } from "../api";

export interface AIAttemptDraft<TArtifact> {
  artifact: TArtifact | null;
  guidanceText: string;
  resolvedGoal: ResolvedGoal | null;
  explanation: string;
  routeId: string;
  provider: string;
  model: string;
  visualContextUsed: boolean;
  issues: string[];
  recheckStatus?: number;
  recheckEngine?: string;
  recheckError?: string;
  rawResponse: unknown;
}

export interface AIAttempt<TArtifact> extends AIAttemptDraft<TArtifact> {
  id: string;
  ordinal: number;
}

interface AttemptHistoryState<TArtifact> {
  attempts: AIAttempt<TArtifact>[];
  activeAttemptId: string | null;
  baselineAttemptId: string | null;
}

type AttemptHistoryAction<TArtifact> =
  | { type: "append"; draft: AIAttemptDraft<TArtifact> }
  | { type: "select-active"; attemptId: string }
  | { type: "select-baseline"; attemptId: string | null }
  | { type: "reset" };

const INITIAL_STATE: AttemptHistoryState<never> = {
  attempts: [],
  activeAttemptId: null,
  baselineAttemptId: null,
};

function findAttempt<TArtifact>(
  attempts: readonly AIAttempt<TArtifact>[],
  attemptId: string | null,
): AIAttempt<TArtifact> | null {
  if (!attemptId) {
    return null;
  }

  return attempts.find((attempt) => attempt.id === attemptId) ?? null;
}

function getDefaultBaselineId<TArtifact>(
  attempts: readonly AIAttempt<TArtifact>[],
  activeAttemptId: string | null,
): string | null {
  const activeAttempt = findAttempt(attempts, activeAttemptId);
  if (!activeAttempt) {
    return null;
  }

  const olderAttempts = attempts.filter(
    (attempt) => attempt.ordinal < activeAttempt.ordinal,
  );

  return olderAttempts.at(-1)?.id ?? null;
}

function attemptHistoryReducer<TArtifact>(
  state: AttemptHistoryState<TArtifact>,
  action: AttemptHistoryAction<TArtifact>,
): AttemptHistoryState<TArtifact> {
  switch (action.type) {
    case "append": {
      const nextAttempt: AIAttempt<TArtifact> = {
        ...action.draft,
        id: `attempt-${state.attempts.length + 1}`,
        ordinal: state.attempts.length + 1,
      };
      const nextAttempts = [...state.attempts, nextAttempt];

      return {
        attempts: nextAttempts,
        activeAttemptId: nextAttempt.id,
        baselineAttemptId: nextAttempts.at(-2)?.id ?? null,
      };
    }

    case "select-active": {
      const activeAttempt = findAttempt(state.attempts, action.attemptId);
      if (!activeAttempt) {
        return state;
      }

      const currentBaseline = findAttempt(
        state.attempts,
        state.baselineAttemptId,
      );
      const keepCurrentBaseline =
        currentBaseline &&
        currentBaseline.id !== activeAttempt.id &&
        currentBaseline.ordinal < activeAttempt.ordinal;

      return {
        ...state,
        activeAttemptId: activeAttempt.id,
        baselineAttemptId: keepCurrentBaseline
          ? currentBaseline.id
          : getDefaultBaselineId(state.attempts, activeAttempt.id),
      };
    }

    case "select-baseline": {
      if (!action.attemptId) {
        return {
          ...state,
          baselineAttemptId: null,
        };
      }

      const activeAttempt = findAttempt(state.attempts, state.activeAttemptId);
      const baselineAttempt = findAttempt(state.attempts, action.attemptId);
      if (!activeAttempt || !baselineAttempt) {
        return state;
      }

      if (
        baselineAttempt.id === activeAttempt.id ||
        baselineAttempt.ordinal >= activeAttempt.ordinal
      ) {
        return state;
      }

      return {
        ...state,
        baselineAttemptId: baselineAttempt.id,
      };
    }

    case "reset":
      return INITIAL_STATE as AttemptHistoryState<TArtifact>;
  }
}

export function useAIAttemptHistory<TArtifact>() {
  const [state, dispatch] = useReducer(
    attemptHistoryReducer<TArtifact>,
    INITIAL_STATE as AttemptHistoryState<TArtifact>,
  );

  const activeAttempt = useMemo(
    () => findAttempt(state.attempts, state.activeAttemptId),
    [state.attempts, state.activeAttemptId],
  );

  const baselineAttempt = useMemo(
    () => findAttempt(state.attempts, state.baselineAttemptId),
    [state.attempts, state.baselineAttemptId],
  );

  return {
    attempts: state.attempts,
    activeAttemptId: state.activeAttemptId,
    baselineAttemptId: state.baselineAttemptId,
    activeAttempt,
    baselineAttempt,
    latestAttempt: state.attempts.at(-1) ?? null,
    appendAttempt: (draft: AIAttemptDraft<TArtifact>) =>
      dispatch({ type: "append", draft }),
    selectAttempt: (attemptId: string) =>
      dispatch({ type: "select-active", attemptId }),
    selectBaseline: (attemptId: string | null) =>
      dispatch({ type: "select-baseline", attemptId }),
    reset: () => dispatch({ type: "reset" }),
  };
}
