/**
 * Purpose: Centralize shared Settings authoring controller state for native and AI-backed editors.
 * Responsibilities: Load Settings inventories, manage AI modal state and draft handoff, coordinate tab-local workspace drafts, and provide shared mutation helpers for reload-backed saves and deletes.
 * Scope: Settings-route authoring control flow only; editor-specific form codecs, inventory row rendering, and API copy stay with each surface.
 * Usage: Called by Settings editors such as render profiles and pipeline JavaScript before rendering the shared Settings authoring shell.
 * Invariants/Assumptions: Each editor owns one local workspace draft session per tab, AI generator/debugger sessions persist through session storage, and inventory mutations downgrade success to a warning when the follow-up refresh fails.
 */

import { useCallback, useEffect, useState } from "react";

import type { ComponentStatus } from "../../api";
import {
  type AIAttempt,
  useAIAttemptHistory,
} from "../../hooks/useAIAttemptHistory";
import { useBeforeUnloadPrompt } from "../../hooks/useBeforeUnloadPrompt";
import { useSessionStorageState } from "../../hooks/useSessionStorageState";
import { getApiErrorMessage } from "../../lib/api-errors";
import { describeAICapability } from "../ai-assistant";
import { useToast } from "../toast";
import {
  type SettingsWorkspaceDraftMode,
  type SettingsWorkspaceDraftSession,
  useSettingsWorkspaceDraftController,
} from "./workspaceDrafts";

export type SettingsAuthoringAISessionSource = "generator" | "debugger";
export type SettingsAuthoringDraftSource =
  | "native"
  | SettingsAuthoringAISessionSource;

interface NamedSettingsRecord {
  name: string;
}

interface MutationToastCopy {
  title: string;
  description: string;
}

interface ConfirmActionOptions {
  title: string;
  description: string;
  confirmLabel: string;
  cancelLabel: string;
  tone: "warning" | "error";
}

interface UseSettingsAuthoringShellOptions<
  TItem extends NamedSettingsRecord,
  TInput,
  TDraft,
> {
  aiStatus?: ComponentStatus | null;
  aiFallbackMessage: string;
  loadErrorMessage: string;
  storageKeys: {
    generatorSession: string;
    debuggerSession: string;
    debuggerTarget: string;
    workspaceDraftSession: string;
  };
  loadInventory: () => Promise<TItem[]>;
  createEmptyInput: () => TInput;
  toInput: (item: TItem) => TInput;
  createDraft: (input: TInput) => TDraft;
  isDirty: (
    session: SettingsWorkspaceDraftSession<
      SettingsAuthoringDraftSource,
      TInput,
      TDraft
    >,
  ) => boolean;
  buildDebuggerReplacePrompt: (
    current: TItem,
    next: TItem,
  ) => Pick<ConfirmActionOptions, "title" | "description">;
  onError?: (error: string) => void;
  onInventoryChange?: (count: number) => void;
}

interface OpenNativeSettingsAuthoringDraftOptions<TItem> {
  mode: SettingsWorkspaceDraftMode;
  item?: TItem;
}

interface RunReloadingMutationOptions<TValue> {
  run: () => Promise<TValue>;
  loading: MutationToastCopy;
  success: MutationToastCopy | ((value: TValue) => MutationToastCopy);
  errorTitle: string;
  errorFallback: string;
}

export function useSettingsAuthoringShell<
  TItem extends NamedSettingsRecord,
  TInput,
  TDraft,
>({
  aiStatus = null,
  aiFallbackMessage,
  loadErrorMessage,
  storageKeys,
  loadInventory: loadInventoryItems,
  createEmptyInput,
  toInput,
  createDraft,
  isDirty,
  buildDebuggerReplacePrompt,
  onError,
  onInventoryChange,
}: UseSettingsAuthoringShellOptions<TItem, TInput, TDraft>) {
  const toast = useToast();
  const [items, setItems] = useState<TItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [debuggingItem, setDebuggingItem, clearDebuggingItem] =
    useSessionStorageState<TItem | null>(storageKeys.debuggerTarget, null);
  const [isAIGeneratorOpen, setIsAIGeneratorOpen] = useState(false);
  const [isAIDebuggerOpen, setIsAIDebuggerOpen] = useState(false);
  const [debuggerResetSignal, setDebuggerResetSignal] = useState(0);
  const [showJson, setShowJson] = useState(false);
  const generatorHistory = useAIAttemptHistory<TItem>({
    storageKey: `${storageKeys.generatorSession}.history`,
  });
  const debuggerHistory = useAIAttemptHistory<TItem>({
    storageKey: `${storageKeys.debuggerSession}.history`,
  });
  const {
    workspaceDraftSession,
    hiddenWorkspaceDraftSession,
    hasDirtySettingsDraft,
    resumeWorkspaceDraft,
    hideWorkspaceDraft,
    discardWorkspaceDraft,
    openNativeWorkspaceDraft,
    openAIWorkspaceDraft,
    returnToAISession,
    updateWorkspaceDraft,
    clearWorkspaceDraftSession,
  } = useSettingsWorkspaceDraftController<
    SettingsAuthoringDraftSource,
    TInput,
    TDraft
  >({
    storageKey: storageKeys.workspaceDraftSession,
    toast,
    clearTransientError: () => setError(null),
    isDirty,
    activateAISession: (source, attemptId) => {
      if (source === "generator") {
        generatorHistory.selectAttempt(attemptId);
        setIsAIDebuggerOpen(false);
        setIsAIGeneratorOpen(false);
        return;
      }

      debuggerHistory.selectAttempt(attemptId);
      setIsAIGeneratorOpen(false);
      setIsAIDebuggerOpen(false);
    },
    openAISession: (source) => {
      if (source === "generator") {
        setIsAIDebuggerOpen(false);
        setIsAIGeneratorOpen(true);
        return;
      }

      setIsAIGeneratorOpen(false);
      setIsAIDebuggerOpen(true);
    },
  });

  const aiCapability = describeAICapability(aiStatus, aiFallbackMessage);
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;

  useBeforeUnloadPrompt(hasDirtySettingsDraft);

  const loadInventory = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const nextItems = await loadInventoryItems();
      setItems(nextItems);
      onInventoryChange?.(nextItems.length);
      return null;
    } catch (err) {
      const message = getApiErrorMessage(err, loadErrorMessage);
      setError(message);
      onError?.(message);
      return message;
    } finally {
      setLoading(false);
    }
  }, [loadErrorMessage, loadInventoryItems, onError, onInventoryChange]);

  useEffect(() => {
    void loadInventory();
  }, [loadInventory]);

  const openNativeEditSession = useCallback(
    async ({ mode, item }: OpenNativeSettingsAuthoringDraftOptions<TItem>) => {
      const nextInitialValue =
        mode === "edit" && item ? toInput(item) : createEmptyInput();

      await openNativeWorkspaceDraft({
        mode,
        originalName: mode === "edit" ? (item?.name ?? null) : null,
        initialValue: nextInitialValue,
        draft: createDraft(nextInitialValue),
      });
    },
    [createDraft, createEmptyInput, openNativeWorkspaceDraft, toInput],
  );

  const openAttemptInSettings = useCallback(
    async (
      source: SettingsAuthoringAISessionSource,
      attempt: AIAttempt<TItem>,
    ) => {
      if (!attempt.artifact) {
        return;
      }

      const nextInitialValue = toInput(attempt.artifact);
      await openAIWorkspaceDraft({
        source,
        attemptId: attempt.id,
        mode: source === "generator" ? "create" : "edit",
        originalName:
          source === "debugger"
            ? (debuggingItem?.name ?? attempt.artifact.name)
            : null,
        initialValue: nextInitialValue,
        draft: createDraft(nextInitialValue),
      });
    },
    [createDraft, debuggingItem?.name, openAIWorkspaceDraft, toInput],
  );

  const handleDraftChange = useCallback(
    (draft: TDraft) => {
      updateWorkspaceDraft(draft);
    },
    [updateWorkspaceDraft],
  );

  const confirmAction = useCallback(
    (options: ConfirmActionOptions) => toast.confirm(options),
    [toast],
  );

  const openGenerator = useCallback(() => {
    hideWorkspaceDraft();
    setIsAIDebuggerOpen(false);
    setIsAIGeneratorOpen(true);
  }, [hideWorkspaceDraft]);

  const openDebugger = useCallback(
    async (item: TItem) => {
      if (debuggingItem && debuggingItem.name !== item.name) {
        const prompt = buildDebuggerReplacePrompt(debuggingItem, item);
        const confirmed = await confirmAction({
          title: prompt.title,
          description: prompt.description,
          confirmLabel: "Start new tuning session",
          cancelLabel: "Keep existing session",
          tone: "warning",
        });
        if (!confirmed) {
          return false;
        }
      }

      hideWorkspaceDraft();
      setIsAIGeneratorOpen(false);

      if (debuggingItem && debuggingItem.name !== item.name) {
        setDebuggerResetSignal((current) => current + 1);
      }

      setDebuggingItem(item);
      setIsAIDebuggerOpen(true);
      return true;
    },
    [
      buildDebuggerReplacePrompt,
      confirmAction,
      debuggingItem,
      hideWorkspaceDraft,
      setDebuggingItem,
    ],
  );

  const runReloadingMutation = useCallback(
    async <TValue>({
      run,
      loading: loadingCopy,
      success,
      errorTitle,
      errorFallback,
    }: RunReloadingMutationOptions<TValue>) => {
      const toastId = toast.show({
        tone: "loading",
        title: loadingCopy.title,
        description: loadingCopy.description,
      });

      try {
        setError(null);
        const value = await run();
        const refreshError = await loadInventory();
        const successCopy =
          typeof success === "function" ? success(value) : success;
        toast.update(toastId, {
          tone: refreshError ? "warning" : "success",
          title: successCopy.title,
          description: refreshError
            ? `${successCopy.description} Saved, but the latest inventory refresh failed: ${refreshError}`
            : successCopy.description,
        });
        return { ok: true as const, value, refreshError };
      } catch (err) {
        const message = getApiErrorMessage(err, errorFallback);
        setError(message);
        onError?.(message);
        toast.update(toastId, {
          tone: "error",
          title: errorTitle,
          description: message,
        });
        return { ok: false as const };
      }
    },
    [loadInventory, onError, toast],
  );

  return {
    items,
    loading,
    error,
    showJson,
    setShowJson,
    aiUnavailable,
    aiUnavailableMessage,
    generatorHistory,
    debuggerHistory,
    debuggingItem,
    setDebuggingItem,
    clearDebuggingItem,
    isAIGeneratorOpen,
    setIsAIGeneratorOpen,
    isAIDebuggerOpen,
    setIsAIDebuggerOpen,
    debuggerResetSignal,
    workspaceDraftSession,
    hiddenWorkspaceDraftSession,
    resumeWorkspaceDraft,
    hideWorkspaceDraft,
    discardWorkspaceDraft,
    returnToAISession,
    clearWorkspaceDraftSession,
    loadInventory,
    openNativeEditSession,
    openAttemptInSettings,
    handleDraftChange,
    confirmAction,
    openGenerator,
    openDebugger,
    runReloadingMutation,
  } as const;
}
