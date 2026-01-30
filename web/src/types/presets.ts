/**
 * Job Preset Types
 *
 * Type definitions for job templates and quick-start presets used throughout
 * the web UI. Provides TypeScript interfaces for preset configuration,
 * resource estimation, and URL pattern matching.
 *
 * @module types/presets
 */

/**
 * Job type discriminator for presets
 */
export type JobType = "scrape" | "crawl" | "research";

/**
 * Resource estimation for a preset
 */
export interface PresetResources {
  /** Estimated time in seconds */
  timeSeconds: number;
  /** CPU intensity: 'low' | 'medium' | 'high' */
  cpu: "low" | "medium" | "high";
  /** Memory intensity: 'low' | 'medium' | 'high' */
  memory: "low" | "medium" | "high";
}

/**
 * Configuration values that can be preset for any job type
 */
export interface PresetConfig {
  // Common options
  headless?: boolean;
  usePlaywright?: boolean;
  timeoutSeconds?: number;
  authProfile?: string;
  authBasic?: string;
  headersRaw?: string;
  cookiesRaw?: string;
  queryRaw?: string;
  loginUrl?: string;
  loginUserSelector?: string;
  loginPassSelector?: string;
  loginSubmitSelector?: string;
  loginUser?: string;
  loginPass?: string;
  extractTemplate?: string;
  extractValidate?: boolean;
  preProcessors?: string;
  postProcessors?: string;
  transformers?: string;
  incremental?: boolean;
  maxDepth?: number;
  maxPages?: number;
  // Job-type specific inputs
  url?: string;
  query?: string;
  urls?: string;
}

/**
 * A job preset definition
 */
export interface JobPreset {
  id: string;
  name: string;
  description: string;
  icon: string; // Emoji character
  jobType: JobType;
  config: PresetConfig;
  resources: PresetResources;
  /** URL patterns for auto-suggestion (regex strings) */
  urlPatterns?: string[];
  /** Use cases/examples */
  useCases: string[];
  /** Whether this is a built-in preset (vs user-saved) */
  isBuiltIn: boolean;
  /** Creation timestamp for custom presets */
  createdAt?: number;
}
