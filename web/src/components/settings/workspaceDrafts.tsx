/**
 * Purpose: Centralize shared Settings authoring workspace-draft state, copy, and AI handoff behavior.
 * Responsibilities: Persist one tab-local Settings draft session, handle close/resume/discard/replace orchestration, and provide shared UI copy for hidden drafts and AI handoff context.
 * Scope: Browser-side Settings authoring workspace flow only; editor-specific form parsing, API persistence, and inventory rendering stay in the owning editor.
 * Usage: Mount from Settings editors that use one local workspace draft per authoring surface and need consistent native-vs-AI handoff behavior.
 * Invariants/Assumptions: Each editor owns at most one workspace draft session per tab, Close is non-destructive, discard always requires explicit confirmation, and AI handoff drafts return to the originating modal when requested.
 */

import { useCallback } from "react";
import { deepEqual } from "../../lib/diff-utils";
import { useSessionStorageState } from "../../hooks/useSessionStorageState";
import type { ToastController } from "../toast";

export type SettingsWorkspaceDraftMode = "create" | "edit";

export interface SettingsWorkspaceDraftSession<
  TSource extends string,
  TInitialValue,
  TDraft,
> {
  source: TSource;
  attemptId: string | null;
  mode: SettingsWorkspaceDraftMode;
  originalName: string | null;
  initialValue: TInitialValue;
  draft: TDraft;
  visible: boolean;
}

interface SettingsWorkspaceDraftControllerOptions<
  TSource extends string,
  TInitialValue,
  TDraft,
> {
  storageKey: string;
  toast: Pick<ToastController, "confirm">;
  clearTransientError: () => void;
  isDirty: (
    session: SettingsWorkspaceDraftSession<TSource, TInitialValue, TDraft>,
  ) => boolean;
  activateAISession: (source: TSource, attemptId: string) => void;
  openAISession: (source: TSource) => void;
}

interface OpenNativeSettingsWorkspaceDraftOptions<TInitialValue, TDraft> {
  mode: SettingsWorkspaceDraftMode;
  originalName: string | null;
  initialValue: TInitialValue;
  draft: TDraft;
}

interface OpenAISettingsWorkspaceDraftOptions<
  TSource extends string,
  TInitialValue,
  TDraft,
> extends OpenNativeSettingsWorkspaceDraftOptions<TInitialValue, TDraft> {
  source: TSource;
  attemptId: string;
}

interface ConfirmDiscardOptions {
  reason?: string;
  title?: string;
}

interface HiddenSettingsDraftNoticeCopy {
  title: string;
  description: string;
  resumeLabel: string;
  discardLabel: string;
}

function buildDiscardCopy(
  source: string,
  isDirty: boolean,
  options?: ConfirmDiscardOptions,
): Required<ConfirmDiscardOptions> {
  const isAIHandoffDraft = source !== "native";

  return {
    title:
      options?.title ??
      (isAIHandoffDraft
        ? "Discard the AI handoff draft?"
        : "Discard the local Settings draft?"),
    reason:
      options?.reason ??
      (isAIHandoffDraft
        ? isDirty
          ? "This removes the local Settings draft for the current AI attempt. Your unsaved edits will be lost."
          : "This removes the current AI handoff draft from Settings. You can still reopen the AI modal itself if you keep that session."
        : isDirty
          ? "This removes the in-progress local Settings draft. Your unsaved edits will be lost."
          : "This removes the current local Settings draft from this tab."),
  };
}

function buildReplaceCopy(
  currentSource: string,
  nextDraftKind: "native" | "ai",
): Required<ConfirmDiscardOptions> {
  return {
    title:
      currentSource === "native"
        ? "Replace the current Settings draft?"
        : "Replace the current AI handoff draft?",
    reason:
      nextDraftKind === "native"
        ? "This opens another local Settings draft and discards the edits you have not saved yet. Keep the current draft if you still need it."
        : currentSource === "native"
          ? "This opens an AI handoff draft and discards the local Settings edits you have not saved yet. Keep the current draft if you still need it."
          : "This attempt will replace the current Settings draft for another AI handoff. Discard the older draft only if you no longer need it.",
  };
}

export function describeHiddenSettingsDraft(
  session: Pick<
    SettingsWorkspaceDraftSession<string, unknown, unknown>,
    "attemptId" | "originalName" | "source"
  >,
  options: {
    isDirty: boolean;
    nativeDraftLabel: string;
  },
): HiddenSettingsDraftNoticeCopy {
  if (session.source === "native") {
    return {
      title: `Local Settings draft for ${session.originalName ?? options.nativeDraftLabel}${
        options.isDirty
          ? " has unsaved edits."
          : " is still available in this tab."
      }`,
      description:
        "Close keeps this draft available in the current tab. Resume it when you want to continue editing, or discard it explicitly once you no longer need it.",
      resumeLabel: "Resume Settings draft",
      discardLabel: "Discard Settings draft",
    };
  }

  return {
    title: `AI handoff draft for Attempt ${session.attemptId?.replace("attempt-", "")}${
      options.isDirty
        ? " has unsaved Settings edits."
        : " is still available in Settings."
    }`,
    description:
      "Close keeps this draft available in the current tab. Resume it when you want to keep editing the local handoff draft, or discard it explicitly once you no longer need it.",
    resumeLabel: "Resume AI handoff draft",
    discardLabel: "Discard handoff draft",
  };
}

export function SettingsAIHandoffContextNotice({
  attemptId,
  submitLabel,
}: {
  attemptId: string;
  submitLabel: string;
}) {
  const attemptOrdinal = attemptId.replace("attempt-", "");

  return (
    <div className="space-y-2">
      <p>You are editing Attempt {attemptOrdinal} from the AI session.</p>
      <p>
        Back to AI session returns to the modal with this working candidate
        preserved locally as-is, even if it is unsaved.
      </p>
      <p>
        {submitLabel} saves to the API, updates the AI attempt, and then returns
        to the modal.
      </p>
      <p>
        Unsaved edits are preserved locally, but they are not reflected in the
        AI attempt history until you save.
      </p>
    </div>
  );
}

export function useSettingsWorkspaceDraftController<
  TSource extends string,
  TInitialValue,
  TDraft,
>({
  storageKey,
  toast,
  clearTransientError,
  isDirty,
  activateAISession,
  openAISession,
}: SettingsWorkspaceDraftControllerOptions<TSource, TInitialValue, TDraft>) {
  const [workspaceDraftSession, setWorkspaceDraftSession, clearWorkspaceDraft] =
    useSessionStorageState<SettingsWorkspaceDraftSession<
      TSource,
      TInitialValue,
      TDraft
    > | null>(storageKey, null);

  const hiddenWorkspaceDraftSession =
    workspaceDraftSession && !workspaceDraftSession.visible
      ? workspaceDraftSession
      : null;
  const hasDirtySettingsDraft = workspaceDraftSession
    ? isDirty(workspaceDraftSession)
    : false;

  const resumeWorkspaceDraft = useCallback(() => {
    setWorkspaceDraftSession((current) =>
      current ? { ...current, visible: true } : current,
    );
  }, [setWorkspaceDraftSession]);

  const hideWorkspaceDraft = useCallback(() => {
    setWorkspaceDraftSession((current) =>
      current ? { ...current, visible: false } : current,
    );
  }, [setWorkspaceDraftSession]);

  const discardWorkspaceDraft = useCallback(
    async (options?: ConfirmDiscardOptions) => {
      if (!workspaceDraftSession) {
        return true;
      }

      const copy = buildDiscardCopy(
        workspaceDraftSession.source,
        isDirty(workspaceDraftSession),
        options,
      );
      const confirmed = await toast.confirm({
        title: copy.title,
        description: copy.reason,
        confirmLabel: "Discard draft",
        cancelLabel: "Keep draft",
        tone: "warning",
      });
      if (!confirmed) {
        return false;
      }

      clearWorkspaceDraft();
      return true;
    },
    [clearWorkspaceDraft, isDirty, toast, workspaceDraftSession],
  );

  const openNativeWorkspaceDraft = useCallback(
    async ({
      mode,
      originalName,
      initialValue,
      draft,
    }: OpenNativeSettingsWorkspaceDraftOptions<TInitialValue, TDraft>) => {
      if (
        workspaceDraftSession?.source === "native" &&
        workspaceDraftSession.mode === mode &&
        workspaceDraftSession.originalName === originalName
      ) {
        resumeWorkspaceDraft();
        return true;
      }

      if (
        workspaceDraftSession &&
        isDirty(workspaceDraftSession) &&
        !(await discardWorkspaceDraft(
          buildReplaceCopy(workspaceDraftSession.source, "native"),
        ))
      ) {
        return false;
      }

      clearTransientError();
      setWorkspaceDraftSession({
        source: "native" as TSource,
        attemptId: null,
        mode,
        originalName,
        initialValue,
        draft,
        visible: true,
      });
      return true;
    },
    [
      clearTransientError,
      discardWorkspaceDraft,
      isDirty,
      resumeWorkspaceDraft,
      setWorkspaceDraftSession,
      workspaceDraftSession,
    ],
  );

  const openAIWorkspaceDraft = useCallback(
    async ({
      source,
      attemptId,
      mode,
      originalName,
      initialValue,
      draft,
    }: OpenAISettingsWorkspaceDraftOptions<TSource, TInitialValue, TDraft>) => {
      if (
        workspaceDraftSession &&
        (workspaceDraftSession.source !== source ||
          workspaceDraftSession.attemptId !== attemptId) &&
        isDirty(workspaceDraftSession) &&
        !(await discardWorkspaceDraft(
          buildReplaceCopy(workspaceDraftSession.source, "ai"),
        ))
      ) {
        return false;
      }

      clearTransientError();
      activateAISession(source, attemptId);
      setWorkspaceDraftSession((current) => {
        if (
          current &&
          current.source === source &&
          current.attemptId === attemptId
        ) {
          return {
            ...current,
            originalName,
            visible: true,
          };
        }

        return {
          source,
          attemptId,
          mode,
          originalName,
          initialValue,
          draft,
          visible: true,
        };
      });
      return true;
    },
    [
      activateAISession,
      clearTransientError,
      discardWorkspaceDraft,
      isDirty,
      setWorkspaceDraftSession,
      workspaceDraftSession,
    ],
  );

  const returnToAISession = useCallback(
    (source: TSource, options?: { preserveDraft?: boolean }) => {
      setWorkspaceDraftSession((current) => {
        if (!current || current.source !== source) {
          return current;
        }

        return options?.preserveDraft === false
          ? null
          : { ...current, visible: false };
      });
      openAISession(source);
    },
    [openAISession, setWorkspaceDraftSession],
  );

  const updateWorkspaceDraft = useCallback(
    (draft: TDraft) => {
      setWorkspaceDraftSession((current) => {
        if (!current || deepEqual(current.draft, draft)) {
          return current;
        }

        return { ...current, draft };
      });
    },
    [setWorkspaceDraftSession],
  );

  return {
    workspaceDraftSession,
    setWorkspaceDraftSession,
    clearWorkspaceDraftSession: clearWorkspaceDraft,
    hiddenWorkspaceDraftSession,
    hasDirtySettingsDraft,
    resumeWorkspaceDraft,
    hideWorkspaceDraft,
    discardWorkspaceDraft,
    openNativeWorkspaceDraft,
    openAIWorkspaceDraft,
    returnToAISession,
    updateWorkspaceDraft,
  } as const;
}
