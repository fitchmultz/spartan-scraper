/**
 * Onboarding State Management Hook
 *
 * Manages the onboarding flow state with localStorage persistence.
 * Tracks completion status, current step, and provides controls for
 * starting, skipping, and resetting the onboarding experience.
 *
 * Features:
 * - First-time visitor detection
 * - Step-by-step progress tracking
 * - Query param override (?showHelp) for testing
 * - Resumable tours
 *
 * @module useOnboarding
 */

import { useState, useEffect, useCallback, useMemo } from "react";

const STORAGE_KEY = "spartan-onboarding";

export interface OnboardingState {
  /** Whether the user has completed the full onboarding tour */
  hasCompletedOnboarding: boolean;
  /** Whether the user has explicitly skipped onboarding */
  hasSkippedOnboarding: boolean;
  /** Whether the user has started the onboarding tour */
  hasStartedOnboarding: boolean;
  /** Current step index (0-based) */
  currentStep: number;
  /** Array of completed step indices */
  completedSteps: number[];
  /** ISO timestamp of first visit */
  firstVisitAt: string | null;
  /** Whether this is the first time the hook has loaded (for showing welcome) */
  isFirstLoad: boolean;
}

export interface OnboardingActions {
  /** Start the onboarding tour from the beginning */
  startOnboarding: () => void;
  /** Skip the onboarding tour entirely */
  skipOnboarding: () => void;
  /** Reset onboarding state to start fresh */
  resetOnboarding: () => void;
  /** Mark the current step as complete and advance */
  completeStep: (stepIndex: number) => void;
  /** Go to a specific step */
  goToStep: (stepIndex: number) => void;
  /** Mark the entire tour as complete */
  finishOnboarding: () => void;
  /** Show welcome modal (if first visit and not skipped/completed) */
  shouldShowWelcome: boolean;
  /** Whether the tour is currently active */
  isTourActive: boolean;
  /** Total number of steps in the tour */
  totalSteps: number;
}

export type UseOnboardingReturn = OnboardingState & OnboardingActions;

const DEFAULT_STATE: OnboardingState = {
  hasCompletedOnboarding: false,
  hasSkippedOnboarding: false,
  hasStartedOnboarding: false,
  currentStep: 0,
  completedSteps: [],
  firstVisitAt: null,
  isFirstLoad: true,
};

const TOTAL_STEPS = 7;

/**
 * Load onboarding state from localStorage.
 */
function loadStoredState(): Partial<OnboardingState> | null {
  if (typeof window === "undefined") return null;
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored);
      if (typeof parsed === "object" && parsed !== null) {
        return parsed as Partial<OnboardingState>;
      }
    }
  } catch {
    // localStorage may be unavailable or data corrupted
  }
  return null;
}

/**
 * Save onboarding state to localStorage.
 */
function saveState(state: OnboardingState): void {
  if (typeof window === "undefined") return;
  try {
    // Don't persist isFirstLoad or isTourActive (runtime state only)
    const { isFirstLoad: _, ...stateToSave } = state;
    localStorage.setItem(STORAGE_KEY, JSON.stringify(stateToSave));
  } catch {
    // localStorage may be unavailable
  }
}

/**
 * Check if URL has ?showHelp query param for forcing onboarding.
 */
function hasShowHelpParam(): boolean {
  if (typeof window === "undefined") return false;
  const params = new URLSearchParams(window.location.search);
  return params.has("showHelp");
}

/**
 * React hook for managing onboarding state and flow.
 *
 * @example
 * ```tsx
 * function App() {
 *   const {
 *     shouldShowWelcome,
 *     startOnboarding,
 *     skipOnboarding,
 *     isTourActive,
 *     currentStep
 *   } = useOnboarding();
 *
 *   return (
 *     <>
 *       {shouldShowWelcome && (
 *         <WelcomeModal
 *           onStart={startOnboarding}
 *           onSkip={skipOnboarding}
 *         />
 *       )}
 *       {isTourActive && (
 *         <OnboardingFlow
 *           currentStep={currentStep}
 *           onComplete={finishOnboarding}
 *         />
 *       )}
 *     </>
 *   );
 * }
 * ```
 */
export function useOnboarding(): UseOnboardingReturn {
  const [state, setState] = useState<OnboardingState>(DEFAULT_STATE);
  const [isLoaded, setIsLoaded] = useState(false);

  // Load from localStorage on mount
  useEffect(() => {
    const stored = loadStoredState();
    const isForced = hasShowHelpParam();

    if (stored) {
      setState({
        ...DEFAULT_STATE,
        ...stored,
        // Reset completion if forced via query param
        hasCompletedOnboarding: isForced
          ? false
          : (stored.hasCompletedOnboarding ?? false),
        hasSkippedOnboarding: isForced
          ? false
          : (stored.hasSkippedOnboarding ?? false),
        // Reset first visit detection if forced
        firstVisitAt: stored.firstVisitAt ?? new Date().toISOString(),
        isFirstLoad: true,
      });
    } else {
      // First-time visitor
      setState({
        ...DEFAULT_STATE,
        firstVisitAt: new Date().toISOString(),
        isFirstLoad: true,
      });
    }

    setIsLoaded(true);
  }, []);

  // Persist to localStorage when state changes
  useEffect(() => {
    if (!isLoaded) return;
    saveState(state);
  }, [state, isLoaded]);

  // Clear isFirstLoad after initial render
  useEffect(() => {
    if (state.isFirstLoad && isLoaded) {
      // Small delay to allow components to mount and check shouldShowWelcome
      const timer = setTimeout(() => {
        setState((prev) => ({ ...prev, isFirstLoad: false }));
      }, 100);
      return () => clearTimeout(timer);
    }
  }, [state.isFirstLoad, isLoaded]);

  /**
   * Start the onboarding tour from the beginning.
   */
  const startOnboarding = useCallback(() => {
    setState((prev) => ({
      ...prev,
      hasStartedOnboarding: true,
      currentStep: 0,
      completedSteps: [],
      hasSkippedOnboarding: false,
    }));
  }, []);

  /**
   * Skip the onboarding tour entirely.
   */
  const skipOnboarding = useCallback(() => {
    setState((prev) => ({
      ...prev,
      hasSkippedOnboarding: true,
      currentStep: 0,
      completedSteps: [],
    }));
  }, []);

  /**
   * Reset onboarding state to start fresh.
   */
  const resetOnboarding = useCallback(() => {
    setState({
      ...DEFAULT_STATE,
      hasStartedOnboarding: true,
      firstVisitAt: new Date().toISOString(),
      isFirstLoad: false,
    });
  }, []);

  /**
   * Mark the current step as complete and advance.
   */
  const completeStep = useCallback((stepIndex: number) => {
    setState((prev) => ({
      ...prev,
      completedSteps: [...new Set([...prev.completedSteps, stepIndex])],
      currentStep: stepIndex + 1,
    }));
  }, []);

  /**
   * Go to a specific step.
   */
  const goToStep = useCallback((stepIndex: number) => {
    setState((prev) => ({
      ...prev,
      currentStep: Math.max(0, Math.min(stepIndex, TOTAL_STEPS - 1)),
    }));
  }, []);

  /**
   * Mark the entire tour as complete.
   */
  const finishOnboarding = useCallback(() => {
    setState((prev) => ({
      ...prev,
      hasCompletedOnboarding: true,
      hasStartedOnboarding: false,
      hasSkippedOnboarding: false,
      completedSteps: Array.from({ length: TOTAL_STEPS }, (_, i) => i),
      currentStep: 0,
    }));
  }, []);

  /**
   * Whether to show the welcome modal.
   * Show if: first load, not completed, not skipped, and is loaded
   */
  const shouldShowWelcome = useMemo(() => {
    if (!isLoaded) return false;
    if (!state.isFirstLoad) return false;
    if (state.hasCompletedOnboarding) return false;
    if (state.hasSkippedOnboarding) return false;
    return true;
  }, [
    isLoaded,
    state.isFirstLoad,
    state.hasCompletedOnboarding,
    state.hasSkippedOnboarding,
  ]);

  /**
   * Whether the tour is currently active.
   * Active if: started but not completed and not skipped
   */
  const isTourActive = useMemo(() => {
    if (state.hasCompletedOnboarding) return false;
    if (state.hasSkippedOnboarding) return false;
    return state.hasStartedOnboarding;
  }, [
    state.hasCompletedOnboarding,
    state.hasSkippedOnboarding,
    state.hasStartedOnboarding,
  ]);

  return {
    ...state,
    startOnboarding,
    skipOnboarding,
    resetOnboarding,
    completeStep,
    goToStep,
    finishOnboarding,
    shouldShowWelcome,
    isTourActive,
    totalSteps: TOTAL_STEPS,
  };
}
