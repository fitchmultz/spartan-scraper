/**
 * Spartan Scraper Web UI - Main Application Component
 *
 * This is the primary React component for the Spartan Scraper web interface.
 * It provides a single-page application for:
 *
 * - Submitting scrape, crawl, and research jobs
 * - Configuring authentication, headers, cookies, and query parameters
 * - Managing extraction templates and validation
 * - Viewing real-time job status and manager state
 * - Browsing and analyzing job results
 *
 * @module App
 */
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  deleteV1JobsById,
  getV1Jobs,
  postV1Crawl,
  postV1Research,
  postV1Scrape,
  getHealthz,
  listTemplates,
  listCrawlStates,
  getV1AuthProfiles,
  getV1Schedules,
  type Job,
  type CrawlState,
  type AuthProfile,
  type ScrapeRequest,
  type CrawlRequest,
  type ResearchRequest,
  type ExtractOptions,
} from "./api";
import { buildApiUrl, getApiBaseUrl } from "./lib/api-config";

type JobEntry = Job;

type EvidenceItem = {
  url: string;
  title: string;
  snippet: string;
  score: number;
  confidence?: number;
  citationUrl?: string;
  clusterId?: string;
};

type ClusterItem = {
  id: string;
  label: string;
  confidence: number;
  evidence: EvidenceItem[];
};

type CitationItem = {
  canonical: string;
  anchor?: string;
  url?: string;
};

type ExtractedData = Record<string, unknown>;

type NormalizedData = Record<string, unknown>;

type CrawlResultItem = {
  url: string;
  status: number;
  title: string;
  text: string;
  links: string[];
  metadata?: Record<string, unknown>;
  extracted?: ExtractedData;
  normalized?: NormalizedData;
};

type ResearchResultItem = {
  summary?: string;
  confidence?: number;
  evidence?: EvidenceItem[];
  clusters?: ClusterItem[];
  citations?: CitationItem[];
};

type ResultItem = CrawlResultItem | ResearchResultItem;

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
  const [cookiesRaw, setCookiesRaw] = useState("");
  const [queryRaw, setQueryRaw] = useState("");
  const [authProfile, setAuthProfile] = useState("");
  const [loginUrl, setLoginUrl] = useState("");
  const [loginUserSelector, setLoginUserSelector] = useState("");
  const [loginPassSelector, setLoginPassSelector] = useState("");
  const [loginSubmitSelector, setLoginSubmitSelector] = useState("");
  const [loginUser, setLoginUser] = useState("");
  const [loginPass, setLoginPass] = useState("");
  const [extractTemplate, setExtractTemplate] = useState("");
  const [extractValidate, setExtractValidate] = useState(false);
  const [preProcessors, setPreProcessors] = useState("");
  const [postProcessors, setPostProcessors] = useState("");
  const [transformers, setTransformers] = useState("");
  const [incremental, setIncremental] = useState(false);
  const [researchQuery, setResearchQuery] = useState("");
  const [researchUrls, setResearchUrls] = useState("");
  const [jobs, setJobs] = useState<JobEntry[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
  const [resultItems, setResultItems] = useState<ResultItem[]>([]);
  const [selectedResultIndex, setSelectedResultIndex] = useState(0);
  const [resultSummary, setResultSummary] = useState<string | null>(null);
  const [resultConfidence, setResultConfidence] = useState<number | null>(null);
  const [resultEvidence, setResultEvidence] = useState<EvidenceItem[]>([]);
  const [resultClusters, setResultClusters] = useState<ClusterItem[]>([]);
  const [resultCitations, setResultCitations] = useState<CitationItem[]>([]);
  const [rawResult, setRawResult] = useState<string | null>(null);
  const [resultFormat, setResultFormat] = useState<string>("jsonl");
  const [managerStatus, setManagerStatus] = useState<{
    queued: number;
    active: number;
  } | null>(null);

  const [profiles, setProfiles] = useState<
    { name: string; parents: string[] }[]
  >([]);
  const [schedules, setSchedules] = useState<
    { id: string; kind: string; intervalSeconds: number; nextRun: string }[]
  >([]);
  const [templates, setTemplates] = useState<string[]>([]);
  const [crawlStates, setCrawlStates] = useState<CrawlState[]>([]);

  const selectedJobIdRef = useRef<string | null>(null);
  const resultFormatRef = useRef<string>("jsonl");

  const headerMap = useMemo(() => parseHeaders(headersRaw), [headersRaw]);
  const cookieList = useMemo(() => parseCookies(cookiesRaw), [cookiesRaw]);
  const queryMap = useMemo(() => parseQueryParams(queryRaw), [queryRaw]);

  const refreshJobs = useCallback(async () => {
    setLoading(true);
    try {
      const { data, error: apiError } = await getV1Jobs({
        baseUrl: getApiBaseUrl(),
      });
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

  const refreshManagerStatus = useCallback(async () => {
    try {
      const { data, error: apiError } = await getHealthz({
        baseUrl: getApiBaseUrl(),
      });
      if (apiError) {
        console.error("Failed to fetch manager status:", apiError);
        return;
      }
      const queueDetails = data?.components?.queue?.details;
      if (queueDetails && typeof queueDetails === "object") {
        const queued =
          typeof queueDetails.queued === "number" ? queueDetails.queued : 0;
        const active =
          typeof queueDetails.active === "number" ? queueDetails.active : 0;
        setManagerStatus({ queued, active });
      }
    } catch (err) {
      console.error("Failed to fetch manager status:", err);
    }
  }, []);

  const refreshProfiles = useCallback(async () => {
    try {
      const { data, error: apiError } = await getV1AuthProfiles({
        baseUrl: getApiBaseUrl(),
      });
      if (apiError) {
        console.error("Failed to fetch profiles:", apiError);
        return;
      }
      const profileList = (data?.profiles ?? [])
        .filter((p) => p.name !== undefined)
        .map((p: AuthProfile) => ({
          name: p.name as string,
          parents: p.parents || [],
        }));
      setProfiles(profileList);
    } catch (err) {
      console.error("Failed to fetch profiles:", err);
    }
  }, []);

  const refreshSchedules = useCallback(async () => {
    try {
      const { data, error: apiError } = await getV1Schedules({
        baseUrl: getApiBaseUrl(),
      });
      if (apiError) {
        console.error("Failed to fetch schedules:", apiError);
        return;
      }
      setSchedules(data?.schedules || []);
    } catch (err) {
      console.error("Failed to fetch schedules:", err);
    }
  }, []);

  const refreshTemplates = useCallback(async () => {
    try {
      const { data, error: apiError } = await listTemplates({
        baseUrl: getApiBaseUrl(),
      });
      if (apiError) {
        console.error("Failed to fetch templates:", apiError);
        return;
      }
      setTemplates(data?.templates || []);
    } catch (err) {
      console.error("Failed to fetch templates:", err);
    }
  }, []);

  const refreshCrawlStates = useCallback(async () => {
    try {
      const { data, error: apiError } = await listCrawlStates({
        baseUrl: getApiBaseUrl(),
      });
      if (apiError) {
        console.error("Failed to fetch crawl states:", apiError);
        return;
      }
      setCrawlStates(data?.crawlStates || []);
    } catch (err) {
      console.error("Failed to fetch crawl states:", err);
    }
  }, []);

  useEffect(() => {
    void refreshJobs();
    void refreshManagerStatus();
    void refreshProfiles();
    void refreshSchedules();
    void refreshTemplates();
    void refreshCrawlStates();
    const handle = window.setInterval(() => {
      void refreshJobs();
      void refreshManagerStatus();
    }, 4000);
    return () => window.clearInterval(handle);
  }, [
    refreshJobs,
    refreshManagerStatus,
    refreshProfiles,
    refreshSchedules,
    refreshTemplates,
    refreshCrawlStates,
  ]);

  useEffect(() => {
    selectedJobIdRef.current = selectedJobId;
  }, [selectedJobId]);

  useEffect(() => {
    resultFormatRef.current = resultFormat;
  }, [resultFormat]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: using refs to avoid circular dependency on loadResults
  useEffect(() => {
    const jobId = selectedJobIdRef.current;
    const fmt = resultFormatRef.current;
    if (jobId) {
      void loadResults(jobId, fmt);
    }
  }, [resultFormat]);

  useEffect(() => {
    if (!headless && usePlaywright) {
      setUsePlaywright(false);
    }
  }, [headless, usePlaywright]);

  useEffect(() => {
    if (resultItems.length === 0) {
      setResultSummary(null);
      setResultConfidence(null);
      setResultEvidence([]);
      setResultClusters([]);
      setResultCitations([]);
      return;
    }
    const item = resultItems[selectedResultIndex];
    if (isResearchResultItem(item)) {
      setResultSummary(item.summary ?? null);
      setResultConfidence(item.confidence ?? null);
      setResultEvidence(item.evidence ?? []);
      setResultClusters(item.clusters ?? []);
      setResultCitations(item.citations ?? []);
    } else {
      setResultSummary(null);
      setResultConfidence(null);
      setResultEvidence([]);
      setResultClusters([]);
      setResultCitations([]);
    }
  }, [selectedResultIndex, resultItems]);

  async function submitScrape() {
    if (!scrapeUrl) {
      setError("Scrape URL is required.");
      return;
    }
    setLoading(true);
    try {
      const { error: apiError } = await postV1Scrape({
        baseUrl: getApiBaseUrl(),
        body: buildScrapeRequest(
          scrapeUrl,
          headless,
          usePlaywright,
          timeoutSeconds,
          authProfile || undefined,
          buildAuth(
            authBasic,
            headerMap,
            cookieList,
            queryMap,
            loginUrl,
            loginUserSelector,
            loginPassSelector,
            loginSubmitSelector,
            loginUser,
            loginPass,
          ),
          {
            template: extractTemplate || undefined,
            validate: extractValidate,
          },
          preProcessors,
          postProcessors,
          transformers,
          incremental,
        ),
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
        baseUrl: getApiBaseUrl(),
        body: buildCrawlRequest(
          crawlUrl,
          maxDepth,
          maxPages,
          headless,
          usePlaywright,
          timeoutSeconds,
          authProfile || undefined,
          buildAuth(
            authBasic,
            headerMap,
            cookieList,
            queryMap,
            loginUrl,
            loginUserSelector,
            loginPassSelector,
            loginSubmitSelector,
            loginUser,
            loginPass,
          ),
          {
            template: extractTemplate || undefined,
            validate: extractValidate,
          },
          preProcessors,
          postProcessors,
          transformers,
          incremental,
        ),
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
        baseUrl: getApiBaseUrl(),
        body: buildResearchRequest(
          researchQuery,
          parseUrlList(researchUrls),
          maxDepth,
          maxPages,
          headless,
          usePlaywright,
          timeoutSeconds,
          authProfile || undefined,
          buildAuth(
            authBasic,
            headerMap,
            cookieList,
            queryMap,
            loginUrl,
            loginUserSelector,
            loginPassSelector,
            loginSubmitSelector,
            loginUser,
            loginPass,
          ),
          {
            template: extractTemplate || undefined,
            validate: extractValidate,
          },
          preProcessors,
          postProcessors,
          transformers,
          incremental,
        ),
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

  async function loadResults(jobId: string, format: string = "jsonl") {
    setSelectedJobId(jobId);
    setResultItems([]);
    setSelectedResultIndex(0);
    setResultSummary(null);
    setResultConfidence(null);
    setResultEvidence([]);
    setResultClusters([]);
    setResultCitations([]);
    setRawResult(null);
    setResultFormat(format);
    try {
      const resultsUrl = buildApiUrl(
        `/v1/jobs/${jobId}/results?format=${format}`,
      );
      const response = await fetch(resultsUrl);

      if (!response.ok) {
        let errorMessage = `Failed to load results (${response.status} ${response.statusText})`;
        try {
          const errorData = (await response.json()) as { error?: string };
          if (errorData.error) {
            errorMessage = errorData.error;
          }
        } catch {
          // If parsing error body fails, use default message
        }
        setError(errorMessage);
        return;
      }

      // Handle based on format
      if (format === "jsonl") {
        const text = await response.text();
        const lines = text.split("\n").filter((line) => line.trim());

        const parsedItems: ResultItem[] = [];
        for (const line of lines) {
          try {
            const parsed = JSON.parse(line);
            parsedItems.push(parsed);
          } catch {
            // Skip malformed JSON lines
          }
        }

        // Check if we had input but failed to parse anything
        if (parsedItems.length === 0 && lines.length > 0) {
          setError("No valid results found. Results file may be corrupted.");
          return;
        }

        setResultItems(parsedItems);
        setRawResult(text);
      } else {
        // For other formats, just store raw text for display
        const text = await response.text();
        setRawResult(text);
        // Don't try to parse JSONL for json/md/csv
        setResultItems([]);
      }
    } catch (err) {
      setError(String(err));
    }
  }

  async function cancelJob(jobId: string) {
    setLoading(true);
    try {
      const { error: apiError } = await deleteV1JobsById({
        baseUrl: getApiBaseUrl(),
        path: { id: jobId },
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

  async function deleteJob(jobId: string) {
    if (!confirm("Are you sure you want to permanently delete this job?")) {
      return;
    }
    setLoading(true);
    try {
      const { error: apiError } = await deleteV1JobsById({
        baseUrl: getApiBaseUrl(),
        path: { id: jobId },
        query: { force: true },
      });
      if (apiError) {
        setError(String(apiError));
        return;
      }
      setError(null);
      await refreshJobs();
      if (selectedJobId === jobId) {
        setSelectedJobId(null);
        setResultItems([]);
      }
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
          {managerStatus !== null ? (
            <>
              <div>Queued: {managerStatus.queued}</div>
              <div>Active: {managerStatus.active}</div>
            </>
          ) : null}
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
          <label htmlFor="auth-profile" style={{ marginTop: 12 }}>
            Auth Profile
          </label>
          <select
            id="auth-profile"
            value={authProfile}
            onChange={(event) => setAuthProfile(event.target.value)}
          >
            <option value="">None</option>
            {profiles.map((p) => (
              <option key={p.name} value={p.name}>
                {p.name}{" "}
                {p.parents.length > 0
                  ? `(extends: ${p.parents.join(", ")})`
                  : ""}
              </option>
            ))}
          </select>
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
          <label htmlFor="cookies-raw" style={{ marginTop: 12 }}>
            Cookies (one per line: name=value)
          </label>
          <textarea
            id="cookies-raw"
            rows={2}
            value={cookiesRaw}
            onChange={(event) => setCookiesRaw(event.target.value)}
            placeholder="session_id=abc123&#10;auth_token=xyz789"
          />
          <label htmlFor="query-raw" style={{ marginTop: 12 }}>
            Query params (one per line: key=value)
          </label>
          <textarea
            id="query-raw"
            rows={2}
            value={queryRaw}
            onChange={(event) => setQueryRaw(event.target.value)}
            placeholder="api_key=your_key&#10;version=v1"
          />
          <details>
            <summary
              style={{
                cursor: "pointer",
                marginBottom: "8px",
                color: "var(--accent)",
              }}
            >
              Login Flow Configuration (Headless Auth)
            </summary>
            <div
              style={{
                marginTop: "12px",
                padding: "12px",
                borderRadius: "12px",
                background: "rgba(0, 0, 0, 0.25)",
              }}
            >
              <label htmlFor="login-url">Login URL</label>
              <input
                id="login-url"
                value={loginUrl}
                onChange={(event) => setLoginUrl(event.target.value)}
                placeholder="https://example.com/login"
              />
              <div className="row" style={{ marginTop: "12px" }}>
                <label>
                  User Selector
                  <input
                    value={loginUserSelector}
                    onChange={(event) =>
                      setLoginUserSelector(event.target.value)
                    }
                    placeholder="#email"
                  />
                </label>
                <label>
                  Pass Selector
                  <input
                    value={loginPassSelector}
                    onChange={(event) =>
                      setLoginPassSelector(event.target.value)
                    }
                    placeholder="#password"
                  />
                </label>
              </div>
              <div className="row" style={{ marginTop: "12px" }}>
                <label>
                  Submit Selector
                  <input
                    value={loginSubmitSelector}
                    onChange={(event) =>
                      setLoginSubmitSelector(event.target.value)
                    }
                    placeholder="button[type=submit]"
                  />
                </label>
              </div>
              <div className="row" style={{ marginTop: "12px" }}>
                <label>
                  Username
                  <input
                    type="text"
                    value={loginUser}
                    onChange={(event) => setLoginUser(event.target.value)}
                    placeholder="you@example.com"
                  />
                </label>
                <label>
                  Password
                  <input
                    type="password"
                    value={loginPass}
                    onChange={(event) => setLoginPass(event.target.value)}
                    placeholder="•••••••"
                  />
                </label>
              </div>
            </div>
          </details>
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
          <details>
            <summary
              style={{
                cursor: "pointer",
                marginBottom: "8px",
                color: "var(--accent)",
              }}
            >
              Pipeline Options
            </summary>
            <div
              style={{
                marginTop: "12px",
                padding: "12px",
                borderRadius: "12px",
                background: "rgba(0, 0, 0, 0.25)",
              }}
            >
              <label htmlFor="scrape-pre-processors">
                Pre-Processors (comma-separated)
              </label>
              <input
                id="scrape-pre-processors"
                value={preProcessors}
                onChange={(event) => setPreProcessors(event.target.value)}
                placeholder="redact, sanitize"
              />
              <label htmlFor="scrape-post-processors" style={{ marginTop: 12 }}>
                Post-Processors (comma-separated)
              </label>
              <input
                id="scrape-post-processors"
                value={postProcessors}
                onChange={(event) => setPostProcessors(event.target.value)}
                placeholder="cleanup, normalize"
              />
              <label htmlFor="scrape-transformers" style={{ marginTop: 12 }}>
                Transformers (comma-separated)
              </label>
              <input
                id="scrape-transformers"
                value={transformers}
                onChange={(event) => setTransformers(event.target.value)}
                placeholder="json-clean, csv-export"
              />
              <label style={{ marginTop: 12 }}>
                <input
                  type="checkbox"
                  checked={incremental}
                  onChange={(event) => setIncremental(event.target.checked)}
                />{" "}
                Incremental (ETag/Hash tracking)
              </label>
            </div>
          </details>
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
          <label htmlFor="crawl-auth-profile" style={{ marginTop: 12 }}>
            Auth Profile
          </label>
          <select
            id="crawl-auth-profile"
            value={authProfile}
            onChange={(event) => setAuthProfile(event.target.value)}
          >
            <option value="">None</option>
            {profiles.map((p) => (
              <option key={p.name} value={p.name}>
                {p.name}{" "}
                {p.parents.length > 0
                  ? `(extends: ${p.parents.join(", ")})`
                  : ""}
              </option>
            ))}
          </select>
          <details>
            <summary
              style={{
                cursor: "pointer",
                marginBottom: "8px",
                color: "var(--accent)",
              }}
            >
              Login Flow Configuration (Headless Auth)
            </summary>
            <div
              style={{
                marginTop: "12px",
                padding: "12px",
                borderRadius: "12px",
                background: "rgba(0, 0, 0, 0.25)",
              }}
            >
              <label htmlFor="crawl-login-url">Login URL</label>
              <input
                id="crawl-login-url"
                value={loginUrl}
                onChange={(event) => setLoginUrl(event.target.value)}
                placeholder="https://example.com/login"
              />
              <div className="row" style={{ marginTop: "12px" }}>
                <label>
                  User Selector
                  <input
                    value={loginUserSelector}
                    onChange={(event) =>
                      setLoginUserSelector(event.target.value)
                    }
                    placeholder="#email"
                  />
                </label>
                <label>
                  Pass Selector
                  <input
                    value={loginPassSelector}
                    onChange={(event) =>
                      setLoginPassSelector(event.target.value)
                    }
                    placeholder="#password"
                  />
                </label>
              </div>
              <div className="row" style={{ marginTop: "12px" }}>
                <label>
                  Submit Selector
                  <input
                    value={loginSubmitSelector}
                    onChange={(event) =>
                      setLoginSubmitSelector(event.target.value)
                    }
                    placeholder="button[type=submit]"
                  />
                </label>
              </div>
              <div className="row" style={{ marginTop: "12px" }}>
                <label>
                  Username
                  <input
                    type="text"
                    value={loginUser}
                    onChange={(event) => setLoginUser(event.target.value)}
                    placeholder="you@example.com"
                  />
                </label>
                <label>
                  Password
                  <input
                    type="password"
                    value={loginPass}
                    onChange={(event) => setLoginPass(event.target.value)}
                    placeholder="•••••••"
                  />
                </label>
              </div>
            </div>
          </details>
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
          <details>
            <summary
              style={{
                cursor: "pointer",
                marginBottom: "8px",
                color: "var(--accent)",
              }}
            >
              Pipeline Options
            </summary>
            <div
              style={{
                marginTop: "12px",
                padding: "12px",
                borderRadius: "12px",
                background: "rgba(0, 0, 0, 0.25)",
              }}
            >
              <label htmlFor="crawl-pre-processors">
                Pre-Processors (comma-separated)
              </label>
              <input
                id="crawl-pre-processors"
                value={preProcessors}
                onChange={(event) => setPreProcessors(event.target.value)}
                placeholder="redact, sanitize"
              />
              <label htmlFor="crawl-post-processors" style={{ marginTop: 12 }}>
                Post-Processors (comma-separated)
              </label>
              <input
                id="crawl-post-processors"
                value={postProcessors}
                onChange={(event) => setPostProcessors(event.target.value)}
                placeholder="cleanup, normalize"
              />
              <label htmlFor="crawl-transformers" style={{ marginTop: 12 }}>
                Transformers (comma-separated)
              </label>
              <input
                id="crawl-transformers"
                value={transformers}
                onChange={(event) => setTransformers(event.target.value)}
                placeholder="json-clean, csv-export"
              />
              <label style={{ marginTop: 12 }}>
                <input
                  type="checkbox"
                  checked={incremental}
                  onChange={(event) => setIncremental(event.target.checked)}
                />{" "}
                Incremental (ETag/Hash tracking)
              </label>
            </div>
          </details>
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
          <label htmlFor="research-auth-profile" style={{ marginTop: 12 }}>
            Auth Profile
          </label>
          <select
            id="research-auth-profile"
            value={authProfile}
            onChange={(event) => setAuthProfile(event.target.value)}
          >
            <option value="">None</option>
            {profiles.map((p) => (
              <option key={p.name} value={p.name}>
                {p.name}{" "}
                {p.parents.length > 0
                  ? `(extends: ${p.parents.join(", ")})`
                  : ""}
              </option>
            ))}
          </select>
          <details>
            <summary
              style={{
                cursor: "pointer",
                marginBottom: "8px",
                color: "var(--accent)",
              }}
            >
              Login Flow Configuration (Headless Auth)
            </summary>
            <div
              style={{
                marginTop: "12px",
                padding: "12px",
                borderRadius: "12px",
                background: "rgba(0, 0, 0, 0.25)",
              }}
            >
              <label htmlFor="research-login-url">Login URL</label>
              <input
                id="research-login-url"
                value={loginUrl}
                onChange={(event) => setLoginUrl(event.target.value)}
                placeholder="https://example.com/login"
              />
              <div className="row" style={{ marginTop: "12px" }}>
                <label>
                  User Selector
                  <input
                    value={loginUserSelector}
                    onChange={(event) =>
                      setLoginUserSelector(event.target.value)
                    }
                    placeholder="#email"
                  />
                </label>
                <label>
                  Pass Selector
                  <input
                    value={loginPassSelector}
                    onChange={(event) =>
                      setLoginPassSelector(event.target.value)
                    }
                    placeholder="#password"
                  />
                </label>
              </div>
              <div className="row" style={{ marginTop: "12px" }}>
                <label>
                  Submit Selector
                  <input
                    value={loginSubmitSelector}
                    onChange={(event) =>
                      setLoginSubmitSelector(event.target.value)
                    }
                    placeholder="button[type=submit]"
                  />
                </label>
              </div>
              <div className="row" style={{ marginTop: "12px" }}>
                <label>
                  Username
                  <input
                    type="text"
                    value={loginUser}
                    onChange={(event) => setLoginUser(event.target.value)}
                    placeholder="you@example.com"
                  />
                </label>
                <label>
                  Password
                  <input
                    type="password"
                    value={loginPass}
                    onChange={(event) => setLoginPass(event.target.value)}
                    placeholder="•••••••"
                  />
                </label>
              </div>
            </div>
          </details>
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
          <details>
            <summary
              style={{
                cursor: "pointer",
                marginBottom: "8px",
                color: "var(--accent)",
              }}
            >
              Pipeline Options
            </summary>
            <div
              style={{
                marginTop: "12px",
                padding: "12px",
                borderRadius: "12px",
                background: "rgba(0, 0, 0, 0.25)",
              }}
            >
              <label htmlFor="research-pre-processors">
                Pre-Processors (comma-separated)
              </label>
              <input
                id="research-pre-processors"
                value={preProcessors}
                onChange={(event) => setPreProcessors(event.target.value)}
                placeholder="redact, sanitize"
              />
              <label
                htmlFor="research-post-processors"
                style={{ marginTop: 12 }}
              >
                Post-Processors (comma-separated)
              </label>
              <input
                id="research-post-processors"
                value={postProcessors}
                onChange={(event) => setPostProcessors(event.target.value)}
                placeholder="cleanup, normalize"
              />
              <label htmlFor="research-transformers" style={{ marginTop: 12 }}>
                Transformers (comma-separated)
              </label>
              <input
                id="research-transformers"
                value={transformers}
                onChange={(event) => setTransformers(event.target.value)}
                placeholder="json-clean, csv-export"
              />
              <label style={{ marginTop: 12 }}>
                <input
                  type="checkbox"
                  checked={incremental}
                  onChange={(event) => setIncremental(event.target.checked)}
                />{" "}
                Incremental (ETag/Hash tracking)
              </label>
            </div>
          </details>
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
                <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
                  {job.status === "succeeded" ? (
                    <button
                      type="button"
                      className="secondary"
                      onClick={() =>
                        void loadResults(job.id ?? "", resultFormat)
                      }
                    >
                      View Results
                    </button>
                  ) : null}
                  {job.status === "queued" || job.status === "running" ? (
                    <button
                      type="button"
                      className="secondary"
                      onClick={() => void cancelJob(job.id ?? "")}
                    >
                      Cancel
                    </button>
                  ) : null}
                  <button
                    type="button"
                    className="secondary"
                    onClick={() => void deleteJob(job.id ?? "")}
                    style={{ color: "#ff6b6b" }}
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
        {selectedJobId ? (
          <div className="panel" style={{ marginTop: 16 }}>
            <h3>Results: {selectedJobId}</h3>
            {resultItems.length > 1 ? (
              <div className="result-navigation">
                <div className="result-counter">
                  Showing {selectedResultIndex + 1} of {resultItems.length}{" "}
                  results
                </div>
                <div className="result-nav-buttons">
                  <button
                    type="button"
                    className="secondary"
                    onClick={() =>
                      setSelectedResultIndex((i) => Math.max(0, i - 1))
                    }
                    disabled={selectedResultIndex === 0}
                  >
                    ← Previous
                  </button>
                  <button
                    type="button"
                    className="secondary"
                    onClick={() =>
                      setSelectedResultIndex((i) =>
                        Math.min(resultItems.length - 1, i + 1),
                      )
                    }
                    disabled={selectedResultIndex === resultItems.length - 1}
                  >
                    Next →
                  </button>
                </div>
              </div>
            ) : null}
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
            {resultItems.length > 0 ? (
              <div style={{ marginTop: 12 }}>
                <h4>Results List</h4>
                <div className="result-items-list">
                  {resultItems.map((item, index) => {
                    const isCrawl = isCrawlResultItem(item);
                    const itemKey = isCrawl ? item.url : `result-${index}`;
                    return (
                      <button
                        key={itemKey}
                        type="button"
                        className={`result-item ${index === selectedResultIndex ? "selected" : ""}`}
                        onClick={() => setSelectedResultIndex(index)}
                      >
                        {isCrawl ? (
                          <>
                            <div className="result-item-header">
                              <span className="result-item-url">
                                {item.url}
                              </span>
                              <span
                                className={`badge ${statusClass(String(item.status))}`}
                              >
                                {item.status}
                              </span>
                            </div>
                            <div className="result-item-title">
                              {item.title || "Untitled"}
                            </div>
                            {item.links?.length ? (
                              <div className="result-item-meta">
                                {item.links.length} links
                              </div>
                            ) : null}
                          </>
                        ) : (
                          <div className="result-item-non-crawl">
                            Result {index + 1} (research/aggregated)
                          </div>
                        )}
                      </button>
                    );
                  })}
                </div>
                {resultItems.length > 0 ? (
                  <details style={{ marginTop: 12 }}>
                    <summary>Normalized Data (Selected Item)</summary>
                    <NormalizedView
                      raw={rawResult ?? ""}
                      index={selectedResultIndex}
                    />
                  </details>
                ) : null}
                <details style={{ marginTop: 8 }}>
                  <summary>Raw output</summary>
                  <pre>{rawResult}</pre>
                </details>
              </div>
            ) : null}
          </div>
        ) : null}
      </section>

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

      <div className="footer">
        Spartan Scraper — build once, deploy everywhere.
      </div>
    </div>
  );
}

function NormalizedView({ raw, index }: { raw: string; index?: number }) {
  try {
    const lines = raw.split("\n").filter((line) => line.trim());
    const targetIndex = index ?? 0;
    if (targetIndex >= lines.length) return null;
    const data = JSON.parse(lines[targetIndex]);
    // Only crawl results have normalized data
    if (!isCrawlResultItem(data) || !data.normalized) {
      return <div>No normalized data found for this result type.</div>;
    }
    return (
      <pre style={{ background: "rgba(0, 50, 50, 0.3)" }}>
        {JSON.stringify(data.normalized, null, 2)}
      </pre>
    );
  } catch {
    return <div>Failed to parse result.</div>;
  }
}

function isCrawlResultItem(item: ResultItem): item is CrawlResultItem {
  return "url" in item && "status" in item;
}

function isResearchResultItem(item: ResultItem): item is ResearchResultItem {
  // Must explicitly NOT be a crawl result (no url/status)
  // AND must have at least one research field
  const isNotCrawl = !("url" in item && "status" in item);
  const hasResearchField =
    "summary" in item ||
    "confidence" in item ||
    "evidence" in item ||
    "clusters" in item ||
    "citations" in item;
  return isNotCrawl && hasResearchField;
}

function statusClass(status: string) {
  switch (status) {
    case "succeeded":
      return "success";
    case "failed":
      return "failed";
    case "canceled":
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

export function parseCookies(raw: string): string[] | undefined {
  if (!raw.trim()) {
    return undefined;
  }
  const cookies: string[] = raw
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
  return cookies.length ? cookies : undefined;
}

export function parseQueryParams(
  raw: string,
): Record<string, string> | undefined {
  if (!raw.trim()) {
    return undefined;
  }
  const params: Record<string, string> = {};
  raw
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
    .forEach((line) => {
      const idx = line.indexOf("=");
      if (idx > 0) {
        const key = line.slice(0, idx).trim();
        const value = line.slice(idx + 1).trim();
        if (key && value) {
          params[key] = value;
        }
      }
    });
  return Object.keys(params).length ? params : undefined;
}

export function parseProcessors(raw: string): string[] | undefined {
  if (!raw.trim()) {
    return undefined;
  }
  const processors = raw
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
  return processors.length ? processors : undefined;
}

export function buildAuth(
  basic: string,
  headers?: Record<string, string>,
  cookies?: string[],
  query?: Record<string, string>,
  loginUrl?: string,
  loginUserSelector?: string,
  loginPassSelector?: string,
  loginSubmitSelector?: string,
  loginUser?: string,
  loginPass?: string,
) {
  if (
    !basic &&
    !headers &&
    !cookies &&
    !query &&
    !loginUrl &&
    !loginUserSelector &&
    !loginPassSelector &&
    !loginSubmitSelector &&
    !loginUser &&
    !loginPass
  ) {
    return undefined;
  }
  return {
    basic: basic || undefined,
    headers,
    cookies,
    query,
    loginUrl: loginUrl || undefined,
    loginUserSelector: loginUserSelector || undefined,
    loginPassSelector: loginPassSelector || undefined,
    loginSubmitSelector: loginSubmitSelector || undefined,
    loginUser: loginUser || undefined,
    loginPass: loginPass || undefined,
  };
}

function parseUrlList(raw: string) {
  return raw
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function buildPipelineOptions(
  preProcessorsRaw: string,
  postProcessorsRaw: string,
  transformersRaw: string,
) {
  const pre = parseProcessors(preProcessorsRaw);
  const post = parseProcessors(postProcessorsRaw);
  const trans = parseProcessors(transformersRaw);

  if (!pre && !post && !trans) {
    return undefined;
  }

  return {
    preProcessors: pre,
    postProcessors: post,
    transformers: trans,
  };
}

export function buildScrapeRequest(
  url: string,
  headless: boolean,
  usePlaywright: boolean,
  timeoutSeconds: number,
  authProfile: string | undefined,
  auth: ReturnType<typeof buildAuth>,
  extract: ExtractOptions | undefined,
  preProcessors: string,
  postProcessors: string,
  transformers: string,
  incremental: boolean,
): ScrapeRequest {
  return {
    url,
    headless,
    playwright: headless ? usePlaywright : false,
    timeoutSeconds,
    authProfile: authProfile || undefined,
    auth,
    extract,
    pipeline: buildPipelineOptions(preProcessors, postProcessors, transformers),
    incremental: incremental || undefined,
  };
}

export function buildCrawlRequest(
  url: string,
  maxDepth: number,
  maxPages: number,
  headless: boolean,
  usePlaywright: boolean,
  timeoutSeconds: number,
  authProfile: string | undefined,
  auth: ReturnType<typeof buildAuth>,
  extract: ExtractOptions | undefined,
  preProcessors: string,
  postProcessors: string,
  transformers: string,
  incremental: boolean,
): CrawlRequest {
  return {
    url,
    maxDepth,
    maxPages,
    headless,
    playwright: headless ? usePlaywright : false,
    timeoutSeconds,
    authProfile: authProfile || undefined,
    auth,
    extract,
    pipeline: buildPipelineOptions(preProcessors, postProcessors, transformers),
    incremental: incremental || undefined,
  };
}

export function buildResearchRequest(
  query: string,
  urls: string[],
  maxDepth: number,
  maxPages: number,
  headless: boolean,
  usePlaywright: boolean,
  timeoutSeconds: number,
  authProfile: string | undefined,
  auth: ReturnType<typeof buildAuth>,
  extract: ExtractOptions | undefined,
  preProcessors: string,
  postProcessors: string,
  transformers: string,
  incremental: boolean,
): ResearchRequest {
  return {
    query,
    urls,
    maxDepth,
    maxPages,
    headless,
    playwright: headless ? usePlaywright : false,
    timeoutSeconds,
    authProfile: authProfile || undefined,
    auth,
    extract,
    pipeline: buildPipelineOptions(preProcessors, postProcessors, transformers),
    incremental: incremental || undefined,
  };
}
