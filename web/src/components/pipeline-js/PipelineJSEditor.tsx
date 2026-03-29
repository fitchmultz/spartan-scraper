/**
 * Purpose: Provide the Settings-route editor for stored pipeline JavaScript configurations.
 * Responsibilities: Load the script inventory, coordinate create/edit/delete flows, preserve AI authoring sessions across Settings handoff, and surface operator feedback through inline state and toasts.
 * Scope: Browser-side pipeline-script management only; runtime execution and matching logic stay on the backend.
 * Usage: Render inside the Settings route without additional providers beyond the app-level toast boundary.
 * Invariants/Assumptions: Script persistence goes through the generated API client, manual AI handoff returns to the same in-session history, and destructive actions use the shared confirmation dialog instead of browser-native prompts.
 */

import { useCallback } from "react";

import type {
  ComponentStatus,
  JsTargetScript,
  PipelineJsInput,
} from "../../api";
import {
  deleteV1PipelineJsByName,
  getV1PipelineJs,
  postV1PipelineJs,
  putV1PipelineJsByName,
} from "../../api";
import type { AIAttempt } from "../../hooks/useAIAttemptHistory";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { AIPipelineJSDebugger } from "../AIPipelineJSDebugger";
import { AIPipelineJSGenerator } from "../AIPipelineJSGenerator";
import { ActionEmptyState } from "../ActionEmptyState";
import { SettingsAuthoringShell } from "../settings/SettingsAuthoringShell";
import {
  SettingsAIHandoffContextNotice,
  type SettingsWorkspaceDraftSession,
} from "../settings/workspaceDrafts";
import {
  type SettingsAuthoringAISessionSource,
  useSettingsAuthoringShell,
} from "../settings/useSettingsAuthoringShell";
import {
  createEmptyPipelineJsInput,
  createScriptFormDraft,
  isScriptDraftDirty,
  PipelineScriptForm,
  toPipelineJsInput,
  type ScriptFormDraft,
} from "./PipelineScriptForm";

type ScriptDraftSessionSource = "native" | SettingsAuthoringAISessionSource;
type ScriptWorkspaceDraftSession = SettingsWorkspaceDraftSession<
  ScriptDraftSessionSource,
  PipelineJsInput,
  ScriptFormDraft
>;

const PIPELINE_JS_GENERATOR_SESSION_KEY =
  "spartan.pipeline-js.ai-generator-session";
const PIPELINE_JS_DEBUGGER_SESSION_KEY =
  "spartan.pipeline-js.ai-debugger-session";
const PIPELINE_JS_DEBUGGER_TARGET_KEY =
  "spartan.pipeline-js.ai-debugger-target";
const PIPELINE_JS_WORKSPACE_DRAFT_SESSION_KEY =
  "spartan.pipeline-js.workspace-draft-session";

interface PipelineJSEditorProps {
  onError?: (error: string) => void;
  aiStatus?: ComponentStatus | null;
  onInventoryChange?: (count: number) => void;
}

function isScriptWorkspaceDraftSessionDirty(
  session: ScriptWorkspaceDraftSession,
): boolean {
  return isScriptDraftDirty(session.draft, session.initialValue);
}

export function PipelineJSEditor({
  onError,
  aiStatus = null,
  onInventoryChange,
}: PipelineJSEditorProps) {
  const loadScripts = useCallback(async () => {
    const response = await getV1PipelineJs({
      baseUrl: getApiBaseUrl(),
    });
    if (response.error) {
      throw new Error(
        getApiErrorMessage(response.error, "Failed to load scripts"),
      );
    }
    return response.data?.scripts || [];
  }, []);

  const {
    items: scripts,
    loading,
    error,
    showJson,
    setShowJson,
    aiUnavailable,
    aiUnavailableMessage,
    generatorHistory,
    debuggerHistory,
    debuggingItem: debuggingScript,
    setDebuggingItem: setDebuggingScript,
    clearDebuggingItem: clearDebuggingScript,
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
  } = useSettingsAuthoringShell<
    JsTargetScript,
    PipelineJsInput,
    ScriptFormDraft
  >({
    aiStatus,
    aiFallbackMessage: "Create and edit scripts manually below.",
    loadErrorMessage: "Failed to load scripts",
    storageKeys: {
      generatorSession: PIPELINE_JS_GENERATOR_SESSION_KEY,
      debuggerSession: PIPELINE_JS_DEBUGGER_SESSION_KEY,
      debuggerTarget: PIPELINE_JS_DEBUGGER_TARGET_KEY,
      workspaceDraftSession: PIPELINE_JS_WORKSPACE_DRAFT_SESSION_KEY,
    },
    loadInventory: loadScripts,
    createEmptyInput: createEmptyPipelineJsInput,
    toInput: toPipelineJsInput,
    createDraft: createScriptFormDraft,
    isDirty: isScriptWorkspaceDraftSessionDirty,
    buildDebuggerReplacePrompt: (current, next) => ({
      title: `Start tuning ${next.name}?`,
      description: `This replaces the in-progress AI tuning session for ${current.name}. Keep the existing session if you still need that candidate or request draft.`,
    }),
    onError,
    onInventoryChange,
  });

  const handleSaveNativeEditSession = async (
    session: ScriptWorkspaceDraftSession,
    input: PipelineJsInput,
  ) => {
    if (session.source !== "native") {
      return;
    }

    const isCreate = session.mode === "create";
    const result = await runReloadingMutation({
      run: async () => {
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
        return (response.data ?? input) as JsTargetScript;
      },
      loading: {
        title: isCreate
          ? input.name
            ? `Creating ${input.name}`
            : "Creating script"
          : `Updating ${session.originalName ?? input.name}`,
        description: isCreate
          ? "Saving the new pipeline JavaScript configuration."
          : "Saving the latest pipeline JavaScript changes.",
      },
      success: {
        title: isCreate ? "Script created" : "Script updated",
        description: isCreate
          ? `${input.name} is ready for pipeline matching.`
          : `${session.originalName ?? input.name} now reflects the latest configuration.`,
      },
      errorTitle: isCreate
        ? "Failed to create script"
        : "Failed to update script",
      errorFallback: isCreate
        ? "Failed to create script"
        : "Failed to update script",
    });

    if (result.ok) {
      clearWorkspaceDraftSession();
    }
  };

  const handleDelete = async (name: string) => {
    const confirmed = await confirmAction({
      title: `Delete ${name}?`,
      description:
        "This removes the saved script configuration from local storage. Matching pages will stop using it immediately.",
      confirmLabel: "Delete script",
      cancelLabel: "Keep script",
      tone: "error",
    });
    if (!confirmed) {
      return;
    }

    await runReloadingMutation({
      run: async () => {
        const response = await deleteV1PipelineJsByName({
          baseUrl: getApiBaseUrl(),
          path: { name },
        });
        if (response.error) {
          throw new Error(
            getApiErrorMessage(response.error, "Failed to delete script"),
          );
        }
      },
      loading: {
        title: `Deleting ${name}`,
        description: "Removing the saved pipeline script.",
      },
      success: {
        title: "Script deleted",
        description: `${name} has been removed.`,
      },
      errorTitle: "Failed to delete script",
      errorFallback: "Failed to delete script",
    });
  };

  const handleManualEditSubmit = async (
    session: ScriptWorkspaceDraftSession,
    input: PipelineJsInput,
  ) => {
    if (session.source === "native" || !session.attemptId) {
      return;
    }

    const result = await runReloadingMutation({
      run: async () => {
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

        return (response.data ?? input) as JsTargetScript;
      },
      loading: {
        title:
          session.mode === "create"
            ? `Creating ${input.name}`
            : `Updating ${input.name}`,
        description:
          "Saving the manually edited script and preserving the AI attempt history.",
      },
      success: {
        title: "Manual edits saved",
        description:
          "The AI attempt now uses your saved script as the retry baseline.",
      },
      errorTitle: "Failed to save script",
      errorFallback: "Failed to save script",
    });

    if (!result.ok) {
      return;
    }

    const history =
      session.source === "generator" ? generatorHistory : debuggerHistory;
    history.replaceArtifact(session.attemptId, result.value, {
      markManualEdit: true,
    });

    if (session.source === "debugger") {
      setDebuggingScript(result.value);
    }

    returnToAISession(session.source, { preserveDraft: false });
  };

  return (
    <SettingsAuthoringShell
      loading={loading}
      loadingLabel="Loading scripts..."
      title="Pipeline JavaScript"
      description="Save host-specific JavaScript only when a page needs repeatable DOM prep, waits, or post-navigation cleanup."
      showJson={showJson}
      onToggleJson={() => setShowJson((current) => !current)}
      createLabel="Create Script"
      onCreate={() => {
        void openNativeEditSession({ mode: "create" });
      }}
      onOpenGenerator={openGenerator}
      aiUnavailable={aiUnavailable}
      aiUnavailableMessage={aiUnavailableMessage}
      error={error}
      hiddenDraftNotice={
        hiddenWorkspaceDraftSession
          ? {
              session: hiddenWorkspaceDraftSession,
              isDirty: isScriptWorkspaceDraftSessionDirty(
                hiddenWorkspaceDraftSession,
              ),
              nativeDraftLabel: "a new pipeline script",
              onResume: resumeWorkspaceDraft,
              onDiscard: () => {
                void discardWorkspaceDraft();
              },
            }
          : null
      }
      draftPanel={
        workspaceDraftSession?.visible ? (
          workspaceDraftSession.source === "native" ? (
            <PipelineScriptForm
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
            <PipelineScriptForm
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
                <SettingsAIHandoffContextNotice
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
        ) : null
      }
      emptyState={
        scripts.length === 0 && !workspaceDraftSession ? (
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
        ) : null
      }
      jsonPanel={
        showJson && scripts.length > 0 ? (
          <div className="max-h-96 overflow-auto rounded bg-gray-900 p-4 text-green-400">
            <pre className="text-sm">{JSON.stringify(scripts, null, 2)}</pre>
          </div>
        ) : null
      }
      aiPanels={
        <>
          <AIPipelineJSGenerator
            isOpen={isAIGeneratorOpen}
            aiStatus={aiStatus}
            history={generatorHistory}
            onEditInSettings={(attempt: AIAttempt<JsTargetScript>) => {
              void openAttemptInSettings("generator", attempt);
            }}
            onClose={() => setIsAIGeneratorOpen(false)}
            storageKey={PIPELINE_JS_GENERATOR_SESSION_KEY}
            onSaved={() => {
              void loadInventory();
            }}
          />

          <AIPipelineJSDebugger
            isOpen={isAIDebuggerOpen}
            aiStatus={aiStatus}
            script={debuggingScript}
            history={debuggerHistory}
            onEditInSettings={(attempt: AIAttempt<JsTargetScript>) => {
              void openAttemptInSettings("debugger", attempt);
            }}
            onClose={() => {
              setIsAIDebuggerOpen(false);
            }}
            onSaved={() => {
              void loadInventory();
            }}
            storageKey={PIPELINE_JS_DEBUGGER_SESSION_KEY}
            resetSignal={debuggerResetSignal}
            onSessionCleared={clearDebuggingScript}
          />
        </>
      }
    >
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
                void openDebugger(script);
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
                void openNativeEditSession({ mode: "edit", item: script });
              }}
              className="text-sm text-blue-600 hover:underline"
            >
              Edit
            </button>
            <button
              type="button"
              onClick={() => {
                void handleDelete(script.name);
              }}
              className="text-sm text-red-600 hover:underline"
            >
              Delete
            </button>
          </div>
        </div>
      ))}
    </SettingsAuthoringShell>
  );
}
