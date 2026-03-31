/**
 * Purpose: Verify the route-aware onboarding flow wires Joyride steps and callbacks correctly.
 * Responsibilities: Assert cross-route target coverage, route-navigation callbacks, and terminal completion/skip behavior.
 * Scope: Onboarding flow integration with the mocked Joyride adapter only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Joyride is mocked and receives the final props that `App.tsx` depends on for onboarding control.
 */

import { act, render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ACTIONS, EVENTS, Joyride, STATUS } from "react-joyride";
import { OnboardingFlow } from "./OnboardingFlow";
import { ONBOARDING_TOTAL_STEPS } from "../lib/onboarding";

vi.mock("react-joyride", () => {
  const JoyrideMock = vi.fn(() => null);

  return {
    Joyride: JoyrideMock,
    ACTIONS: {
      PREV: "prev",
      NEXT: "next",
      SKIP: "skip",
      CLOSE: "close",
    },
    EVENTS: {
      STEP_AFTER: "step:after",
      TARGET_NOT_FOUND: "error:target_not_found",
    },
    STATUS: {
      FINISHED: "finished",
      SKIPPED: "skipped",
    },
  };
});

function getJoyrideProps() {
  const firstCall = vi.mocked(Joyride).mock.calls[0];
  if (!firstCall) {
    throw new Error("Joyride was not rendered");
  }

  return firstCall[0] as unknown as {
    onEvent: (data: {
      action: string;
      index: number;
      status: string;
      type: string;
    }) => void;
    steps: { target: string }[];
  };
}

describe("OnboardingFlow", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("covers every major route in the guided tour", () => {
    render(
      <OnboardingFlow
        isRunning
        currentStep={0}
        currentRoute="jobs"
        onComplete={vi.fn()}
      />,
    );

    const props = getJoyrideProps();

    expect(props.steps).toHaveLength(ONBOARDING_TOTAL_STEPS);
    expect(props.steps.map((step) => step.target)).toEqual(
      expect.arrayContaining([
        '[data-tour="jobs-dashboard"], body',
        '[data-tour="command-palette"], body',
        '[data-tour="job-wizard-header"], body',
        '[data-tour="wizard-steps"], body',
        '[data-tour="job-results"], body',
        '[data-tour="templates-workspace"], body',
        '[data-tour="automation-hub"], body',
        '[data-tour="settings-workspace"], body',
      ]),
    );
  });

  it("requests route navigation when the next step lives on a different route", () => {
    const onRouteChange = vi.fn();
    const onStepChange = vi.fn();

    render(
      <OnboardingFlow
        isRunning
        currentStep={1}
        currentRoute="jobs"
        onComplete={vi.fn()}
        onRouteChange={onRouteChange}
        onStepChange={onStepChange}
      />,
    );

    const { onEvent } = getJoyrideProps();

    act(() => {
      onEvent({
        action: ACTIONS.NEXT,
        index: 1,
        status: "running",
        type: EVENTS.STEP_AFTER,
      });
    });

    expect(onStepChange).toHaveBeenCalledWith(2);
    expect(onRouteChange).toHaveBeenCalledWith("new-job");
  });

  it("calls completion and skip callbacks for terminal states", () => {
    const onComplete = vi.fn();
    const onSkip = vi.fn();

    render(
      <OnboardingFlow
        isRunning
        currentStep={0}
        currentRoute="jobs"
        onComplete={onComplete}
        onSkip={onSkip}
      />,
    );

    const { onEvent } = getJoyrideProps();

    act(() => {
      onEvent({
        action: ACTIONS.NEXT,
        index: ONBOARDING_TOTAL_STEPS - 1,
        status: STATUS.FINISHED,
        type: EVENTS.STEP_AFTER,
      });
    });

    expect(onComplete).toHaveBeenCalledTimes(1);

    act(() => {
      onEvent({
        action: ACTIONS.SKIP,
        index: 1,
        status: STATUS.SKIPPED,
        type: EVENTS.STEP_AFTER,
      });
    });

    expect(onSkip).toHaveBeenCalledTimes(1);
  });
});
