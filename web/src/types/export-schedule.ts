/**
 * Export Schedule Types Module
 *
 * Centralizes all export schedule-related type definitions used across the
 * export schedule management components and hooks.
 *
 * This module does NOT handle:
 * - API response types (those come from ../api)
 * - Runtime validation or type guards
 * - Business logic or state management
 *
 * @module types/export-schedule
 */

import type {
  ExportSchedule,
  ExportScheduleRequest,
  ExportHistoryRecord,
} from "../api";

/**
 * Props for the ExportScheduleManager component
 */
export interface ExportScheduleManagerProps {
  schedules: ExportSchedule[];
  onRefresh: () => void;
  onCreate: (request: ExportScheduleRequest) => Promise<void>;
  onUpdate: (id: string, request: ExportScheduleRequest) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onToggleEnabled: (id: string, enabled: boolean) => Promise<void>;
  onGetHistory: (
    id: string,
    limit?: number,
    offset?: number,
  ) => Promise<{
    records: ExportHistoryRecord[];
    total: number;
  }>;
  loading?: boolean;
}

/**
 * Form data structure for export schedule creation/editing
 * Uses string types for numeric inputs to allow empty values during editing
 */
export interface ExportScheduleFormData {
  name: string;
  enabled: boolean;
  // Filters
  filterJobKinds: Array<"scrape" | "crawl" | "research">;
  filterJobStatus: Array<"completed" | "failed" | "succeeded" | "canceled">;
  filterTags: string; // newline-separated for textarea
  filterHasResults: boolean;
  // Export
  format: "json" | "jsonl" | "md" | "csv" | "xlsx";
  destinationType: "local" | "webhook";
  pathTemplate: string;
  // Local config (conditional)
  localPath: string;
  // Webhook config (conditional)
  webhookUrl: string;
  // Retry
  maxRetries: number;
  baseDelayMs: number;
}

/**
 * Props for the ExportScheduleList component
 */
export interface ExportScheduleListProps {
  schedules: ExportSchedule[];
  deleteConfirmId: string | null;
  onEdit: (schedule: ExportSchedule) => void;
  onDelete: (id: string) => void;
  onToggleEnabled: (id: string, enabled: boolean) => void;
  onViewHistory: (schedule: ExportSchedule) => void;
  onDeleteConfirm: (id: string | null) => void;
}

/**
 * Props for the ExportScheduleListItem component
 */
export interface ExportScheduleListItemProps {
  schedule: ExportSchedule;
  isDeleting: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  onViewHistory: () => void;
  onDeleteConfirm: () => void;
}

/**
 * Props for the ExportScheduleForm component
 */
export interface ExportScheduleFormProps {
  formData: ExportScheduleFormData;
  formError: string | null;
  formSubmitting: boolean;
  isEditing: boolean;
  onChange: (data: Partial<ExportScheduleFormData>) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

/**
 * Props for the ExportScheduleHistory component
 */
export interface ExportScheduleHistoryProps {
  scheduleName: string;
  records: ExportHistoryRecord[];
  total: number;
  limit: number;
  offset: number;
  loading: boolean;
  onClose: () => void;
  onPageChange: (offset: number) => void;
}
