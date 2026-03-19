/**
 * Purpose: Centralize watch-management UI types and component props.
 * Responsibilities: Define form state, list row actions, manual-check modal props, and persisted history inspection props.
 * Scope: Web watch-management typing only; network contracts come from `../api`.
 * Usage: Import from watch containers, managers, and modal components to keep prop contracts aligned.
 * Invariants/Assumptions: API-generated watch contracts remain the source of truth for persisted entities, and UI-only state lives in these prop helpers rather than the generated client.
 */

import type {
  Watch,
  WatchCheckHistoryResponse,
  WatchCheckInspection,
  WatchCheckResult,
  WatchInput,
} from "../api";

export interface WatchManagerProps {
  watches: Watch[];
  onRefresh: () => void;
  onCreate: (watch: WatchInput) => Promise<void>;
  onUpdate: (id: string, watch: WatchInput) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onCheck: (id: string) => Promise<WatchCheckResult | undefined>;
  onLoadHistory: (
    watchId: string,
    limit: number,
    offset: number,
  ) => Promise<WatchCheckHistoryResponse | undefined>;
  onLoadHistoryDetail: (
    watchId: string,
    checkId: string,
  ) => Promise<WatchCheckInspection | undefined>;
  loading?: boolean;
}

export interface WatchFormData {
  url: string;
  selector: string;
  intervalSeconds: number;
  enabled: boolean;
  diffFormat: "unified" | "html-side-by-side" | "html-inline";
  notifyOnChange: boolean;
  webhookUrl: string;
  webhookSecret: string;
  headless: boolean;
  usePlaywright: boolean;
  extractMode: "" | "text" | "html" | "markdown";
  minChangeSize: string;
  ignorePatterns: string;
  screenshotEnabled: boolean;
  screenshotFullPage: boolean;
  screenshotFormat: "png" | "jpeg";
  visualDiffThreshold: string;
  jobTriggerKind: "" | "scrape" | "crawl" | "research";
  jobTriggerRequest: string;
}

export interface WatchListProps {
  watches: Watch[];
  checkingId: string | null;
  historyLoadingId: string | null;
  deleteConfirmId: string | null;
  onEdit: (watch: Watch) => void;
  onDelete: (id: string) => void;
  onCheck: (watch: Watch) => void;
  onHistory: (watch: Watch) => void;
  onDeleteConfirm: (id: string | null) => void;
}

export interface WatchListItemProps {
  watch: Watch;
  isChecking: boolean;
  isHistoryLoading: boolean;
  isDeleting: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onCheck: () => void;
  onHistory: () => void;
  onDeleteConfirm: () => void;
}

export interface WatchFormProps {
  formData: WatchFormData;
  formError: string | null;
  formSubmitting: boolean;
  isEditing: boolean;
  onChange: (data: Partial<WatchFormData>) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

export interface CheckResultModalProps {
  result: WatchCheckResult;
  inspection: WatchCheckInspection | null;
  onClose: () => void;
  onOpenHistory: (checkId: string) => void;
}

export interface WatchHistoryModalProps {
  watch: Watch;
  records: WatchCheckInspection[];
  total: number;
  limit: number;
  offset: number;
  loading: boolean;
  selectedCheck: WatchCheckInspection | null;
  selectedCheckLoading: boolean;
  onClose: () => void;
  onSelectCheck: (checkId: string) => void;
  onPageChange: (offset: number) => void;
}
