/**
 * Purpose: Render operator-friendly AI candidate previews for render profiles and pipeline JS scripts.
 * Responsibilities: Show latest-only highlights, show field-level diffs against a previous candidate, and preserve raw JSON access with safe fallback behavior.
 * Scope: Shared artifact preview surface for AI automation generator and debugger modals.
 * Usage: Mount inside `AIAuthoringAttemptPanel` for previous/latest candidate content.
 * Invariants/Assumptions: Unchanged fields stay hidden in comparison mode, raw JSON remains available on demand, and unsupported runtime changes default to raw JSON.
 */

import { useEffect, useMemo, useState } from "react";
import type { JsTargetScript, RenderProfile } from "../api";
import {
  summarizePipelineScriptCandidateDiff,
  summarizeRenderProfileCandidateDiff,
  type CandidateDiffSummary,
  type FieldChange,
} from "../lib/diff-utils";
import { formatDisplayValue, truncateEnd } from "../lib/formatting";

type ArtifactKind = "render-profile" | "pipeline-js";

type CandidateArtifact = RenderProfile | JsTargetScript;

interface AICandidateDiffViewProps {
  artifactKind: ArtifactKind;
  previousArtifact?: CandidateArtifact | null;
  latestArtifact?: CandidateArtifact | null;
}

function formatCandidateValue(value: unknown): string {
  if (Array.isArray(value) || (value && typeof value === "object")) {
    return truncateEnd(JSON.stringify(value, null, 2), 800, "-");
  }

  return formatDisplayValue(value, {
    nullLabel: "null",
    undefinedLabel: "undefined",
    maxLength: 800,
  });
}

function CandidateFieldChangeRow({ change }: { change: FieldChange }) {
  return (
    <section aria-label={change.field} className="diff-field-change">
      <div className="diff-field-header" role="presentation">
        <span className="diff-field-name">{change.field}</span>
      </div>
      <div className="diff-field-values">
        <div className="diff-field-old">
          <span className="diff-field-label">Previous</span>
          <pre className="ai-candidate-field-pre">
            {formatCandidateValue(change.oldValue)}
          </pre>
        </div>
        <div className="diff-field-new">
          <span className="diff-field-label">Latest</span>
          <pre className="ai-candidate-field-pre">
            {formatCandidateValue(change.newValue)}
          </pre>
        </div>
      </div>
    </section>
  );
}

export function AICandidateDiffView({
  artifactKind,
  previousArtifact = null,
  latestArtifact = null,
}: AICandidateDiffViewProps) {
  const summary: CandidateDiffSummary = useMemo(() => {
    if (artifactKind === "render-profile") {
      return summarizeRenderProfileCandidateDiff(
        previousArtifact as RenderProfile | null,
        latestArtifact as RenderProfile | null,
      );
    }

    return summarizePipelineScriptCandidateDiff(
      previousArtifact as JsTargetScript | null,
      latestArtifact as JsTargetScript | null,
    );
  }, [artifactKind, previousArtifact, latestArtifact]);
  const [showRawJson, setShowRawJson] = useState(
    summary.shouldShowRawJsonByDefault,
  );

  useEffect(() => {
    setShowRawJson(summary.shouldShowRawJsonByDefault);
  }, [summary.shouldShowRawJsonByDefault]);

  if (!latestArtifact) {
    return null;
  }

  const hasComparison = !!previousArtifact;

  return (
    <div className="ai-candidate-diff space-y-3">
      {hasComparison ? (
        summary.shouldShowRawJsonByDefault ? (
          <div className="ai-candidate-diff-callout" role="status">
            {summary.rawJsonReason}
          </div>
        ) : summary.changes.length > 0 ? (
          <section className="space-y-2" aria-label="Changed fields">
            {summary.changes.map((change) => (
              <CandidateFieldChangeRow
                key={change.path ?? change.field}
                change={change}
              />
            ))}
          </section>
        ) : (
          <div className="diff-empty">
            No high-signal field changes were summarized.
          </div>
        )
      ) : summary.latestFields.length > 0 ? (
        <dl
          className="ai-candidate-field-list"
          aria-label="Candidate highlights"
        >
          {summary.latestFields.map((field) => (
            <div key={field.path} className="ai-candidate-field-row">
              <dt className="ai-candidate-field-name">{field.label}</dt>
              <dd className="ai-candidate-field-value">
                <pre className="ai-candidate-field-pre">
                  {formatCandidateValue(field.value)}
                </pre>
              </dd>
            </div>
          ))}
        </dl>
      ) : (
        <div className="diff-empty">No summarized fields available.</div>
      )}

      <button
        type="button"
        className="button-secondary ai-candidate-raw-toggle"
        onClick={() => setShowRawJson((current) => !current)}
      >
        {showRawJson ? "Hide raw JSON" : "Show raw JSON"}
      </button>

      {showRawJson ? (
        <div className="ai-candidate-raw-grid">
          {hasComparison ? (
            <div className="space-y-2">
              <span className="ai-candidate-raw-label">Previous</span>
              <pre className="ai-candidate-raw-json">
                {JSON.stringify(previousArtifact, null, 2)}
              </pre>
            </div>
          ) : null}
          <div className="space-y-2">
            <span className="ai-candidate-raw-label">
              {hasComparison ? "Latest" : "Candidate"}
            </span>
            <pre className="ai-candidate-raw-json">
              {JSON.stringify(latestArtifact, null, 2)}
            </pre>
          </div>
        </div>
      ) : null}
    </div>
  );
}
