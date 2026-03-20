/**
 * Purpose: Render guided settings inventory panels for auth profiles, schedules, and crawl-state records.
 * Responsibilities: Show configured data when present, explain each capability when empty, and keep crawl-state pagination wired.
 * Scope: Settings route inventory panels only; shell framing and editor surfaces stay outside this component.
 * Usage: Render from the Settings route with current data and navigation callbacks.
 * Invariants/Assumptions: Empty sections should still explain what the capability is for and what to do next without overwhelming fresh installs.
 */

import { useEffect, useState, type ReactNode } from "react";
import type { CrawlState } from "../api";
import { formatDateTime } from "../lib/formatting";
import { ActionEmptyState } from "./ActionEmptyState";

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
  onCreateJob: () => void;
  onOpenAutomation: () => void;
  onOpenJobs: () => void;
}

interface SectionFrameProps {
  title: string;
  description: string;
  children: ReactNode;
}

function SectionFrame({ title, description, children }: SectionFrameProps) {
  return (
    <section className="panel info-section">
      <div style={{ marginBottom: 16 }}>
        <h2 style={{ marginBottom: 4 }}>{title}</h2>
        <p style={{ margin: 0, opacity: 0.8 }}>{description}</p>
      </div>
      {children}
    </section>
  );
}

export function InfoSections({
  profiles,
  schedules,
  crawlStates,
  crawlStatesPage,
  crawlStatesTotal,
  crawlStatesPerPage,
  onCrawlStatesPageChange,
  onCreateJob,
  onOpenAutomation,
  onOpenJobs,
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
      <SectionFrame
        title="Auth Profiles"
        description="Reusable credentials and inherited auth state for sites that need more than anonymous access."
      >
        {profiles.length === 0 ? (
          <ActionEmptyState
            eyebrow="Optional capability"
            title="No reusable auth profiles yet"
            description="You only need auth profiles when a target requires login, cookies, or shared headers. Start with a normal job first, then save auth once you know the workflow needs it."
            actions={[{ label: "Create job", onClick: onCreateJob }]}
          />
        ) : (
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
        )}
      </SectionFrame>

      <SectionFrame
        title="Schedules"
        description="Recurring work built on top of successful jobs once you are ready to automate."
      >
        {schedules.length === 0 ? (
          <ActionEmptyState
            eyebrow="Automation-ready later"
            title="No recurring schedules yet"
            description="Schedules are optional. Run a job manually first, open the succeeded job you trust, then promote that verified result into automation when you want hands-off repetition."
            actions={[
              { label: "Review jobs", onClick: onOpenJobs },
              {
                label: "Open automation",
                onClick: onOpenAutomation,
                tone: "secondary",
              },
            ]}
          />
        ) : (
          <div className="job-list">
            {schedules.map((schedule) => (
              <div key={schedule.id} className="job-item">
                <div>{schedule.kind}</div>
                <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                  <div>ID: {schedule.id}</div>
                  <div>Interval: {schedule.intervalSeconds}s</div>
                  <div>Next: {formatDateTime(schedule.nextRun)}</div>
                </div>
              </div>
            ))}
          </div>
        )}
      </SectionFrame>

      <SectionFrame
        title="Crawl States"
        description="Incremental tracking history that appears after crawl workflows have something worth resuming."
      >
        {crawlStates.length === 0 ? (
          <ActionEmptyState
            eyebrow="Appears after incremental runs"
            title="No crawl state has been recorded yet"
            description="Crawl state is created once Spartan has incremental crawl history to maintain. There is nothing to clean up or inspect until those runs exist."
            actions={[{ label: "Create crawl job", onClick: onCreateJob }]}
          />
        ) : (
          <>
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
                    {state.etag ? <div>ETag: {state.etag}</div> : null}
                    {state.lastScraped ? (
                      <div>Scraped: {formatDateTime(state.lastScraped)}</div>
                    ) : null}
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </SectionFrame>
    </div>
  );
}
