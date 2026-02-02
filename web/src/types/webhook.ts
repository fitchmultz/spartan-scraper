/**
 * Webhook Types Module
 *
 * Centralizes all webhook-related type definitions used across the
 * webhook delivery dashboard components.
 *
 * This module does NOT handle:
 * - API response types (those come from ../api)
 * - Runtime validation or type guards
 * - Business logic or state management
 *
 * @module types/webhook
 */

import type { WebhookDeliveryRecord } from "../api";

/**
 * Delivery status for filtering
 */
export type DeliveryStatus = "pending" | "delivered" | "failed" | "all";

/**
 * UI-facing delivery record (same as API type but re-exported for consistency)
 */
export type DeliveryRecord = WebhookDeliveryRecord;

/**
 * Filter state for webhook deliveries
 */
export interface DeliveryFilters {
  jobId: string;
  status: DeliveryStatus;
}

/**
 * Props for the WebhookDeliveryContainer component
 */
export type WebhookDeliveryContainerProps = Record<string, never>;

/**
 * Props for the WebhookDeliveries component (main presentation)
 */
export interface WebhookDeliveriesProps {
  deliveries: DeliveryRecord[];
  total: number;
  loading: boolean;
  filters: DeliveryFilters;
  limit: number;
  offset: number;
  selectedDelivery: DeliveryRecord | null;
  detailLoading: boolean;
  onRefresh: () => void;
  onFilterChange: (filters: DeliveryFilters) => void;
  onPageChange: (offset: number) => void;
  onViewDetail: (delivery: DeliveryRecord) => void;
  onCloseDetail: () => void;
}

/**
 * Props for the WebhookDeliveryList component
 */
export interface WebhookDeliveryListProps {
  deliveries: DeliveryRecord[];
  onViewDetail: (delivery: DeliveryRecord) => void;
}

/**
 * Props for the WebhookDeliveryDetail component
 */
export interface WebhookDeliveryDetailProps {
  delivery: DeliveryRecord;
  loading: boolean;
  onClose: () => void;
}

/**
 * Props for the WebhookDeliveryFilters component
 */
export interface WebhookDeliveryFiltersProps {
  filters: DeliveryFilters;
  onChange: (filters: DeliveryFilters) => void;
  onApply: () => void;
  onReset: () => void;
}
