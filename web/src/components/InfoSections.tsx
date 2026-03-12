/**
 * Info Sections Component
 *
 * Displays informational sections showing configured resources: auth profiles,
 * recurring schedules, extraction templates, and crawl state tracking for incremental
 * crawling. Each section is rendered conditionally based on data availability.
 *
 * @module InfoSections
 */
import { useState, useEffect } from "react";
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

  const maxPage = Math.ceil(crawlStatesTotal / crawlStatesPerPage);

  return (
    <>
      {profiles.length > 0 && (
        <section className="panel" style={{ marginTop: 16 }}>
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
        <section className="panel" style={{ marginTop: 16 }}>
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
        <section className="panel" style={{ marginTop: 16 }}>
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
                  onChange={(e) => {
                    const page = parseInt(e.target.value, 10);
                    if (
                      Number.isInteger(page) &&
                      page >= 1 &&
                      page <= maxPage
                    ) {
                      setJumpInputValue(e.target.value);
                    }
                  }}
                />
                <button
                  type="button"
                  onClick={() => {
                    const page = parseInt(jumpInputValue, 10);
                    if (page >= 1 && page <= maxPage) {
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
    </>
  );
}
