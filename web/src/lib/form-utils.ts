/**
 * Form Utilities Module
 *
 * Shared utility functions for parsing raw form inputs (headers, cookies, query params)
 * and building structured request objects for the API. Provides type-safe builders
 * for ScrapeRequest, CrawlRequest, and ResearchRequest, along with type guards for
 * result item discrimination.
 *
 * @module form-utils
 */
import type {
  AiExtractOptions,
  AuthOptions,
  CrawlRequest,
  ExtractOptions,
  NetworkInterceptConfig,
  PipelineOptions,
  ResearchRequest,
  ScrapeRequest,
  WebhookConfig,
} from "../api";
import type { FormController } from "../hooks/useFormState";
import type { CrawlResultItem, ResearchResultItem, ResultItem } from "../types";
import {
  parseLineSeparatedMap,
  parseOptionalList,
  splitAndTrim,
} from "./input-parsing";

export function parseHeaders(raw: string): Record<string, string> | undefined {
  return parseLineSeparatedMap(raw, ":");
}

export function parseCookies(raw: string): string[] | undefined {
  return parseOptionalList(raw, "\n");
}

export function parseQueryParams(
  raw: string,
): Record<string, string> | undefined {
  return parseLineSeparatedMap(raw, "=");
}

export function parseProcessors(raw: string): string[] | undefined {
  return parseOptionalList(raw, ",");
}

export function parsePatternList(raw: string): string[] | undefined {
  return parseOptionalList(raw, ",");
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
  return splitAndTrim(raw, ",");
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

export function parseAIExtractSchemaText(
  raw: string,
): Record<string, unknown> | undefined {
  if (!raw.trim()) {
    return undefined;
  }

  const parsed = JSON.parse(raw) as unknown;
  if (!parsed || Array.isArray(parsed) || typeof parsed !== "object") {
    throw new Error("AI schema must be a JSON object");
  }

  return parsed as Record<string, unknown>;
}

export function buildAIExtractOptions(
  enabled: boolean,
  mode: "natural_language" | "schema_guided",
  prompt: string,
  schemaText: string,
  fields: string,
): AiExtractOptions | undefined {
  if (!enabled) {
    return undefined;
  }

  const parsedFields = splitAndTrim(fields, ",");
  const trimmedPrompt = prompt.trim();
  const parsedSchema =
    mode === "schema_guided" ? parseAIExtractSchemaText(schemaText) : undefined;

  return {
    enabled: true,
    mode,
    prompt:
      mode === "natural_language" && trimmedPrompt ? trimmedPrompt : undefined,
    schema: parsedSchema,
    fields: parsedFields.length > 0 ? parsedFields : undefined,
  };
}

type SharedFormFields = Pick<
  FormController,
  | "authProfile"
  | "authBasic"
  | "headersRaw"
  | "cookiesRaw"
  | "queryRaw"
  | "loginUrl"
  | "loginUserSelector"
  | "loginPassSelector"
  | "loginSubmitSelector"
  | "loginUser"
  | "loginPass"
  | "extractTemplate"
  | "extractValidate"
  | "preProcessors"
  | "postProcessors"
  | "transformers"
  | "webhookUrl"
  | "webhookEvents"
  | "webhookSecret"
  | "interceptEnabled"
  | "interceptURLPatterns"
  | "interceptResourceTypes"
  | "interceptCaptureRequestBody"
  | "interceptCaptureResponseBody"
  | "interceptMaxBodySize"
>;

export type SharedRequestConfig = {
  authProfile?: string;
  auth?: AuthOptions;
  extract: ExtractOptions;
  pipeline?: PipelineOptions;
  preProcessors: string;
  postProcessors: string;
  transformers: string;
  webhook?: WebhookConfig;
  networkIntercept?: NetworkInterceptConfig;
};

export function buildSharedRequestConfig(
  form: SharedFormFields,
  interceptMaxEntries = 1000,
): SharedRequestConfig {
  return {
    authProfile: form.authProfile || undefined,
    auth: buildAuth(
      form.authBasic,
      parseHeaders(form.headersRaw),
      parseCookies(form.cookiesRaw),
      parseQueryParams(form.queryRaw),
      form.loginUrl,
      form.loginUserSelector,
      form.loginPassSelector,
      form.loginSubmitSelector,
      form.loginUser,
      form.loginPass,
    ),
    extract: {
      template: form.extractTemplate || undefined,
      validate: form.extractValidate,
    },
    pipeline: buildPipelineOptions(
      form.preProcessors,
      form.postProcessors,
      form.transformers,
    ),
    preProcessors: form.preProcessors,
    postProcessors: form.postProcessors,
    transformers: form.transformers,
    webhook: buildWebhookConfig(
      form.webhookUrl,
      form.webhookEvents,
      form.webhookSecret,
    ),
    networkIntercept: buildNetworkInterceptConfig(
      form.interceptEnabled,
      form.interceptURLPatterns,
      form.interceptResourceTypes,
      form.interceptCaptureRequestBody,
      form.interceptCaptureResponseBody,
      form.interceptMaxBodySize,
      interceptMaxEntries,
    ),
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
  aiExtract?: AiExtractOptions,
): ScrapeRequest {
  // Merge AI options into extract options if provided
  const mergedExtract: ExtractOptions | undefined =
    extract || aiExtract
      ? {
          ...extract,
          ai: aiExtract,
        }
      : undefined;

  return {
    url,
    headless,
    playwright: headless ? usePlaywright : false,
    timeoutSeconds,
    authProfile: authProfile || undefined,
    auth,
    extract: mergedExtract,
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
  aiExtract?: AiExtractOptions,
): CrawlRequest {
  const mergedExtract: ExtractOptions | undefined =
    extract || aiExtract
      ? {
          ...extract,
          ai: aiExtract,
        }
      : undefined;

  return {
    url,
    maxDepth,
    maxPages,
    headless,
    playwright: headless ? usePlaywright : false,
    timeoutSeconds,
    authProfile: authProfile || undefined,
    auth,
    extract: mergedExtract,
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
  aiExtract?: AiExtractOptions,
): ResearchRequest {
  const mergedExtract: ExtractOptions | undefined =
    extract || aiExtract
      ? {
          ...extract,
          ai: aiExtract,
        }
      : undefined;

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
    extract: mergedExtract,
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
