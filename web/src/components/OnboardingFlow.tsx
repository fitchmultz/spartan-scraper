/**
 * Purpose: Render the route-aware onboarding tour that introduces the major web workflows.
 * Responsibilities: Transform shared onboarding step config into Joyride steps, synchronize route changes for cross-route tour progress, and report completion/skip events back to onboarding state.
 * Scope: Guided-tour presentation only.
 * Usage: Mount once from `App.tsx` with the current onboarding state and route key.
 * Invariants/Assumptions: Tour targets must exist on the active route or fall back safely to `body`, and the route key must stay aligned with the application shell's top-level routes.
 */

import { useEffect, useMemo } from "react";
import Joyride, {
  ACTIONS,
  EVENTS,
  STATUS,
  type CallBackProps,
  type Step,
  type Styles,
} from "react-joyride";
import {
  ONBOARDING_TOUR_STEPS,
  ONBOARDING_TOTAL_STEPS,
  type OnboardingRouteKey,
} from "../lib/onboarding";

export interface OnboardingFlowProps {
  isRunning: boolean;
  currentStep: number;
  currentRoute: OnboardingRouteKey;
  onComplete: () => void;
  onSkip?: () => void;
  onStepChange?: (stepIndex: number) => void;
  onRouteChange?: (route: OnboardingRouteKey) => void;
}

const joyrideStyles: Partial<Styles> = {
  options: {
    arrowColor: "var(--panel, #1a1a24)",
    backgroundColor: "var(--panel, #1a1a24)",
    overlayColor: "rgba(0, 0, 0, 0.7)",
    primaryColor: "var(--accent, #ffb700)",
    textColor: "var(--text, #fff)",
    zIndex: 1100,
  },
  tooltip: {
    backgroundColor: "var(--panel, #1a1a24)",
    border: "1px solid var(--stroke, rgba(255,255,255,0.1))",
    borderRadius: 16,
    boxShadow: "0 16px 48px rgba(0,0,0,0.5)",
    fontSize: "0.9rem",
    padding: 20,
  },
};

export function OnboardingFlow({
  isRunning,
  currentStep,
  currentRoute,
  onComplete,
  onSkip,
  onStepChange,
  onRouteChange,
}: OnboardingFlowProps) {
  const steps = useMemo<Step[]>(
    () =>
      ONBOARDING_TOUR_STEPS.map((step) => ({
        target: step.target,
        placement: step.placement ?? "bottom",
        disableBeacon: step.disableBeacon ?? false,
        content: (
          <div>
            <h3 style={{ margin: "0 0 8px", fontSize: "1rem" }}>
              {step.title}
            </h3>
            <p style={{ margin: 0, lineHeight: 1.5 }}>{step.body}</p>
            {step.bullets?.length ? (
              <ul
                style={{ margin: "12px 0 0", paddingLeft: 18, lineHeight: 1.7 }}
              >
                {step.bullets.map((bullet) => (
                  <li key={bullet}>{bullet}</li>
                ))}
              </ul>
            ) : null}
          </div>
        ),
      })),
    [],
  );

  useEffect(() => {
    if (!isRunning) {
      return;
    }

    const activeStep = ONBOARDING_TOUR_STEPS[currentStep];
    if (!activeStep?.route || activeStep.route === currentRoute) {
      return;
    }

    onRouteChange?.(activeStep.route);
  }, [currentRoute, currentStep, isRunning, onRouteChange]);

  const handleCallback = (data: CallBackProps) => {
    const { action, index, status, type } = data;
    const isTerminal = status === STATUS.FINISHED || status === STATUS.SKIPPED;
    const isStepEvent =
      type === EVENTS.STEP_AFTER || type === EVENTS.TARGET_NOT_FOUND;
    const isNavigationalAction =
      action === ACTIONS.NEXT || action === ACTIONS.PREV;

    if (!isTerminal && isStepEvent && isNavigationalAction) {
      const direction = action === ACTIONS.PREV ? -1 : 1;
      const nextIndex = Math.max(
        0,
        Math.min(index + direction, ONBOARDING_TOTAL_STEPS - 1),
      );
      const nextRoute = ONBOARDING_TOUR_STEPS[nextIndex]?.route;

      if (nextRoute && nextRoute !== currentRoute) {
        onRouteChange?.(nextRoute);
      }

      onStepChange?.(nextIndex);
    }

    if (status === STATUS.FINISHED) {
      onComplete();
      return;
    }

    if (
      status === STATUS.SKIPPED ||
      action === ACTIONS.SKIP ||
      action === ACTIONS.CLOSE
    ) {
      onSkip?.();
    }
  };

  return (
    <Joyride
      steps={steps}
      run={isRunning}
      stepIndex={currentStep}
      continuous
      showProgress
      showSkipButton
      scrollToFirstStep
      disableOverlayClose={false}
      spotlightClicks={false}
      styles={joyrideStyles}
      locale={{
        back: "Back",
        close: "Close",
        last: "Finish",
        next: "Next",
        skip: "Skip tour",
        open: "Open",
      }}
      callback={handleCallback}
    />
  );
}
