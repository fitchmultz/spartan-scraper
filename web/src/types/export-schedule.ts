/**
 * Purpose: Define shared type contracts for export schedule.
 * Responsibilities: Export reusable TypeScript types and aliases that keep the surrounding feature consistent.
 * Scope: Type-level contracts only; runtime logic stays in implementation modules.
 * Usage: Import these types from adjacent feature, route, and test modules.
 * Invariants/Assumptions: The exported types should reflect the current source-of-truth contracts without introducing runtime side effects.
 */

import type {
  ComponentStatus,
  ExportInspection,
  ExportOutcomeListResponse,
  ExportSchedule,
  ExportScheduleRequest,
} from "../api";
import type { ExportSchedulePromotionSeed } from "./promotion";

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
  ) => Promise<ExportOutcomeListResponse>;
  loading?: boolean;
  aiStatus?: ComponentStatus | null;
  promotionSeed?: ExportSchedulePromotionSeed | null;
  onClearPromotionSeed?: () => void;
  onOpenSourceJob?: (jobId: string) => void;
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
  // Optional result transform
  transformExpression: string;
  transformLanguage: "jmespath" | "jsonata";
  // Optional export shaping
  shapeTopLevelFields: string;
  shapeNormalizedFields: string;
  shapeEvidenceFields: string;
  shapeSummaryFields: string;
  shapeFieldLabels: string;
  shapeEmptyValue: string;
  shapeMultiValueJoin: string;
  shapeMarkdownTitle: string;
}

/**
 * Props for the ExportScheduleList component
 */
export interface ExportScheduleListProps {
  schedules: ExportSchedule[];
  historyLoadingId: string | null;
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
  isHistoryLoading: boolean;
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
  aiStatus?: ComponentStatus | null;
  promotionSeed?: ExportSchedulePromotionSeed | null;
  onClearPromotionSeed?: () => void;
  onOpenSourceJob?: (jobId: string) => void;
}

/**
 * Props for the ExportScheduleHistory component
 */
export interface ExportScheduleHistoryProps {
  scheduleName: string;
  records: ExportInspection[];
  total: number;
  limit: number;
  offset: number;
  loading: boolean;
  onClose: () => void;
  onPageChange: (offset: number) => void;
}
