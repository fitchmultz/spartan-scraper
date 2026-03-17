/**
 * Purpose: Render compact settings inventory panels for auth profiles, schedules, and crawl-state records.
 * Responsibilities: Show only populated settings sections, keep crawl-state pagination wired, and present runtime inventory without route-level layout hacks.
 * Scope: Settings route inventory panels only; shell framing and editor surfaces stay outside this component.
 * Usage: Render from the Settings route with current profile, schedule, and crawl-state data plus pagination callbacks.
 * Invariants/Assumptions: Empty sections stay hidden, pagination is parent-owned, and spacing comes from shared layout containers instead of per-panel margins.
 */
import { useEffect, useState } from "react";
import type { CrawlState } from "../api";
import { formatDateTime } from "../lib/formatting";

interface InfoSectionsProps {
  profiles: Array<{ name: string; parents: string[] }>;
  schedules: Array<{
    id: string;
    kind: string;
    intervalSeconds: number;
    nextRun: string;
  }>;
  crawlStates: CrawlState[];
  crawlStatesPage: number;
  crawlStatesTotal: number;
  crawlStatesPerPage: number;
  onCrawlStatesPageChange: (page: number) => void;
}

export function InfoSections({
  profiles,
  schedules,
  crawlStates,
  crawlStatesPage,
  crawlStatesTotal,
  crawlStatesPerPage,
  onCrawlStatesPageChange,
}: InfoSectionsProps) {
  const [jumpInputValue, setJumpInputValue] = useState(
    crawlStatesPage.toString(),
  );

  useEffect(() => {
    setJumpInputValue(crawlStatesPage.toString());
  }, [crawlStatesPage]);

  const maxPage = Math.max(1, Math.ceil(crawlStatesTotal / crawlStatesPerPage));

  return (
    <div className="info-sections">
      {profiles.length > 0 && (
        <section className="panel info-section">
          <h2>Auth Profiles</h2>
          <div className="job-list">
            {profiles.map((profile) => (
              <div key={profile.name} className="job-item">
                <div>{profile.name}</div>
                <div style={{ fontSize: "0.8em", color: "#666" }}>
                  {profile.parents.length > 0
                    ? `Parents: ${profile.parents.join(", ")}`
                    : "No parents"}
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      {schedules.length > 0 && (
        <section className="panel info-section">
          <h2>Schedules</h2>
          <div className="job-list">
            {schedules.map((sched) => (
              <div key={sched.id} className="job-item">
                <div>{sched.kind}</div>
                <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                  <div>ID: {sched.id}</div>
                  <div>Interval: {sched.intervalSeconds}s</div>
                  <div>Next: {formatDateTime(sched.nextRun)}</div>
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      {crawlStates.length > 0 && (
        <section className="panel info-section">
          <h2>Crawl States (Incremental Tracking)</h2>

          {crawlStatesTotal > crawlStatesPerPage ? (
            <div className="pagination-controls" style={{ marginBottom: 12 }}>
              <button
                type="button"
                disabled={crawlStatesPage <= 1}
                onClick={() => onCrawlStatesPageChange(crawlStatesPage - 1)}
              >
                Previous
              </button>

              <span className="pagination-info">
                Page {crawlStatesPage} of {maxPage} ({crawlStatesTotal} total)
              </span>

              <button
                type="button"
                disabled={crawlStatesPage >= maxPage}
                onClick={() => onCrawlStatesPageChange(crawlStatesPage + 1)}
              >
                Next
              </button>

              <div className="pagination-jump">
                <input
                  type="number"
                  min="1"
                  max={maxPage}
                  value={jumpInputValue}
                  onChange={(event) => setJumpInputValue(event.target.value)}
                />
                <button
                  type="button"
                  onClick={() => {
                    const page = Number.parseInt(jumpInputValue, 10);
                    if (
                      Number.isInteger(page) &&
                      page >= 1 &&
                      page <= maxPage
                    ) {
                      onCrawlStatesPageChange(page);
                    }
                  }}
                >
                  Go
                </button>
              </div>
            </div>
          ) : null}

          <div className="job-list">
            {crawlStates.map((state) => (
              <div key={state.url} className="job-item">
                <div style={{ wordBreak: "break-all" }}>{state.url}</div>
                <div
                  style={{
                    display: "flex",
                    gap: 8,
                    flexWrap: "wrap",
                    fontSize: "0.8em",
                  }}
                >
                  {state.etag && <div>ETag: {state.etag}</div>}
                  {state.lastScraped && (
                    <div>Scraped: {formatDateTime(state.lastScraped)}</div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
