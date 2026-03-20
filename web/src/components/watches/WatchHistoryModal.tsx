/**
 * Purpose: Render persisted watch check history and detailed inspection inside a modal workflow.
 * Responsibilities: Show paginated history rows, load-sensitive selection state, artifact previews, and guided next steps for the selected check.
 * Scope: Watch history presentation only; loading and API orchestration stay in the parent container.
 * Usage: Mount from `WatchManager` after loading persisted watch history for a specific watch.
 * Invariants/Assumptions: Records are already sanitized transport-safe inspections, and `selectedCheck` belongs to the current watch when present.
 */

import type { WatchArtifact } from "../../api";
import type { WatchHistoryModalProps } from "../../types/watch";
import { formatDateTime } from "../../lib/formatting";
import {
  getWatchArtifactLabel,
  getWatchArtifactUrl,
} from "../../lib/watch-utils";
import { getWatchCheckStatusTone } from "../../lib/status-display";
import { CapabilityActionList } from "../CapabilityActionList";
import { StatusPill } from "../StatusPill";
import { ActionEmptyState } from "../ActionEmptyState";

function WatchArtifactGrid({ artifacts }: { artifacts: WatchArtifact[] }) {
  if (!artifacts?.length) {
    return null;
  }

  return (
    <div style={{ marginTop: 16 }}>
      <strong>Artifacts</strong>
      <div
        style={{
          display: "grid",
          gap: 12,
          gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
          marginTop: 8,
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
              {artifact.contentType.startsWith("image/") && downloadUrl ? (
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
  );
}

export function WatchHistoryModal({
  watch,
  records,
  total,
  limit,
  offset,
  loading,
  selectedCheck,
  selectedCheckLoading,
  onClose,
  onSelectCheck,
  onPageChange,
}: WatchHistoryModalProps) {
  const currentPage = Math.floor(offset / limit) + 1;
  const totalPages = Math.max(1, Math.ceil(total / limit));

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
          maxWidth: 1180,
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
            <h3 style={{ margin: 0 }}>Watch History: {watch.url}</h3>
            <p style={{ margin: "8px 0 0", color: "var(--muted)" }}>
              Inspect recent checks, compare saved artifacts, and copy the next
              action without leaving Automation.
            </p>
          </div>
          <button type="button" onClick={onClose} className="secondary">
            Close
          </button>
        </div>

        {loading ? (
          <div role="status" aria-live="polite" style={{ padding: 24 }}>
            <ActionEmptyState
              eyebrow="History"
              title="Loading watch history"
              description="Fetching saved checks and inspection summaries for this watch."
            />
          </div>
        ) : records.length === 0 ? (
          <div style={{ padding: 24 }}>
            <ActionEmptyState
              eyebrow="History"
              title="No watch history found yet"
              description="Run a manual check or wait for the scheduler to record one."
            />
          </div>
        ) : (
          <>
            <div
              className="row"
              style={{
                justifyContent: "space-between",
                alignItems: "center",
                marginBottom: 16,
                fontSize: 13,
                color: "var(--muted)",
              }}
            >
              <span>
                Showing {offset + 1}-{Math.min(offset + records.length, total)}{" "}
                of {total}
              </span>
              <span>
                Page {currentPage} of {totalPages}
              </span>
            </div>

            <div
              style={{
                display: "grid",
                gap: 16,
                gridTemplateColumns: "minmax(280px, 360px) minmax(0, 1fr)",
              }}
            >
              <div
                style={{
                  display: "grid",
                  gap: 8,
                  alignContent: "start",
                }}
              >
                {records.map((record) => {
                  const selected = record.id === selectedCheck?.id;
                  return (
                    <button
                      key={record.id}
                      type="button"
                      onClick={() => onSelectCheck(record.id)}
                      className="secondary"
                      style={{
                        textAlign: "left",
                        padding: 12,
                        borderColor: selected
                          ? "var(--accent)"
                          : "var(--stroke)",
                        background: selected
                          ? "rgba(59, 130, 246, 0.08)"
                          : undefined,
                      }}
                    >
                      <div
                        className="row"
                        style={{ justifyContent: "space-between", gap: 8 }}
                      >
                        <StatusPill
                          label={record.status}
                          tone={getWatchCheckStatusTone(record.status)}
                        />
                        <span
                          style={{
                            color: "var(--muted)",
                            fontSize: 12,
                            fontFamily: "monospace",
                          }}
                        >
                          {record.id}
                        </span>
                      </div>
                      <div style={{ fontWeight: 600, marginTop: 8 }}>
                        {record.title}
                      </div>
                      <div
                        style={{
                          marginTop: 6,
                          fontSize: 13,
                          color: "var(--muted)",
                        }}
                      >
                        {record.message}
                      </div>
                      <div
                        style={{
                          marginTop: 8,
                          fontSize: 12,
                          color: "var(--muted)",
                        }}
                      >
                        {formatDateTime(record.checkedAt)}
                      </div>
                    </button>
                  );
                })}
              </div>

              <div
                className="panel"
                style={{
                  padding: 16,
                  border: "1px solid var(--stroke)",
                  minHeight: 320,
                }}
              >
                {selectedCheckLoading ? (
                  <div role="status" aria-live="polite" style={{ padding: 24 }}>
                    <ActionEmptyState
                      eyebrow="History"
                      title="Loading check details"
                      description="Fetching saved artifacts, diff output, and recommended next steps for this run."
                    />
                  </div>
                ) : selectedCheck ? (
                  <>
                    <div
                      className="row"
                      style={{
                        justifyContent: "space-between",
                        alignItems: "flex-start",
                        gap: 16,
                      }}
                    >
                      <div>
                        <div
                          className="row"
                          style={{
                            alignItems: "center",
                            gap: 8,
                            marginBottom: 8,
                          }}
                        >
                          <StatusPill
                            label={selectedCheck.status}
                            tone={getWatchCheckStatusTone(selectedCheck.status)}
                          />
                          <span style={{ color: "var(--muted)", fontSize: 12 }}>
                            {formatDateTime(selectedCheck.checkedAt)}
                          </span>
                        </div>
                        <h4 style={{ margin: 0 }}>{selectedCheck.title}</h4>
                        <p style={{ margin: "8px 0 0", color: "var(--muted)" }}>
                          {selectedCheck.message}
                        </p>
                      </div>
                      <div
                        style={{
                          fontFamily: "monospace",
                          fontSize: 12,
                          color: "var(--muted)",
                          textAlign: "right",
                        }}
                      >
                        <div>{selectedCheck.id}</div>
                        <div>{selectedCheck.watchId}</div>
                      </div>
                    </div>

                    <div
                      style={{
                        display: "grid",
                        gridTemplateColumns:
                          "repeat(auto-fit, minmax(180px, 1fr))",
                        gap: 12,
                        marginTop: 16,
                      }}
                    >
                      <div>
                        <strong>URL</strong>
                        <div style={{ wordBreak: "break-word" }}>
                          {selectedCheck.url}
                        </div>
                      </div>
                      <div>
                        <strong>Changed</strong>
                        <div>{selectedCheck.changed ? "Yes" : "No"}</div>
                      </div>
                      <div>
                        <strong>Baseline</strong>
                        <div>{selectedCheck.baseline ? "Yes" : "No"}</div>
                      </div>
                      <div>
                        <strong>Triggered jobs</strong>
                        <div>{selectedCheck.triggeredJobs?.length || 0}</div>
                      </div>
                    </div>

                    {selectedCheck.error ? (
                      <div
                        style={{
                          marginTop: 16,
                          padding: 12,
                          borderRadius: 12,
                          border: "1px solid rgba(239, 68, 68, 0.3)",
                          background: "rgba(239, 68, 68, 0.08)",
                        }}
                      >
                        <strong>Error</strong>
                        <p style={{ margin: "8px 0 0" }}>
                          {selectedCheck.error}
                        </p>
                      </div>
                    ) : null}

                    <WatchArtifactGrid
                      artifacts={selectedCheck.artifacts || []}
                    />

                    {selectedCheck.diffText ? (
                      <div style={{ marginTop: 16 }}>
                        <strong>Diff</strong>
                        <pre
                          style={{
                            backgroundColor: "var(--bg-alt)",
                            padding: 16,
                            borderRadius: 12,
                            overflow: "auto",
                            fontSize: 12,
                            lineHeight: 1.5,
                            maxHeight: 280,
                            marginTop: 8,
                          }}
                        >
                          {selectedCheck.diffText}
                        </pre>
                      </div>
                    ) : null}

                    {selectedCheck.actions?.length ? (
                      <div style={{ marginTop: 16 }}>
                        <strong>Recommended next steps</strong>
                        <div style={{ marginTop: 8 }}>
                          <CapabilityActionList
                            actions={selectedCheck.actions}
                            onNavigate={(path) => {
                              window.location.assign(path);
                            }}
                            onRefresh={async () => undefined}
                          />
                        </div>
                      </div>
                    ) : null}
                  </>
                ) : (
                  <div
                    style={{
                      textAlign: "center",
                      padding: 40,
                      color: "var(--muted)",
                    }}
                  >
                    Select a check to inspect its details.
                  </div>
                )}
              </div>
            </div>

            {totalPages > 1 ? (
              <div
                className="row"
                style={{ justifyContent: "center", gap: 8, marginTop: 16 }}
              >
                <button
                  type="button"
                  onClick={() => onPageChange(offset - limit)}
                  disabled={offset === 0}
                  className="secondary"
                >
                  Previous
                </button>
                <button
                  type="button"
                  onClick={() => onPageChange(offset + limit)}
                  disabled={offset + limit >= total}
                  className="secondary"
                >
                  Next
                </button>
              </div>
            ) : null}
          </>
        )}
      </div>
    </div>
  );
}
