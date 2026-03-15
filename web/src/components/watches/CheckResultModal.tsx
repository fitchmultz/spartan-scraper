/**
 * CheckResultModal Component
 *
 * Displays the result of a manual watch check in a modal dialog.
 * Shows whether content changed, when it was checked, any errors,
 * and the diff text if available.
 *
 * This component does NOT handle:
 * - Triggering watch checks (parent handles that)
 * - State management for check results
 * - API calls to perform checks
 *
 * @module components/watches/CheckResultModal
 */

import type { CheckResultModalProps } from "../../types/watch";
import { formatDateTime } from "../../lib/formatting";

/**
 * Modal component for displaying watch check results
 */
export function CheckResultModal({ result, onClose }: CheckResultModalProps) {
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
          maxWidth: 700,
          width: "100%",
          maxHeight: "80vh",
          overflow: "auto",
        }}
      >
        <div
          className="row"
          style={{
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: 16,
          }}
        >
          <h3 style={{ margin: 0 }}>Check Result</h3>
          <button type="button" onClick={onClose} className="secondary">
            Close
          </button>
        </div>

        <div style={{ marginBottom: 16 }}>
          <div className="row" style={{ gap: 16, marginBottom: 8 }}>
            <span>
              <strong>Changed:</strong>{" "}
              <span
                style={{
                  color: result.changed ? "#22c55e" : "var(--muted)",
                  fontWeight: 600,
                }}
              >
                {result.changed ? "Yes" : "No"}
              </span>
            </span>
            <span>
              <strong>Checked At:</strong>{" "}
              {formatDateTime(result.checkedAt, "Never")}
            </span>
          </div>
          {result.error && (
            <div
              style={{
                padding: 12,
                backgroundColor: "rgba(239, 68, 68, 0.1)",
                borderRadius: 8,
                color: "#ef4444",
                marginTop: 8,
              }}
            >
              <strong>Error:</strong> {result.error}
            </div>
          )}
          {result.triggeredJobs && result.triggeredJobs.length > 0 && (
            <div
              style={{
                padding: 12,
                backgroundColor: "rgba(34, 197, 94, 0.1)",
                borderRadius: 8,
                color: "#22c55e",
                marginTop: 8,
              }}
            >
              <strong>Triggered Jobs:</strong> {result.triggeredJobs.join(", ")}
            </div>
          )}
        </div>

        {result.diffText && (
          <div>
            <h4 style={{ marginBottom: 8 }}>Diff</h4>
            <pre
              style={{
                backgroundColor: "var(--bg-alt)",
                padding: 16,
                borderRadius: 8,
                overflow: "auto",
                fontSize: 12,
                lineHeight: 1.5,
                maxHeight: 300,
              }}
            >
              {result.diffText}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
}
