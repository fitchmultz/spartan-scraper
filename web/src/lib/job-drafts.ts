/**
 * Purpose: Centralize per-job draft defaults, local field extraction, and preset-config assembly for the guided job wizard and expert forms.
 * Responsibilities: Define local draft field shapes, provide empty draft factories, translate live form state into preset configs, and hydrate local draft fields back from stored configs.
 * Scope: Web job creation draft/preset helpers only.
 * Usage: Import from job forms, the job submission container, and wizard hooks when reading or writing draft state.
 * Invariants/Assumptions: Preset configs remain the canonical serializable draft format, form-controller state owns shared runtime/extraction fields, and job-specific local fields are limited to URL/query/device/scope inputs that do not live in `useFormState`.
 */

import type { DeviceEmulation } from "../api";
import type { FormController } from "../hooks/useFormState";
import type { JobType, PresetConfig } from "../types/presets";

export interface ScrapeDraftFields {
  url: string;
  device: DeviceEmulation | null;
}

export interface CrawlDraftFields {
  url: string;
  sitemapURL: string;
  sitemapOnly: boolean;
  includePatterns: string;
  excludePatterns: string;
  device: DeviceEmulation | null;
}

export interface ResearchDraftFields {
  query: string;
  urls: string;
  device: DeviceEmulation | null;
}

export interface JobDraftLocalState {
  scrape: ScrapeDraftFields;
  crawl: CrawlDraftFields;
  research: ResearchDraftFields;
}

export function createInitialJobDraftLocalState(): JobDraftLocalState {
  return {
    scrape: {
      url: "",
      device: null,
    },
    crawl: {
      url: "",
      sitemapURL: "",
      sitemapOnly: false,
      includePatterns: "",
      excludePatterns: "",
      device: null,
    },
    research: {
      query: "",
      urls: "",
      device: null,
    },
  };
}

export function buildSharedPresetConfig(form: FormController): PresetConfig {
  return {
    headless: form.headless,
    usePlaywright: form.usePlaywright,
    timeoutSeconds: form.timeoutSeconds,
    authProfile: form.authProfile,
    authBasic: form.authBasic,
    headersRaw: form.headersRaw,
    cookiesRaw: form.cookiesRaw,
    queryRaw: form.queryRaw,
    proxyUrl: form.proxyUrl,
    proxyUsername: form.proxyUsername,
    proxyPassword: form.proxyPassword,
    proxyRegion: form.proxyRegion,
    proxyRequiredTags: form.proxyRequiredTags,
    proxyExcludeProxyIds: form.proxyExcludeProxyIds,
    loginUrl: form.loginUrl,
    loginUserSelector: form.loginUserSelector,
    loginPassSelector: form.loginPassSelector,
    loginSubmitSelector: form.loginSubmitSelector,
    loginUser: form.loginUser,
    loginPass: form.loginPass,
    extractTemplate: form.extractTemplate,
    extractValidate: form.extractValidate,
    aiExtractEnabled: form.aiExtractEnabled,
    aiExtractMode: form.aiExtractMode,
    aiExtractPrompt: form.aiExtractPrompt,
    aiExtractSchema: form.aiExtractSchema,
    aiExtractFields: form.aiExtractFields,
    agenticResearchEnabled: form.agenticResearchEnabled,
    agenticResearchInstructions: form.agenticResearchInstructions,
    agenticResearchMaxRounds: form.agenticResearchMaxRounds,
    agenticResearchMaxFollowUpUrls: form.agenticResearchMaxFollowUpUrls,
    preProcessors: form.preProcessors,
    postProcessors: form.postProcessors,
    transformers: form.transformers,
    incremental: form.incremental,
    maxDepth: form.maxDepth,
    maxPages: form.maxPages,
    webhookUrl: form.webhookUrl,
    webhookEvents: form.webhookEvents,
    webhookSecret: form.webhookSecret,
    screenshotEnabled: form.screenshotEnabled,
    screenshotFullPage: form.screenshotFullPage,
    screenshotFormat: form.screenshotFormat,
    screenshotQuality: form.screenshotQuality,
    screenshotWidth: form.screenshotWidth,
    screenshotHeight: form.screenshotHeight,
    interceptEnabled: form.interceptEnabled,
    interceptURLPatterns: form.interceptURLPatterns,
    interceptResourceTypes: form.interceptResourceTypes,
    interceptCaptureRequestBody: form.interceptCaptureRequestBody,
    interceptCaptureResponseBody: form.interceptCaptureResponseBody,
    interceptMaxBodySize: form.interceptMaxBodySize,
    interceptMaxEntries: form.interceptMaxEntries,
  };
}

export function buildPresetConfig(
  jobType: JobType,
  form: FormController,
  localState: JobDraftLocalState,
): PresetConfig {
  const shared = buildSharedPresetConfig(form);

  if (jobType === "scrape") {
    return {
      ...shared,
      url: localState.scrape.url,
      device: localState.scrape.device ?? undefined,
    };
  }

  if (jobType === "crawl") {
    return {
      ...shared,
      url: localState.crawl.url,
      sitemapURL: localState.crawl.sitemapURL,
      sitemapOnly: localState.crawl.sitemapOnly,
      includePatterns: localState.crawl.includePatterns,
      excludePatterns: localState.crawl.excludePatterns,
      device: localState.crawl.device ?? undefined,
    };
  }

  return {
    ...shared,
    query: localState.research.query,
    urls: localState.research.urls,
    device: localState.research.device ?? undefined,
  };
}

export function extractLocalDraftFields(
  jobType: JobType,
  config: PresetConfig | null | undefined,
) {
  const defaults = createInitialJobDraftLocalState();

  if (!config) {
    return defaults[jobType];
  }

  if (jobType === "scrape") {
    return {
      url: config.url ?? defaults.scrape.url,
      device: config.device ?? defaults.scrape.device,
    } satisfies ScrapeDraftFields;
  }

  if (jobType === "crawl") {
    return {
      url: config.url ?? defaults.crawl.url,
      sitemapURL: config.sitemapURL ?? defaults.crawl.sitemapURL,
      sitemapOnly: config.sitemapOnly ?? defaults.crawl.sitemapOnly,
      includePatterns: config.includePatterns ?? defaults.crawl.includePatterns,
      excludePatterns: config.excludePatterns ?? defaults.crawl.excludePatterns,
      device: config.device ?? defaults.crawl.device,
    } satisfies CrawlDraftFields;
  }

  return {
    query: config.query ?? defaults.research.query,
    urls: config.urls ?? defaults.research.urls,
    device: config.device ?? defaults.research.device,
  } satisfies ResearchDraftFields;
}
