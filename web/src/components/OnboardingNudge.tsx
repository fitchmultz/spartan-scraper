/**
 * Purpose: Show a lightweight first-run onboarding prompt without blocking access to the product.
 * Responsibilities: Offer clear actions to start the full tour, open shortcut help, or dismiss the hint while keeping the app interactive.
 * Scope: First-run onboarding nudge UI only.
 * Usage: Mount near the top of the application shell and control visibility from `useOnboarding()`.
 * Invariants/Assumptions: The nudge should never trap focus like a modal, and shortcut help remains available whether or not operators start the tour.
 */

import { ShortcutHint } from "./ShortcutHint";

interface OnboardingNudgeProps {
  isVisible: boolean;
  onStartTour: () => void;
  onOpenHelp: () => void;
  onDismiss: () => void;
  isMac?: boolean;
}

export function OnboardingNudge({
  isVisible,
  onStartTour,
  onOpenHelp,
  onDismiss,
  isMac,
}: OnboardingNudgeProps) {
  if (!isVisible) {
    return null;
  }

  return (
    <aside className="onboarding-nudge panel" aria-label="Getting started">
      <div className="onboarding-nudge__copy">
        <div className="onboarding-nudge__eyebrow">New here?</div>
        <h2>Explore the product without blocking your work</h2>
        <p>
          Spartan Scraper now teaches progressively. Start the full product
          tour, or keep working and use the visible command palette and shortcut
          help whenever you need them.
        </p>
      </div>

      <div className="onboarding-nudge__actions">
        <button type="button" onClick={onStartTour}>
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
