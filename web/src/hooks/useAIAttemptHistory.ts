/**
 * Purpose: Manage full per-session AI authoring attempt history for generator and debugger modals.
 * Responsibilities: Store ordered attempts, track the selected attempt and comparison baseline, support in-place manual-edit replacement, and optionally persist tab-scoped session history across component unmounts.
 * Scope: Web-only session state for AI authoring flows.
 * Usage: Call from parent editors or AI modals and append a normalized attempt after each successful AI response.
 * Invariants/Assumptions: New attempts become selected automatically, baselines must always be older than the selected attempt, replacing an attempt keeps later history intact, and storage persistence must fail open to in-memory behavior.
 */

import { useEffect, useMemo, useReducer } from "react";
import type { ResolvedGoal } from "../api";

export interface AIAttemptManualEdit {
  edited: boolean;
  count: number;
  updatedAt: string | null;
}

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
  manualEdit?: Partial<AIAttemptManualEdit>;
}

export interface AIAttempt<TArtifact> extends AIAttemptDraft<TArtifact> {
  id: string;
  ordinal: number;
  manualEdit: AIAttemptManualEdit;
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
  | {
      type: "replace-artifact";
      attemptId: string;
      artifact: TArtifact;
      markManualEdit: boolean;
    }
  | { type: "reset" };

const EMPTY_MANUAL_EDIT: AIAttemptManualEdit = {
  edited: false,
  count: 0,
  updatedAt: null,
};

const INITIAL_STATE: AttemptHistoryState<never> = {
  attempts: [],
  activeAttemptId: null,
  baselineAttemptId: null,
};

function getSessionStorage(): Storage | null {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    return window.sessionStorage;
  } catch {
    return null;
  }
}

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

function normalizeAttempt<TArtifact>(
  attempt: Partial<AIAttempt<TArtifact>> | null | undefined,
  index: number,
): AIAttempt<TArtifact> | null {
  if (!attempt) {
    return null;
  }

  return {
    artifact:
      (attempt.artifact as AIAttempt<TArtifact>["artifact"] | undefined) ??
      null,
    guidanceText: attempt.guidanceText ?? "",
    resolvedGoal: attempt.resolvedGoal ?? null,
    explanation: attempt.explanation ?? "",
    routeId: attempt.routeId ?? "",
    provider: attempt.provider ?? "",
    model: attempt.model ?? "",
    visualContextUsed: attempt.visualContextUsed ?? false,
    issues: Array.isArray(attempt.issues) ? attempt.issues : [],
    recheckStatus: attempt.recheckStatus,
    recheckEngine: attempt.recheckEngine,
    recheckError: attempt.recheckError,
    rawResponse: attempt.rawResponse,
    id: attempt.id?.trim() || `attempt-${index + 1}`,
    ordinal:
      typeof attempt.ordinal === "number" && attempt.ordinal > 0
        ? attempt.ordinal
        : index + 1,
    manualEdit: {
      ...EMPTY_MANUAL_EDIT,
      ...(attempt.manualEdit ?? {}),
    },
  };
}

function normalizeAttemptHistoryState<TArtifact>(
  value: unknown,
): AttemptHistoryState<TArtifact> {
  if (!value || typeof value !== "object") {
    return INITIAL_STATE as AttemptHistoryState<TArtifact>;
  }

  const candidate = value as Partial<AttemptHistoryState<TArtifact>>;
  const attempts = Array.isArray(candidate.attempts)
    ? candidate.attempts
        .map((attempt, index) => normalizeAttempt(attempt, index))
        .filter((attempt): attempt is AIAttempt<TArtifact> => attempt !== null)
    : [];

  const activeAttemptId =
    typeof candidate.activeAttemptId === "string" &&
    attempts.some((attempt) => attempt.id === candidate.activeAttemptId)
      ? candidate.activeAttemptId
      : (attempts.at(-1)?.id ?? null);

  const baselineAttemptId =
    typeof candidate.baselineAttemptId === "string" &&
    attempts.some((attempt) => attempt.id === candidate.baselineAttemptId)
      ? candidate.baselineAttemptId
      : getDefaultBaselineId(attempts, activeAttemptId);

  return {
    attempts,
    activeAttemptId,
    baselineAttemptId,
  };
}

function readPersistedState<TArtifact>(storageKey: string) {
  const storage = getSessionStorage();
  if (!storage) {
    return INITIAL_STATE as AttemptHistoryState<TArtifact>;
  }

  try {
    const raw = storage.getItem(storageKey);
    if (!raw) {
      return INITIAL_STATE as AttemptHistoryState<TArtifact>;
    }

    return normalizeAttemptHistoryState<TArtifact>(JSON.parse(raw));
  } catch {
    storage.removeItem(storageKey);
    return INITIAL_STATE as AttemptHistoryState<TArtifact>;
  }
}

function persistState<TArtifact>(
  storageKey: string,
  state: AttemptHistoryState<TArtifact>,
) {
  const storage = getSessionStorage();
  if (!storage) {
    return;
  }

  try {
    if (state.attempts.length === 0) {
      storage.removeItem(storageKey);
      return;
    }

    storage.setItem(storageKey, JSON.stringify(state));
  } catch {
    // Ignore storage failures; in-memory history still works.
  }
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
        manualEdit: {
          ...EMPTY_MANUAL_EDIT,
          ...action.draft.manualEdit,
        },
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

    case "replace-artifact": {
      const nextAttempts = state.attempts.map((attempt) => {
        if (attempt.id !== action.attemptId) {
          return attempt;
        }

        return {
          ...attempt,
          artifact: action.artifact,
          manualEdit: action.markManualEdit
            ? {
                edited: true,
                count: attempt.manualEdit.count + 1,
                updatedAt: new Date().toISOString(),
              }
            : attempt.manualEdit,
        };
      });

      return {
        attempts: nextAttempts,
        activeAttemptId: action.attemptId,
        baselineAttemptId: getDefaultBaselineId(nextAttempts, action.attemptId),
      };
    }

    case "reset":
      return INITIAL_STATE as AttemptHistoryState<TArtifact>;
  }
}

export interface AIAttemptHistoryController<TArtifact> {
  attempts: AIAttempt<TArtifact>[];
  activeAttemptId: string | null;
  baselineAttemptId: string | null;
  activeAttempt: AIAttempt<TArtifact> | null;
  baselineAttempt: AIAttempt<TArtifact> | null;
  latestAttempt: AIAttempt<TArtifact> | null;
  appendAttempt: (draft: AIAttemptDraft<TArtifact>) => void;
  selectAttempt: (attemptId: string) => void;
  selectBaseline: (attemptId: string | null) => void;
  replaceArtifact: (
    attemptId: string,
    artifact: TArtifact,
    options?: { markManualEdit?: boolean },
  ) => void;
  reset: () => void;
}

interface UseAIAttemptHistoryOptions {
  storageKey?: string;
}

export function useAIAttemptHistory<TArtifact>(
  options?: UseAIAttemptHistoryOptions,
): AIAttemptHistoryController<TArtifact> {
  const [state, dispatch] = useReducer(
    attemptHistoryReducer<TArtifact>,
    options?.storageKey,
    (storageKey) =>
      storageKey
        ? readPersistedState<TArtifact>(storageKey)
        : (INITIAL_STATE as AttemptHistoryState<TArtifact>),
  );

  useEffect(() => {
    if (!options?.storageKey) {
      return;
    }

    persistState(options.storageKey, state);
  }, [options?.storageKey, state]);

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
    replaceArtifact: (attemptId, artifact, options) =>
      dispatch({
        type: "replace-artifact",
        attemptId,
        artifact,
        markManualEdit: options?.markManualEdit ?? true,
      }),
    reset: () => dispatch({ type: "reset" }),
  };
}
