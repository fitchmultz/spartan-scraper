/**
 * Purpose: Share pure reader helpers across the split saved-results reader modules.
 * Responsibilities: Provide summary/navigator formatting, crawl-detail guards, and research evidence rendering helpers without carrying route state.
 * Scope: Saved-results reader helpers only.
 * Usage: Imported by the split reader components under `results-viewer/`.
 * Invariants/Assumptions: Helper output remains presentation-ready, evidence field rendering stays consistent across research detail surfaces, and crawl-detail checks rely on normalized result structure.
 */

import type { EvidenceItem, ResearchResultItem, ResultItem } from "../../types";
import { isCrawlResultItem } from "../../lib/form-utils";

function getFieldDisplayValues(
  field: NonNullable<EvidenceItem["fields"]>[string],
) {
  if (field.values && field.values.length > 0) {
    return field.values;
  }
  return [];
}

export function renderEvidenceFields(fields: EvidenceItem["fields"]) {
  if (!fields || Object.keys(fields).length === 0) {
    return null;
  }

  return (
    <div className="results-viewer__field-block">
      <div className="results-viewer__section-label">Extracted fields</div>
      <div className="job-list">
        {Object.entries(fields).map(([name, field]) => {
          const values = getFieldDisplayValues(field);
          return (
            <div key={name} className="job-item">
              <div style={{ fontWeight: 600 }}>{name}</div>
              {values.length > 0 ? (
                <ul style={{ margin: "6px 0 0", paddingLeft: 18 }}>
                  {values.map((value) => (
                    <li key={`${name}-${value}`}>{value}</li>
                  ))}
                </ul>
              ) : (
                <div style={{ color: "var(--text-muted)" }}>
                  No string values returned.
                </div>
              )}
              {field.rawObject ? (
                <pre style={{ marginTop: 8, whiteSpace: "pre-wrap" }}>
                  {field.rawObject}
                </pre>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function truncateExcerpt(text?: string): string | null {
  if (!text) {
    return null;
  }

  const normalized = text.replace(/\s+/g, " ").trim();
  if (!normalized) {
    return null;
  }

  if (normalized.length <= 420) {
    return normalized;
  }

  return `${normalized.slice(0, 417)}…`;
}

function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) {
    return text;
  }

  return `${text.slice(0, Math.max(0, maxLength - 1)).trimEnd()}…`;
}

export function getResearchResultTitle(
  item: ResearchResultItem,
  index: number,
): string {
  return item.query?.trim() || `Research result ${index + 1}`;
}

export function getResearchSummaryPreview(
  summary?: string | null,
  maxLength = 180,
): string | null {
  const normalized = truncateExcerpt(summary ?? undefined);
  if (!normalized) {
    return null;
  }
  return truncateText(normalized, maxLength);
}

export function getResearchResultMeta(
  item: ResearchResultItem,
  fallback: string,
): string {
  const parts: string[] = [];

  if (item.evidence && item.evidence.length > 0) {
    parts.push(`${item.evidence.length} evidence`);
  }
  if (item.clusters && item.clusters.length > 0) {
    parts.push(`${item.clusters.length} clusters`);
  }
  if (item.citations && item.citations.length > 0) {
    parts.push(`${item.citations.length} citations`);
  }

  return parts.join(" · ") || fallback;
}

export function getResearchSourceLabels(evidence: EvidenceItem[]): string[] {
  const seen = new Set<string>();
  const labels: string[] = [];

  for (const item of evidence) {
    const label = (item.title || item.url || "").trim();
    if (!label || seen.has(label)) {
      continue;
    }
    seen.add(label);
    labels.push(label);
    if (labels.length >= 3) {
      break;
    }
  }

  return labels;
}

export function hasStructuredSelectedItem(item: ResultItem | null): boolean {
  return (
    !!item &&
    isCrawlResultItem(item) &&
    !!(
      (item.normalized && Object.keys(item.normalized).length > 0) ||
      (item.extracted && Object.keys(item.extracted).length > 0) ||
      (item.metadata && Object.keys(item.metadata).length > 0)
    )
  );
}
