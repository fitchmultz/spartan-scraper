/**
 * Normalized View Component
 *
 * Displays normalized data from the selected result item.
 * Returns null if the selected item is missing and shows a message when the current
 * result shape does not provide normalized output.
 *
 * @module NormalizedView
 */
import { isCrawlResultItem } from "../lib/form-utils";
import type { ResultItem } from "../types";

export function NormalizedView({ item }: { item: ResultItem | null }) {
  if (!item) {
    return null;
  }

  if (!isCrawlResultItem(item) || !item.normalized) {
    return <div>No normalized data found for this result type.</div>;
  }

  return (
    <pre style={{ background: "rgba(0, 50, 50, 0.3)" }}>
      {JSON.stringify(item.normalized, null, 2)}
    </pre>
  );
}
