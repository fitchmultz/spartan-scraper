/**
 * Batch Utilities Module
 *
 * Shared utility functions for batch operations including URL list parsing,
 * CSV/JSON file parsing, batch request building, and batch status calculations.
 *
 * @module batch-utils
 */

import type {
  AiExtractOptions,
  AuthOptions,
  BatchCrawlRequest,
  BatchJobStats,
  BatchJobRequest,
  BatchResearchRequest,
  BatchScrapeRequest,
  DeviceEmulation,
  ExtractOptions,
  NetworkInterceptConfig,
  PipelineOptions,
  ResearchAgenticConfig,
  ScreenshotConfig,
  WebhookConfig,
} from "../api";
import { splitAndTrim } from "./input-parsing";

/**
 * Parse URL list from textarea input (newline or comma-separated)
 */
export function parseUrlList(input: string): string[] {
  return splitAndTrim(input, /[\n,]/);
}

/**
 * Validate that all URLs are valid
 * Returns array of invalid URLs, empty array if all valid
 */
export function validateUrls(urls: string[]): string[] {
  return urls.filter((url) => {
    try {
      new URL(url);
      return false;
    } catch {
      return true;
    }
  });
}

/**
 * Validate batch size (max 100)
 */
export function validateBatchSize(urls: string[]): {
  valid: boolean;
  error?: string;
} {
  const MAX_BATCH_SIZE = 100;
  if (urls.length === 0) {
    return { valid: false, error: "At least one URL is required" };
  }
  if (urls.length > MAX_BATCH_SIZE) {
    return { valid: false, error: `Maximum ${MAX_BATCH_SIZE} URLs allowed` };
  }
  return { valid: true };
}

/**
 * Parse CSV text to BatchJobRequest array
 * Assumes first column is URL, optional second column is method
 */
export function parseBatchCSV(csvText: string): BatchJobRequest[] {
  const lines = csvText.split("\n").filter((line) => line.trim());
  if (lines.length === 0) return [];

  // Detect if first line is header
  const firstLine = lines[0].toLowerCase();
  const hasHeader = firstLine.includes("url") && !firstLine.startsWith("http");
  const startIdx = hasHeader ? 1 : 0;

  return lines.slice(startIdx).map((line) => {
    const cols = line.split(",").map((col) => col.trim());
    const url = cols[0];
    const methodStr = cols[1]?.toUpperCase() || "GET";
    const validMethods: Array<BatchJobRequest["method"]> = [
      "GET",
      "POST",
      "PUT",
      "DELETE",
      "PATCH",
      "HEAD",
      "OPTIONS",
    ];
    const method = validMethods.includes(methodStr as BatchJobRequest["method"])
      ? (methodStr as BatchJobRequest["method"])
      : "GET";
    const body = cols[2] || undefined;
    const contentType = cols[3] || undefined;

    const result: BatchJobRequest = {
      url,
      method,
      body,
      contentType,
    };

    return result;
  });
}

/**
 * Parse JSON text to BatchJobRequest array
 */
export function parseBatchJSON(jsonText: string): BatchJobRequest[] {
  const parsed = JSON.parse(jsonText) as unknown;

  if (!Array.isArray(parsed)) {
    throw new Error("JSON must be an array of job objects");
  }

  return parsed.map((item: unknown) => {
    if (typeof item !== "object" || item === null) {
      throw new Error("Each item must be an object");
    }
    const obj = item as Record<string, unknown>;

    if (typeof obj.url !== "string") {
      throw new Error("Each item must have a 'url' string property");
    }

    const methodStr =
      typeof obj.method === "string" ? obj.method.toUpperCase() : "GET";
    const validMethods: Array<BatchJobRequest["method"]> = [
      "GET",
      "POST",
      "PUT",
      "DELETE",
      "PATCH",
      "HEAD",
      "OPTIONS",
    ];
    const method = validMethods.includes(methodStr as BatchJobRequest["method"])
      ? (methodStr as BatchJobRequest["method"])
      : "GET";

    const result: BatchJobRequest = {
      url: obj.url,
      method,
      body: typeof obj.body === "string" ? obj.body : undefined,
      contentType:
        typeof obj.contentType === "string" ? obj.contentType : undefined,
    };

    if (typeof obj.headers === "object" && obj.headers !== null) {
      result.headers = obj.headers as Record<string, string>;
    }

    return result;
  });
}

/**
 * Calculate progress percentage from batch stats
 */
export function calculateBatchProgress(
  stats: BatchJobStats,
  total: number,
): number {
  if (total === 0) return 0;
  const completed = stats.succeeded + stats.failed + stats.canceled;
  return Math.round((completed / total) * 100);
}

/**
 * Check if batch status is terminal (completed, failed, partial, or canceled)
 */
export function isTerminalStatus(
  status: string,
): status is "completed" | "failed" | "partial" | "canceled" {
  return ["completed", "failed", "partial", "canceled"].includes(status);
}

/**
 * Build batch scrape request
 */
export function buildBatchScrapeRequest(
  urls: string[],
  headless: boolean,
  usePlaywright: boolean,
  timeoutSeconds: number,
  authProfile: string | undefined,
  auth: AuthOptions | undefined,
  extract: ExtractOptions | undefined,
  pipeline: PipelineOptions | undefined,
  incremental: boolean,
  webhook: WebhookConfig | undefined,
  screenshot: ScreenshotConfig | undefined,
  device: DeviceEmulation | undefined,
  networkIntercept: NetworkInterceptConfig | undefined,
  aiExtract?: AiExtractOptions,
): BatchScrapeRequest {
  const jobs: BatchJobRequest[] = urls.map((url) => ({ url }));
  const mergedExtract: ExtractOptions | undefined =
    extract || aiExtract
      ? {
          ...extract,
          ai: aiExtract,
        }
      : undefined;

  return {
    jobs,
    headless,
    playwright: headless ? usePlaywright : false,
    timeoutSeconds,
    authProfile,
    auth,
    extract: mergedExtract,
    pipeline,
    incremental: incremental || undefined,
    webhook,
    screenshot,
    device,
    networkIntercept,
  };
}

/**
 * Build batch crawl request
 */
export function buildBatchCrawlRequest(
  urls: string[],
  maxDepth: number,
  maxPages: number,
  headless: boolean,
  usePlaywright: boolean,
  timeoutSeconds: number,
  authProfile: string | undefined,
  auth: AuthOptions | undefined,
  extract: ExtractOptions | undefined,
  pipeline: PipelineOptions | undefined,
  incremental: boolean,
  webhook: WebhookConfig | undefined,
  screenshot: ScreenshotConfig | undefined,
  device: DeviceEmulation | undefined,
  networkIntercept: NetworkInterceptConfig | undefined,
  aiExtract?: AiExtractOptions,
): BatchCrawlRequest {
  const jobs: BatchJobRequest[] = urls.map((url) => ({ url }));
  const mergedExtract: ExtractOptions | undefined =
    extract || aiExtract
      ? {
          ...extract,
          ai: aiExtract,
        }
      : undefined;

  return {
    jobs,
    maxDepth,
    maxPages,
    headless,
    playwright: headless ? usePlaywright : false,
    timeoutSeconds,
    authProfile,
    auth,
    extract: mergedExtract,
    pipeline,
    incremental: incremental || undefined,
    webhook,
    screenshot,
    device,
    networkIntercept,
  };
}

/**
 * Build batch research request
 */
export function buildBatchResearchRequest(
  urls: string[],
  query: string,
  maxDepth: number,
  maxPages: number,
  headless: boolean,
  usePlaywright: boolean,
  timeoutSeconds: number,
  authProfile: string | undefined,
  auth: AuthOptions | undefined,
  extract: ExtractOptions | undefined,
  pipeline: PipelineOptions | undefined,
  webhook: WebhookConfig | undefined,
  screenshot: ScreenshotConfig | undefined,
  device: DeviceEmulation | undefined,
  networkIntercept: NetworkInterceptConfig | undefined,
  aiExtract?: AiExtractOptions,
  agentic?: ResearchAgenticConfig,
): BatchResearchRequest {
  const jobs: BatchJobRequest[] = urls.map((url) => ({ url }));
  const mergedExtract: ExtractOptions | undefined =
    extract || aiExtract
      ? {
          ...extract,
          ai: aiExtract,
        }
      : undefined;

  return {
    jobs,
    query,
    maxDepth,
    maxPages,
    headless,
    playwright: headless ? usePlaywright : false,
    timeoutSeconds,
    authProfile,
    auth,
    extract: mergedExtract,
    pipeline,
    webhook,
    screenshot,
    device,
    networkIntercept,
    agentic,
  };
}

/**
 * Format batch status for display
 */
export function formatBatchStatus(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

/**
 * Get CSS class for batch status
 */
export function getStatusClass(status: string): string {
  switch (status) {
    case "completed":
      return "success";
    case "processing":
      return "running";
    case "failed":
    case "canceled":
      return "failed";
    case "partial":
      return "warning";
    default:
      return "";
  }
}
