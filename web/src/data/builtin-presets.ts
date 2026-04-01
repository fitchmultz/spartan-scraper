/**
 * Purpose: Provide the builtin presets module for this repository.
 * Responsibilities: Define the file-local logic, exports, and helpers that belong to this module.
 * Scope: This file only; broader orchestration stays in adjacent modules.
 * Usage: Import from the owning package or feature surface.
 * Invariants/Assumptions: The file should stay aligned with surrounding source-of-truth contracts and avoid hidden side effects.
 */

import type { JobPreset } from "../types/presets";

/**
 * Built-in preset for static HTML pages using HTTP fetch.
 * Fast and lightweight for simple sites without JavaScript.
 */
const staticPagePreset: JobPreset = {
  id: "static-page",
  name: "Static Page",
  description: "Fast HTTP fetch for static HTML content",
  icon: "📄",
  jobType: "scrape",
  config: {
    headless: false,
    timeoutSeconds: 30,
  },
  resources: {
    timeSeconds: 10,
    cpu: "low",
    memory: "low",
  },
  useCases: ["Static HTML sites", "Documentation", "Simple blogs"],
  isBuiltIn: true,
};

/**
 * Built-in preset for React and modern SPA applications using Playwright.
 * Handles JavaScript-heavy sites with full browser automation.
 */
const spaReactPreset: JobPreset = {
  id: "spa-react",
  name: "SPA / React App",
  description: "Full browser automation for React/Vue/Angular apps",
  icon: "⚛️",
  jobType: "scrape",
  config: {
    headless: true,
    usePlaywright: true,
    timeoutSeconds: 60,
  },
  resources: {
    timeSeconds: 45,
    cpu: "high",
    memory: "high",
  },
  useCases: ["React apps", "Vue apps", "Angular apps", "Dynamic content"],
  isBuiltIn: true,
};

/**
 * Built-in preset for JavaScript-heavy sites using Chromedp.
 * Alternative to Playwright for sites that work well with Chromium.
 */
const spaChromedpPreset: JobPreset = {
  id: "spa-chromedp",
  name: "SPA / Chromedp",
  description: "Chromium-based headless for JavaScript sites",
  icon: "🌐",
  jobType: "scrape",
  config: {
    headless: true,
    usePlaywright: false,
    timeoutSeconds: 45,
  },
  resources: {
    timeSeconds: 30,
    cpu: "medium",
    memory: "medium",
  },
  useCases: ["JavaScript-heavy sites", "Single-page apps", "AJAX content"],
  isBuiltIn: true,
};

/**
 * Built-in preset for blog articles with content extraction.
 * Optimized for article content with extraction template.
 */
const blogArticlePreset: JobPreset = {
  id: "blog-article",
  name: "Blog Article",
  description: "Article extraction with content-focused template",
  icon: "📝",
  jobType: "scrape",
  config: {
    headless: true,
    extractTemplate: "article",
    timeoutSeconds: 45,
  },
  resources: {
    timeSeconds: 20,
    cpu: "medium",
    memory: "low",
  },
  useCases: ["Medium posts", "Blog articles", "News articles"],
  isBuiltIn: true,
};

/**
 * Built-in preset for e-commerce product pages.
 * Uses Playwright for dynamic product data and extraction template.
 */
const productPagePreset: JobPreset = {
  id: "product-page",
  name: "Product Page",
  description: "E-commerce scraping with product extraction",
  icon: "🛍️",
  jobType: "scrape",
  config: {
    headless: true,
    usePlaywright: true,
    extractTemplate: "product",
    timeoutSeconds: 60,
  },
  resources: {
    timeSeconds: 45,
    cpu: "high",
    memory: "medium",
  },
  useCases: ["E-commerce sites", "Product catalogs", "Marketplaces"],
  isBuiltIn: true,
};

/**
 * Built-in preset for API documentation crawling.
 * Crawls documentation sites with incremental support.
 */
const docsCrawlPreset: JobPreset = {
  id: "docs-crawl",
  name: "Documentation Crawl",
  description: "Crawl API docs and technical documentation",
  icon: "📚",
  jobType: "crawl",
  config: {
    headless: false,
    maxDepth: 3,
    maxPages: 500,
    extractTemplate: "article",
    incremental: true,
  },
  resources: {
    timeSeconds: 300,
    cpu: "medium",
    memory: "medium",
  },
  useCases: ["API docs", "Technical documentation", "Wiki sites"],
  isBuiltIn: true,
};

/**
 * Built-in preset for site audits.
 * Crawls with headless browser for SEO and broken link checking.
 */
const siteAuditPreset: JobPreset = {
  id: "site-audit",
  name: "Site Audit",
  description: "SEO audit and broken link detection crawl",
  icon: "🔍",
  jobType: "crawl",
  config: {
    headless: true,
    maxDepth: 2,
    maxPages: 100,
    timeoutSeconds: 45,
  },
  resources: {
    timeSeconds: 180,
    cpu: "high",
    memory: "high",
  },
  useCases: ["SEO audits", "Broken link checking", "Site analysis"],
  isBuiltIn: true,
};

/**
 * Built-in preset for research queries.
 * Multi-source research with balanced settings.
 */
const researchQueryPreset: JobPreset = {
  id: "research-query",
  name: "Research Query",
  description: "Multi-source research with content aggregation",
  icon: "🔬",
  jobType: "research",
  config: {
    headless: true,
    maxDepth: 1,
    maxPages: 50,
    timeoutSeconds: 60,
  },
  resources: {
    timeSeconds: 120,
    cpu: "high",
    memory: "medium",
  },
  useCases: [
    "Multi-source research",
    "Content aggregation",
    "Comparison studies",
  ],
  isBuiltIn: true,
};

/**
 * Built-in preset for API endpoints.
 * Fast JSON API scraping with appropriate headers.
 */
const apiEndpointPreset: JobPreset = {
  id: "api-endpoint",
  name: "API Endpoint",
  description: "Fast scraping for REST APIs and JSON endpoints",
  icon: "🔌",
  jobType: "scrape",
  config: {
    headless: false,
    timeoutSeconds: 30,
    headersRaw: "Accept: application/json",
  },
  resources: {
    timeSeconds: 5,
    cpu: "low",
    memory: "low",
  },
  useCases: ["REST APIs", "JSON endpoints", "Data feeds"],
  isBuiltIn: true,
};

/**
 * Built-in preset for GitHub repositories.
 * Optimized for GitHub repo documentation and README files.
 */
const githubRepoPreset: JobPreset = {
  id: "github-repo",
  name: "GitHub Repository",
  description: "Crawl GitHub repos for documentation and READMEs",
  icon: "🐙",
  jobType: "crawl",
  config: {
    headless: false,
    maxDepth: 2,
    maxPages: 50,
  },
  urlPatterns: ["github\\.com/[^/]+/[^/]+"],
  resources: {
    timeSeconds: 60,
    cpu: "low",
    memory: "low",
  },
  useCases: ["Repository docs", "README files", "Wiki pages"],
  isBuiltIn: true,
};

/**
 * All built-in presets available in the application.
 */
export const BUILTIN_PRESETS: JobPreset[] = [
  staticPagePreset,
  spaReactPreset,
  spaChromedpPreset,
  blogArticlePreset,
  productPagePreset,
  docsCrawlPreset,
  siteAuditPreset,
  researchQueryPreset,
  apiEndpointPreset,
  githubRepoPreset,
];

/**
 * Get a built-in preset by its ID.
 *
 * @param id - The preset ID to look up
 * @returns The preset if found, undefined otherwise
 */
export function getBuiltinPresetById(id: string): JobPreset | undefined {
  return BUILTIN_PRESETS.find((preset) => preset.id === id);
}

/**
 * Get all presets for a specific job type.
 *
 * @param jobType - The job type to filter by
 * @returns Array of presets for the given job type
 */
export function getPresetsByJobType(
  jobType: JobPreset["jobType"],
): JobPreset[] {
  return BUILTIN_PRESETS.filter((preset) => preset.jobType === jobType);
}
