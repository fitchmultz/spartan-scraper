/**
 * Purpose: Provide skeleton placeholders for jobs dashboard cards during loading and state restoration.
 * Responsibilities: Mirror the card silhouette closely enough to stabilize layout and communicate loading.
 * Scope: Jobs dashboard loading UI only.
 * Usage: Render in dashboard lanes while jobs data or saved jobs-route context is being restored.
 * Invariants/Assumptions: Skeletons are decorative placeholders and must remain aria-hidden.
 */

export function JobCardSkeleton() {
  return (
    <div
      className="job-card-skeleton"
      data-testid="job-card-skeleton"
      aria-hidden="true"
    >
      <div className="job-card-skeleton__header">
        <div className="skeleton job-card-skeleton__pill" />
        <div className="skeleton job-card-skeleton__pill job-card-skeleton__pill--sm" />
      </div>
      <div className="skeleton job-card-skeleton__title" />
      <div className="skeleton job-card-skeleton__meta" />
      <div className="skeleton job-card-skeleton__progress" />
      <div className="job-card-skeleton__timeline">
        <div className="skeleton job-card-skeleton__tile" />
        <div className="skeleton job-card-skeleton__tile" />
        <div className="skeleton job-card-skeleton__tile" />
      </div>
    </div>
  );
}
