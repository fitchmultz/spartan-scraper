/**
 * Results loading and direct export utilities for saved job results.
 *
 * @module results
 */

import type { ExportShapeConfig, ResultTransformConfig } from "../api";
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

export interface ResultExportRequest {
  format?: "jsonl" | "json" | "md" | "csv" | "xlsx";
  shape?: ExportShapeConfig;
  transform?: ResultTransformConfig;
}

export interface ResultExportResponse {
  content: string;
  filename: string;
  contentType: string;
  isBinary: boolean;
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
 * Parse JSON array formatted results.
 *
 * Used by paginated jsonl responses, which return a structured JSON array plus
 * pagination headers instead of newline-delimited raw file contents.
 */
export function parseJsonArrayResults(text: string): JsonlParseResult {
  try {
    const parsed = JSON.parse(text) as unknown;

    if (!Array.isArray(parsed)) {
      return {
        error: "Expected paginated results to be a JSON array.",
        raw: text,
      };
    }

    return {
      data: parsed,
      raw: JSON.stringify(parsed, null, 2),
    };
  } catch {
    return {
      error: "Failed to parse paginated results response.",
      raw: text,
    };
  }
}

/**
 * Build the raw-results URL for a given job ID and format.
 *
 * Includes pagination parameters for jsonl format (limit, offset) but not for other formats.
 */
export function buildResultsUrl(
  jobId: string,
  format: string,
  limit?: number,
  offset?: number,
): string {
  let path = `/v1/jobs/${jobId}/results?format=${format}`;

  if (format === "jsonl" && limit !== undefined && offset !== undefined) {
    path += `&limit=${limit}&offset=${offset}`;
  }

  return buildApiUrl(path);
}

/**
 * Parse error response from API.
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

  if (parsedItems.length === 0 && lines.length > 0) {
    return {
      error: "No valid results found. Results file may be corrupted.",
      raw: text,
    };
  }

  return { data: parsedItems, raw: text };
}

/**
 * Load raw saved results for a job from the API.
 */
export async function loadResults(
  jobId: string,
  format: string = "jsonl",
  page: number = 1,
  resultsPerPage: number = 100,
): Promise<ResultsResponse> {
  try {
    const offset = (page - 1) * resultsPerPage;
    const response = await fetch(
      buildResultsUrl(jobId, format, resultsPerPage, offset),
    );

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
      const contentType = response.headers.get("Content-Type") ?? "";
      const parsed = contentType.includes("application/json")
        ? parseJsonArrayResults(text)
        : parseJsonlResults(text);
      const totalCountHeader = response.headers.get("X-Total-Count");
      if (totalCountHeader) {
        const totalCount = Number.parseInt(totalCountHeader, 10);
        if (Number.isFinite(totalCount)) {
          return { ...parsed, totalCount };
        }
      }
      return parsed;
    }

    const text = await response.text();
    return { raw: text };
  } catch (err) {
    return { error: String(err) };
  }
}

export async function exportResults(
  jobId: string,
  request: ResultExportRequest,
): Promise<ResultExportResponse> {
  const response = await fetch(buildApiUrl(`/v1/jobs/${jobId}/export`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const errorMessage = await parseErrorResponse(
      response,
      response.status,
      response.statusText,
    );
    throw new Error(errorMessage);
  }

  const contentType =
    response.headers.get("Content-Type") ?? "application/octet-stream";
  const requestedFormat = request.format || "jsonl";
  const filename = parseExportFilename(
    response.headers.get("Content-Disposition"),
    `${jobId}.${requestedFormat}`,
  );
  const isBinary = contentType.includes(
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  );

  if (isBinary) {
    const buffer = await response.arrayBuffer();
    const bytes = new Uint8Array(buffer);
    let binary = "";
    for (let i = 0; i < bytes.byteLength; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    return {
      content: btoa(binary),
      filename,
      contentType,
      isBinary: true,
    };
  }

  return {
    content: await response.text(),
    filename,
    contentType,
    isBinary: false,
  };
}

function parseExportFilename(
  contentDisposition: string | null,
  fallback: string,
): string {
  if (!contentDisposition) {
    return fallback;
  }

  const starMatch = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (starMatch?.[1]) {
    try {
      return decodeURIComponent(starMatch[1]);
    } catch {
      return starMatch[1];
    }
  }

  const match = contentDisposition.match(/filename="?([^";]+)"?/i);
  return match?.[1] || fallback;
}
