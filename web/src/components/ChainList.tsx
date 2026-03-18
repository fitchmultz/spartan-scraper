/**
 * Purpose: Render the saved job-chain inventory with inline expand, submit, and delete controls.
 * Responsibilities: Display chain summaries, collect optional JSON overrides for submissions, and route destructive/transient feedback through the shared toast system.
 * Scope: Chain list presentation and local interaction state only.
 * Usage: Mount inside the automation route with authoritative chain data and mutation callbacks supplied by the parent container.
 * Invariants/Assumptions: Override input must parse as JSON before submission, only one submit modal is open at a time, and deletions require explicit confirmation through the shared dialog layer.
 */
import { useState, useCallback } from "react";
import type { JobChain, ChainCreateRequest } from "../api";
import { getApiErrorMessage } from "../lib/api-errors";
import { formatDateTime } from "../lib/formatting";
import { ActionEmptyState } from "./ActionEmptyState";
import { useToast } from "./toast";

export type { ChainCreateRequest };

interface ChainListProps {
  chains: JobChain[];
  onRefresh: () => void;
  onDelete: (id: string) => Promise<void>;
  onSubmit: (id: string, overrides?: Record<string, unknown>) => Promise<void>;
  loading?: boolean;
  onCreateClick?: () => void;
}

export function ChainList({
  chains,
  onRefresh,
  onDelete,
  onSubmit,
  loading = false,
  onCreateClick,
}: ChainListProps) {
  const toast = useToast();
  const [expandedChain, setExpandedChain] = useState<string | null>(null);
  const [submittingChain, setSubmittingChain] = useState<string | null>(null);
  const [showOverridesModal, setShowOverridesModal] = useState(false);
  const [overridesInput, setOverridesInput] = useState("{}");
  const [overridesError, setOverridesError] = useState<string | null>(null);

  const toggleExpand = useCallback((chainId: string) => {
    setExpandedChain((current) => (current === chainId ? null : chainId));
  }, []);

  const handleDelete = useCallback(
    async (chainId: string) => {
      const confirmed = await toast.confirm({
        title: "Delete this chain?",
        description:
          "This removes the saved workflow definition. Existing jobs already created from it are not affected.",
        confirmLabel: "Delete chain",
        cancelLabel: "Keep chain",
        tone: "error",
      });
      if (!confirmed) {
        return;
      }
      try {
        await onDelete(chainId);
        toast.show({
          tone: "success",
          title: "Chain deleted",
          description: "The saved workflow has been removed.",
        });
      } catch (err) {
        console.error("Failed to delete chain:", err);
        toast.show({
          tone: "error",
          title: "Failed to delete chain",
          description: getApiErrorMessage(
            err,
            "Unable to delete the selected chain.",
          ),
        });
      }
    },
    [onDelete, toast],
  );

  const openSubmitModal = useCallback((chainId: string) => {
    setSubmittingChain(chainId);
    setOverridesInput("{}");
    setOverridesError(null);
    setShowOverridesModal(true);
  }, []);

  const closeSubmitModal = useCallback(() => {
    setShowOverridesModal(false);
    setSubmittingChain(null);
    setOverridesError(null);
  }, []);

  const handleSubmitWithOverrides = useCallback(async () => {
    if (!submittingChain) return;

    // Validate JSON
    let overrides: Record<string, unknown> | undefined;
    try {
      const trimmed = overridesInput.trim();
      if (trimmed && trimmed !== "{}") {
        overrides = JSON.parse(trimmed) as Record<string, unknown>;
      }
    } catch (err) {
      setOverridesError(`Invalid JSON: ${String(err)}`);
      return;
    }

    try {
      await onSubmit(submittingChain, overrides);
      closeSubmitModal();
      toast.show({
        tone: "success",
        title: "Chain submitted",
        description: "The workflow is now queued with the selected overrides.",
      });
    } catch (err) {
      console.error("Failed to submit chain:", err);
      toast.show({
        tone: "error",
        title: "Failed to submit chain",
        description: getApiErrorMessage(
          err,
          "Unable to start the selected chain.",
        ),
      });
    }
  }, [closeSubmitModal, onSubmit, overridesInput, submittingChain, toast]);

  // Find the chain being submitted for display
  const submittingChainData = chains.find((c) => c.id === submittingChain);

  if (chains.length === 0) {
    return (
      <div className="panel">
        <ActionEmptyState
          eyebrow="Automation"
          title="No job chains yet"
          description="Create a reusable workflow when you want one action to fan out into multiple dependent jobs."
          actions={[
            ...(onCreateClick
              ? [{ label: "Create chain", onClick: onCreateClick }]
              : []),
            { label: "Refresh", onClick: onRefresh, tone: "secondary" },
          ]}
        />
      </div>
    );
  }

  return (
    <div className="panel">
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 16,
        }}
      >
        <h2>Job Chains ({chains.length})</h2>
        <div style={{ display: "flex", gap: 8 }}>
          <button
            type="button"
            className="secondary"
            onClick={onRefresh}
            disabled={loading}
          >
            {loading ? "Loading..." : "Refresh"}
          </button>
          {onCreateClick && (
            <button type="button" onClick={onCreateClick}>
              Create Chain
            </button>
          )}
        </div>
      </div>

      <div
        className="chain-list"
        style={{ display: "flex", flexDirection: "column", gap: 12 }}
      >
        {chains.map((chain) => {
          const isExpanded = expandedChain === chain.id;

          return (
            <div
              key={chain.id}
              className="chain-item"
              style={{
                border: "1px solid var(--border)",
                borderRadius: 8,
                padding: 16,
                background: "var(--panel-bg)",
              }}
            >
              {/* Header */}
              <button
                type="button"
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                  width: "100%",
                  background: "none",
                  border: "none",
                  padding: 0,
                  cursor: "pointer",
                  textAlign: "left",
                }}
                onClick={() => toggleExpand(chain.id)}
                aria-expanded={isExpanded}
                aria-label={`Chain ${chain.name}`}
              >
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 12,
                    flex: 1,
                    minWidth: 0,
                  }}
                >
                  <span
                    className="badge"
                    style={{
                      padding: "4px 8px",
                      borderRadius: 4,
                      fontSize: 12,
                      fontWeight: 600,
                      textTransform: "uppercase",
                      backgroundColor: "#e0e7ff",
                      color: "#4338ca",
                      flexShrink: 0,
                    }}
                  >
                    Chain
                  </span>
                  <span
                    style={{
                      fontWeight: 600,
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                    title={chain.name}
                  >
                    {chain.name}
                  </span>
                  <span
                    style={{
                      color: "var(--text-muted)",
                      fontSize: 14,
                      flexShrink: 0,
                    }}
                  >
                    {chain.id.slice(0, 8)}...
                  </span>
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
                  <span style={{ color: "var(--text-muted)", fontSize: 14 }}>
                    {chain.definition.nodes.length} node
                    {chain.definition.nodes.length !== 1 ? "s" : ""}
                  </span>
                  <span style={{ fontSize: 12 }}>{isExpanded ? "▼" : "▶"}</span>
                </div>
              </button>

              {/* Description */}
              {chain.description && (
                <div
                  style={{
                    marginTop: 8,
                    fontSize: 14,
                    color: "var(--text-muted)",
                  }}
                >
                  {chain.description}
                </div>
              )}

              {/* Expanded details */}
              {isExpanded && (
                <div
                  style={{
                    marginTop: 16,
                    paddingTop: 16,
                    borderTop: "1px solid var(--border)",
                  }}
                >
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "flex-start",
                      marginBottom: 12,
                      flexWrap: "wrap",
                      gap: 12,
                    }}
                  >
                    <div style={{ fontSize: 13, color: "var(--text-muted)" }}>
                      <div>ID: {chain.id}</div>
                      <div>Created: {formatDateTime(chain.createdAt)}</div>
                      <div>Updated: {formatDateTime(chain.updatedAt)}</div>
                      <div>Nodes: {chain.definition.nodes.length}</div>
                      <div>Edges: {chain.definition.edges.length}</div>
                    </div>
                    <div style={{ display: "flex", gap: 8 }}>
                      <button
                        type="button"
                        className="secondary"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDelete(chain.id);
                        }}
                        style={{ color: "#ff6b6b" }}
                      >
                        Delete
                      </button>
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          openSubmitModal(chain.id);
                        }}
                      >
                        Submit
                      </button>
                    </div>
                  </div>

                  {/* Node list */}
                  <div style={{ marginTop: 12 }}>
                    <h4 style={{ fontSize: 14, marginBottom: 8 }}>Nodes</h4>
                    <div
                      style={{
                        maxHeight: 200,
                        overflow: "auto",
                        border: "1px solid var(--border)",
                        borderRadius: 4,
                      }}
                    >
                      <table style={{ width: "100%", fontSize: 13 }}>
                        <thead>
                          <tr style={{ background: "var(--bg)" }}>
                            <th
                              style={{
                                textAlign: "left",
                                padding: 8,
                                fontWeight: 600,
                              }}
                            >
                              ID
                            </th>
                            <th
                              style={{
                                textAlign: "left",
                                padding: 8,
                                fontWeight: 600,
                              }}
                            >
                              Kind
                            </th>
                            <th
                              style={{
                                textAlign: "left",
                                padding: 8,
                                fontWeight: 600,
                              }}
                            >
                              Name
                            </th>
                          </tr>
                        </thead>
                        <tbody>
                          {chain.definition.nodes.map((node) => (
                            <tr
                              key={node.id}
                              style={{ borderTop: "1px solid var(--border)" }}
                            >
                              <td style={{ padding: 8 }}>{node.id}</td>
                              <td style={{ padding: 8 }}>
                                <span
                                  className="badge"
                                  style={{
                                    fontSize: 11,
                                    textTransform: "uppercase",
                                  }}
                                >
                                  {node.kind}
                                </span>
                              </td>
                              <td style={{ padding: 8 }}>
                                {node.metadata?.name || "-"}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>

                  {/* Edges */}
                  {chain.definition.edges.length > 0 && (
                    <div style={{ marginTop: 12 }}>
                      <h4 style={{ fontSize: 14, marginBottom: 8 }}>
                        Dependencies
                      </h4>
                      <div
                        style={{
                          display: "flex",
                          flexWrap: "wrap",
                          gap: 8,
                        }}
                      >
                        {chain.definition.edges.map((edge) => (
                          <span
                            key={`${edge.from}-${edge.to}`}
                            className="badge"
                            style={{
                              fontSize: 12,
                              background: "var(--bg)",
                            }}
                          >
                            {edge.from} → {edge.to}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* Overrides Modal */}
      {showOverridesModal && (
        <div
          style={{
            position: "fixed",
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: "rgba(0, 0, 0, 0.5)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex: 1000,
          }}
          onClick={(e) => {
            if (e.target === e.currentTarget) {
              closeSubmitModal();
            }
          }}
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              closeSubmitModal();
            }
          }}
          role="dialog"
          aria-modal="true"
        >
          <div
            style={{
              background: "var(--panel-bg)",
              borderRadius: 8,
              padding: 24,
              maxWidth: 600,
              width: "90%",
              maxHeight: "80vh",
              overflow: "auto",
            }}
          >
            <h3 style={{ marginTop: 0 }}>
              Submit Chain: {submittingChainData?.name}
            </h3>
            <p style={{ fontSize: 14, color: "var(--text-muted)" }}>
              Optionally provide parameter overrides as JSON (keyed by node ID):
            </p>
            <textarea
              value={overridesInput}
              onChange={(e) => {
                setOverridesInput(e.target.value);
                setOverridesError(null);
              }}
              placeholder='{"step1": {"url": "https://example.com"}}'
              style={{
                width: "100%",
                minHeight: 150,
                fontFamily: "monospace",
                fontSize: 13,
                padding: 12,
                border: overridesError
                  ? "1px solid var(--error)"
                  : "1px solid var(--border)",
                borderRadius: 4,
                background: "var(--bg)",
                color: "var(--text)",
                resize: "vertical",
              }}
            />
            {overridesError && (
              <p style={{ color: "var(--error)", fontSize: 13, marginTop: 8 }}>
                {overridesError}
              </p>
            )}
            <div
              style={{
                display: "flex",
                justifyContent: "flex-end",
                gap: 8,
                marginTop: 16,
              }}
            >
              <button
                type="button"
                className="secondary"
                onClick={closeSubmitModal}
              >
                Cancel
              </button>
              <button type="button" onClick={handleSubmitWithOverrides}>
                Submit
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
