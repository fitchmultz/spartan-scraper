/**
 * Purpose: Render the guided job-wizard stepper and expose controlled step navigation.
 * Responsibilities: Show step labels, progress state, accessibility metadata, and guarded navigation for completed/current steps.
 * Scope: Guided job wizard stepper only.
 * Usage: Render from `JobSubmissionContainer` above the active wizard step panel.
 * Invariants/Assumptions: The step order is fixed at basics → runtime → extraction → review, and future steps must stay disabled until earlier steps are completed.
 */

import type { WizardStepId } from "./useJobWizard";

const STEP_LABELS: Record<WizardStepId, string> = {
  basics: "Basics",
  runtime: "Runtime",
  extraction: "Extraction",
  review: "Review",
};

const STEP_ORDER: WizardStepId[] = [
  "basics",
  "runtime",
  "extraction",
  "review",
];

interface WizardStepperProps {
  activeStep: WizardStepId;
  completedSteps: WizardStepId[];
  onStepChange: (step: WizardStepId) => void;
}

export function WizardStepper({
  activeStep,
  completedSteps,
  onStepChange,
}: WizardStepperProps) {
  const activeIndex = STEP_ORDER.indexOf(activeStep);

  return (
    <div className="job-wizard__stepper-wrap" data-tour="wizard-steps">
      <p className="job-wizard__step-count">
        Step {activeIndex + 1} of {STEP_ORDER.length}: {STEP_LABELS[activeStep]}
      </p>
      <ol className="job-wizard__stepper" aria-label="Job setup progress">
        {STEP_ORDER.map((step, index) => {
          const isActive = step === activeStep;
          const isComplete = completedSteps.includes(step);
          const isEnabled = isActive || isComplete || index <= activeIndex;

          return (
            <li key={step}>
              <button
                type="button"
                className={`job-wizard__step${isActive ? " is-active" : ""}${isComplete ? " is-complete" : ""}`}
                onClick={() => onStepChange(step)}
                disabled={!isEnabled}
                aria-current={isActive ? "step" : undefined}
              >
                <span className="job-wizard__step-index">Step {index + 1}</span>
                <span className="job-wizard__step-label">
                  {STEP_LABELS[step]}
                </span>
              </button>
            </li>
          );
        })}
      </ol>
    </div>
  );
}
