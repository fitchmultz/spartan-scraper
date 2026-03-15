/**
 * Watch Types Module
 *
 * Centralizes all watch-related type definitions used across the watch management
 * components and hooks. This module provides a single source of truth for watch
 * form data structures and component prop types.
 *
 * This module does NOT handle:
 * - API response types (those come from ../api)
 * - Runtime validation or type guards
 * - Business logic or state management
 *
 * @module types/watch
 */

import type { Watch, WatchInput, WatchCheckResult } from "../api";

/**
 * Props for the WatchManager component
 */
export interface WatchManagerProps {
  watches: Watch[];
  onRefresh: () => void;
  onCreate: (watch: WatchInput) => Promise<void>;
  onUpdate: (id: string, watch: WatchInput) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onCheck: (id: string) => Promise<WatchCheckResult | undefined>;
  loading?: boolean;
}

/**
 * Form data structure for watch creation/editing
 * Uses string types for numeric inputs to allow empty values during editing
 */
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

/**
 * Props for the WatchList component
 */
export interface WatchListProps {
  watches: Watch[];
  checkingId: string | null;
  deleteConfirmId: string | null;
  onEdit: (watch: Watch) => void;
  onDelete: (id: string) => void;
  onCheck: (watch: Watch) => void;
  onDeleteConfirm: (id: string | null) => void;
}

/**
 * Props for the WatchListItem component
 */
export interface WatchListItemProps {
  watch: Watch;
  isChecking: boolean;
  isDeleting: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onCheck: () => void;
  onDeleteConfirm: () => void;
}

/**
 * Props for the WatchForm component
 */
export interface WatchFormProps {
  formData: WatchFormData;
  formError: string | null;
  formSubmitting: boolean;
  isEditing: boolean;
  onChange: (data: Partial<WatchFormData>) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

/**
 * Props for the CheckResultModal component
 */
export interface CheckResultModalProps {
  result: WatchCheckResult;
  onClose: () => void;
}
