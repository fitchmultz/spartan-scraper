/**
 * Purpose: Render the webhook-delivery dashboard for the Automation route.
 * Responsibilities: Compose filters, list rendering, pagination, detail view, and guided empty states for both first-run and filtered-no-match scenarios.
 * Scope: Presentation only; API state and mutations live in `WebhookDeliveryContainer`.
 * Usage: Mount from the webhook delivery container with authoritative delivery data and callbacks.
 * Invariants/Assumptions: Empty delivery history should still suggest a next step, and filtered empties should make it easy to recover the last visible results.
 */

import type { WebhookDeliveriesProps } from "../../types/webhook";
import { ActionEmptyState } from "../ActionEmptyState";
import { WebhookDeliveryFilters } from "./WebhookDeliveryFilters";
import { WebhookDeliveryList } from "./WebhookDeliveryList";
import { WebhookDeliveryDetail } from "./WebhookDeliveryDetail";

export function WebhookDeliveries({
  deliveries,
  total,
  loading,
  filters,
  limit,
  offset,
  selectedDelivery,
  detailLoading,
  onRefresh,
  onFilterChange,
  onPageChange,
  onViewDetail,
  onCloseDetail,
}: WebhookDeliveriesProps) {
  const hasMore = offset + limit < total;
  const hasPrev = offset > 0;
  const currentPage = Math.floor(offset / limit) + 1;
  const totalPages = Math.ceil(total / limit) || 1;

  return (
    <div className="panel">
      <div
        className="row"
        style={{
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 16,
        }}
      >
        <h2 style={{ margin: 0 }}>Webhook Deliveries</h2>
        <div className="row" style={{ gap: 8 }}>
          <button
            type="button"
            onClick={onRefresh}
            disabled={loading}
            className="secondary"
          >
            {loading ? "Loading..." : "Refresh"}
          </button>
        </div>
      </div>

      <p style={{ color: "var(--muted)", marginBottom: 16, fontSize: 14 }}>
        View webhook delivery history for debugging and monitoring. Filter by
        job ID or status to troubleshoot delivery issues.
      </p>

      <WebhookDeliveryFilters
        filters={filters}
        onChange={onFilterChange}
        onApply={onRefresh}
        onReset={() =>
          onFilterChange({
            jobId: "",
            status: "all",
          })
        }
      />

      {deliveries.length === 0 && !loading ? (
        <ActionEmptyState
          eyebrow="Automation"
          title={
            filters.jobId || filters.status !== "all"
              ? "No webhook deliveries match these filters"
              : "No webhook deliveries yet"
          }
          description={
            filters.jobId || filters.status !== "all"
              ? "Clear the current filters or refresh after a new delivery is attempted."
              : "Webhook deliveries will appear here after jobs or exports run with webhook notification configured."
          }
          actions={
            filters.jobId || filters.status !== "all"
              ? [
                  {
                    label: "Reset filters",
                    onClick: () =>
                      onFilterChange({
                        jobId: "",
                        status: "all",
                      }),
                  },
                  {
                    label: "Refresh",
                    onClick: onRefresh,
                    tone: "secondary",
                  },
                ]
              : [{ label: "Refresh", onClick: onRefresh }]
          }
        />
      ) : (
        <>
          <WebhookDeliveryList
            deliveries={deliveries}
            onViewDetail={onViewDetail}
          />

          {/* Pagination */}
          {total > 0 && (
            <div
              className="row"
              style={{
                justifyContent: "space-between",
                alignItems: "center",
                marginTop: 16,
                paddingTop: 16,
                borderTop: "1px solid var(--stroke)",
              }}
            >
              <div style={{ fontSize: 14, color: "var(--muted)" }}>
                Showing {offset + 1}-{Math.min(offset + limit, total)} of{" "}
                {total}
              </div>
              <div className="row" style={{ gap: 8 }}>
                <button
                  type="button"
                  onClick={() => onPageChange(offset - limit)}
                  disabled={!hasPrev || loading}
                  className="secondary"
                >
                  Previous
                </button>
                <span
                  style={{
                    fontSize: 14,
                    padding: "8px 12px",
                    color: "var(--muted)",
                  }}
                >
                  Page {currentPage} of {totalPages}
                </span>
                <button
                  type="button"
                  onClick={() => onPageChange(offset + limit)}
                  disabled={!hasMore || loading}
                  className="secondary"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </>
      )}

      {selectedDelivery && (
        <WebhookDeliveryDetail
          delivery={selectedDelivery}
          loading={detailLoading}
          onClose={onCloseDetail}
        />
      )}
    </div>
  );
}
