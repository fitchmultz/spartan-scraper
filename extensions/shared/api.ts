/**
 * Spartan Scraper Extension - API Client
 *
 * Client for the Spartan Scraper API.
 * All API calls go through the background script to avoid CORS issues.
 */

import type {
  ErrorResponse,
  Job,
  ScrapeRequest,
  TemplatesResponse,
} from "./types.js";

const API_VERSION = "v1";

/** API error class */
export class APIError extends Error {
  constructor(
    message: string,
    public statusCode?: number,
    public code?: string,
  ) {
    super(message);
    this.name = "APIError";
  }
}

/**
 * Make an API request to the Spartan server
 */
async function apiRequest<T>(
  baseUrl: string,
  apiKey: string,
  method: "GET" | "POST",
  endpoint: string,
  body?: unknown,
): Promise<T> {
  const url = `${baseUrl}/${API_VERSION}${endpoint}`;

  const headers: Record<string, string> = {
    Accept: "application/json",
    "X-API-Key": apiKey,
  };

  if (body) {
    headers["Content-Type"] = "application/json";
  }

  const response = await fetch(url, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
    let errorCode: string | undefined;

    try {
      const errorData = (await response.json()) as ErrorResponse;
      if (errorData.error) {
        errorMessage = errorData.error;
      }
      errorCode = errorData.code;
    } catch {
      // Use default error message if parsing fails
    }

    throw new APIError(errorMessage, response.status, errorCode);
  }

  return response.json() as Promise<T>;
}

/**
 * Get list of available extraction templates
 */
export async function getTemplates(
  baseUrl: string,
  apiKey: string,
): Promise<string[]> {
  const response = await apiRequest<TemplatesResponse>(
    baseUrl,
    apiKey,
    "GET",
    "/templates",
  );
  return response.templates;
}

/**
 * Create a new scrape job
 */
export async function createScrapeJob(
  baseUrl: string,
  apiKey: string,
  request: ScrapeRequest,
): Promise<Job> {
  return apiRequest<Job>(baseUrl, apiKey, "POST", "/scrape", request);
}

/**
 * Get job status by ID
 */
export async function getJobStatus(
  baseUrl: string,
  apiKey: string,
  jobId: string,
): Promise<Job> {
  return apiRequest<Job>(baseUrl, apiKey, "GET", `/jobs/${jobId}`);
}

/**
 * Test API connectivity with the provided settings
 */
export async function testConnection(
  baseUrl: string,
  apiKey: string,
): Promise<{ success: boolean; message: string }> {
  try {
    // Try to fetch templates as a lightweight test
    await getTemplates(baseUrl, apiKey);
    return { success: true, message: "Connection successful" };
  } catch (err) {
    if (err instanceof APIError) {
      if (err.statusCode === 401) {
        return { success: false, message: "Invalid API key" };
      }
      return { success: false, message: err.message };
    }
    if (err instanceof TypeError && err.message.includes("fetch")) {
      return {
        success: false,
        message: "Cannot connect to server. Is Spartan running?",
      };
    }
    return {
      success: false,
      message: err instanceof Error ? err.message : "Unknown error",
    };
  }
}
