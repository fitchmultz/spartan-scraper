/**
 * Purpose: Render operator-friendly AI candidate previews for render profiles and pipeline JavaScript scripts.
 * Responsibilities: Show selected-only highlights, show field-level diffs against a chosen baseline, and preserve raw artifact JSON access.
 * Scope: Shared artifact preview surface for AI automation generator and debugger modals.
 * Usage: Mount inside `AIAuthoringAttemptPanel` for selected and baseline candidate content.
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
  baselineArtifact?: CandidateArtifact | null;
  selectedArtifact?: CandidateArtifact | null;
  baselineLabel?: string;
  selectedLabel?: string;
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

function CandidateFieldChangeRow({
  change,
  baselineLabel,
  selectedLabel,
}: {
  change: FieldChange;
  baselineLabel: string;
  selectedLabel: string;
}) {
  return (
    <section aria-label={change.field} className="diff-field-change">
      <div className="diff-field-header" role="presentation">
        <span className="diff-field-name">{change.field}</span>
      </div>
      <div className="diff-field-values">
        <div className="diff-field-old">
          <span className="diff-field-label">{baselineLabel}</span>
          <pre className="ai-candidate-field-pre">
            {formatCandidateValue(change.oldValue)}
          </pre>
        </div>
        <div className="diff-field-new">
          <span className="diff-field-label">{selectedLabel}</span>
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
  baselineArtifact = null,
  selectedArtifact = null,
  baselineLabel = "Comparison baseline",
  selectedLabel = "Selected candidate",
}: AICandidateDiffViewProps) {
  const summary: CandidateDiffSummary = useMemo(() => {
    if (artifactKind === "render-profile") {
      return summarizeRenderProfileCandidateDiff(
        baselineArtifact as RenderProfile | null,
        selectedArtifact as RenderProfile | null,
      );
    }

    return summarizePipelineScriptCandidateDiff(
      baselineArtifact as JsTargetScript | null,
      selectedArtifact as JsTargetScript | null,
    );
  }, [artifactKind, baselineArtifact, selectedArtifact]);
  const [showRawJson, setShowRawJson] = useState(
    summary.shouldShowRawJsonByDefault,
  );

  useEffect(() => {
    setShowRawJson(summary.shouldShowRawJsonByDefault);
  }, [summary.shouldShowRawJsonByDefault]);

  if (!selectedArtifact) {
    return null;
  }

  const hasComparison = !!baselineArtifact;

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
                baselineLabel={baselineLabel}
                selectedLabel={selectedLabel}
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
              <span className="ai-candidate-raw-label">{baselineLabel}</span>
              <pre className="ai-candidate-raw-json">
                {JSON.stringify(baselineArtifact, null, 2)}
              </pre>
            </div>
          ) : null}
          <div className="space-y-2">
            <span className="ai-candidate-raw-label">{selectedLabel}</span>
            <pre className="ai-candidate-raw-json">
              {JSON.stringify(selectedArtifact, null, 2)}
            </pre>
          </div>
        </div>
      ) : null}
    </div>
  );
}
