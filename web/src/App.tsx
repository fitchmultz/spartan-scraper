import { useCallback, useEffect, useMemo, useState } from "react";
import {
  getV1Jobs,
  postV1Crawl,
  postV1Research,
  postV1Scrape,
  type Job,
} from "./api";

type JobEntry = Job;

const defaultHeaders = "";

export function App() {
  const [scrapeUrl, setScrapeUrl] = useState("");
  const [crawlUrl, setCrawlUrl] = useState("");
  const [headless, setHeadless] = useState(false);
  const [usePlaywright, setUsePlaywright] = useState(false);
  const [maxDepth, setMaxDepth] = useState(2);
  const [maxPages, setMaxPages] = useState(200);
  const [timeoutSeconds, setTimeoutSeconds] = useState(30);
  const [authBasic, setAuthBasic] = useState("");
  const [headersRaw, setHeadersRaw] = useState(defaultHeaders);
  const [extractTemplate, setExtractTemplate] = useState("");
  const [extractValidate, setExtractValidate] = useState(false);
  const [researchQuery, setResearchQuery] = useState("");
  const [researchUrls, setResearchUrls] = useState("");
  const [jobs, setJobs] = useState<JobEntry[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
  const [resultSummary, setResultSummary] = useState<string | null>(null);
  const [resultConfidence, setResultConfidence] = useState<number | null>(null);
  const [resultEvidence, setResultEvidence] = useState<
    {
      url: string;
      title: string;
      snippet: string;
      score: number;
      confidence?: number;
      citationUrl?: string;
      clusterId?: string;
    }[]
  >([]);
  const [resultClusters, setResultClusters] = useState<
    {
      id: string;
      label: string;
      confidence: number;
      evidence: {
        url: string;
        title: string;
        score: number;
        confidence?: number;
        citationUrl?: string;
      }[];
    }[]
  >([]);
  const [resultCitations, setResultCitations] = useState<
    { canonical: string; anchor?: string; url?: string }[]
  >([]);
  const [rawResult, setRawResult] = useState<string | null>(null);

  const headerMap = useMemo(() => parseHeaders(headersRaw), [headersRaw]);

  const refreshJobs = useCallback(async () => {
    setLoading(true);
    try {
      const { data, error: apiError } = await getV1Jobs({ baseUrl: "" });
      if (apiError) {
        setError(String(apiError));
        return;
      }
      setJobs(data?.jobs ?? []);
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

  useEffect(() => {
    if (!headless && usePlaywright) {
      setUsePlaywright(false);
    }
  }, [headless, usePlaywright]);

  async function submitScrape() {
    if (!scrapeUrl) {
      setError("Scrape URL is required.");
      return;
    }
    setLoading(true);
    try {
      const { error: apiError } = await postV1Scrape({
        baseUrl: "",
        body: {
          url: scrapeUrl,
          headless,
          playwright: headless ? usePlaywright : false,
          timeoutSeconds,
          auth: buildAuth(authBasic, headerMap),
          extract: {
            template: extractTemplate || undefined,
            validate: extractValidate,
          },
        },
      });
      if (apiError) {
        setError(String(apiError));
        return;
      }
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
      const { error: apiError } = await postV1Crawl({
        baseUrl: "",
        body: {
          url: crawlUrl,
          maxDepth,
          maxPages,
          headless,
          playwright: headless ? usePlaywright : false,
          timeoutSeconds,
          auth: buildAuth(authBasic, headerMap),
          extract: {
            template: extractTemplate || undefined,
            validate: extractValidate,
          },
        },
      });
      if (apiError) {
        setError(String(apiError));
        return;
      }
      setError(null);
      await refreshJobs();
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  async function submitResearch() {
    if (!researchQuery || !researchUrls) {
      setError("Research query and URLs are required.");
      return;
    }
    setLoading(true);
    try {
      const { error: apiError } = await postV1Research({
        baseUrl: "",
        body: {
          query: researchQuery,
          urls: parseUrlList(researchUrls),
          maxDepth,
          maxPages,
          headless,
          playwright: headless ? usePlaywright : false,
          timeoutSeconds,
          auth: buildAuth(authBasic, headerMap),
          extract: {
            template: extractTemplate || undefined,
            validate: extractValidate,
          },
        },
      });
      if (apiError) {
        setError(String(apiError));
        return;
      }
      setError(null);
      await refreshJobs();
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }

  async function loadResults(jobId: string) {
    setSelectedJobId(jobId);
    setResultSummary(null);
    setResultConfidence(null);
    setResultEvidence([]);
    setResultClusters([]);
    setResultCitations([]);
    setRawResult(null);
    try {
      const response = await fetch(`/v1/jobs/${jobId}/results`);
      const text = await response.text();
      const firstLine = text.split("\n").find((line) => line.trim());
      if (firstLine) {
        const parsed = JSON.parse(firstLine);
        if (parsed?.summary) {
          setResultSummary(parsed.summary);
        }
        if (typeof parsed?.confidence === "number") {
          setResultConfidence(parsed.confidence);
        }
        if (Array.isArray(parsed?.evidence)) {
          setResultEvidence(parsed.evidence);
        }
        if (Array.isArray(parsed?.clusters)) {
          setResultClusters(parsed.clusters);
        }
        if (Array.isArray(parsed?.citations)) {
          setResultCitations(parsed.citations);
        }
      }
      setRawResult(text);
    } catch (err) {
      setError(String(err));
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
          <div>Playwright: {usePlaywright ? "Enabled" : "Disabled"}</div>
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
              <input
                type="checkbox"
                checked={usePlaywright}
                disabled={!headless}
                onChange={(event) => setUsePlaywright(event.target.checked)}
              />{" "}
              Playwright
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
          <div className="row" style={{ marginTop: 12 }}>
            <label>
              Extract Template
              <input
                value={extractTemplate}
                onChange={(e) => setExtractTemplate(e.target.value)}
                placeholder="default, article, product..."
              />
            </label>
            <label>
              <input
                type="checkbox"
                checked={extractValidate}
                onChange={(e) => setExtractValidate(e.target.checked)}
              />{" "}
              Validate Schema
            </label>
          </div>
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
              <input
                type="checkbox"
                checked={usePlaywright}
                disabled={!headless}
                onChange={(event) => setUsePlaywright(event.target.checked)}
              />{" "}
              Playwright
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

        <div className="panel">
          <h2>Deep Research</h2>
          <label htmlFor="research-query">Research query</label>
          <input
            id="research-query"
            value={researchQuery}
            onChange={(event) => setResearchQuery(event.target.value)}
            placeholder="pricing model, security posture, roadmap..."
          />
          <label htmlFor="research-urls" style={{ marginTop: 12 }}>
            Source URLs (comma-separated)
          </label>
          <textarea
            id="research-urls"
            rows={3}
            value={researchUrls}
            onChange={(event) => setResearchUrls(event.target.value)}
            placeholder="https://example.com, https://example.com/docs"
          />
          <div className="row" style={{ marginTop: 12 }}>
            <label>
              Max depth
              <input
                type="number"
                min={0}
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
              <input
                type="checkbox"
                checked={usePlaywright}
                disabled={!headless}
                onChange={(event) => setUsePlaywright(event.target.checked)}
              />{" "}
              Playwright
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
            <button type="button" onClick={() => void submitResearch()}>
              Run Research
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => {
                setResearchQuery("");
                setResearchUrls("");
              }}
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
                {job.status === "succeeded" ? (
                  <button
                    type="button"
                    className="secondary"
                    onClick={() => void loadResults(job.id ?? "")}
                  >
                    View Results
                  </button>
                ) : null}
              </div>
            ))
          )}
        </div>
        {selectedJobId ? (
          <div className="panel" style={{ marginTop: 16 }}>
            <h3>Results: {selectedJobId}</h3>
            {typeof resultConfidence === "number" ? (
              <div className="badge running" style={{ marginBottom: 8 }}>
                Confidence {resultConfidence.toFixed(2)}
              </div>
            ) : null}
            {resultSummary ? <p>{resultSummary}</p> : null}
            {resultClusters.length > 0 ? (
              <div style={{ marginTop: 12 }}>
                <h4>Evidence Clusters</h4>
                <div className="job-list">
                  {resultClusters.map((cluster) => (
                    <div key={cluster.id} className="job-item">
                      <div>{cluster.label || cluster.id}</div>
                      <div className="badge running">
                        Confidence {cluster.confidence.toFixed(2)}
                      </div>
                      <div>{cluster.evidence.length} sources</div>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
            {resultCitations.length > 0 ? (
              <div style={{ marginTop: 12 }}>
                <h4>Citations</h4>
                <div className="job-list">
                  {resultCitations.map((citation) => {
                    const target =
                      citation.anchor && citation.canonical
                        ? `${citation.canonical}#${citation.anchor}`
                        : citation.canonical || citation.url || "";
                    return (
                      <div key={target} className="job-item">
                        <a href={target} target="_blank" rel="noreferrer">
                          {target}
                        </a>
                      </div>
                    );
                  })}
                </div>
              </div>
            ) : null}
            {resultEvidence.length > 0 ? (
              <div className="job-list" style={{ marginTop: 12 }}>
                {resultEvidence.slice(0, 10).map((item) => (
                  <div
                    key={`${item.url}-${item.score}-${item.clusterId ?? ""}`}
                    className="job-item"
                  >
                    <div>{item.title || item.url}</div>
                    <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                      <div className="badge running">
                        Score {item.score.toFixed(2)}
                      </div>
                      {typeof item.confidence === "number" ? (
                        <div className="badge running">
                          Confidence {item.confidence.toFixed(2)}
                        </div>
                      ) : null}
                      {item.clusterId ? (
                        <div className="badge running">{item.clusterId}</div>
                      ) : null}
                    </div>
                    {item.citationUrl ? (
                      <a
                        href={item.citationUrl}
                        target="_blank"
                        rel="noreferrer"
                      >
                        {item.citationUrl}
                      </a>
                    ) : null}
                    <div>{item.snippet}</div>
                  </div>
                ))}
              </div>
            ) : null}
            {rawResult ? (
              <div style={{ marginTop: 12 }}>
                <details>
                  <summary>Normalized Data (First Item)</summary>
                  <NormalizedView raw={rawResult} />
                </details>
                <details style={{ marginTop: 8 }}>
                  <summary>Raw output</summary>
                  <pre>{rawResult}</pre>
                </details>
              </div>
            ) : null}
          </div>
        ) : null}
      </section>

      <div className="footer">
        Spartan Scraper — build once, deploy everywhere.
      </div>
    </div>
  );
}

function NormalizedView({ raw }: { raw: string }) {
  try {
    const firstLine = raw.split("\n").find((line) => line.trim());
    if (!firstLine) return null;
    const data = JSON.parse(firstLine);
    if (!data.normalized) return <div>No normalized data found.</div>;
    return (
      <pre style={{ background: "rgba(0, 50, 50, 0.3)" }}>
        {JSON.stringify(data.normalized, null, 2)}
      </pre>
    );
  } catch {
    return <div>Failed to parse result.</div>;
  }
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

function parseUrlList(raw: string) {
  return raw
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}
