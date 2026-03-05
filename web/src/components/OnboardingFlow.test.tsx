/**
 * Tests for OnboardingFlow callback behavior.
 *
 * @module OnboardingFlow.test
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, act } from "@testing-library/react";
import Joyride, { ACTIONS, EVENTS, STATUS } from "react-joyride";
import { OnboardingFlow } from "./OnboardingFlow";

vi.mock("react-joyride", () => {
  const JoyrideMock = vi.fn(() => null);

  return {
    default: JoyrideMock,
    ACTIONS: {
      PREV: "prev",
      NEXT: "next",
      SKIP: "skip",
      CLOSE: "close",
    },
    EVENTS: {
      STEP_AFTER: "step:after",
      TARGET_NOT_FOUND: "target:not_found",
    },
    STATUS: {
      FINISHED: "finished",
      SKIPPED: "skipped",
    },
  };
});

type JoyrideProps = {
  callback: (data: {
    action: string;
    index: number;
    status: string;
    type: string;
  }) => void;
};

function getCallback(): JoyrideProps["callback"] {
  const joyrideMock = vi.mocked(Joyride);
  const firstCall = joyrideMock.mock.calls[0];
  if (!firstCall) {
    throw new Error("Joyride was not rendered");
  }

  const [props] = firstCall;
  const callback = (props as unknown as JoyrideProps).callback;

  if (!callback) {
    throw new Error("Joyride callback prop was not provided");
  }

  return callback;
}

describe("OnboardingFlow", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("advances and rewinds steps from Joyride callback events", () => {
    const onStepChange = vi.fn();

    render(
      <OnboardingFlow
        isRunning
        currentStep={2}
        onComplete={vi.fn()}
        onStepChange={onStepChange}
      />,
    );

    const callback = getCallback();

    act(() => {
      callback({
        action: ACTIONS.NEXT,
        index: 2,
        status: "running",
        type: EVENTS.STEP_AFTER,
      });
    });

    expect(onStepChange).toHaveBeenLastCalledWith(3);

    act(() => {
      callback({
        action: ACTIONS.PREV,
        index: 2,
        status: "running",
        type: EVENTS.STEP_AFTER,
      });
    });

    expect(onStepChange).toHaveBeenLastCalledWith(1);

    act(() => {
      callback({
        action: ACTIONS.NEXT,
        index: 3,
        status: "running",
        type: EVENTS.TARGET_NOT_FOUND,
      });
    });

    expect(onStepChange).toHaveBeenLastCalledWith(4);
  });

  it("calls completion and skip callbacks for terminal states", () => {
    const onComplete = vi.fn();
    const onSkip = vi.fn();
    const onStepChange = vi.fn();

    render(
      <OnboardingFlow
        isRunning
        currentStep={0}
        onComplete={onComplete}
        onSkip={onSkip}
        onStepChange={onStepChange}
      />,
    );

    const callback = getCallback();

    act(() => {
      callback({
        action: ACTIONS.NEXT,
        index: 7,
        status: STATUS.FINISHED,
        type: EVENTS.STEP_AFTER,
      });
    });

    expect(onComplete).toHaveBeenCalledTimes(1);
    expect(onSkip).not.toHaveBeenCalled();

    act(() => {
      callback({
        action: ACTIONS.CLOSE,
        index: 1,
        status: "running",
        type: EVENTS.STEP_AFTER,
      });
    });

    expect(onSkip).toHaveBeenCalledTimes(1);
    expect(onStepChange).not.toHaveBeenCalled();

    act(() => {
      callback({
        action: ACTIONS.SKIP,
        index: 2,
        status: STATUS.SKIPPED,
        type: EVENTS.STEP_AFTER,
      });
    });

    expect(onSkip).toHaveBeenCalledTimes(2);
    expect(onStepChange).not.toHaveBeenCalled();
  });
});
