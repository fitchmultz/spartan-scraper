/**
 * Purpose: Render the immediate result of a manual watch check with a bridge into persisted history inspection.
 * Responsibilities: Show the just-completed outcome, fall back to transient result fields while history detail loads, and surface next-step actions once the persisted inspection is available.
 * Scope: Manual watch-check modal presentation only; check execution and history loading stay in parent components.
 * Usage: Render from `WatchManager` after `onCheck` completes.
 * Invariants/Assumptions: The manual check has already completed, and `inspection` either represents the same `checkId` or is null while the persisted detail is loading.
 */

import type { CheckResultModalProps } from "../../types/watch";
import { formatDateTime } from "../../lib/formatting";
import {
  getWatchArtifactLabel,
  getWatchArtifactUrl,
} from "../../lib/watch-utils";
import { getWatchCheckStatusTone } from "../../lib/status-display";
import { CapabilityActionList } from "../CapabilityActionList";
import { StatusPill } from "../StatusPill";

export function CheckResultModal({
  result,
  inspection,
  onClose,
  onOpenHistory,
}: CheckResultModalProps) {
  const title =
    inspection?.title ||
    (result.baseline
      ? "Baseline recorded"
      : result.changed
        ? "Change detected"
        : result.error
          ? "Check failed"
          : "No change detected");
  const message =
    inspection?.message ||
    (result.error
      ? result.error
      : result.baseline
        ? "The first successful check saved a comparison baseline for this watch."
        : result.changed
          ? "Spartan detected a change during this manual watch check."
          : "The latest manual check matched the saved baseline.");
  const status: "baseline" | "changed" | "failed" | "unchanged" =
    inspection?.status ||
    (result.baseline
      ? "baseline"
      : result.changed
        ? "changed"
        : result.error
          ? "failed"
          : "unchanged");
  const artifacts = inspection?.artifacts || result.artifacts || [];

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        backgroundColor: "rgba(0, 0, 0, 0.7)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
        padding: 20,
      }}
    >
      <div
        className="panel"
        style={{
          maxWidth: 860,
          width: "100%",
          maxHeight: "90vh",
          overflow: "auto",
        }}
      >
        <div
          className="row"
          style={{
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: 16,
            gap: 16,
          }}
        >
          <div>
            <div className="row" style={{ alignItems: "center", gap: 8 }}>
              <StatusPill
                label={status}
                tone={getWatchCheckStatusTone(status)}
              />
              <span style={{ color: "var(--muted)", fontSize: 12 }}>
                {formatDateTime(result.checkedAt, "Never")}
              </span>
            </div>
            <h3 style={{ margin: "8px 0 0" }}>{title}</h3>
            <p style={{ margin: "8px 0 0", color: "var(--muted)" }}>
              {message}
            </p>
          </div>
          <div className="row" style={{ gap: 8 }}>
            {result.checkId ? (
              <button
                type="button"
                onClick={() => onOpenHistory(result.checkId || "")}
                className="secondary"
              >
                View history
              </button>
            ) : null}
            <button type="button" onClick={onClose} className="secondary">
              Close
            </button>
          </div>
        </div>

        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
            gap: 12,
            marginBottom: 16,
          }}
        >
          <div>
            <strong>Watch ID</strong>
            <div>{result.watchId}</div>
          </div>
          <div>
            <strong>Check ID</strong>
            <div>{result.checkId || "Saving..."}</div>
          </div>
          <div>
            <strong>Changed</strong>
            <div>{result.changed ? "Yes" : "No"}</div>
          </div>
          <div>
            <strong>Baseline</strong>
            <div>{result.baseline ? "Yes" : "No"}</div>
          </div>
        </div>

        {result.triggeredJobs && result.triggeredJobs.length > 0 ? (
          <div
            style={{
              padding: 12,
              backgroundColor: "rgba(34, 197, 94, 0.1)",
              borderRadius: 12,
              color: "#22c55e",
              marginBottom: 16,
            }}
          >
            <strong>Triggered Jobs:</strong> {result.triggeredJobs.join(", ")}
          </div>
        ) : null}

        {artifacts.length > 0 ? (
          <div style={{ marginBottom: 16 }}>
            <h4 style={{ marginBottom: 8 }}>Artifacts</h4>
            <div
              style={{
                display: "grid",
                gap: 12,
                gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
              }}
            >
              {artifacts.map((artifact) => {
                const downloadUrl = getWatchArtifactUrl(artifact);
                return (
                  <div
                    key={`${artifact.kind}-${artifact.downloadUrl}`}
                    style={{
                      backgroundColor: "var(--bg-alt)",
                      borderRadius: 12,
                      padding: 12,
                    }}
                  >
                    <div style={{ fontWeight: 600, marginBottom: 8 }}>
                      {getWatchArtifactLabel(artifact.kind)}
                    </div>
                    {artifact.contentType.startsWith("image/") &&
                    downloadUrl ? (
                      <img
                        src={downloadUrl}
                        alt={getWatchArtifactLabel(artifact.kind)}
                        style={{
                          width: "100%",
                          maxHeight: 180,
                          objectFit: "contain",
                          borderRadius: 8,
                          marginBottom: 8,
                          backgroundColor: "rgba(255,255,255,0.04)",
                        }}
                      />
                    ) : null}
                    <div style={{ fontSize: 12, color: "var(--muted)" }}>
                      {artifact.filename}
                    </div>
                    <div style={{ fontSize: 12, color: "var(--muted)" }}>
                      {artifact.contentType}
                      {artifact.byteSize ? ` · ${artifact.byteSize} bytes` : ""}
                    </div>
                    {downloadUrl ? (
                      <a
                        href={downloadUrl}
                        target="_blank"
                        rel="noreferrer"
                        style={{ display: "inline-block", marginTop: 8 }}
                      >
                        Open artifact
                      </a>
                    ) : null}
                  </div>
                );
              })}
            </div>
          </div>
        ) : null}

        {(inspection?.actions?.length || 0) > 0 ? (
          <div style={{ marginBottom: 16 }}>
            <h4 style={{ marginBottom: 8 }}>Recommended next steps</h4>
            <CapabilityActionList
              actions={inspection?.actions || []}
              onNavigate={(path) => {
                window.location.assign(path);
              }}
              onRefresh={async () => undefined}
            />
          </div>
        ) : null}

        {inspection?.diffText || result.diffText ? (
          <div>
            <h4 style={{ marginBottom: 8 }}>Diff</h4>
            <pre
              style={{
                backgroundColor: "var(--bg-alt)",
                padding: 16,
                borderRadius: 12,
                overflow: "auto",
                fontSize: 12,
                lineHeight: 1.5,
                maxHeight: 320,
              }}
            >
              {inspection?.diffText || result.diffText}
            </pre>
          </div>
        ) : null}
      </div>
    </div>
  );
}
