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
import { useCallback, useEffect, useRef, useState } from "react";
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
  type ScrapeRequest,
  type CrawlRequest,
  type ResearchRequest,
} from "./api";
import { buildApiUrl, getApiBaseUrl } from "./lib/api-config";
import { loadResults as loadResultsUtil } from "./lib/results";
import { Hero } from "./components/Hero";
import { JobList } from "./components/JobList";
import { ResultsViewer } from "./components/ResultsViewer";
import { InfoSections } from "./components/InfoSections";
import { ScrapeForm } from "./components/ScrapeForm";
import { CrawlForm } from "./components/CrawlForm";
import { ResearchForm } from "./components/ResearchForm";
import type {
  JobEntry,
  ResultItem,
  EvidenceItem,
  ClusterItem,
  CitationItem,
} from "./types";

const defaultHeaders = "";

export function App() {
  const [headless, setHeadless] = useState(false);
  const [usePlaywright, setUsePlaywright] = useState(false);
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
  const [maxDepth, setMaxDepth] = useState(2);
  const [maxPages, setMaxPages] = useState(200);

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
  const [currentPage, setCurrentPage] = useState(1);
  const [totalResults, setTotalResults] = useState(0);
  const [resultsPerPage] = useState(100);

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
  const [crawlStates, setCrawlStates] = useState<import("./api").CrawlState[]>(
    [],
  );

  const selectedJobIdRef = useRef<string | null>(null);
  const resultFormatRef = useRef<string>("jsonl");

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
        .map((p) => ({
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

  const loadResults = useCallback(
    async (jobId: string, format: string = "jsonl", page: number = 1) => {
      setSelectedJobId(jobId);
      setResultFormat(format);

      if (page === 1) {
        setCurrentPage(1);
        setTotalResults(0);
        setResultItems([]);
        setSelectedResultIndex(0);
        setResultSummary(null);
        setResultConfidence(null);
        setResultEvidence([]);
        setResultClusters([]);
        setResultCitations([]);
        setRawResult(null);
      }

      const result = await loadResultsUtil(jobId, format, page, resultsPerPage);

      if (result.error) {
        setError(result.error);
        return;
      }

      // Handle jsonl pagination response headers
      if (format === "jsonl" && result.data) {
        try {
          const resultsUrl = buildApiUrl(
            `/v1/jobs/${jobId}/results?format=${format}&limit=${resultsPerPage}&offset=${(page - 1) * resultsPerPage}`,
          );
          const response = await fetch(resultsUrl, { method: "HEAD" });
          const totalCountStr = response.headers.get("X-Total-Count");
          if (totalCountStr) {
            setTotalResults(parseInt(totalCountStr, 10));
          }
        } catch {
          // Ignore header fetch errors; results are still valid
        }

        setResultItems(result.data as ResultItem[]);
        setRawResult(JSON.stringify(result.data, null, 2));
      } else if (result.raw) {
        // For other formats, store raw text for display
        setRawResult(result.raw);
        setResultItems([]);
      }
    },
    [resultsPerPage],
  );

  // biome-ignore lint/correctness/useExhaustiveDependencies: using refs to avoid circular dependency on loadResults
  useEffect(() => {
    const jobId = selectedJobIdRef.current;
    const fmt = resultFormatRef.current;
    if (jobId) {
      void loadResults(jobId, fmt);
    }
  }, [resultFormat, loadResults]);

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
    if (item && "summary" in item) {
      setResultSummary((item as { summary?: string }).summary ?? null);
      setResultConfidence((item as { confidence?: number }).confidence ?? null);
      setResultEvidence((item as { evidence?: EvidenceItem[] }).evidence ?? []);
      setResultClusters((item as { clusters?: ClusterItem[] }).clusters ?? []);
      setResultCitations(
        (item as { citations?: CitationItem[] }).citations ?? [],
      );
    } else {
      setResultSummary(null);
      setResultConfidence(null);
      setResultEvidence([]);
      setResultClusters([]);
      setResultCitations([]);
    }
  }, [selectedResultIndex, resultItems]);

  const handleSubmitScrape = useCallback(
    async (request: ScrapeRequest) => {
      setLoading(true);
      try {
        const { error: apiError } = await postV1Scrape({
          baseUrl: getApiBaseUrl(),
          body: request,
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
    },
    [refreshJobs],
  );

  const handleSubmitCrawl = useCallback(
    async (request: CrawlRequest) => {
      setLoading(true);
      try {
        const { error: apiError } = await postV1Crawl({
          baseUrl: getApiBaseUrl(),
          body: request,
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
    },
    [refreshJobs],
  );

  const handleSubmitResearch = useCallback(
    async (request: ResearchRequest) => {
      setLoading(true);
      try {
        const { error: apiError } = await postV1Research({
          baseUrl: getApiBaseUrl(),
          body: request,
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
    },
    [refreshJobs],
  );

  const cancelJob = useCallback(
    async (jobId: string) => {
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
    },
    [refreshJobs],
  );

  const deleteJob = useCallback(
    async (jobId: string) => {
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
    },
    [refreshJobs, selectedJobId],
  );

  return (
    <div className="app">
      <Hero
        loading={loading}
        managerStatus={managerStatus}
        jobsCount={jobs.length}
        headless={headless}
        usePlaywright={usePlaywright}
      />

      <section className="grid">
        <ScrapeForm
          headless={headless}
          setHeadless={setHeadless}
          usePlaywright={usePlaywright}
          setUsePlaywright={setUsePlaywright}
          timeoutSeconds={timeoutSeconds}
          setTimeoutSeconds={setTimeoutSeconds}
          authProfile={authProfile}
          setAuthProfile={setAuthProfile}
          authBasic={authBasic}
          setAuthBasic={setAuthBasic}
          headersRaw={headersRaw}
          setHeadersRaw={setHeadersRaw}
          cookiesRaw={cookiesRaw}
          setCookiesRaw={setCookiesRaw}
          queryRaw={queryRaw}
          setQueryRaw={setQueryRaw}
          loginUrl={loginUrl}
          setLoginUrl={setLoginUrl}
          loginUserSelector={loginUserSelector}
          setLoginUserSelector={setLoginUserSelector}
          loginPassSelector={loginPassSelector}
          setLoginPassSelector={setLoginPassSelector}
          loginSubmitSelector={loginSubmitSelector}
          setLoginSubmitSelector={setLoginSubmitSelector}
          loginUser={loginUser}
          setLoginUser={setLoginUser}
          loginPass={loginPass}
          setLoginPass={setLoginPass}
          extractTemplate={extractTemplate}
          setExtractTemplate={setExtractTemplate}
          extractValidate={extractValidate}
          setExtractValidate={setExtractValidate}
          preProcessors={preProcessors}
          setPreProcessors={setPreProcessors}
          postProcessors={postProcessors}
          setPostProcessors={setPostProcessors}
          transformers={transformers}
          setTransformers={setTransformers}
          incremental={incremental}
          setIncremental={setIncremental}
          profiles={profiles}
          onSubmit={handleSubmitScrape}
          loading={loading}
        />

        <CrawlForm
          headless={headless}
          setHeadless={setHeadless}
          usePlaywright={usePlaywright}
          setUsePlaywright={setUsePlaywright}
          timeoutSeconds={timeoutSeconds}
          setTimeoutSeconds={setTimeoutSeconds}
          authProfile={authProfile}
          setAuthProfile={setAuthProfile}
          authBasic={authBasic}
          setAuthBasic={setAuthBasic}
          headersRaw={headersRaw}
          setHeadersRaw={setHeadersRaw}
          cookiesRaw={cookiesRaw}
          setCookiesRaw={setCookiesRaw}
          queryRaw={queryRaw}
          setQueryRaw={setQueryRaw}
          loginUrl={loginUrl}
          setLoginUrl={setLoginUrl}
          loginUserSelector={loginUserSelector}
          setLoginUserSelector={setLoginUserSelector}
          loginPassSelector={loginPassSelector}
          setLoginPassSelector={setLoginPassSelector}
          loginSubmitSelector={loginSubmitSelector}
          setLoginSubmitSelector={setLoginSubmitSelector}
          loginUser={loginUser}
          setLoginUser={setLoginUser}
          loginPass={loginPass}
          setLoginPass={setLoginPass}
          extractTemplate={extractTemplate}
          setExtractTemplate={setExtractTemplate}
          extractValidate={extractValidate}
          setExtractValidate={setExtractValidate}
          preProcessors={preProcessors}
          setPreProcessors={setPreProcessors}
          postProcessors={postProcessors}
          setPostProcessors={setPostProcessors}
          transformers={transformers}
          setTransformers={setTransformers}
          incremental={incremental}
          setIncremental={setIncremental}
          profiles={profiles}
          onSubmit={handleSubmitCrawl}
          loading={loading}
        />

        <ResearchForm
          maxDepth={maxDepth}
          setMaxDepth={setMaxDepth}
          maxPages={maxPages}
          setMaxPages={setMaxPages}
          headless={headless}
          setHeadless={setHeadless}
          usePlaywright={usePlaywright}
          setUsePlaywright={setUsePlaywright}
          timeoutSeconds={timeoutSeconds}
          setTimeoutSeconds={setTimeoutSeconds}
          authProfile={authProfile}
          setAuthProfile={setAuthProfile}
          authBasic={authBasic}
          setAuthBasic={setAuthBasic}
          headersRaw={headersRaw}
          setHeadersRaw={setHeadersRaw}
          cookiesRaw={cookiesRaw}
          setCookiesRaw={setCookiesRaw}
          queryRaw={queryRaw}
          setQueryRaw={setQueryRaw}
          loginUrl={loginUrl}
          setLoginUrl={setLoginUrl}
          loginUserSelector={loginUserSelector}
          setLoginUserSelector={setLoginUserSelector}
          loginPassSelector={loginPassSelector}
          setLoginPassSelector={setLoginPassSelector}
          loginSubmitSelector={loginSubmitSelector}
          setLoginSubmitSelector={setLoginSubmitSelector}
          loginUser={loginUser}
          setLoginUser={setLoginUser}
          loginPass={loginPass}
          setLoginPass={setLoginPass}
          extractTemplate={extractTemplate}
          setExtractTemplate={setExtractTemplate}
          extractValidate={extractValidate}
          setExtractValidate={setExtractValidate}
          preProcessors={preProcessors}
          setPreProcessors={setPreProcessors}
          postProcessors={postProcessors}
          setPostProcessors={setPostProcessors}
          transformers={transformers}
          setTransformers={setTransformers}
          incremental={incremental}
          setIncremental={setIncremental}
          profiles={profiles}
          onSubmit={handleSubmitResearch}
          loading={loading}
        />
      </section>

      <JobList
        jobs={jobs}
        error={error}
        onViewResults={loadResults}
        onCancel={cancelJob}
        onDelete={deleteJob}
        onRefresh={refreshJobs}
      />

      <ResultsViewer
        jobId={selectedJobId}
        resultItems={resultItems}
        selectedResultIndex={selectedResultIndex}
        setSelectedResultIndex={setSelectedResultIndex}
        resultSummary={resultSummary}
        resultConfidence={resultConfidence}
        resultEvidence={resultEvidence}
        resultClusters={resultClusters}
        resultCitations={resultCitations}
        rawResult={rawResult}
        resultFormat={resultFormat}
        currentPage={currentPage}
        totalResults={totalResults}
        resultsPerPage={resultsPerPage}
        onLoadPage={setCurrentPage}
      />

      <InfoSections
        profiles={profiles}
        schedules={schedules}
        templates={templates}
        crawlStates={crawlStates}
      />

      <div className="footer">
        Spartan Scraper — build once, deploy everywhere.
      </div>
    </div>
  );
}
