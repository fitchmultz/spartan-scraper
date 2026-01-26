/**
 * Info Sections Component
 *
 * Displays informational sections showing configured resources: auth profiles,
 * recurring schedules, extraction templates, and crawl state tracking for incremental
 * crawling. Each section is rendered conditionally based on data availability.
 *
 * @module InfoSections
 */
import type { CrawlState } from "../api";

interface InfoSectionsProps {
  profiles: Array<{ name: string; parents: string[] }>;
  schedules: Array<{
    id: string;
    kind: string;
    intervalSeconds: number;
    nextRun: string;
  }>;
  templates: string[];
  crawlStates: CrawlState[];
}

export function InfoSections({
  profiles,
  schedules,
  templates,
  crawlStates,
}: InfoSectionsProps) {
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
                  <div>Next: {new Date(sched.nextRun).toLocaleString()}</div>
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      {templates.length > 0 && (
        <section className="panel" style={{ marginTop: 16 }}>
          <h2>Extraction Templates</h2>
          <div className="job-list">
            {templates.map((name) => (
              <div key={name} className="job-item">
                <div>{name}</div>
              </div>
            ))}
          </div>
        </section>
      )}

      {crawlStates.length > 0 && (
        <section className="panel" style={{ marginTop: 16 }}>
          <h2>Crawl States (Incremental Tracking)</h2>
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
                    <div>
                      Scraped: {new Date(state.lastScraped).toLocaleString()}
                    </div>
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
