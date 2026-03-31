/**
 * Purpose: Coordinate webhook-delivery inspection state for the automation route.
 * Responsibilities: Load paginated delivery records, apply route-local filters, fetch individual delivery detail records, and feed the lazy-loaded webhook dashboard.
 * Scope: Webhook inspection route coordination only; presentation lives in `WebhookDeliveries` and transport stays in the generated API client.
 * Usage: Render from the automation route when operators need to inspect persisted delivery history.
 * Invariants/Assumptions: Server data is authoritative, detail lookups may refine list rows with fuller payloads, and client-side status filtering remains a temporary view-layer refinement.
 */

import {
  useCallback,
  useEffect,
  useState,
  Suspense,
  lazy,
  useMemo,
} from "react";
import {
  getV1WebhooksDeliveries,
  getV1WebhooksDeliveriesById,
  type WebhookDeliveryRecord,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { reportRuntimeError } from "../../lib/runtime-errors";
import type { DeliveryFilters } from "../../types/webhook";

const WebhookDeliveries = lazy(() =>
  import("./WebhookDeliveries").then((mod) => ({
    default: mod.WebhookDeliveries,
  })),
);

const DEFAULT_LIMIT = 20;

export function WebhookDeliveryContainer() {
  // List state
  const [deliveries, setDeliveries] = useState<WebhookDeliveryRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  // Pagination state
  const [offset, setOffset] = useState(0);
  const limit = DEFAULT_LIMIT;

  // Filter state
  const [filters, setFilters] = useState<DeliveryFilters>({
    jobId: "",
    status: "all",
  });

  // Detail view state
  const [selectedDelivery, setSelectedDelivery] =
    useState<WebhookDeliveryRecord | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  /**
   * Refresh deliveries list with current filters and pagination
   */
  const refreshDeliveries = useCallback(async () => {
    setLoading(true);
    try {
      const { data, error } = await getV1WebhooksDeliveries({
        baseUrl: getApiBaseUrl(),
        query: {
          job_id: filters.jobId || undefined,
          limit,
          offset,
        },
      });

      if (error) {
        reportRuntimeError("Failed to load webhook deliveries", error);
        return;
      }

      setDeliveries(data?.deliveries || []);
      setTotal(data?.total || 0);
    } catch (err) {
      reportRuntimeError("Error loading webhook deliveries", err);
    } finally {
      setLoading(false);
    }
  }, [filters.jobId, offset]);

  /**
   * Load single delivery detail
   */
  const loadDeliveryDetail = useCallback(
    async (id: string): Promise<WebhookDeliveryRecord | null> => {
      setDetailLoading(true);
      try {
        const { data, error } = await getV1WebhooksDeliveriesById({
          baseUrl: getApiBaseUrl(),
          path: { id },
        });

        if (error) {
          reportRuntimeError("Failed to load delivery detail", error);
          return null;
        }

        return data || null;
      } catch (err) {
        reportRuntimeError("Error loading delivery detail", err);
        return null;
      } finally {
        setDetailLoading(false);
      }
    },
    [],
  );

  /**
   * Handle filter changes
   */
  const handleFilterChange = useCallback((newFilters: DeliveryFilters) => {
    setFilters(newFilters);
    // Reset to first page when filters change
    setOffset(0);
  }, []);

  /**
   * Handle pagination changes
   */
  const handlePageChange = useCallback((newOffset: number) => {
    setOffset(newOffset);
  }, []);

  /**
   * Handle viewing delivery detail
   */
  const handleViewDetail = useCallback(
    async (delivery: WebhookDeliveryRecord) => {
      // If we already have the full record with all fields, use it
      // Otherwise fetch the detail
      if (delivery.id) {
        const detail = await loadDeliveryDetail(delivery.id);
        if (detail) {
          setSelectedDelivery(detail);
          return;
        }
      }
      // Fallback to the list record
      setSelectedDelivery(delivery);
    },
    [loadDeliveryDetail],
  );

  /**
   * Handle closing detail view
   */
  const handleCloseDetail = useCallback(() => {
    setSelectedDelivery(null);
  }, []);

  // Load deliveries on mount and when filters/pagination change
  useEffect(() => {
    refreshDeliveries();
  }, [refreshDeliveries]);

  // Filter deliveries by status on the client side
  // (since the API may not support status filtering)
  const filteredDeliveries = useMemo(() => {
    if (filters.status === "all") {
      return deliveries;
    }
    return deliveries.filter((d) => d.status?.toLowerCase() === filters.status);
  }, [deliveries, filters.status]);

  return (
    <section id="webhook-deliveries">
      <Suspense
        fallback={
          <div className="loading-placeholder">
            Loading webhook delivery dashboard...
          </div>
        }
      >
        <WebhookDeliveries
          deliveries={filteredDeliveries}
          total={total}
          loading={loading}
          filters={filters}
          limit={limit}
          offset={offset}
          selectedDelivery={selectedDelivery}
          detailLoading={detailLoading}
          onRefresh={refreshDeliveries}
          onFilterChange={handleFilterChange}
          onPageChange={handlePageChange}
          onViewDetail={handleViewDetail}
          onCloseDetail={handleCloseDetail}
        />
      </Suspense>
    </section>
  );
}
