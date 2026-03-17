/**
 * Purpose: Render the sticky guided job-wizard footer actions for navigation, reset, and submission.
 * Responsibilities: Present step-aware primary/secondary actions, draft metadata, and disabled/loading states.
 * Scope: Guided job wizard footer actions only.
 * Usage: Render from `JobSubmissionContainer` below the active wizard step panel.
 * Invariants/Assumptions: Navigation order follows the fixed wizard step order and the final step swaps the primary action from Next to Submit.
 */

import type { JobType } from "../../types/presets";
import type { WizardStepId } from "./useJobWizard";

interface WizardActionsProps {
  activeStep: WizardStepId;
  activeTab: JobType;
  loading: boolean;
  draftSavedAt: number | null;
  onBack: () => void;
  onNext: () => void;
  onSubmit: () => void;
  onResetDraft: () => void;
}

function formatDraftSavedAt(draftSavedAt: number | null): string {
  if (!draftSavedAt) {
    return "Draft active";
  }

  return `Draft saved ${new Date(draftSavedAt).toLocaleTimeString([], {
    hour: "numeric",
    minute: "2-digit",
  })}`;
}

function submitLabelFor(jobType: JobType): string {
  if (jobType === "scrape") {
    return "Deploy Scrape";
  }
  if (jobType === "crawl") {
    return "Launch Crawl";
  }
  return "Run Research";
}

export function WizardActions({
  activeStep,
  activeTab,
  loading,
  draftSavedAt,
  onBack,
  onNext,
  onSubmit,
  onResetDraft,
}: WizardActionsProps) {
  const isReview = activeStep === "review";
  const isBasics = activeStep === "basics";

  return (
    <div className="job-wizard__actions">
      <div className="job-wizard__actions-meta">
        <button
          type="button"
          className="secondary job-wizard__reset"
          onClick={onResetDraft}
        >
          Reset Draft
        </button>
        <span className="job-wizard__draft-status">
          {formatDraftSavedAt(draftSavedAt)}
        </span>
      </div>

      <div className="job-wizard__actions-buttons">
        <button
          type="button"
          className="secondary"
          onClick={onBack}
          disabled={isBasics}
        >
          Back
        </button>
        {isReview ? (
          <button type="button" onClick={onSubmit} disabled={loading}>
            {loading ? "Submitting..." : submitLabelFor(activeTab)}
          </button>
        ) : (
          <button type="button" onClick={onNext} disabled={loading}>
            Next
          </button>
        )}
      </div>
    </div>
  );
}
