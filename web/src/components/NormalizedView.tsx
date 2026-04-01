/**
 * Purpose: Render the normalized view UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
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
