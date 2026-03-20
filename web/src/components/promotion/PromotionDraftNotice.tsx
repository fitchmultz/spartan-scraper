/**
 * Purpose: Show visible source-job lineage and review guidance for promotion-seeded destination drafts.
 * Responsibilities: Summarize carried-forward fields, remaining operator decisions, unsupported carry-forward, and a fast path back to the source job.
 * Scope: Shared destination-draft notice only; routing and draft state stay in parent containers.
 * Usage: Render inside template, watch, or export-schedule workspaces whenever a promotion seed is active.
 * Invariants/Assumptions: Notices only display browser-safe sanitized source context, unsupported carry-forward must stay explicit, and operators should always have a direct way back to the source job.
 */

import type { PromotionSeed } from "../../types/promotion";

interface PromotionDraftNoticeProps {
  title: string;
  description: string;
  seed: PromotionSeed;
  onOpenSourceJob?: (jobId: string) => void;
  onClear?: () => void;
}

export function PromotionDraftNotice({
  title,
  description,
  seed,
  onOpenSourceJob,
  onClear,
}: PromotionDraftNoticeProps) {
  return (
    <section className="promotion-notice" aria-label={title}>
      <div className="promotion-notice__header">
        <div>
          <div className="results-viewer__section-label">
            Verified job promotion
          </div>
          <h4>{title}</h4>
          <p>{description}</p>
        </div>
        <div className="promotion-notice__actions">
          {onOpenSourceJob ? (
            <button
              type="button"
              className="secondary"
              onClick={() => onOpenSourceJob(seed.source.jobId)}
            >
              Open source job
            </button>
          ) : null}
          {onClear ? (
            <button type="button" className="secondary" onClick={onClear}>
              Clear promoted draft
            </button>
          ) : null}
        </div>
      </div>

      <div className="promotion-notice__source">
        <strong>{seed.source.label}</strong>
        <span>{seed.source.value}</span>
        <span>Job {seed.source.jobId}</span>
        <span>{seed.source.jobKind} workflow</span>
      </div>

      <div className="promotion-notice__grid">
        <div>
          <h5>Carried forward</h5>
          <ul>
            {seed.carriedForward.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </div>
        <div>
          <h5>Review before save</h5>
          <ul>
            {seed.remainingDecisions.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </div>
      </div>

      {seed.unsupportedCarryForward.length > 0 ? (
        <div className="promotion-notice__unsupported">
          <h5>Not carried forward</h5>
          <ul>
            {seed.unsupportedCarryForward.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}
