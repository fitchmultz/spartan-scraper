/**
 * WebhookDeliveryContainer - Container component for webhook delivery history
 *
 * This component encapsulates all webhook delivery-related state and operations:
 * - Loading and displaying webhook delivery records
 * - Filtering by job ID and status
 * - Pagination (limit/offset)
 * - Viewing delivery details
 *
 * It does NOT handle:
 * - Job submission or results viewing
 * - Webhook configuration (see WebhookConfig component)
 * - Retry/resend functionality (future enhancement)
 *
 * @module WebhookDeliveryContainer
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
        console.error("Failed to load webhook deliveries:", error);
        return;
      }

      setDeliveries(data?.deliveries || []);
      setTotal(data?.total || 0);
    } catch (err) {
      console.error("Error loading webhook deliveries:", err);
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
          console.error("Failed to load delivery detail:", error);
          return null;
        }

        return data || null;
      } catch (err) {
        console.error("Error loading delivery detail:", err);
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
