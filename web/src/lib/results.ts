/**
 * Purpose: Results loading, export, and browser-download utilities for saved job results.
 * Responsibilities: Build result URLs, load paginated results, trigger server-side exports, and initiate browser file downloads for export content.
 * Scope: Pure data and DOM-download helpers; no React state or component logic.
 * Usage: Import from results-explorer hooks and anywhere that needs to load, export, or download job results.
 * Invariants/Assumptions: Export responses conform to the `ExportOutcomeResponse` API contract; binary content is base64-encoded.
 *
 * @module results
 */

import type {
  ExportInspection,
  ExportOutcomeResponse,
  ExportShapeConfig,
  ResultTransformConfig,
} from "../api";
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
  outcome: ExportInspection;
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
  const params = new URLSearchParams({ format });
  if (format === "jsonl" && typeof limit === "number") {
    params.set("limit", String(limit));
  }
  if (format === "jsonl" && typeof offset === "number") {
    params.set("offset", String(offset));
  }
  return `/v1/jobs/${jobId}/results?${params.toString()}`;
}

export async function loadResults(
  jobId: string,
  format: string,
  page?: number,
  perPage?: number,
): Promise<ResultsResponse> {
  try {
    const url =
      format === "jsonl" && page && perPage
        ? buildResultsUrl(jobId, format, perPage, (page - 1) * perPage)
        : buildResultsUrl(jobId, format);

    const response = await fetch(buildApiUrl(url));

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

  const payload = (await response.json()) as ExportOutcomeResponse;
  const outcome = payload.export;
  if (!outcome) {
    throw new Error("Export response did not include an export outcome.");
  }

  const artifact = outcome.artifact;
  return {
    outcome,
    content: artifact?.content ?? "",
    filename: artifact?.filename ?? `${jobId}.${request.format || "jsonl"}`,
    contentType: artifact?.contentType ?? "application/octet-stream",
    isBinary: artifact?.encoding === "base64",
  };
}

function parseJsonlResults(text: string): JsonlParseResult {
  const trimmed = text.trim();
  if (!trimmed) {
    return { data: [], raw: text };
  }

  const lines = trimmed.split(/\r?\n/).filter(Boolean);
  try {
    const data = lines.map((line) => JSON.parse(line));
    return {
      data,
      raw: JSON.stringify(data, null, 2),
    };
  } catch {
    return {
      error: "Failed to parse JSONL results response.",
      raw: text,
    };
  }
}

async function parseErrorResponse(
  response: Response,
  status: number,
  statusText: string,
): Promise<string> {
  const text = await response.text();
  if (!text) {
    return `${status} ${statusText}`;
  }

  try {
    const parsed = JSON.parse(text) as { error?: string };
    if (parsed.error) {
      return parsed.error;
    }
  } catch {
    // Fall through and return the raw text.
  }

  return text;
}

/**
 * Decode a base64 string to an ArrayBuffer for binary Blob construction.
 */
export function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);
  for (let index = 0; index < binaryString.length; index++) {
    bytes[index] = binaryString.charCodeAt(index);
  }
  return bytes.buffer;
}

/**
 * Trigger a browser file download for the given content.
 *
 * Creates a temporary Blob, appends an anchor element, clicks it, then cleans up.
 * Supports both text and base64-encoded binary content.
 */
export function downloadFile(
  content: string,
  filename: string,
  mimeType: string,
  isBinary = false,
): void {
  const blob = isBinary
    ? new Blob([base64ToArrayBuffer(content)], { type: mimeType })
    : new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
