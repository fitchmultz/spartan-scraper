/**
 * Results loading utilities for job results API.
 *
 * This module provides functions for fetching and parsing job results from the API.
 * It handles HTTP requests, error parsing, JSONL parsing, and different result formats.
 *
 * @module results
 */

import { buildApiUrl } from "./api-config";

/**
 * Response shape for results loading operations.
 */
export interface ResultsResponse {
  error?: string;
  data?: unknown[];
  raw?: string;
  isBinary?: boolean;
  totalCount?: number;
}

/**
 * Result of JSONL parsing operations.
 */
export interface JsonlParseResult {
  data?: unknown[];
  raw: string;
  error?: string;
}

/**
 * Build the results URL for a given job ID and format.
 *
 * Includes pagination parameters for jsonl format (limit, offset) but not for other formats.
 * Supports optional transformation parameters for applying JMESPath/JSONata expressions.
 *
 * @param jobId - The job ID to fetch results for
 * @param format - The result format (e.g., "jsonl", "json", "csv", "md")
 * @param limit - Pagination limit for jsonl format
 * @param offset - Pagination offset for jsonl format
 * @param transformExpression - Optional JMESPath/JSONata expression to transform results
 * @param transformLanguage - Optional transformation language ("jmespath" or "jsonata")
 * @returns The complete URL for fetching results
 */
export function buildResultsUrl(
  jobId: string,
  format: string,
  limit?: number,
  offset?: number,
  transformExpression?: string,
  transformLanguage?: "jmespath" | "jsonata",
): string {
  let path = `/v1/jobs/${jobId}/results?format=${format}`;

  // Only add pagination for jsonl format
  if (format === "jsonl" && limit !== undefined && offset !== undefined) {
    path += `&limit=${limit}&offset=${offset}`;
  }

  // Add transform parameters if provided
  if (transformExpression) {
    path += `&transform_expression=${encodeURIComponent(transformExpression)}`;
    if (transformLanguage) {
      path += `&transform_language=${transformLanguage}`;
    }
  }

  return buildApiUrl(path);
}

/**
 * Parse error response from API.
 *
 * Tries to parse response body as JSON with an "error" field.
 * Falls back to default error message with status code and text.
 *
 * @param response - The HTTP response object
 * @param status - HTTP status code
 * @param statusText - HTTP status text
 * @returns The parsed error message
 */
export async function parseErrorResponse(
  response: Response,
  status: number,
  statusText: string,
): Promise<string> {
  try {
    const errorData = (await response.json()) as { error?: string };
    if (errorData.error) {
      return errorData.error;
    }
  } catch {
    // If parsing error body fails, use default message
  }
  return `Failed to load results (${status} ${statusText})`;
}

/**
 * Parse JSONL formatted results.
 *
 * Splits text by newlines, filters empty lines, and parses each line as JSON.
 * Skips malformed JSON lines but preserves raw text.
 *
 * @param text - The raw JSONL text
 * @returns The parsed results or error if all lines are malformed
 */
export function parseJsonlResults(text: string): JsonlParseResult {
  const lines = text.split("\n").filter((line) => line.trim());

  const parsedItems: unknown[] = [];
  for (const line of lines) {
    try {
      const parsed = JSON.parse(line);
      parsedItems.push(parsed);
    } catch {
      // Skip malformed JSON lines
    }
  }

  // Check if we had input but failed to parse anything
  if (parsedItems.length === 0 && lines.length > 0) {
    return {
      error: "No valid results found. Results file may be corrupted.",
      raw: text,
    };
  }

  return { data: parsedItems, raw: text };
}

/**
 * Load results for a job from the API.
 *
 * Fetches results in the specified format, handles HTTP errors, and parses JSONL responses.
 * For jsonl format, returns parsed data; for other formats, returns raw text.
 * Supports optional transformation parameters for applying JMESPath/JSONata expressions.
 *
 * @param jobId - The job ID to fetch results for
 * @param format - The result format (defaults to "jsonl")
 * @param page - The page number for jsonl pagination (1-based)
 * @param resultsPerPage - Number of results per page for jsonl format
 * @param transformExpression - Optional JMESPath/JSONata expression to transform results
 * @param transformLanguage - Optional transformation language ("jmespath" or "jsonata")
 * @returns The results response with data, raw text, or error
 */
export async function loadResults(
  jobId: string,
  format: string = "jsonl",
  page: number = 1,
  resultsPerPage: number = 100,
  transformExpression?: string,
  transformLanguage?: "jmespath" | "jsonata",
): Promise<ResultsResponse> {
  try {
    const offset = (page - 1) * resultsPerPage;
    const resultsUrl = buildResultsUrl(
      jobId,
      format,
      resultsPerPage,
      offset,
      transformExpression,
      transformLanguage,
    );
    const response = await fetch(resultsUrl);

    if (!response.ok) {
      const errorMessage = await parseErrorResponse(
        response,
        response.status,
        response.statusText,
      );
      return { error: errorMessage };
    }

    if (format === "jsonl") {
      const text = await response.text();
      const parsed = parseJsonlResults(text);
      const totalCountHeader = response.headers.get("X-Total-Count");
      if (totalCountHeader) {
        const totalCount = Number.parseInt(totalCountHeader, 10);
        if (Number.isFinite(totalCount)) {
          return { ...parsed, totalCount };
        }
      }
      return parsed;
    } else if (format === "xlsx") {
      // For binary formats, convert to base64 for transport
      const buffer = await response.arrayBuffer();
      const bytes = new Uint8Array(buffer);
      let binary = "";
      for (let i = 0; i < bytes.byteLength; i++) {
        binary += String.fromCharCode(bytes[i]);
      }
      const base64 = btoa(binary);
      return { raw: base64, isBinary: true };
    } else {
      // For other formats, just store raw text for display
      const text = await response.text();
      return { raw: text };
    }
  } catch (err) {
    return { error: String(err) };
  }
}
