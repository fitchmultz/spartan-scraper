/**
 * Spartan Scraper Extension - Shared Types
 *
 * TypeScript type definitions for the browser extension.
 * Mirrors the Spartan API types from openapi.yaml.
 */

/** Extension settings stored in chrome.storage */
export interface ExtensionSettings {
  /** Base URL for the Spartan API server */
  apiUrl: string;
  /** API key for authentication */
  apiKey: string;
  /** Default template to use for quick scrapes */
  defaultTemplate: string;
  /** Whether to use headless mode by default */
  defaultHeadless: boolean;
}

/** Default settings values */
export const DEFAULT_SETTINGS: ExtensionSettings = {
  apiUrl: "http://localhost:8741",
  apiKey: "",
  defaultTemplate: "article",
  defaultHeadless: true,
};

/** Job status from Spartan API */
export type JobStatus = "queued" | "running" | "succeeded" | "failed" | "canceled";

/** Job kind from Spartan API */
export type JobKind = "scrape" | "crawl" | "research";

/** Job response from Spartan API */
export interface Job {
  id: string;
  kind: JobKind;
  status: JobStatus;
  createdAt: string;
  updatedAt: string;
  params?: Record<string, unknown>;
  error?: string;
  screenshotPath?: string;
}

/** Scrape request payload */
export interface ScrapeRequest {
  url: string;
  method?: "GET" | "POST" | "PUT" | "DELETE" | "PATCH" | "HEAD" | "OPTIONS";
  body?: string;
  contentType?: string;
  headless?: boolean;
  playwright?: boolean;
  authProfile?: string;
  auth?: AuthOptions;
  extract?: ExtractOptions;
  pipeline?: PipelineOptions;
  timeoutSeconds?: number;
  incremental?: boolean;
}

/** Authentication options */
export interface AuthOptions {
  headers?: Record<string, string>;
  cookies?: Record<string, string>;
  bearerToken?: string;
  username?: string;
  password?: string;
}

/** Extraction options */
export interface ExtractOptions {
  template?: string;
  validate?: boolean;
  ai?: AIExtractOptions;
}

/** AI extraction options */
export interface AIExtractOptions {
  enabled: boolean;
  mode?: "natural_language" | "schema_guided";
  prompt?: string;
  schema?: Record<string, unknown>;
  fields?: string[];
}

/** Pipeline options */
export interface PipelineOptions {
  transformers?: string[];
  filters?: string[];
  hooks?: PipelineHooks;
}

/** Pipeline hooks configuration */
export interface PipelineHooks {
  preFetch?: string;
  postFetch?: string;
  preExtract?: string;
  postExtract?: string;
  preOutput?: string;
}

/** Templates list response */
export interface TemplatesResponse {
  templates: string[];
}

/** Error response from API */
export interface ErrorResponse {
  error: string;
  code?: string;
}

/** Message types for communication between extension components */
export interface Message {
  type: MessageType;
  payload?: unknown;
}

export type MessageType =
  | "GET_CURRENT_TAB"
  | "GET_CURRENT_TAB_RESPONSE"
  | "CREATE_SCRAPE_JOB"
  | "CREATE_SCRAPE_JOB_RESPONSE"
  | "GET_JOB_STATUS"
  | "GET_JOB_STATUS_RESPONSE"
  | "GET_TEMPLATES"
  | "GET_TEMPLATES_RESPONSE"
  | "OPEN_OPTIONS_PAGE"
  | "SHOW_NOTIFICATION"
  | "CONTEXT_MENU_SCRAPE";

/** Current tab info */
export interface TabInfo {
  url: string;
  title: string;
  favicon?: string;
}

/** Scrape form state */
export interface ScrapeFormState {
  url: string;
  template: string;
  headless: boolean;
  isSubmitting: boolean;
  jobId?: string;
  error?: string;
}
