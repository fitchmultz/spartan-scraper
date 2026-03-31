/**
 * Purpose: Centralize shared tab-scoped AI authoring session state for generator and debugger modals.
 * Responsibilities: Own persisted draft state, wire attempt history, derive session-draft status, and provide consistent reset/discard confirmation flows.
 * Scope: Web-only AI authoring modal session orchestration; artifact-specific request and save logic stay in the caller.
 * Usage: Call from AI generator/debugger components and pass the returned state, history, and session actions into the modal UI.
 * Invariants/Assumptions: Session drafts persist in browser sessionStorage when a storage key is supplied, reset preserves current request inputs while clearing attempts, and discard clears both draft state and attempt history before closing.
 */

import {
  useCallback,
  useMemo,
  type Dispatch,
  type SetStateAction,
} from "react";
import {
  useAIAttemptHistory,
  type AIAttempt,
  type AIAttemptHistoryController,
} from "../../hooks/useAIAttemptHistory";
import { useSessionStorageState } from "../../hooks/useSessionStorageState";
import { useToast } from "../toast";

interface UseAIAuthoringSessionOptions<TArtifact, TState> {
  storageKey?: string;
  initialState: TState | (() => TState);
  providedHistory?: AIAttemptHistoryController<TArtifact>;
  hasSessionDraft: (state: TState, attemptCount: number) => boolean;
  clearError: (state: TState) => TState;
  resetDescription: string;
  discardDescription: string;
  onClose: () => void;
  onSessionCleared?: () => void;
}

interface AIAuthoringSessionController<TArtifact, TState> {
  state: TState;
  setState: Dispatch<SetStateAction<TState>>;
  history: AIAttemptHistoryController<TArtifact>;
  activeAttempt: AIAttempt<TArtifact> | null;
  baselineAttempt: AIAttempt<TArtifact> | null;
  latestAttempt: AIAttempt<TArtifact> | null;
  hasSessionDraft: boolean;
  clearSession: () => void;
  resetSession: () => Promise<void>;
  discardSession: () => Promise<void>;
}

export function useAIAuthoringSession<TArtifact, TState>({
  storageKey,
  initialState,
  providedHistory,
  hasSessionDraft: hasSessionDraftState,
  clearError,
  resetDescription,
  discardDescription,
  onClose,
  onSessionCleared,
}: UseAIAuthoringSessionOptions<
  TArtifact,
  TState
>): AIAuthoringSessionController<TArtifact, TState> {
  const toast = useToast();
  const [state, setState, clearState] = useSessionStorageState<TState>(
    storageKey ?? null,
    initialState,
  );
  const ownedHistory = useAIAttemptHistory<TArtifact>(
    storageKey ? { storageKey: `${storageKey}.history` } : undefined,
  );
  const history = providedHistory ?? ownedHistory;

  const hasSessionDraft = useMemo(
    () => hasSessionDraftState(state, history.attempts.length),
    [hasSessionDraftState, history.attempts.length, state],
  );

  const clearSession = useCallback(() => {
    clearState();
    history.reset();
    onSessionCleared?.();
  }, [clearState, history, onSessionCleared]);

  const resetSession = useCallback(async () => {
    const confirmed = await toast.confirm({
      title: "Reset this AI session?",
      description: resetDescription,
      confirmLabel: "Reset session",
      cancelLabel: "Keep session",
      tone: "warning",
    });
    if (!confirmed) {
      return;
    }

    history.reset();
    setState((previous) => clearError(previous));
  }, [clearError, history, resetDescription, setState, toast]);

  const discardSession = useCallback(async () => {
    if (!hasSessionDraft) {
      clearSession();
      onClose();
      return;
    }

    const confirmed = await toast.confirm({
      title: "Discard this AI session?",
      description: discardDescription,
      confirmLabel: "Discard session",
      cancelLabel: "Keep session",
      tone: "warning",
    });
    if (!confirmed) {
      return;
    }

    clearSession();
    onClose();
  }, [clearSession, discardDescription, hasSessionDraft, onClose, toast]);

  return {
    state,
    setState,
    history,
    activeAttempt: history.activeAttempt,
    baselineAttempt: history.baselineAttempt,
    latestAttempt: history.latestAttempt,
    hasSessionDraft,
    clearSession,
    resetSession,
    discardSession,
  };
}
