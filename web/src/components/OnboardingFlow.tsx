/**
 * Purpose: Render the application onboarding tour with route-aware guidance for the current web workflow model.
 * Responsibilities: Define Joyride steps, keep onboarding copy aligned with visible UI targets, and report progress/completion events back to onboarding state.
 * Scope: Web onboarding tour presentation only.
 * Usage: Mount once from `App.tsx` and drive it with `useOnboarding` state.
 * Invariants/Assumptions: Tour steps must target visible elements in the current shell, the guided job wizard is the default new-job experience, and total step count stays aligned with `ONBOARDING_TOTAL_STEPS`.
 */

import Joyride, {
  ACTIONS,
  EVENTS,
  STATUS,
  type CallBackProps,
  type Step,
  type Styles,
} from "react-joyride";
import { ONBOARDING_TOTAL_STEPS } from "../lib/onboarding";

export interface OnboardingFlowProps {
  /** Whether the tour is running */
  isRunning: boolean;
  /** Current step index */
  currentStep: number;
  /** Callback when tour completes or is skipped */
  onComplete: () => void;
  /** Callback when tour is skipped */
  onSkip?: () => void;
  /** Callback when step changes */
  onStepChange?: (stepIndex: number) => void;
}

/**
 * Tour steps definition.
 * Each step targets a specific element with data-tour attribute.
 */
const TOUR_STEPS: Step[] = [
  {
    target: "body",
    content: (
      <div>
        <h3 style={{ margin: "0 0 8px", fontSize: "1.1rem" }}>
          Welcome to Spartan Scraper!
        </h3>
        <p style={{ margin: 0, lineHeight: 1.5 }}>
          This quick tour walks through the new guided job workflow, presets,
          and the fastest ways to launch real work.
        </p>
      </div>
    ),
    placement: "center",
    disableBeacon: true,
  },
  {
    target: '[data-tour="quickstart"]',
    content: (
      <div>
        <h3 style={{ margin: "0 0 8px", fontSize: "1rem" }}>
          Quick Start Panel
        </h3>
        <p style={{ margin: 0, lineHeight: 1.5 }}>
          Browse saved and built-in presets, switch job types quickly, and save
          your current workflow once you have a setup worth reusing.
        </p>
      </div>
    ),
    placement: "bottom",
    title: "Quick Start",
  },
  {
    target: '[data-tour="job-type-selection"]',
    content: (
      <div>
        <h3 style={{ margin: "0 0 8px", fontSize: "1rem" }}>Three Job Types</h3>
        <p style={{ margin: "0 0 12px", lineHeight: 1.5 }}>
          Start by choosing the workflow that matches your intent:
        </p>
        <ul style={{ margin: 0, paddingLeft: 16, lineHeight: 1.8 }}>
          <li>
            <strong>Scrape</strong> — Single-page extraction with precise
            control
          </li>
          <li>
            <strong>Crawl</strong> — Multi-page crawling with explicit bounds
          </li>
          <li>
            <strong>Research</strong> — Multi-source synthesis driven by a query
          </li>
        </ul>
      </div>
    ),
    placement: "bottom",
    title: "Job Types",
  },
  {
    target: '[data-tour="wizard-steps"]',
    content: (
      <div>
        <h3 style={{ margin: "0 0 8px", fontSize: "1rem" }}>
          Guided Setup Steps
        </h3>
        <p style={{ margin: 0, lineHeight: 1.5 }}>
          Move through Basics, Runtime, Extraction, and Review. Spartan blocks
          the next step only when the required inputs for the current stage are
          missing.
        </p>
      </div>
    ),
    placement: "bottom",
    title: "Wizard Steps",
  },
  {
    target: '[data-tour="expert-mode"]',
    content: (
      <div>
        <h3 style={{ margin: "0 0 8px", fontSize: "1rem" }}>Expert Mode</h3>
        <p style={{ margin: 0, lineHeight: 1.5 }}>
          Switch to the dense full-form editor whenever you want every advanced
          control on one screen. Your draft carries across both modes.
        </p>
      </div>
    ),
    placement: "left",
    title: "Expert Mode",
  },
  {
    target: '[data-tour="quickstart"]',
    content: (
      <div>
        <h3 style={{ margin: "0 0 8px", fontSize: "1rem" }}>
          AI Assistance and Presets
        </h3>
        <p style={{ margin: 0, lineHeight: 1.5 }}>
          The quick-start rail is also where you can open AI preview and AI
          template actions, so you can debug extraction or generate helpers
          without leaving the job-creation workspace.
        </p>
      </div>
    ),
    placement: "bottom",
    title: "AI Assistance",
  },
  {
    target: '[data-tour="command-palette"]',
    content: (
      <div>
        <h3 style={{ margin: "0 0 8px", fontSize: "1rem" }}>Command Palette</h3>
        <p style={{ margin: "0 0 12px", lineHeight: 1.5 }}>
          Access everything quickly with the command palette. Press{" "}
          <kbd
            style={{
              background: "rgba(255,255,255,0.1)",
              padding: "2px 6px",
              borderRadius: 4,
              fontFamily: "inherit",
            }}
          >
            Ctrl+K
          </kbd>{" "}
          (or{" "}
          <kbd
            style={{
              background: "rgba(255,255,255,0.1)",
              padding: "2px 6px",
              borderRadius: 4,
              fontFamily: "inherit",
            }}
          >
            ⌘K
          </kbd>
          ) to:
        </p>
        <ul style={{ margin: 0, paddingLeft: 16, lineHeight: 1.8 }}>
          <li>Submit jobs quickly</li>
          <li>Navigate between sections</li>
          <li>Access recent jobs and presets</li>
          <li>Restart this tour anytime</li>
        </ul>
      </div>
    ),
    placement: "bottom",
    title: "Quick Access",
  },
  {
    target: "body",
    content: (
      <div>
        <h3 style={{ margin: "0 0 8px", fontSize: "1.1rem" }}>
          You're All Set!
        </h3>
        <p style={{ margin: "0 0 12px", lineHeight: 1.5 }}>
          You're ready to start scraping. Remember:
        </p>
        <ul style={{ margin: 0, paddingLeft: 16, lineHeight: 1.8 }}>
          <li>
            Press{" "}
            <kbd
              style={{
                background: "rgba(255,255,255,0.1)",
                padding: "2px 6px",
                borderRadius: 4,
                fontFamily: "inherit",
              }}
            >
              ?
            </kbd>{" "}
            for keyboard shortcuts (outside text inputs)
          </li>
          <li>
            Use{" "}
            <kbd
              style={{
                background: "rgba(255,255,255,0.1)",
                padding: "2px 6px",
                borderRadius: 4,
                fontFamily: "inherit",
              }}
            >
              Ctrl+K
            </kbd>{" "}
            for the command palette
          </li>
          <li>Hover over fields for contextual help</li>
          <li>Add ?showHelp to the URL to replay this tour</li>
        </ul>
        <p
          style={{
            margin: "12px 0 0",
            fontStyle: "italic",
            color: "var(--muted, #a0a0a8)",
          }}
        >
          Happy scraping!
        </p>
      </div>
    ),
    placement: "center",
    disableBeacon: true,
  },
];

if (import.meta.env.DEV && TOUR_STEPS.length !== ONBOARDING_TOTAL_STEPS) {
  console.warn(
    `[OnboardingFlow] TOUR_STEPS has ${TOUR_STEPS.length} steps but ONBOARDING_TOTAL_STEPS is ${ONBOARDING_TOTAL_STEPS}.`,
  );
}

/**
 * Custom styles for react-joyride that match the app theme.
 */
const joyrideStyles: Partial<Styles> = {
  options: {
    arrowColor: "var(--panel, #1a1a24)",
    backgroundColor: "var(--panel, #1a1a24)",
    overlayColor: "rgba(0, 0, 0, 0.7)",
    primaryColor: "var(--accent, #ffb700)",
    textColor: "var(--text, #fff)",
    zIndex: 1000,
  },
  tooltip: {
    backgroundColor: "var(--panel, #1a1a24)",
    border: "1px solid var(--stroke, rgba(255,255,255,0.1))",
    borderRadius: 16,
    boxShadow: "0 16px 48px rgba(0,0,0,0.5)",
    fontSize: "0.9rem",
    padding: 20,
  },
  tooltipContainer: {
    textAlign: "left",
  },
  tooltipTitle: {
    fontSize: "1rem",
    fontWeight: 600,
    margin: "0 0 12px",
    color: "var(--text, #fff)",
  },
  tooltipContent: {
    padding: 0,
  },
  buttonNext: {
    backgroundColor: "var(--accent, #ffb700)",
    color: "#1a1200",
    fontWeight: 600,
    padding: "8px 16px",
    borderRadius: 8,
    border: "none",
    cursor: "pointer",
    fontSize: "0.85rem",
  },
  buttonBack: {
    backgroundColor: "transparent",
    color: "var(--muted, #a0a0a8)",
    fontWeight: 500,
    padding: "8px 16px",
    borderRadius: 8,
    border: "none",
    cursor: "pointer",
    fontSize: "0.85rem",
    marginRight: 8,
  },
  buttonSkip: {
    backgroundColor: "transparent",
    color: "var(--muted, #a0a0a8)",
    fontWeight: 400,
    padding: "8px 16px",
    borderRadius: 8,
    border: "none",
    cursor: "pointer",
    fontSize: "0.8rem",
  },
  buttonClose: {
    color: "var(--muted, #a0a0a8)",
    background: "transparent",
    border: "none",
    cursor: "pointer",
    fontSize: "1.2rem",
    padding: 4,
  },
  overlay: {
    backdropFilter: "blur(4px)",
  },
  spotlight: {
    backgroundColor: "transparent",
    border: "2px solid var(--accent, #ffb700)",
    borderRadius: 12,
    boxShadow: "0 0 0 9999px rgba(0, 0, 0, 0.7)",
  },
};

/**
 * Onboarding Flow Component
 *
 * Wraps react-joyride with app-specific configuration and styling.
 * Provides a guided tour of the application's key features.
 *
 * @example
 * ```tsx
 * <OnboardingFlow
 *   isRunning={isTourActive}
 *   currentStep={currentStep}
 *   onComplete={finishOnboarding}
 *   onSkip={skipOnboarding}
 *   onStepChange={setCurrentStep}
 * />
 * ```
 */
export function OnboardingFlow({
  isRunning,
  currentStep,
  onComplete,
  onSkip,
  onStepChange,
}: OnboardingFlowProps) {
  const handleCallback = (data: CallBackProps) => {
    const { action, index, status, type } = data;

    const isTerminal = status === STATUS.FINISHED || status === STATUS.SKIPPED;
    const isStepEvent =
      type === EVENTS.STEP_AFTER || type === EVENTS.TARGET_NOT_FOUND;
    const isNavigationalAction =
      action === ACTIONS.NEXT || action === ACTIONS.PREV;

    if (!isTerminal && isStepEvent && isNavigationalAction) {
      const direction = action === ACTIONS.PREV ? -1 : 1;
      onStepChange?.(index + direction);
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
      steps={TOUR_STEPS}
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
