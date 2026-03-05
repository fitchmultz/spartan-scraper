/**
 * Normalized View Component
 *
 * Displays the normalized data field from a crawl result item. Parses the raw JSONL
 * output, extracts the target line by index, and pretty-prints the normalized data.
 * Returns null if parsing fails or if the result type doesn't contain normalized data.
 *
 * @module NormalizedView
 */
import { isCrawlResultItem } from "../lib/form-utils";
import type { ResultItem } from "../types";

export function NormalizedView({
  raw,
  index,
}: {
  raw: string;
  index?: number;
}) {
  try {
    const lines = raw.split("\n").filter((line) => line.trim());
    const targetIndex = index ?? 0;
    if (targetIndex >= lines.length) return null;
    const data = JSON.parse(lines[targetIndex]) as ResultItem;
    if (!isCrawlResultItem(data) || !data.normalized) {
      return <div>No normalized data found for this result type.</div>;
    }
    return (
      <pre style={{ background: "rgba(0, 50, 50, 0.3)" }}>
        {JSON.stringify(data.normalized, null, 2)}
      </pre>
    );
  } catch {
    return <div>Failed to parse result.</div>;
  }
}
