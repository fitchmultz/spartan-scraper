/**
 * Purpose: Centralize shared constants and inline style tokens for export schedule authoring sections.
 * Responsibilities: Expose select options and shared section/field style objects so the extracted form sections stay visually aligned.
 * Scope: Export schedule form presentation constants only; business logic and dialog orchestration stay elsewhere.
 * Usage: Import from the focused export schedule section modules.
 * Invariants/Assumptions: These constants remain local to export schedule authoring and should not become a generic cross-feature style system.
 */

export const JOB_KIND_OPTIONS = [
  { value: "scrape", label: "Scrape" },
  { value: "crawl", label: "Crawl" },
  { value: "research", label: "Research" },
] as const;

export const JOB_STATUS_OPTIONS = [
  { value: "completed", label: "Completed" },
  { value: "succeeded", label: "Succeeded" },
  { value: "failed", label: "Failed" },
  { value: "canceled", label: "Canceled" },
] as const;

export const FORMAT_OPTIONS = [
  { value: "json", label: "JSON" },
  { value: "jsonl", label: "JSON Lines" },
  { value: "md", label: "Markdown" },
  { value: "csv", label: "CSV" },
  { value: "xlsx", label: "Excel (XLSX)" },
] as const;

export const DESTINATION_OPTIONS = [
  { value: "local", label: "Local File" },
  { value: "webhook", label: "Webhook" },
] as const;

export const sectionStyle = {
  marginBottom: 24,
  padding: 16,
  backgroundColor: "var(--bg-alt)",
  borderRadius: 8,
} as const;

export const sectionTitleStyle = {
  margin: "0 0 12px 0",
  fontSize: 14,
} as const;
export const fieldLabelStyle = {
  display: "block",
  marginBottom: 4,
  fontSize: 13,
} as const;
export const mutedInlineStyle = {
  color: "var(--muted)",
  fontSize: 13,
} as const;
export const codeTextareaStyle = {
  width: "100%",
  fontFamily: "monospace",
  fontSize: 12,
} as const;
