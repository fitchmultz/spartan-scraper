/**
 * Purpose: Provide the Settings-route editor for stored pipeline JavaScript configurations.
 * Responsibilities: Load the script inventory, coordinate create/edit/delete flows, preserve AI authoring sessions across Settings handoff, and surface operator feedback through inline state and toasts.
 * Scope: Browser-side pipeline-script management only; runtime execution and matching logic stay on the backend.
 * Usage: Render inside the Settings route without additional providers beyond the app-level toast boundary.
 * Invariants/Assumptions: Script persistence goes through the generated API client, manual AI handoff returns to the same in-session history, and destructive actions use the shared confirmation dialog instead of browser-native prompts.
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  deleteV1PipelineJsByName,
  getV1PipelineJs,
  postV1PipelineJs,
  putV1PipelineJsByName,
  type ComponentStatus,
  type JsTargetScript,
  type PipelineJsInput,
} from "../../api";
import {
  type AIAttempt,
  useAIAttemptHistory,
} from "../../hooks/useAIAttemptHistory";
import { useBeforeUnloadPrompt } from "../../hooks/useBeforeUnloadPrompt";
import { useSessionStorageState } from "../../hooks/useSessionStorageState";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { deepEqual } from "../../lib/diff-utils";
import { AIPipelineJSDebugger } from "../AIPipelineJSDebugger";
import { AIPipelineJSGenerator } from "../AIPipelineJSGenerator";
import { ActionEmptyState } from "../ActionEmptyState";
import { AIUnavailableNotice, describeAICapability } from "../ai-assistant";
import { ResumableSettingsDraftNotice } from "../settings/ResumableSettingsDraftNotice";
import { useToast } from "../toast";

type AISessionSource = "generator" | "debugger";
type ScriptDraftSessionSource = "native" | AISessionSource;

const PIPELINE_JS_GENERATOR_SESSION_KEY =
  "spartan.pipeline-js.ai-generator-session";
const PIPELINE_JS_DEBUGGER_SESSION_KEY =
  "spartan.pipeline-js.ai-debugger-session";
const PIPELINE_JS_DEBUGGER_TARGET_KEY =
  "spartan.pipeline-js.ai-debugger-target";
const PIPELINE_JS_WORKSPACE_DRAFT_SESSION_KEY =
  "spartan.pipeline-js.workspace-draft-session";

interface ScriptFormDraft {
  formData: PipelineJsInput;
  hostPatternInput: string;
  selectorInput: string;
}

interface ScriptWorkspaceDraftSession {
  source: ScriptDraftSessionSource;
  attemptId: string | null;
  mode: "create" | "edit";
  originalName: string | null;
  initialValue: PipelineJsInput;
  draft: ScriptFormDraft;
  visible: boolean;
}

interface PipelineJSEditorProps {
  onError?: (error: string) => void;
  aiStatus?: ComponentStatus | null;
  onInventoryChange?: (count: number) => void;
}

function createEmptyPipelineJsInput(): PipelineJsInput {
  return {
    name: "",
    hostPatterns: [],
    engine: undefined,
    preNav: undefined,
    postNav: undefined,
    selectors: undefined,
  };
}

function toPipelineJsInput(script: JsTargetScript): PipelineJsInput {
  return {
    name: script.name,
    hostPatterns: [...script.hostPatterns],
    engine: script.engine,
    preNav: script.preNav,
    postNav: script.postNav,
    selectors: script.selectors ? [...script.selectors] : undefined,
  };
}

function createScriptFormDraft(seed: PipelineJsInput): ScriptFormDraft {
  return {
    formData: seed,
    hostPatternInput: seed.hostPatterns.join(", "),
    selectorInput: seed.selectors?.join(", ") || "",
  };
}

function buildPipelineJsInputFromDraft(
  draft: ScriptFormDraft,
): PipelineJsInput {
  const hostPatterns = draft.hostPatternInput
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean);
  const selectors = draft.selectorInput
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean);

  return {
    ...draft.formData,
    engine: draft.formData.engine || undefined,
    hostPatterns,
    selectors: selectors.length > 0 ? selectors : undefined,
  };
}

function isScriptDraftDirty(
  draft: ScriptFormDraft,
  initialValue: PipelineJsInput,
): boolean {
  return !deepEqual(buildPipelineJsInputFromDraft(draft), initialValue);
}

function isScriptWorkspaceDraftSessionDirty(
  session: ScriptWorkspaceDraftSession,
): boolean {
  return isScriptDraftDirty(session.draft, session.initialValue);
}

function ManualEditContextNotice({
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

export function PipelineJSEditor({
  onError,
  aiStatus = null,
  onInventoryChange,
}: PipelineJSEditorProps) {
  const toast = useToast();
  const [scripts, setScripts] = useState<JsTargetScript[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [
    workspaceDraftSession,
    setWorkspaceDraftSession,
    clearWorkspaceDraftSession,
  ] = useSessionStorageState<ScriptWorkspaceDraftSession | null>(
    PIPELINE_JS_WORKSPACE_DRAFT_SESSION_KEY,
    null,
  );
  const [debuggingScript, setDebuggingScript, clearDebuggingScript] =
    useSessionStorageState<JsTargetScript | null>(
      PIPELINE_JS_DEBUGGER_TARGET_KEY,
      null,
    );
  const [isAIGeneratorOpen, setIsAIGeneratorOpen] = useState(false);
  const [isAIDebuggerOpen, setIsAIDebuggerOpen] = useState(false);
  const [debuggerResetSignal, setDebuggerResetSignal] = useState(0);
  const [showJson, setShowJson] = useState(false);
  const generatorHistory = useAIAttemptHistory<JsTargetScript>({
    storageKey: `${PIPELINE_JS_GENERATOR_SESSION_KEY}.history`,
  });
  const debuggerHistory = useAIAttemptHistory<JsTargetScript>({
    storageKey: `${PIPELINE_JS_DEBUGGER_SESSION_KEY}.history`,
  });

  const aiCapability = describeAICapability(
    aiStatus,
    "Create and edit scripts manually below.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;
  const hiddenWorkspaceDraftSession =
    workspaceDraftSession && !workspaceDraftSession.visible
      ? workspaceDraftSession
      : null;
  const hasDirtySettingsDraft = workspaceDraftSession
    ? isScriptWorkspaceDraftSessionDirty(workspaceDraftSession)
    : false;

  useBeforeUnloadPrompt(hasDirtySettingsDraft);

  const loadScripts = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await getV1PipelineJs({
        baseUrl: getApiBaseUrl(),
      });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to load scripts"),
        );
      }
      const nextScripts = response.data?.scripts || [];
      setScripts(nextScripts);
      onInventoryChange?.(nextScripts.length);
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to load scripts");
      setError(message);
      onError?.(message);
    } finally {
      setLoading(false);
    }
  }, [onError, onInventoryChange]);

  useEffect(() => {
    loadScripts();
  }, [loadScripts]);

  const hideWorkspaceDraft = useCallback(() => {
    setWorkspaceDraftSession((current) =>
      current ? { ...current, visible: false } : current,
    );
  }, [setWorkspaceDraftSession]);

  const handleSaveNativeEditSession = async (
    session: ScriptWorkspaceDraftSession,
    input: PipelineJsInput,
  ) => {
    if (session.source !== "native") {
      return;
    }

    const isCreate = session.mode === "create";
    const actionLabel = isCreate
      ? input.name
        ? `Creating ${input.name}`
        : "Creating script"
      : `Updating ${session.originalName ?? input.name}`;
    const toastId = toast.show({
      tone: "loading",
      title: actionLabel,
      description: isCreate
        ? "Saving the new pipeline JavaScript configuration."
        : "Saving the latest pipeline JavaScript changes.",
    });

    try {
      setError(null);
      const response = isCreate
        ? await postV1PipelineJs({
            baseUrl: getApiBaseUrl(),
            body: input,
          })
        : await putV1PipelineJsByName({
            baseUrl: getApiBaseUrl(),
            path: { name: session.originalName ?? input.name },
            body: input,
          });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(
            response.error,
            isCreate ? "Failed to create script" : "Failed to update script",
          ),
        );
      }
      await loadScripts();
      clearWorkspaceDraftSession();
      toast.update(toastId, {
        tone: "success",
        title: isCreate ? "Script created" : "Script updated",
        description: isCreate
          ? `${input.name} is ready for pipeline matching.`
          : `${session.originalName ?? input.name} now reflects the latest configuration.`,
      });
    } catch (err) {
      const message = getApiErrorMessage(
        err,
        isCreate ? "Failed to create script" : "Failed to update script",
      );
      setError(message);
      onError?.(message);
      toast.update(toastId, {
        tone: "error",
        title: isCreate ? "Failed to create script" : "Failed to update script",
        description: message,
      });
    }
  };

  const handleDelete = async (name: string) => {
    const confirmed = await toast.confirm({
      title: `Delete ${name}?`,
      description:
        "This removes the saved script configuration from local storage. Matching pages will stop using it immediately.",
      confirmLabel: "Delete script",
      cancelLabel: "Keep script",
      tone: "error",
    });
    if (!confirmed) return;

    const toastId = toast.show({
      tone: "loading",
      title: `Deleting ${name}`,
      description: "Removing the saved pipeline script.",
    });
    try {
      setError(null);
      const response = await deleteV1PipelineJsByName({
        baseUrl: getApiBaseUrl(),
        path: { name },
      });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to delete script"),
        );
      }
      await loadScripts();
      toast.update(toastId, {
        tone: "success",
        title: "Script deleted",
        description: `${name} has been removed.`,
      });
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to delete script");
      setError(message);
      onError?.(message);
      toast.update(toastId, {
        tone: "error",
        title: "Failed to delete script",
        description: message,
      });
    }
  };

  const discardWorkspaceDraft = useCallback(
    async (options?: { reason?: string; title?: string }) => {
      if (!workspaceDraftSession) {
        return true;
      }

      const isAIHandoffDraft = workspaceDraftSession.source !== "native";
      const confirmed = await toast.confirm({
        title:
          options?.title ??
          (isAIHandoffDraft
            ? "Discard the AI handoff draft?"
            : "Discard the local Settings draft?"),
        description:
          options?.reason ??
          (isAIHandoffDraft
            ? isScriptWorkspaceDraftSessionDirty(workspaceDraftSession)
              ? "This removes the local Settings draft for the current AI attempt. Your unsaved edits will be lost."
              : "This removes the current AI handoff draft from Settings. You can still reopen the AI modal itself if you keep that session."
            : isScriptWorkspaceDraftSessionDirty(workspaceDraftSession)
              ? "This removes the in-progress local Settings draft. Your unsaved edits will be lost."
              : "This removes the current local Settings draft from this tab."),
        confirmLabel: "Discard draft",
        cancelLabel: "Keep draft",
        tone: "warning",
      });
      if (!confirmed) {
        return false;
      }

      clearWorkspaceDraftSession();
      return true;
    },
    [clearWorkspaceDraftSession, toast, workspaceDraftSession],
  );

  const openNativeEditSession = async (options: {
    mode: "create" | "edit";
    script?: JsTargetScript;
  }) => {
    const nextOriginalName =
      options.mode === "edit" ? (options.script?.name ?? null) : null;
    const nextInitialValue =
      options.mode === "edit" && options.script
        ? toPipelineJsInput(options.script)
        : createEmptyPipelineJsInput();

    if (
      workspaceDraftSession?.source === "native" &&
      workspaceDraftSession.mode === options.mode &&
      workspaceDraftSession.originalName === nextOriginalName
    ) {
      setWorkspaceDraftSession((current) =>
        current ? { ...current, visible: true } : current,
      );
      return;
    }

    if (
      workspaceDraftSession &&
      isScriptWorkspaceDraftSessionDirty(workspaceDraftSession) &&
      !(await discardWorkspaceDraft({
        title:
          workspaceDraftSession.source === "native"
            ? "Replace the current Settings draft?"
            : "Replace the current AI handoff draft?",
        reason:
          "This opens another local Settings draft and discards the edits you have not saved yet. Keep the current draft if you still need it.",
      }))
    ) {
      return;
    }

    setError(null);
    setWorkspaceDraftSession({
      source: "native",
      attemptId: null,
      mode: options.mode,
      originalName: nextOriginalName,
      initialValue: nextInitialValue,
      draft: createScriptFormDraft(nextInitialValue),
      visible: true,
    });
  };

  const openAttemptInSettings = async (
    source: AISessionSource,
    attempt: AIAttempt<JsTargetScript>,
  ) => {
    if (!attempt.artifact) {
      return;
    }

    if (
      workspaceDraftSession &&
      (workspaceDraftSession.source !== source ||
        workspaceDraftSession.attemptId !== attempt.id) &&
      isScriptWorkspaceDraftSessionDirty(workspaceDraftSession) &&
      !(await discardWorkspaceDraft({
        title:
          workspaceDraftSession.source === "native"
            ? "Replace the current Settings draft?"
            : "Replace the current AI handoff draft?",
        reason:
          workspaceDraftSession.source === "native"
            ? "This opens an AI handoff draft and discards the local Settings edits you have not saved yet. Keep the current draft if you still need it."
            : "This attempt will replace the current Settings draft for another AI handoff. Discard the older draft only if you no longer need it.",
      }))
    ) {
      return;
    }

    setError(null);

    if (source === "generator") {
      generatorHistory.selectAttempt(attempt.id);
      setIsAIGeneratorOpen(false);
    } else {
      debuggerHistory.selectAttempt(attempt.id);
      setIsAIDebuggerOpen(false);
    }

    const nextMode = source === "generator" ? "create" : "edit";
    const nextOriginalName =
      source === "debugger"
        ? (debuggingScript?.name ?? attempt.artifact.name)
        : null;
    const nextInitialValue = toPipelineJsInput(attempt.artifact);

    setWorkspaceDraftSession((current) => {
      if (
        current &&
        current.source === source &&
        current.attemptId === attempt.id
      ) {
        return {
          ...current,
          originalName: nextOriginalName,
          visible: true,
        };
      }

      return {
        source,
        attemptId: attempt.id,
        mode: nextMode,
        originalName: nextOriginalName,
        initialValue: nextInitialValue,
        draft: createScriptFormDraft(nextInitialValue),
        visible: true,
      };
    });
  };

  const returnToAISession = (
    source: AISessionSource,
    options?: { preserveDraft?: boolean },
  ) => {
    setWorkspaceDraftSession((current) => {
      if (!current || current.source !== source) {
        return current;
      }

      return options?.preserveDraft === false
        ? null
        : { ...current, visible: false };
    });

    if (source === "generator") {
      setIsAIGeneratorOpen(true);
      return;
    }

    setIsAIDebuggerOpen(true);
  };

  const handleDraftChange = useCallback(
    (draft: ScriptFormDraft) => {
      setWorkspaceDraftSession((current) => {
        if (!current || deepEqual(current.draft, draft)) {
          return current;
        }

        return { ...current, draft };
      });
    },
    [setWorkspaceDraftSession],
  );

  const handleManualEditSubmit = async (
    session: ScriptWorkspaceDraftSession,
    input: PipelineJsInput,
  ) => {
    if (session.source === "native" || !session.attemptId) {
      return;
    }
    const actionLabel =
      session.mode === "create"
        ? `Creating ${input.name}`
        : `Updating ${input.name}`;
    const toastId = toast.show({
      tone: "loading",
      title: actionLabel,
      description:
        "Saving the manually edited script and preserving the AI attempt history.",
    });

    try {
      setError(null);
      const response =
        session.mode === "edit"
          ? await putV1PipelineJsByName({
              baseUrl: getApiBaseUrl(),
              path: { name: session.originalName ?? input.name },
              body: input,
            })
          : await postV1PipelineJs({
              baseUrl: getApiBaseUrl(),
              body: input,
            });

      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to save script"),
        );
      }

      const savedScript = (response.data ?? input) as JsTargetScript;
      const history =
        session.source === "generator" ? generatorHistory : debuggerHistory;
      history.replaceArtifact(session.attemptId, savedScript, {
        markManualEdit: true,
      });

      if (session.source === "debugger") {
        setDebuggingScript(savedScript);
      }

      await loadScripts();
      toast.update(toastId, {
        tone: "success",
        title: "Manual edits saved",
        description:
          "The AI attempt now uses your saved script as the retry baseline.",
      });
      returnToAISession(session.source, { preserveDraft: false });
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to save script");
      setError(message);
      onError?.(message);
      toast.update(toastId, {
        tone: "error",
        title: "Failed to save script",
        description: message,
      });
    }
  };

  const handleOpenGenerator = () => {
    hideWorkspaceDraft();
    setIsAIGeneratorOpen(true);
  };

  const handleOpenDebugger = async (script: JsTargetScript) => {
    if (
      debuggingScript &&
      debuggingScript.name !== script.name &&
      !(await toast.confirm({
        title: `Start tuning ${script.name}?`,
        description: `This replaces the in-progress AI tuning session for ${debuggingScript.name}. Keep the existing session if you still need that candidate or request draft.`,
        confirmLabel: "Start new tuning session",
        cancelLabel: "Keep existing session",
        tone: "warning",
      }))
    ) {
      return;
    }

    hideWorkspaceDraft();

    if (debuggingScript && debuggingScript.name !== script.name) {
      setDebuggerResetSignal((current) => current + 1);
    }

    setDebuggingScript(script);
    setIsAIDebuggerOpen(true);
  };

  if (loading) {
    return <div className="p-4 text-center">Loading scripts...</div>;
  }

  return (
    <div className="space-y-4">
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          gap: 12,
          flexWrap: "wrap",
        }}
      >
        <div>
          <h2 style={{ marginBottom: 4 }}>Pipeline JavaScript</h2>
          <p style={{ margin: 0, opacity: 0.8 }}>
            Save host-specific JavaScript only when a page needs repeatable DOM
            prep, waits, or post-navigation cleanup.
          </p>
        </div>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          <button
            type="button"
            onClick={() => setShowJson(!showJson)}
            className="secondary"
          >
            {showJson ? "Hide JSON" : "Show JSON"}
          </button>
          <button
            type="button"
            onClick={handleOpenGenerator}
            disabled={aiUnavailable}
            title={aiUnavailableMessage ?? undefined}
            className={aiUnavailable ? "secondary" : undefined}
          >
            Generate with AI
          </button>
          <button
            type="button"
            onClick={() => {
              void openNativeEditSession({ mode: "create" });
            }}
          >
            Create Script
          </button>
        </div>
      </div>

      {aiUnavailableMessage ? (
        <AIUnavailableNotice message={aiUnavailableMessage} />
      ) : null}

      {error ? (
        <div className="error" role="alert">
          {error}
        </div>
      ) : null}

      {hiddenWorkspaceDraftSession ? (
        <ResumableSettingsDraftNotice
          title={
            hiddenWorkspaceDraftSession.source === "native"
              ? `Local Settings draft for ${
                  hiddenWorkspaceDraftSession.originalName ??
                  "a new pipeline script"
                }${
                  isScriptWorkspaceDraftSessionDirty(
                    hiddenWorkspaceDraftSession,
                  )
                    ? " has unsaved edits."
                    : " is still available in this tab."
                }`
              : `AI handoff draft for Attempt ${hiddenWorkspaceDraftSession.attemptId?.replace(
                  "attempt-",
                  "",
                )}${
                  isScriptWorkspaceDraftSessionDirty(
                    hiddenWorkspaceDraftSession,
                  )
                    ? " has unsaved Settings edits."
                    : " is still available in Settings."
                }`
          }
          description={
            hiddenWorkspaceDraftSession.source === "native"
              ? "Close keeps this draft available in the current tab. Resume it when you want to continue editing, or discard it explicitly once you no longer need it."
              : "Close keeps this draft available in the current tab. Resume it when you want to keep editing the local handoff draft, or discard it explicitly once you no longer need it."
          }
          resumeLabel={
            hiddenWorkspaceDraftSession.source === "native"
              ? "Resume Settings draft"
              : "Resume AI handoff draft"
          }
          discardLabel={
            hiddenWorkspaceDraftSession.source === "native"
              ? "Discard Settings draft"
              : "Discard handoff draft"
          }
          onResume={() =>
            setWorkspaceDraftSession((current) =>
              current ? { ...current, visible: true } : current,
            )
          }
          onDiscard={() => {
            void discardWorkspaceDraft();
          }}
        />
      ) : null}

      {workspaceDraftSession?.visible ? (
        workspaceDraftSession.source === "native" ? (
          <ScriptForm
            key={`native-${workspaceDraftSession.mode}-${workspaceDraftSession.originalName ?? "create"}`}
            initialValue={workspaceDraftSession.initialValue}
            draft={workspaceDraftSession.draft}
            savedValue={
              workspaceDraftSession.mode === "edit"
                ? workspaceDraftSession.initialValue
                : undefined
            }
            lockName={workspaceDraftSession.mode === "edit"}
            title={
              workspaceDraftSession.mode === "edit"
                ? "Edit Saved Script"
                : "Create New Script"
            }
            cancelLabel="Close"
            submitLabel={
              workspaceDraftSession.mode === "edit" ? "Update" : "Create"
            }
            onDraftChange={handleDraftChange}
            onSubmit={(input) => {
              void handleSaveNativeEditSession(workspaceDraftSession, input);
            }}
            onCancel={hideWorkspaceDraft}
            onDiscard={() => {
              void discardWorkspaceDraft();
            }}
          />
        ) : (
          <ScriptForm
            key={`manual-${workspaceDraftSession.source}-${workspaceDraftSession.attemptId}`}
            initialValue={workspaceDraftSession.initialValue}
            draft={workspaceDraftSession.draft}
            savedValue={workspaceDraftSession.initialValue}
            onDraftChange={handleDraftChange}
            lockName={workspaceDraftSession.mode === "edit"}
            title={
              workspaceDraftSession.mode === "edit"
                ? "Edit Script from AI Session"
                : "Create Script from AI Session"
            }
            contextNotice={
              <ManualEditContextNotice
                attemptId={workspaceDraftSession.attemptId ?? "attempt-"}
                submitLabel={
                  workspaceDraftSession.mode === "edit" ? "Update" : "Create"
                }
              />
            }
            cancelLabel="Back to AI session"
            submitLabel={
              workspaceDraftSession.mode === "edit" ? "Update" : "Create"
            }
            onSubmit={(input) => {
              void handleManualEditSubmit(workspaceDraftSession, input);
            }}
            onCancel={() => {
              if (workspaceDraftSession.source !== "native") {
                returnToAISession(workspaceDraftSession.source);
              }
            }}
          />
        )
      ) : null}

      {scripts.length === 0 && !workspaceDraftSession ? (
        <ActionEmptyState
          eyebrow="Optional page-specific hook"
          title="No pipeline scripts yet"
          description="Most sites do not need custom JavaScript in the fetch pipeline. Add a script once a host needs repeatable wait selectors, DOM normalization, or pre-navigation setup."
          actions={[
            {
              label: "Create your first script",
              onClick: () => {
                void openNativeEditSession({ mode: "create" });
              },
            },
          ]}
        />
      ) : null}

      {showJson && scripts.length > 0 ? (
        <div className="max-h-96 overflow-auto rounded bg-gray-900 p-4 text-green-400">
          <pre className="text-sm">{JSON.stringify(scripts, null, 2)}</pre>
        </div>
      ) : null}

      <AIPipelineJSGenerator
        isOpen={isAIGeneratorOpen}
        aiStatus={aiStatus}
        history={generatorHistory}
        onEditInSettings={(attempt) => {
          void openAttemptInSettings("generator", attempt);
        }}
        onClose={() => setIsAIGeneratorOpen(false)}
        storageKey={PIPELINE_JS_GENERATOR_SESSION_KEY}
        onSaved={() => {
          void loadScripts();
        }}
      />

      <AIPipelineJSDebugger
        isOpen={isAIDebuggerOpen}
        aiStatus={aiStatus}
        script={debuggingScript}
        history={debuggerHistory}
        onEditInSettings={(attempt) => {
          void openAttemptInSettings("debugger", attempt);
        }}
        onClose={() => {
          setIsAIDebuggerOpen(false);
        }}
        onSaved={() => {
          void loadScripts();
        }}
        storageKey={PIPELINE_JS_DEBUGGER_SESSION_KEY}
        resetSignal={debuggerResetSignal}
        onSessionCleared={clearDebuggingScript}
      />

      <div className="space-y-2">
        {scripts.map((script) => (
          <div
            key={script.name}
            className="flex items-start justify-between rounded border p-4 hover:bg-gray-50"
          >
            <div className="flex-1">
              <h3 className="font-medium">{script.name}</h3>
              <p className="text-sm text-gray-600">
                Hosts: {script.hostPatterns.join(", ")}
              </p>
              {script.engine ? (
                <span className="mr-2 rounded bg-purple-100 px-2 py-0.5 text-xs text-purple-800">
                  {script.engine}
                </span>
              ) : null}
              {script.preNav ? (
                <span className="mr-2 rounded bg-green-100 px-2 py-0.5 text-xs text-green-800">
                  pre-nav
                </span>
              ) : null}
              {script.postNav ? (
                <span className="mr-2 rounded bg-blue-100 px-2 py-0.5 text-xs text-blue-800">
                  post-nav
                </span>
              ) : null}
              {script.selectors && script.selectors.length > 0 ? (
                <span className="rounded bg-orange-100 px-2 py-0.5 text-xs text-orange-800">
                  {script.selectors.length} selector
                  {script.selectors.length !== 1 ? "s" : ""}
                </span>
              ) : null}
            </div>
            <div className="space-x-2">
              <button
                type="button"
                onClick={() => {
                  void handleOpenDebugger(script);
                }}
                disabled={aiUnavailable}
                title={aiUnavailableMessage ?? undefined}
                className={`text-sm ${
                  aiUnavailable
                    ? "cursor-not-allowed text-gray-400"
                    : "text-purple-600 hover:underline"
                }`}
              >
                {debuggingScript?.name === script.name
                  ? "Resume AI tuning"
                  : "Tune with AI"}
              </button>
              <button
                type="button"
                onClick={() => {
                  void openNativeEditSession({ mode: "edit", script });
                }}
                className="text-sm text-blue-600 hover:underline"
              >
                Edit
              </button>
              <button
                type="button"
                onClick={() => handleDelete(script.name)}
                className="text-sm text-red-600 hover:underline"
              >
                Delete
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

interface ScriptFormProps {
  script?: JsTargetScript;
  initialValue?: PipelineJsInput;
  draft?: ScriptFormDraft;
  savedValue?: PipelineJsInput;
  lockName?: boolean;
  title?: string;
  contextNotice?: ReactNode;
  cancelLabel?: string;
  discardLabel?: string;
  submitLabel?: string;
  onDraftChange?: (draft: ScriptFormDraft) => void;
  onSubmit: (input: PipelineJsInput) => void;
  onCancel: () => void;
  onDiscard?: () => void;
}

function ScriptForm({
  script,
  initialValue,
  draft,
  savedValue,
  lockName = false,
  title,
  contextNotice,
  cancelLabel = "Cancel",
  discardLabel = "Discard draft",
  submitLabel,
  onDraftChange,
  onSubmit,
  onCancel,
  onDiscard,
}: ScriptFormProps) {
  const seed =
    initialValue ??
    (script ? toPipelineJsInput(script) : createEmptyPipelineJsInput());
  const seedDraft = draft ?? createScriptFormDraft(seed);

  const [formData, setFormData] = useState<PipelineJsInput>(seedDraft.formData);
  const [hostPatternInput, setHostPatternInput] = useState(
    seedDraft.hostPatternInput,
  );
  const [selectorInput, setSelectorInput] = useState(seedDraft.selectorInput);

  const currentDraft = useMemo<ScriptFormDraft>(
    () => ({
      formData,
      hostPatternInput,
      selectorInput,
    }),
    [formData, hostPatternInput, selectorInput],
  );

  useEffect(() => {
    onDraftChange?.(currentDraft);
  }, [currentDraft, onDraftChange]);

  const syncState = useMemo<"clean" | "dirty" | null>(() => {
    const baselineValue = savedValue ?? seed;
    const workingValue = buildPipelineJsInputFromDraft(currentDraft);
    const isDirty = !deepEqual(workingValue, baselineValue);

    if (isDirty) {
      return "dirty";
    }

    return savedValue ? "clean" : null;
  }, [currentDraft, savedValue, seed]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(buildPipelineJsInputFromDraft(currentDraft));
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="space-y-4 rounded border bg-gray-50 p-4"
    >
      <h3 className="font-medium">
        {title ?? (script ? "Edit Script" : "Create New Script")}
      </h3>

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

      <div>
        <label htmlFor="script-name" className="mb-1 block text-sm font-medium">
          Name
        </label>
        <input
          id="script-name"
          type="text"
          value={formData.name}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
          className="w-full rounded border px-3 py-2"
          required
          disabled={lockName || !!script}
        />
      </div>

      <div>
        <label
          htmlFor="script-host-patterns"
          className="mb-1 block text-sm font-medium"
        >
          Host Patterns (comma-separated)
        </label>
        <input
          id="script-host-patterns"
          type="text"
          value={hostPatternInput}
          onChange={(e) => setHostPatternInput(e.target.value)}
          placeholder="example.com, *.example.com"
          className="w-full rounded border px-3 py-2"
          required
        />
        <p className="mt-1 text-xs text-gray-500">
          Examples: example.com, *.example.com
        </p>
      </div>

      <div>
        <label
          htmlFor="script-engine"
          className="mb-1 block text-sm font-medium"
        >
          Engine
        </label>
        <select
          id="script-engine"
          value={formData.engine || ""}
          onChange={(e) =>
            setFormData({
              ...formData,
              engine: e.target.value
                ? (e.target.value as PipelineJsInput["engine"])
                : undefined,
            })
          }
          className="w-full rounded border px-3 py-2"
        >
          <option value="">Any</option>
          <option value="chromedp">ChromeDP</option>
          <option value="playwright">Playwright</option>
        </select>
      </div>

      <div>
        <label
          htmlFor="script-pre-nav"
          className="mb-1 block text-sm font-medium"
        >
          Pre-Navigation JavaScript
        </label>
        <textarea
          id="script-pre-nav"
          value={formData.preNav || ""}
          onChange={(e) =>
            setFormData({ ...formData, preNav: e.target.value || undefined })
          }
          placeholder="// JavaScript to run before navigation"
          className="w-full rounded border px-3 py-2 font-mono text-sm"
          rows={4}
        />
      </div>

      <div>
        <label
          htmlFor="script-post-nav"
          className="mb-1 block text-sm font-medium"
        >
          Post-Navigation JavaScript
        </label>
        <textarea
          id="script-post-nav"
          value={formData.postNav || ""}
          onChange={(e) =>
            setFormData({ ...formData, postNav: e.target.value || undefined })
          }
          placeholder="// JavaScript to run after navigation"
          className="w-full rounded border px-3 py-2 font-mono text-sm"
          rows={4}
        />
      </div>

      <div>
        <label
          htmlFor="script-selectors"
          className="mb-1 block text-sm font-medium"
        >
          Wait Selectors (comma-separated)
        </label>
        <input
          id="script-selectors"
          type="text"
          value={selectorInput}
          onChange={(e) => setSelectorInput(e.target.value)}
          placeholder="#content, .article, [data-loaded]"
          className="w-full rounded border px-3 py-2"
        />
        <p className="mt-1 text-xs text-gray-500">
          CSS selectors to wait for before considering page loaded
        </p>
      </div>

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
          {submitLabel ?? (script ? "Update" : "Create")}
        </button>
      </div>
    </form>
  );
}
