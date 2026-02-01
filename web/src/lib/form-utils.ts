/**
 * Form Utilities Module
 *
 * Shared utility functions for parsing raw form inputs (headers, cookies, query params)
 * and building structured request objects for the API. Provides type-safe builders
 * for ScrapeRequest, CrawlRequest, and ResearchRequest, along with type guards for
 * result item discrimination and status CSS class mapping.
 *
 * @module form-utils
 */
import type {
  AuthOptions,
  ExtractOptions,
  PipelineOptions,
  ScrapeRequest,
  CrawlRequest,
  ResearchRequest,
  WebhookConfig,
  NetworkInterceptConfig,
} from "../api";
import type { ResultItem, CrawlResultItem, ResearchResultItem } from "../types";

export function parseHeaders(raw: string): Record<string, string> | undefined {
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

export function parsePatternList(raw: string): string[] | undefined {
  if (!raw.trim()) {
    return undefined;
  }
  const patterns = raw
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
  return patterns.length ? patterns : undefined;
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
): AuthOptions | undefined {
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

export function buildPipelineOptions(
  preProcessorsRaw: string,
  postProcessorsRaw: string,
  transformersRaw: string,
): PipelineOptions | undefined {
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

export function parseUrlList(raw: string): string[] {
  return raw
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function buildWebhookConfig(
  url: string,
  events: string[],
  secret: string,
): WebhookConfig | undefined {
  if (!url.trim()) {
    return undefined;
  }
  const validEvents = events.length > 0 ? events : ["completed"];
  return {
    url: url.trim(),
    events: validEvents as WebhookConfig["events"],
    secret: secret || undefined,
  };
}

export function buildNetworkInterceptConfig(
  enabled: boolean,
  urlPatternsRaw: string,
  resourceTypes: string[],
  captureRequestBody: boolean,
  captureResponseBody: boolean,
  maxBodySize: number,
  maxEntries: number,
): NetworkInterceptConfig | undefined {
  if (!enabled) {
    return undefined;
  }
  return {
    enabled: true,
    urlPatterns: parsePatternList(urlPatternsRaw),
    resourceTypes:
      resourceTypes.length > 0
        ? (resourceTypes as NetworkInterceptConfig["resourceTypes"])
        : undefined,
    captureRequestBody,
    captureResponseBody,
    maxBodySize,
    maxEntries,
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
  webhook?: WebhookConfig,
  device?: import("../api").DeviceEmulation,
  networkIntercept?: NetworkInterceptConfig,
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
    webhook,
    device,
    networkIntercept,
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
  sitemapURL?: string,
  sitemapOnly?: boolean,
  webhook?: WebhookConfig,
  includePatterns?: string[],
  excludePatterns?: string[],
  device?: import("../api").DeviceEmulation,
  networkIntercept?: NetworkInterceptConfig,
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
    sitemapURL: sitemapURL || undefined,
    sitemapOnly: sitemapOnly || undefined,
    webhook,
    includePatterns,
    excludePatterns,
    device,
    networkIntercept,
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
  webhook?: WebhookConfig,
  device?: import("../api").DeviceEmulation,
  networkIntercept?: NetworkInterceptConfig,
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
    webhook,
    device,
    networkIntercept,
  };
}

export function isCrawlResultItem(item: ResultItem): item is CrawlResultItem {
  return "url" in item && "status" in item;
}

export function isResearchResultItem(
  item: ResultItem,
): item is ResearchResultItem {
  const isNotCrawl = !("url" in item && "status" in item);
  const hasResearchField =
    "summary" in item ||
    "confidence" in item ||
    "evidence" in item ||
    "clusters" in item ||
    "citations" in item;
  return isNotCrawl && hasResearchField;
}

export function statusClass(status: string): string {
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
