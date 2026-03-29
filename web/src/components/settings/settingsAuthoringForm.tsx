/**
 * Purpose: Share Settings authoring field codecs and draft-form chrome across browser-runtime editors.
 * Responsibilities: Normalize comma-list, optional number, and optional JSON field codecs; compute draft sync state; and render the shared draft status, context notice, validation error, and action row chrome.
 * Scope: Settings authoring helpers only; editor-specific field layouts and API payload schemas stay with each authoring form.
 * Usage: Imported by render-profile and pipeline-script forms to keep browser-runtime authoring behavior aligned.
 * Invariants/Assumptions: Invalid codec parsing should mark the draft dirty, blank optional fields resolve to undefined, and shared form chrome preserves the existing Settings editor semantics.
 */

import type { FormEvent, ReactNode } from "react";

import { deepEqual } from "../../lib/diff-utils";

export type SettingsDraftSyncState = "clean" | "dirty" | null;

export function formatCommaSeparatedList(values?: string[] | null): string {
  return values?.join(", ") ?? "";
}

export function parseCommaSeparatedList(value: string): string[] {
  return value
    .split(",")
    .map((part) => part.trim())
    .filter(Boolean);
}

export function formatOptionalJSON(value: unknown): string {
  if (value === undefined || value === null) {
    return "";
  }

  return JSON.stringify(value, null, 2);
}

export function parseOptionalJSON<T>(
  label: string,
  value: string,
): T | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }

  try {
    return JSON.parse(trimmed) as T;
  } catch (error) {
    throw new Error(
      `${label} must be valid JSON${
        error instanceof Error && error.message ? `: ${error.message}` : ""
      }`,
    );
  }
}

export function parseOptionalJSONObject<T extends object>(
  label: string,
  value: string,
): T | undefined {
  const parsed = parseOptionalJSON<unknown>(label, value);
  if (parsed === undefined) {
    return undefined;
  }
  if (parsed === null || Array.isArray(parsed) || typeof parsed !== "object") {
    throw new Error(`${label} must be a JSON object`);
  }

  return parsed as T;
}

export function parseOptionalNumber(value: string): number | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }

  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : undefined;
}

export function getSettingsDraftSyncState<TDraft, TValue>(options: {
  draft: TDraft;
  initialValue: TValue;
  savedValue?: TValue;
  buildValue: (draft: TDraft) => TValue;
}): SettingsDraftSyncState {
  const { draft, initialValue, savedValue, buildValue } = options;
  const hasSavedValue = savedValue !== undefined;
  const baselineValue = hasSavedValue ? savedValue : initialValue;

  try {
    return deepEqual(buildValue(draft), baselineValue)
      ? hasSavedValue
        ? "clean"
        : null
      : "dirty";
  } catch {
    return "dirty";
  }
}

interface SettingsDraftFormProps {
  title: string;
  syncState: SettingsDraftSyncState;
  contextNotice?: ReactNode;
  error?: string | null;
  cancelLabel: string;
  discardLabel?: string;
  submitLabel: string;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onCancel: () => void;
  onDiscard?: () => void;
  children: ReactNode;
}

export function SettingsDraftForm({
  title,
  syncState,
  contextNotice,
  error,
  cancelLabel,
  discardLabel,
  submitLabel,
  onSubmit,
  onCancel,
  onDiscard,
  children,
}: SettingsDraftFormProps) {
  return (
    <form
      onSubmit={onSubmit}
      className="space-y-4 rounded border bg-gray-50 p-4"
    >
      <h3 className="font-medium">{title}</h3>

      {syncState ? (
        <div
          role="status"
          aria-live="polite"
          className={`rounded-md border px-3 py-2 text-sm ${
            syncState === "dirty"
              ? "border-amber-300 bg-amber-50 text-amber-900"
              : "border-emerald-300 bg-emerald-50 text-emerald-900"
          }`}
        >
          {syncState === "dirty" ? "Unsaved changes" : "In sync with saved"}
        </div>
      ) : null}

      {contextNotice ? (
        <div className="rounded-md border border-purple-200 bg-purple-50 p-3 text-sm text-purple-900">
          {contextNotice}
        </div>
      ) : null}

      {error ? (
        <div className="error" role="alert">
          {error}
        </div>
      ) : null}

      {children}

      <div className="flex justify-end space-x-2">
        {onDiscard ? (
          <button
            type="button"
            onClick={onDiscard}
            className="rounded border px-4 py-2 hover:bg-gray-100"
          >
            {discardLabel}
          </button>
        ) : null}
        <button
          type="button"
          onClick={onCancel}
          className="rounded border px-4 py-2 hover:bg-gray-100"
        >
          {cancelLabel}
        </button>
        <button
          type="submit"
          className="rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
        >
          {submitLabel}
        </button>
      </div>
    </form>
  );
}
