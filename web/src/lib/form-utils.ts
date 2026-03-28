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
  ResearchAgenticConfig,
  ResearchRequest,
  ScrapeRequest,
  ScreenshotConfig,
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
  proxyUrl?: string,
  proxyUsername?: string,
  proxyPassword?: string,
  proxyRegion?: string,
  proxyRequiredTags?: string[],
  proxyExcludeProxyIds?: string[],
): AuthOptions | undefined {
  const trimmedProxyUrl = proxyUrl?.trim();
  const trimmedProxyUsername = proxyUsername?.trim();
  const trimmedProxyPassword = proxyPassword?.trim();
  const normalizedProxyRequiredTags =
    proxyRequiredTags && proxyRequiredTags.length > 0
      ? proxyRequiredTags
      : undefined;
  const normalizedProxyExcludeProxyIds =
    proxyExcludeProxyIds && proxyExcludeProxyIds.length > 0
      ? proxyExcludeProxyIds
      : undefined;
  const normalizedProxyRegion = proxyRegion?.trim() || undefined;
  const hasProxyHints =
    !!normalizedProxyRegion ||
    !!normalizedProxyRequiredTags ||
    !!normalizedProxyExcludeProxyIds;

  if (
    (trimmedProxyUrl || trimmedProxyUsername || trimmedProxyPassword) &&
    hasProxyHints
  ) {
    throw new Error(
      "Direct proxy settings and proxy-pool selection hints are mutually exclusive.",
    );
  }
  if (!trimmedProxyUrl && (trimmedProxyUsername || trimmedProxyPassword)) {
    throw new Error("Proxy username/password require a proxy URL.");
  }

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
    !loginPass &&
    !trimmedProxyUrl &&
    !hasProxyHints
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
    proxy: trimmedProxyUrl
      ? {
          url: trimmedProxyUrl,
          username: trimmedProxyUsername || undefined,
          password: trimmedProxyPassword || undefined,
        }
      : undefined,
    proxyHints: hasProxyHints
      ? {
          preferred_region: normalizedProxyRegion,
          required_tags: normalizedProxyRequiredTags,
          exclude_proxy_ids: normalizedProxyExcludeProxyIds,
        }
      : undefined,
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

export function mergeAIExtractOptions(
  extract: ExtractOptions | undefined,
  aiExtract?: AiExtractOptions,
): ExtractOptions | undefined {
  return extract || aiExtract
    ? {
        ...extract,
        ai: aiExtract,
      }
    : undefined;
}

export function buildBrowserRuntimeFields(options: {
  headless: boolean;
  playwright: boolean;
  visual?: boolean;
}) {
  return {
    headless: options.headless,
    ...(options.headless ? { playwright: options.playwright } : {}),
    ...(typeof options.visual === "boolean" ? { visual: options.visual } : {}),
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

export function buildScreenshotConfig(
  enabled: boolean,
  fullPage: boolean,
  format: "png" | "jpeg",
  quality: number,
  width: number,
  height: number,
): ScreenshotConfig | undefined {
  if (!enabled) {
    return undefined;
  }
  return {
    enabled: true,
    fullPage,
    format,
    quality: format === "jpeg" ? quality : undefined,
    width: width > 0 ? width : undefined,
    height: height > 0 ? height : undefined,
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

export function buildResearchAgenticOptions(
  enabled: boolean,
  instructions: string,
  maxRounds: number,
  maxFollowUpUrls: number,
): ResearchAgenticConfig | undefined {
  if (!enabled) {
    return undefined;
  }

  return {
    enabled: true,
    instructions: instructions.trim() || undefined,
    maxRounds,
    maxFollowUpUrls,
  };
}

type SharedFormFields = Pick<
  FormController,
  | "headless"
  | "authProfile"
  | "authBasic"
  | "headersRaw"
  | "cookiesRaw"
  | "queryRaw"
  | "proxyUrl"
  | "proxyUsername"
  | "proxyPassword"
  | "proxyRegion"
  | "proxyRequiredTags"
  | "proxyExcludeProxyIds"
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
  | "screenshotEnabled"
  | "screenshotFullPage"
  | "screenshotFormat"
  | "screenshotQuality"
  | "screenshotWidth"
  | "screenshotHeight"
  | "interceptEnabled"
  | "interceptURLPatterns"
  | "interceptResourceTypes"
  | "interceptCaptureRequestBody"
  | "interceptCaptureResponseBody"
  | "interceptMaxBodySize"
  | "interceptMaxEntries"
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
  screenshot?: ScreenshotConfig;
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
      form.proxyUrl,
      form.proxyUsername,
      form.proxyPassword,
      form.proxyRegion,
      parseOptionalList(form.proxyRequiredTags, ","),
      parseOptionalList(form.proxyExcludeProxyIds, ","),
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
    screenshot: buildScreenshotConfig(
      form.headless && form.screenshotEnabled,
      form.screenshotFullPage,
      form.screenshotFormat,
      form.screenshotQuality,
      form.screenshotWidth,
      form.screenshotHeight,
    ),
    networkIntercept: buildNetworkInterceptConfig(
      form.headless && form.interceptEnabled,
      form.interceptURLPatterns,
      form.interceptResourceTypes,
      form.interceptCaptureRequestBody,
      form.interceptCaptureResponseBody,
      form.interceptMaxBodySize,
      form.interceptMaxEntries || interceptMaxEntries,
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
  screenshot?: ScreenshotConfig,
  device?: import("../api").DeviceEmulation,
  networkIntercept?: NetworkInterceptConfig,
  aiExtract?: AiExtractOptions,
): ScrapeRequest {
  return {
    url,
    ...buildBrowserRuntimeFields({
      headless,
      playwright: usePlaywright,
    }),
    timeoutSeconds,
    authProfile: authProfile || undefined,
    auth,
    extract: mergeAIExtractOptions(extract, aiExtract),
    pipeline: buildPipelineOptions(preProcessors, postProcessors, transformers),
    incremental: incremental || undefined,
    webhook,
    screenshot,
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
  screenshot?: ScreenshotConfig,
  device?: import("../api").DeviceEmulation,
  networkIntercept?: NetworkInterceptConfig,
  aiExtract?: AiExtractOptions,
): CrawlRequest {
  return {
    url,
    maxDepth,
    maxPages,
    ...buildBrowserRuntimeFields({
      headless,
      playwright: usePlaywright,
    }),
    timeoutSeconds,
    authProfile: authProfile || undefined,
    auth,
    extract: mergeAIExtractOptions(extract, aiExtract),
    pipeline: buildPipelineOptions(preProcessors, postProcessors, transformers),
    incremental: incremental || undefined,
    sitemapURL: sitemapURL || undefined,
    sitemapOnly: sitemapOnly || undefined,
    webhook,
    includePatterns,
    excludePatterns,
    screenshot,
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
  screenshot?: ScreenshotConfig,
  device?: import("../api").DeviceEmulation,
  networkIntercept?: NetworkInterceptConfig,
  aiExtract?: AiExtractOptions,
  agentic?: ResearchAgenticConfig,
): ResearchRequest {
  return {
    query,
    urls,
    maxDepth,
    maxPages,
    ...buildBrowserRuntimeFields({
      headless,
      playwright: usePlaywright,
    }),
    timeoutSeconds,
    authProfile: authProfile || undefined,
    auth,
    extract: mergeAIExtractOptions(extract, aiExtract),
    pipeline: buildPipelineOptions(preProcessors, postProcessors, transformers),
    webhook,
    screenshot,
    device,
    networkIntercept,
    agentic,
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
