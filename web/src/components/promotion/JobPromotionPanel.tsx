/**
 * Purpose: Present the in-context job-to-automation chooser on the verified results route.
 * Responsibilities: Explain the three destination choices, show what each path carries forward, keep unsupported options explicit, and trigger canonical destination handoff.
 * Scope: Results-route promotion chooser only; destination draft rendering lives in the target workspaces.
 * Usage: Render on `/jobs/:id` once authoritative job detail is available and promotion is allowed.
 * Invariants/Assumptions: Only succeeded jobs are eligible, one chooser owns all destination choices, and disabled options must explain why they are unavailable instead of disappearing.
 */

import type {
  PromotionDestination,
  PromotionOption,
} from "../../types/promotion";

interface JobPromotionPanelProps {
  options: PromotionOption[];
  onPromote: (destination: PromotionDestination) => void;
}

export function JobPromotionPanel({
  options,
  onPromote,
}: JobPromotionPanelProps) {
  return (
    <section className="results-explorer__promotion-panel">
      <div className="results-explorer__drawer-header">
        <div>
          <div className="results-viewer__section-label">
            Verified job promotion
          </div>
          <h4>Turn this successful job into reusable automation</h4>
          <p className="form-help">
            Choose the destination that matches what you want next. Spartan
            seeds a real draft in the canonical workspace, keeps unsupported
            carry-forward explicit, and never saves automatically.
          </p>
        </div>
      </div>

      <div className="results-explorer__promotion-grid">
        {options.map((option) => (
          <article
            key={option.destination}
            className={`results-explorer__promotion-card ${option.eligible ? "" : "is-disabled"}`}
          >
            <div className="results-explorer__promotion-card-head">
              <div>
                <h5>{option.title}</h5>
                <p>{option.description}</p>
              </div>
              <span
                className={`results-explorer__readiness results-explorer__readiness--${option.eligible ? "recommended" : "limited"}`}
              >
                {option.eligible ? "ready" : "limited"}
              </span>
            </div>

            <div className="results-explorer__promotion-copy">
              <strong>Spartan carries forward</strong>
              <ul>
                {option.seed.carriedForward.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>

            <div className="results-explorer__promotion-copy">
              <strong>You still confirm</strong>
              <ul>
                {option.seed.remainingDecisions.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>

            {option.eligibilityMessage ? (
              <div className="results-explorer__surface-note">
                {option.eligibilityMessage}
              </div>
            ) : null}

            <button
              type="button"
              onClick={() => onPromote(option.destination)}
              disabled={!option.eligible}
            >
              {option.actionLabel}
            </button>
          </article>
        ))}
      </div>
    </section>
  );
}
