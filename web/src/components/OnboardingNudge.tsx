/**
 * Purpose: Show a lightweight first-run onboarding prompt without blocking access to the product.
 * Responsibilities: Offer clear actions to start the full tour, open shortcut help, or dismiss the hint while keeping the app interactive.
 * Scope: First-run onboarding nudge UI only.
 * Usage: Mount near the top of the application shell and control visibility from `useOnboarding()`.
 * Invariants/Assumptions: The nudge should never trap focus like a modal, shortcut help remains available whether or not operators start the tour, and first-run copy should explain optional subsystem readiness without turning them into blockers.
 */

import type { HealthResponse } from "../api";
import { ShortcutHint } from "./ShortcutHint";

interface OnboardingNudgeProps {
  isVisible: boolean;
  onStartTour: () => void;
  onOpenHelp: () => void;
  onDismiss: () => void;
  onCreateJob: () => void;
  health: HealthResponse | null;
  hasTemplates: boolean;
  isMac?: boolean;
}

export function OnboardingNudge({
  isVisible,
  onStartTour,
  onOpenHelp,
  onDismiss,
  onCreateJob,
  health,
  hasTemplates,
  isMac,
}: OnboardingNudgeProps) {
  if (!isVisible) {
    return null;
  }

  const browserReady = health?.components?.browser?.status === "ok";
  const aiReady = health?.components?.ai?.status === "ok";

  return (
    <aside className="onboarding-nudge panel" aria-label="Getting started">
      <div className="onboarding-nudge__copy">
        <div className="onboarding-nudge__eyebrow">First run</div>
        <h2>Start with one working job</h2>
        <p>
          Spartan is ready for a guided first run. Submit a scrape first, then
          come back for templates, automation, and deeper runtime tuning.
        </p>

        <ul className="onboarding-nudge__checklist">
          <li>
            {browserReady
              ? "Browser automation is ready if you need it."
              : "Browser automation is optional and can be enabled later."}
          </li>
          <li>
            {aiReady
              ? "AI helpers are available."
              : "AI helpers are optional and currently unavailable."}
          </li>
          <li>
            {hasTemplates
              ? "Templates are already available to reuse."
              : "You can create templates later without blocking the first run."}
          </li>
        </ul>
      </div>

      <div className="onboarding-nudge__actions">
        <button type="button" onClick={onCreateJob}>
          Create first job
        </button>
        <button type="button" className="secondary" onClick={onStartTour}>
          Start product tour
        </button>
        <button type="button" className="secondary" onClick={onOpenHelp}>
          View shortcuts <ShortcutHint shortcut="?" isMac={isMac} />
        </button>
        <button type="button" className="secondary" onClick={onDismiss}>
          Dismiss
        </button>
      </div>
    </aside>
  );
}
