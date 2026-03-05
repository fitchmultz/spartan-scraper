/**
 * WebhookDeliveryFilters Component
 *
 * Filter controls for the webhook delivery dashboard.
 * Provides inputs for job ID filtering and status selection.
 *
 * This component does NOT handle:
 * - API calls
 * - State persistence
 *
 * @module components/webhooks/WebhookDeliveryFilters
 */

import type {
  WebhookDeliveryFiltersProps,
  DeliveryStatus,
} from "../../types/webhook";

const STATUS_OPTIONS: { value: DeliveryStatus; label: string }[] = [
  { value: "all", label: "All Statuses" },
  { value: "pending", label: "Pending" },
  { value: "delivered", label: "Delivered" },
  { value: "failed", label: "Failed" },
];

export function WebhookDeliveryFilters({
  filters,
  onChange,
  onApply,
  onReset,
}: WebhookDeliveryFiltersProps) {
  const handleJobIdChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onChange({
      ...filters,
      jobId: e.target.value,
    });
  };

  const handleStatusChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    onChange({
      ...filters,
      status: e.target.value as DeliveryStatus,
    });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onApply();
  };

  return (
    <form
      onSubmit={handleSubmit}
      style={{
        marginBottom: 16,
        padding: 16,
        backgroundColor: "var(--surface)",
        borderRadius: 8,
        border: "1px solid var(--stroke)",
      }}
    >
      <div
        className="row"
        style={{
          gap: 16,
          flexWrap: "wrap",
          alignItems: "flex-end",
        }}
      >
        <div style={{ flex: "1 1 200px" }}>
          <label
            htmlFor="webhook-filter-job-id"
            style={{
              display: "block",
              fontSize: 14,
              marginBottom: 4,
              color: "var(--muted)",
            }}
          >
            Job ID
          </label>
          <input
            id="webhook-filter-job-id"
            type="text"
            value={filters.jobId}
            onChange={handleJobIdChange}
            placeholder="Filter by job ID..."
            style={{ width: "100%" }}
          />
        </div>

        <div style={{ flex: "0 0 150px" }}>
          <label
            htmlFor="webhook-filter-status"
            style={{
              display: "block",
              fontSize: 14,
              marginBottom: 4,
              color: "var(--muted)",
            }}
          >
            Status
          </label>
          <select
            id="webhook-filter-status"
            value={filters.status}
            onChange={handleStatusChange}
            style={{ width: "100%" }}
          >
            {STATUS_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>

        <div className="row" style={{ gap: 8 }}>
          <button type="submit" className="secondary">
            Apply
          </button>
          <button type="button" onClick={onReset} className="secondary">
            Reset
          </button>
        </div>
      </div>
    </form>
  );
}
