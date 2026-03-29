/**
 * Purpose: Provide the Settings-route editor for saved render profiles.
 * Responsibilities: Load the render-profile inventory, coordinate create/edit/delete flows, preserve AI authoring sessions across Settings handoff, and route transient operator feedback through the shared toast system.
 * Scope: Browser-side render-profile management only; fetch execution and runtime matching stay outside this component.
 * Usage: Render inside the Settings route with the app-level toast provider already mounted.
 * Invariants/Assumptions: Profiles are persisted through the generated API client, manual AI handoff returns to the same in-session history, and API errors remain visible in both inline state and toasts.
 */

import { useCallback } from "react";

import type {
  ComponentStatus,
  RenderProfile,
  RenderProfileInput,
} from "../../api";
import {
  deleteV1RenderProfilesByName,
  getV1RenderProfiles,
  postV1RenderProfiles,
  putV1RenderProfilesByName,
} from "../../api";
import type { AIAttempt } from "../../hooks/useAIAttemptHistory";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { AIRenderProfileDebugger } from "../AIRenderProfileDebugger";
import { AIRenderProfileGenerator } from "../AIRenderProfileGenerator";
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
  createEmptyRenderProfileInput,
  createProfileFormDraft,
  isProfileDraftDirty,
  RenderProfileForm,
  toRenderProfileInput,
  type ProfileFormDraft,
} from "./RenderProfileForm";

type ProfileDraftSessionSource = "native" | SettingsAuthoringAISessionSource;
type ProfileWorkspaceDraftSession = SettingsWorkspaceDraftSession<
  ProfileDraftSessionSource,
  RenderProfileInput,
  ProfileFormDraft
>;

const RENDER_PROFILE_GENERATOR_SESSION_KEY =
  "spartan.render-profile.ai-generator-session";
const RENDER_PROFILE_DEBUGGER_SESSION_KEY =
  "spartan.render-profile.ai-debugger-session";
const RENDER_PROFILE_DEBUGGER_TARGET_KEY =
  "spartan.render-profile.ai-debugger-target";
const RENDER_PROFILE_WORKSPACE_DRAFT_SESSION_KEY =
  "spartan.render-profile.workspace-draft-session";

interface RenderProfileEditorProps {
  onError?: (error: string) => void;
  aiStatus?: ComponentStatus | null;
  onInventoryChange?: (count: number) => void;
}

function isProfileWorkspaceDraftSessionDirty(
  session: ProfileWorkspaceDraftSession,
): boolean {
  return isProfileDraftDirty(session.draft, session.initialValue);
}

export function RenderProfileEditor({
  onError,
  aiStatus = null,
  onInventoryChange,
}: RenderProfileEditorProps) {
  const loadProfiles = useCallback(async () => {
    const response = await getV1RenderProfiles({
      baseUrl: getApiBaseUrl(),
    });
    if (response.error) {
      throw new Error(
        getApiErrorMessage(response.error, "Failed to load profiles"),
      );
    }
    return response.data?.profiles || [];
  }, []);

  const {
    items: profiles,
    loading,
    error,
    showJson,
    setShowJson,
    aiUnavailable,
    aiUnavailableMessage,
    generatorHistory,
    debuggerHistory,
    debuggingItem: debuggingProfile,
    setDebuggingItem: setDebuggingProfile,
    clearDebuggingItem: clearDebuggingProfile,
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
    RenderProfile,
    RenderProfileInput,
    ProfileFormDraft
  >({
    aiStatus,
    aiFallbackMessage: "Create and edit profiles manually below.",
    loadErrorMessage: "Failed to load profiles",
    storageKeys: {
      generatorSession: RENDER_PROFILE_GENERATOR_SESSION_KEY,
      debuggerSession: RENDER_PROFILE_DEBUGGER_SESSION_KEY,
      debuggerTarget: RENDER_PROFILE_DEBUGGER_TARGET_KEY,
      workspaceDraftSession: RENDER_PROFILE_WORKSPACE_DRAFT_SESSION_KEY,
    },
    loadInventory: loadProfiles,
    createEmptyInput: createEmptyRenderProfileInput,
    toInput: toRenderProfileInput,
    createDraft: createProfileFormDraft,
    isDirty: isProfileWorkspaceDraftSessionDirty,
    buildDebuggerReplacePrompt: (current, next) => ({
      title: `Start tuning ${next.name}?`,
      description: `This replaces the in-progress AI tuning session for ${current.name}. Keep the existing session if you still need that candidate or request draft.`,
    }),
    onError,
    onInventoryChange,
  });

  const handleSaveNativeEditSession = async (
    session: ProfileWorkspaceDraftSession,
    input: RenderProfileInput,
  ) => {
    if (session.source !== "native") {
      return;
    }

    const isCreate = session.mode === "create";
    const result = await runReloadingMutation({
      run: async () => {
        const response = isCreate
          ? await postV1RenderProfiles({
              baseUrl: getApiBaseUrl(),
              body: input,
            })
          : await putV1RenderProfilesByName({
              baseUrl: getApiBaseUrl(),
              path: { name: session.originalName ?? input.name },
              body: input,
            });
        if (response.error) {
          throw new Error(
            getApiErrorMessage(
              response.error,
              isCreate
                ? "Failed to create profile"
                : "Failed to update profile",
            ),
          );
        }
        return (response.data ?? input) as RenderProfile;
      },
      loading: {
        title: isCreate
          ? input.name
            ? `Creating ${input.name}`
            : "Creating render profile"
          : `Updating ${session.originalName ?? input.name}`,
        description: isCreate
          ? "Saving the new render profile configuration."
          : "Saving the latest render profile changes.",
      },
      success: {
        title: isCreate ? "Render profile created" : "Render profile updated",
        description: isCreate
          ? `${input.name} is now available for fetch configuration.`
          : `${session.originalName ?? input.name} now reflects the latest configuration.`,
      },
      errorTitle: isCreate
        ? "Failed to create render profile"
        : "Failed to update render profile",
      errorFallback: isCreate
        ? "Failed to create profile"
        : "Failed to update profile",
    });

    if (result.ok) {
      clearWorkspaceDraftSession();
    }
  };

  const handleDelete = async (name: string) => {
    const confirmed = await confirmAction({
      title: `Delete ${name}?`,
      description:
        "This removes the saved render profile from local storage. Jobs that reference it will need a different profile.",
      confirmLabel: "Delete profile",
      cancelLabel: "Keep profile",
      tone: "error",
    });
    if (!confirmed) {
      return;
    }

    await runReloadingMutation({
      run: async () => {
        const response = await deleteV1RenderProfilesByName({
          baseUrl: getApiBaseUrl(),
          path: { name },
        });
        if (response.error) {
          throw new Error(
            getApiErrorMessage(response.error, "Failed to delete profile"),
          );
        }
      },
      loading: {
        title: `Deleting ${name}`,
        description: "Removing the saved render profile.",
      },
      success: {
        title: "Render profile deleted",
        description: `${name} has been removed.`,
      },
      errorTitle: "Failed to delete render profile",
      errorFallback: "Failed to delete profile",
    });
  };

  const handleManualEditSubmit = async (
    session: ProfileWorkspaceDraftSession,
    input: RenderProfileInput,
  ) => {
    if (session.source === "native" || !session.attemptId) {
      return;
    }

    const result = await runReloadingMutation({
      run: async () => {
        const response =
          session.mode === "edit"
            ? await putV1RenderProfilesByName({
                baseUrl: getApiBaseUrl(),
                path: { name: session.originalName ?? input.name },
                body: input,
              })
            : await postV1RenderProfiles({
                baseUrl: getApiBaseUrl(),
                body: input,
              });

        if (response.error) {
          throw new Error(
            getApiErrorMessage(response.error, "Failed to save render profile"),
          );
        }

        return (response.data ?? input) as RenderProfile;
      },
      loading: {
        title:
          session.mode === "create"
            ? `Creating ${input.name}`
            : `Updating ${input.name}`,
        description:
          "Saving the manually edited render profile and preserving the AI attempt history.",
      },
      success: {
        title: "Manual edits saved",
        description:
          "The AI attempt now uses your saved render profile as the retry baseline.",
      },
      errorTitle: "Failed to save render profile",
      errorFallback: "Failed to save render profile",
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
      setDebuggingProfile(result.value);
    }

    returnToAISession(session.source, { preserveDraft: false });
  };

  return (
    <SettingsAuthoringShell
      loading={loading}
      loadingLabel="Loading profiles..."
      title="Render Profiles"
      description="Save reusable fetch and browser overrides only for hosts that need a repeatable runtime strategy."
      showJson={showJson}
      onToggleJson={() => setShowJson((current) => !current)}
      createLabel="Create Profile"
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
              isDirty: isProfileWorkspaceDraftSessionDirty(
                hiddenWorkspaceDraftSession,
              ),
              nativeDraftLabel: "a new render profile",
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
            <RenderProfileForm
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
                  ? "Edit Saved Profile"
                  : "Create New Profile"
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
            <RenderProfileForm
              key={`manual-${workspaceDraftSession.source}-${workspaceDraftSession.attemptId}`}
              initialValue={workspaceDraftSession.initialValue}
              draft={workspaceDraftSession.draft}
              savedValue={workspaceDraftSession.initialValue}
              onDraftChange={handleDraftChange}
              lockName={workspaceDraftSession.mode === "edit"}
              title={
                workspaceDraftSession.mode === "edit"
                  ? "Edit Profile from AI Session"
                  : "Create Profile from AI Session"
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
        profiles.length === 0 && !workspaceDraftSession ? (
          <ActionEmptyState
            eyebrow="Optional runtime override"
            title="No saved render profiles yet"
            description="Most jobs can use Spartan's default runtime selection. Add a render profile only when a host needs a stable override for browser usage, engine choice, or rate limits."
            actions={[
              {
                label: "Create your first profile",
                onClick: () => {
                  void openNativeEditSession({ mode: "create" });
                },
              },
            ]}
          />
        ) : null
      }
      jsonPanel={
        showJson && profiles.length > 0 ? (
          <div className="max-h-96 overflow-auto rounded bg-gray-900 p-4 text-green-400">
            <pre className="text-sm">{JSON.stringify(profiles, null, 2)}</pre>
          </div>
        ) : null
      }
      aiPanels={
        <>
          <AIRenderProfileGenerator
            isOpen={isAIGeneratorOpen}
            aiStatus={aiStatus}
            history={generatorHistory}
            onEditInSettings={(attempt: AIAttempt<RenderProfile>) => {
              void openAttemptInSettings("generator", attempt);
            }}
            onClose={() => setIsAIGeneratorOpen(false)}
            storageKey={RENDER_PROFILE_GENERATOR_SESSION_KEY}
            onSaved={() => {
              void loadInventory();
            }}
          />

          <AIRenderProfileDebugger
            isOpen={isAIDebuggerOpen}
            aiStatus={aiStatus}
            profile={debuggingProfile}
            history={debuggerHistory}
            onEditInSettings={(attempt: AIAttempt<RenderProfile>) => {
              void openAttemptInSettings("debugger", attempt);
            }}
            onClose={() => {
              setIsAIDebuggerOpen(false);
            }}
            onSaved={() => {
              void loadInventory();
            }}
            storageKey={RENDER_PROFILE_DEBUGGER_SESSION_KEY}
            resetSignal={debuggerResetSignal}
            onSessionCleared={clearDebuggingProfile}
          />
        </>
      }
    >
      {profiles.map((profile) => (
        <div
          key={profile.name}
          className="flex items-start justify-between rounded border p-4 hover:bg-gray-50"
        >
          <div>
            <h3 className="font-medium">{profile.name}</h3>
            <p className="text-sm text-gray-600">
              Hosts: {profile.hostPatterns.join(", ")}
            </p>
            {profile.forceEngine ? (
              <span className="rounded bg-blue-100 px-2 py-0.5 text-xs text-blue-800">
                {profile.forceEngine}
              </span>
            ) : null}
          </div>
          <div className="space-x-2">
            <button
              type="button"
              onClick={() => {
                void openDebugger(profile);
              }}
              disabled={aiUnavailable}
              title={aiUnavailableMessage ?? undefined}
              className={`text-sm ${
                aiUnavailable
                  ? "cursor-not-allowed text-gray-400"
                  : "text-purple-600 hover:underline"
              }`}
            >
              {debuggingProfile?.name === profile.name
                ? "Resume AI tuning"
                : "Tune with AI"}
            </button>
            <button
              type="button"
              onClick={() => {
                void openNativeEditSession({ mode: "edit", item: profile });
              }}
              className="text-sm text-blue-600 hover:underline"
            >
              Edit
            </button>
            <button
              type="button"
              onClick={() => {
                void handleDelete(profile.name);
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
