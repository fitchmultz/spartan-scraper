/**
 * Purpose: Render the immediate result of a manual watch check using the canonical persisted inspection envelope.
 * Responsibilities: Show the just-completed inspection, artifact previews, diff content, and next-step actions with a direct jump into saved history.
 * Scope: Manual watch-check modal presentation only; check execution and history loading stay in parent components.
 * Usage: Render from `WatchManager` after `onCheck` completes.
 * Invariants/Assumptions: `inspection` already represents the persisted check returned from the canonical watch-check endpoint.
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
  inspection,
  onClose,
  onOpenHistory,
}: CheckResultModalProps) {
  const artifacts = inspection.artifacts || [];

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
                label={inspection.status}
                tone={getWatchCheckStatusTone(inspection.status)}
              />
              <span style={{ color: "var(--muted)", fontSize: 12 }}>
                {formatDateTime(inspection.checkedAt, "Never")}
              </span>
            </div>
            <h3 style={{ margin: "8px 0 0" }}>{inspection.title}</h3>
            <p style={{ margin: "8px 0 0", color: "var(--muted)" }}>
              {inspection.message}
            </p>
          </div>
          <div className="row" style={{ gap: 8 }}>
            <button
              type="button"
              onClick={() => onOpenHistory(inspection.id)}
              className="secondary"
            >
              View history
            </button>
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
            <div>{inspection.watchId}</div>
          </div>
          <div>
            <strong>Check ID</strong>
            <div>{inspection.id}</div>
          </div>
          <div>
            <strong>Changed</strong>
            <div>{inspection.changed ? "Yes" : "No"}</div>
          </div>
          <div>
            <strong>Baseline</strong>
            <div>{inspection.baseline ? "Yes" : "No"}</div>
          </div>
        </div>

        {inspection.triggeredJobs && inspection.triggeredJobs.length > 0 ? (
          <div
            style={{
              padding: 12,
              backgroundColor: "rgba(34, 197, 94, 0.1)",
              borderRadius: 12,
              color: "#22c55e",
              marginBottom: 16,
            }}
          >
            <strong>Triggered Jobs:</strong>{" "}
            {inspection.triggeredJobs.join(", ")}
          </div>
        ) : null}

        {inspection.error ? (
          <div
            style={{
              marginBottom: 16,
              padding: 12,
              borderRadius: 12,
              border: "1px solid rgba(239, 68, 68, 0.3)",
              background: "rgba(239, 68, 68, 0.08)",
            }}
          >
            <strong>Error</strong>
            <p style={{ margin: "8px 0 0" }}>{inspection.error}</p>
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

        {inspection.actions?.length ? (
          <div style={{ marginBottom: 16 }}>
            <h4 style={{ marginBottom: 8 }}>Recommended next steps</h4>
            <CapabilityActionList
              actions={inspection.actions}
              onNavigate={(path) => {
                window.location.assign(path);
              }}
              onRefresh={async () => undefined}
            />
          </div>
        ) : null}

        {inspection.status !== "failed" && inspection.diffText ? (
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
              {inspection.diffText}
            </pre>
          </div>
        ) : null}
      </div>
    </div>
  );
}
