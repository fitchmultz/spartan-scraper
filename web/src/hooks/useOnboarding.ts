/**
 * Purpose: Persist progressive onboarding state for the web app across sessions.
 * Responsibilities: Track first-run nudges, full-tour progress, and onboarding reset semantics in localStorage-backed state.
 * Scope: Onboarding state management only; rendering lives in route help, nudge, and tour components.
 * Usage: Call from `App.tsx` once and pass the returned state/actions into the onboarding UI surfaces.
 * Invariants/Assumptions: Restarting onboarding should launch the full tour immediately, route help stays available after tour skip, and `?showHelp` forces a fresh tour without restoring the old blocking modal model.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { ONBOARDING_TOTAL_STEPS } from "../lib/onboarding";

const STORAGE_KEY = "spartan-onboarding";

export interface OnboardingState {
  hasCompletedOnboarding: boolean;
  hasSkippedOnboarding: boolean;
  hasStartedOnboarding: boolean;
  hasDismissedFirstRunHint: boolean;
  currentStep: number;
  completedSteps: number[];
  firstVisitAt: string | null;
  isFirstLoad: boolean;
}

export interface OnboardingActions {
  startOnboarding: () => void;
  skipOnboarding: () => void;
  resetOnboarding: () => void;
  completeStep: (stepIndex: number) => void;
  goToStep: (stepIndex: number) => void;
  finishOnboarding: () => void;
  dismissFirstRunHint: () => void;
  shouldShowFirstRunHint: boolean;
  isTourActive: boolean;
  totalSteps: number;
}

export type UseOnboardingReturn = OnboardingState & OnboardingActions;

export interface UseOnboardingOptions {
  hasStartedWork?: boolean;
}

const DEFAULT_STATE: OnboardingState = {
  hasCompletedOnboarding: false,
  hasSkippedOnboarding: false,
  hasStartedOnboarding: false,
  hasDismissedFirstRunHint: false,
  currentStep: 0,
  completedSteps: [],
  firstVisitAt: null,
  isFirstLoad: true,
};

function hasShowHelpParam(): boolean {
  if (typeof window === "undefined") {
    return false;
  }

  return new URLSearchParams(window.location.search).has("showHelp");
}

function loadStoredState(): Partial<OnboardingState> | null {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (!stored) {
      return null;
    }

    const parsed = JSON.parse(stored);
    return typeof parsed === "object" && parsed !== null
      ? (parsed as Partial<OnboardingState>)
      : null;
  } catch {
    return null;
  }
}

function saveState(state: OnboardingState): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    const { isFirstLoad: _runtimeOnly, ...persisted } = state;
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(persisted));
  } catch {
    // ignore storage failures
  }
}

function buildInitialState(): OnboardingState {
  const stored = loadStoredState();
  const forced = hasShowHelpParam();

  if (!stored) {
    return {
      ...DEFAULT_STATE,
      hasStartedOnboarding: forced,
      hasDismissedFirstRunHint: forced,
      firstVisitAt: new Date().toISOString(),
      isFirstLoad: !forced,
    };
  }

  return {
    ...DEFAULT_STATE,
    ...stored,
    hasCompletedOnboarding: forced
      ? false
      : (stored.hasCompletedOnboarding ?? false),
    hasSkippedOnboarding: forced
      ? false
      : (stored.hasSkippedOnboarding ?? false),
    hasStartedOnboarding: forced
      ? true
      : (stored.hasStartedOnboarding ?? false),
    hasDismissedFirstRunHint: forced
      ? true
      : (stored.hasDismissedFirstRunHint ?? false),
    currentStep: forced ? 0 : (stored.currentStep ?? 0),
    firstVisitAt: stored.firstVisitAt ?? new Date().toISOString(),
    isFirstLoad: forced
      ? false
      : !stored.hasCompletedOnboarding &&
        !stored.hasSkippedOnboarding &&
        !(stored.hasDismissedFirstRunHint ?? false),
  };
}

export function useOnboarding(
  options: UseOnboardingOptions = {},
): UseOnboardingReturn {
  const { hasStartedWork = false } = options;
  const [state, setState] = useState<OnboardingState>(() =>
    buildInitialState(),
  );

  const updateState = useCallback(
    (
      updater:
        | OnboardingState
        | ((previousState: OnboardingState) => OnboardingState),
    ) => {
      setState((previousState) => {
        const nextState =
          typeof updater === "function" ? updater(previousState) : updater;
        saveState(nextState);
        return nextState;
      });
    },
    [],
  );

  const startOnboarding = useCallback(() => {
    updateState((previous) => ({
      ...previous,
      hasStartedOnboarding: true,
      hasSkippedOnboarding: false,
      hasCompletedOnboarding: false,
      hasDismissedFirstRunHint: true,
      currentStep: 0,
      completedSteps: [],
      isFirstLoad: false,
    }));
  }, [updateState]);

  const skipOnboarding = useCallback(() => {
    updateState((previous) => ({
      ...previous,
      hasStartedOnboarding: false,
      hasSkippedOnboarding: true,
      hasDismissedFirstRunHint: true,
      currentStep: 0,
      completedSteps: [],
      isFirstLoad: false,
    }));
  }, [updateState]);

  const resetOnboarding = useCallback(() => {
    updateState((previous) => ({
      ...previous,
      hasCompletedOnboarding: false,
      hasSkippedOnboarding: false,
      hasStartedOnboarding: true,
      hasDismissedFirstRunHint: true,
      currentStep: 0,
      completedSteps: [],
      isFirstLoad: false,
    }));
  }, [updateState]);

  const completeStep = useCallback(
    (stepIndex: number) => {
      updateState((previous) => ({
        ...previous,
        completedSteps: [...new Set([...previous.completedSteps, stepIndex])],
        currentStep: Math.min(stepIndex + 1, ONBOARDING_TOTAL_STEPS - 1),
      }));
    },
    [updateState],
  );

  const goToStep = useCallback(
    (stepIndex: number) => {
      updateState((previous) => ({
        ...previous,
        currentStep: Math.max(
          0,
          Math.min(stepIndex, ONBOARDING_TOTAL_STEPS - 1),
        ),
      }));
    },
    [updateState],
  );

  const finishOnboarding = useCallback(() => {
    updateState((previous) => ({
      ...previous,
      hasCompletedOnboarding: true,
      hasStartedOnboarding: false,
      hasSkippedOnboarding: false,
      hasDismissedFirstRunHint: true,
      completedSteps: Array.from(
        { length: ONBOARDING_TOTAL_STEPS },
        (_, index) => index,
      ),
      currentStep: 0,
      isFirstLoad: false,
    }));
  }, [updateState]);

  const dismissFirstRunHint = useCallback(() => {
    updateState((previous) => ({
      ...previous,
      hasDismissedFirstRunHint: true,
      isFirstLoad: false,
    }));
  }, [updateState]);

  useEffect(() => {
    if (!hasStartedWork || state.hasDismissedFirstRunHint) {
      return;
    }

    updateState((previous) => ({
      ...previous,
      hasDismissedFirstRunHint: true,
      isFirstLoad: false,
    }));
  }, [hasStartedWork, state.hasDismissedFirstRunHint, updateState]);

  const shouldShowFirstRunHint = useMemo(
    () =>
      !hasStartedWork &&
      !state.hasCompletedOnboarding &&
      !state.hasSkippedOnboarding &&
      !state.hasDismissedFirstRunHint &&
      state.isFirstLoad,
    [
      hasStartedWork,
      state.hasCompletedOnboarding,
      state.hasDismissedFirstRunHint,
      state.hasSkippedOnboarding,
      state.isFirstLoad,
    ],
  );

  const isTourActive = useMemo(
    () =>
      !state.hasCompletedOnboarding &&
      !state.hasSkippedOnboarding &&
      state.hasStartedOnboarding,
    [
      state.hasCompletedOnboarding,
      state.hasSkippedOnboarding,
      state.hasStartedOnboarding,
    ],
  );

  return {
    ...state,
    startOnboarding,
    skipOnboarding,
    resetOnboarding,
    completeStep,
    goToStep,
    finishOnboarding,
    dismissFirstRunHint,
    shouldShowFirstRunHint,
    isTourActive,
    totalSteps: ONBOARDING_TOTAL_STEPS,
  };
}
