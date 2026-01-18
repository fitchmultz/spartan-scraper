import { useCallback, useEffect, useMemo, useState } from "react";
import { OpenAPI, getV1Jobs, postV1Crawl, postV1Scrape, type Job } from "./api";

type JobEntry = Job;

const defaultHeaders = "";

export function App() {
  const [scrapeUrl, setScrapeUrl] = useState("");
  const [crawlUrl, setCrawlUrl] = useState("");
  const [headless, setHeadless] = useState(false);
  const [maxDepth, setMaxDepth] = useState(2);
  const [maxPages, setMaxPages] = useState(200);
  const [timeoutSeconds, setTimeoutSeconds] = useState(30);
  const [authBasic, setAuthBasic] = useState("");
  const [headersRaw, setHeadersRaw] = useState(defaultHeaders);
  const [jobs, setJobs] = useState<JobEntry[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    OpenAPI.BASE = "";
  }, []);

  const headerMap = useMemo(() => parseHeaders(headersRaw), [headersRaw]);

  const refreshJobs = useCallback(async () => {
    setLoading(true);
    try {
      const response = await getV1Jobs();
      setJobs(response.jobs ?? []);
      setError(null);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refreshJobs();
    const handle = window.setInterval(() => void refreshJobs(), 4000);
    return () => window.clearInterval(handle);
  }, [refreshJobs]);

  async function submitScrape() {
    if (!scrapeUrl) {
      setError("Scrape URL is required.");
      return;
    }
    setLoading(true);
    try {
      await postV1Scrape({
        requestBody: {
          url: scrapeUrl,
          headless,
          timeoutSeconds,
          auth: buildAuth(authBasic, headerMap),
        },
      });
      setError(null);
      await refreshJobs();
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  async function submitCrawl() {
    if (!crawlUrl) {
      setError("Crawl URL is required.");
      return;
    }
    setLoading(true);
    try {
      await postV1Crawl({
        requestBody: {
          url: crawlUrl,
          maxDepth,
          maxPages,
          headless,
          timeoutSeconds,
          auth: buildAuth(authBasic, headerMap),
        },
      });
      setError(null);
      await refreshJobs();
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="app">
      <section className="hero">
        <div className="hero-card">
          <div className="kicker">Operation Spartan</div>
          <h1>Spartan Scraper Command Center</h1>
          <p>
            Unified scraping and automation. Single pages, site-wide crawls,
            headless login flows, and durable job tracking.
          </p>
        </div>
        <div className="stats">
          <h3>Live Signals</h3>
          <div>{loading ? "Refreshing…" : "Standing by"}</div>
          <div>Total jobs: {jobs.length}</div>
          <div>Headless mode: {headless ? "Enabled" : "Disabled"}</div>
        </div>
      </section>

      <section className="grid">
        <div className="panel">
          <h2>Scrape a Page</h2>
          <label htmlFor="scrape-url">Target URL</label>
          <input
            id="scrape-url"
            value={scrapeUrl}
            onChange={(event) => setScrapeUrl(event.target.value)}
            placeholder="https://example.com"
          />
          <div className="row" style={{ marginTop: 12 }}>
            <label>
              <input
                type="checkbox"
                checked={headless}
                onChange={(event) => setHeadless(event.target.checked)}
              />{" "}
              Headless
            </label>
            <label>
              Timeout (s)
              <input
                type="number"
                min={5}
                value={timeoutSeconds}
                onChange={(event) =>
                  setTimeoutSeconds(Number(event.target.value))
                }
              />
            </label>
          </div>
          <label htmlFor="auth-basic" style={{ marginTop: 12 }}>
            Basic auth (user:pass)
          </label>
          <input
            id="auth-basic"
            value={authBasic}
            onChange={(event) => setAuthBasic(event.target.value)}
          />
          <label htmlFor="headers-raw" style={{ marginTop: 12 }}>
            Extra headers (one per line: Key: Value)
          </label>
          <textarea
            id="headers-raw"
            rows={3}
            value={headersRaw}
            onChange={(event) => setHeadersRaw(event.target.value)}
          />
          <div style={{ marginTop: 16, display: "flex", gap: 12 }}>
            <button type="button" onClick={() => void submitScrape()}>
              Deploy Scrape
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => setScrapeUrl("")}
            >
              Clear
            </button>
          </div>
        </div>

        <div className="panel">
          <h2>Crawl a Site</h2>
          <label htmlFor="crawl-url">Root URL</label>
          <input
            id="crawl-url"
            value={crawlUrl}
            onChange={(event) => setCrawlUrl(event.target.value)}
            placeholder="https://example.com"
          />
          <div className="row" style={{ marginTop: 12 }}>
            <label>
              Max depth
              <input
                type="number"
                min={1}
                value={maxDepth}
                onChange={(event) => setMaxDepth(Number(event.target.value))}
              />
            </label>
            <label>
              Max pages
              <input
                type="number"
                min={1}
                value={maxPages}
                onChange={(event) => setMaxPages(Number(event.target.value))}
              />
            </label>
          </div>
          <div className="row" style={{ marginTop: 12 }}>
            <label>
              <input
                type="checkbox"
                checked={headless}
                onChange={(event) => setHeadless(event.target.checked)}
              />{" "}
              Headless
            </label>
            <label>
              Timeout (s)
              <input
                type="number"
                min={5}
                value={timeoutSeconds}
                onChange={(event) =>
                  setTimeoutSeconds(Number(event.target.value))
                }
              />
            </label>
          </div>
          <div style={{ marginTop: 16, display: "flex", gap: 12 }}>
            <button type="button" onClick={() => void submitCrawl()}>
              Launch Crawl
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => setCrawlUrl("")}
            >
              Clear
            </button>
          </div>
        </div>
      </section>

      <section className="panel">
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <h2>Active Jobs</h2>
          <button
            type="button"
            className="secondary"
            onClick={() => void refreshJobs()}
          >
            Refresh
          </button>
        </div>
        {error ? <p className="error">{error}</p> : null}
        <div className="job-list" style={{ marginTop: 12 }}>
          {jobs.length === 0 ? (
            <div>No jobs yet. Submit a scrape or crawl.</div>
          ) : (
            jobs.map((job) => (
              <div key={job.id} className="job-item">
                <div>{job.id}</div>
                <div>
                  <span className={`badge ${statusClass(job.status ?? "")}`}>
                    {job.status}
                  </span>{" "}
                  {job.kind}
                </div>
                <div>Updated: {job.updatedAt}</div>
                {job.error ? <div>Error: {job.error}</div> : null}
              </div>
            ))
          )}
        </div>
      </section>

      <div className="footer">
        Spartan Scraper — build once, deploy everywhere.
      </div>
    </div>
  );
}

function statusClass(status: string) {
  switch (status) {
    case "succeeded":
      return "success";
    case "failed":
      return "failed";
    case "running":
      return "running";
    default:
      return "";
  }
}

function parseHeaders(raw: string) {
  if (!raw.trim()) {
    return undefined;
  }
  const headers: Record<string, string> = {};
  raw
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
    .forEach((line) => {
      const idx = line.indexOf(":");
      if (idx > 0) {
        const key = line.slice(0, idx).trim();
        const value = line.slice(idx + 1).trim();
        if (key && value) {
          headers[key] = value;
        }
      }
    });
  return Object.keys(headers).length ? headers : undefined;
}

function buildAuth(basic: string, headers?: Record<string, string>) {
  if (!basic && !headers) {
    return undefined;
  }
  return {
    basic: basic || undefined,
    headers,
  };
}
