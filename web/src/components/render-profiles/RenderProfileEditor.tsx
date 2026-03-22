/**
 * Purpose: Provide the Settings-route editor for saved render profiles.
 * Responsibilities: Load the render-profile inventory, coordinate create/edit/delete flows, preserve AI authoring sessions across Settings handoff, and route transient operator feedback through the shared toast system.
 * Scope: Browser-side render-profile management only; fetch execution and runtime matching stay outside this component.
 * Usage: Render inside the Settings route with the app-level toast provider already mounted.
 * Invariants/Assumptions: Profiles are persisted through the generated API client, manual AI handoff returns to the same in-session history, and API errors remain visible in both inline state and toasts.
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  deleteV1RenderProfilesByName,
  getV1RenderProfiles,
  postV1RenderProfiles,
  putV1RenderProfilesByName,
  type ComponentStatus,
  type RenderProfile,
  type RenderProfileInput,
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
import { AIRenderProfileDebugger } from "../AIRenderProfileDebugger";
import { AIRenderProfileGenerator } from "../AIRenderProfileGenerator";
import { ActionEmptyState } from "../ActionEmptyState";
import { AIUnavailableNotice, describeAICapability } from "../ai-assistant";
import { ResumableSettingsDraftNotice } from "../settings/ResumableSettingsDraftNotice";
import { useToast } from "../toast";

type AISessionSource = "generator" | "debugger";
type ProfileDraftSessionSource = "native" | AISessionSource;

const RENDER_PROFILE_GENERATOR_SESSION_KEY =
  "spartan.render-profile.ai-generator-session";
const RENDER_PROFILE_DEBUGGER_SESSION_KEY =
  "spartan.render-profile.ai-debugger-session";
const RENDER_PROFILE_DEBUGGER_TARGET_KEY =
  "spartan.render-profile.ai-debugger-target";
const RENDER_PROFILE_WORKSPACE_DRAFT_SESSION_KEY =
  "spartan.render-profile.workspace-draft-session";

interface ProfileFormDraft {
  formData: RenderProfileInput;
  hostPatternInput: string;
  jsHeavyThresholdInput: string;
  rateLimitQPSInput: string;
  rateLimitBurstInput: string;
  waitJSON: string;
  blockJSON: string;
  timeoutsJSON: string;
  screenshotJSON: string;
  deviceJSON: string;
}

interface ProfileWorkspaceDraftSession {
  source: ProfileDraftSessionSource;
  attemptId: string | null;
  mode: "create" | "edit";
  originalName: string | null;
  initialValue: RenderProfileInput;
  draft: ProfileFormDraft;
  visible: boolean;
}

interface RenderProfileEditorProps {
  onError?: (error: string) => void;
  aiStatus?: ComponentStatus | null;
  onInventoryChange?: (count: number) => void;
}

function createEmptyRenderProfileInput(): RenderProfileInput {
  return {
    name: "",
    hostPatterns: [],
    forceEngine: undefined,
    preferHeadless: undefined,
    neverHeadless: undefined,
    assumeJsHeavy: undefined,
    jsHeavyThreshold: undefined,
    rateLimitQPS: undefined,
    rateLimitBurst: undefined,
    block: undefined,
    wait: undefined,
    timeouts: undefined,
    screenshot: undefined,
    device: undefined,
  };
}

function toRenderProfileInput(profile: RenderProfile): RenderProfileInput {
  return {
    name: profile.name,
    hostPatterns: [...profile.hostPatterns],
    forceEngine: profile.forceEngine,
    preferHeadless: profile.preferHeadless,
    neverHeadless: profile.neverHeadless,
    assumeJsHeavy: profile.assumeJsHeavy,
    jsHeavyThreshold: profile.jsHeavyThreshold,
    rateLimitQPS: profile.rateLimitQPS,
    rateLimitBurst: profile.rateLimitBurst,
    block: profile.block,
    wait: profile.wait,
    timeouts: profile.timeouts,
    screenshot: profile.screenshot,
    device: profile.device,
  };
}

function stringifyOptionalJSON(value: unknown): string {
  if (!value) {
    return "";
  }
  return JSON.stringify(value, null, 2);
}

function parseOptionalJSON<T>(label: string, value: string): T | undefined {
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

function parseOptionalNumber(value: string): number | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function createProfileFormDraft(seed: RenderProfileInput): ProfileFormDraft {
  return {
    formData: seed,
    hostPatternInput: seed.hostPatterns.join(", "),
    jsHeavyThresholdInput: seed.jsHeavyThreshold?.toString() || "",
    rateLimitQPSInput: seed.rateLimitQPS?.toString() || "",
    rateLimitBurstInput: seed.rateLimitBurst?.toString() || "",
    waitJSON: stringifyOptionalJSON(seed.wait),
    blockJSON: stringifyOptionalJSON(seed.block),
    timeoutsJSON: stringifyOptionalJSON(seed.timeouts),
    screenshotJSON: stringifyOptionalJSON(seed.screenshot),
    deviceJSON: stringifyOptionalJSON(seed.device),
  };
}

function buildRenderProfileInputFromDraft(
  draft: ProfileFormDraft,
): RenderProfileInput {
  const patterns = draft.hostPatternInput
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean);

  return {
    ...draft.formData,
    hostPatterns: patterns,
    forceEngine: draft.formData.forceEngine || undefined,
    preferHeadless: draft.formData.preferHeadless ? true : undefined,
    neverHeadless: draft.formData.neverHeadless ? true : undefined,
    assumeJsHeavy: draft.formData.assumeJsHeavy ? true : undefined,
    jsHeavyThreshold: parseOptionalNumber(draft.jsHeavyThresholdInput),
    rateLimitQPS: parseOptionalNumber(draft.rateLimitQPSInput),
    rateLimitBurst: parseOptionalNumber(draft.rateLimitBurstInput),
    wait: parseOptionalJSON<RenderProfileInput["wait"]>(
      "Wait configuration",
      draft.waitJSON,
    ),
    block: parseOptionalJSON<RenderProfileInput["block"]>(
      "Block configuration",
      draft.blockJSON,
    ),
    timeouts: parseOptionalJSON<RenderProfileInput["timeouts"]>(
      "Timeout configuration",
      draft.timeoutsJSON,
    ),
    screenshot: parseOptionalJSON<RenderProfileInput["screenshot"]>(
      "Screenshot configuration",
      draft.screenshotJSON,
    ),
    device: parseOptionalJSON<RenderProfileInput["device"]>(
      "Device configuration",
      draft.deviceJSON,
    ),
  };
}

function isProfileDraftDirty(
  draft: ProfileFormDraft,
  initialValue: RenderProfileInput,
): boolean {
  try {
    return !deepEqual(buildRenderProfileInputFromDraft(draft), initialValue);
  } catch {
    return true;
  }
}

function isProfileWorkspaceDraftSessionDirty(
  session: ProfileWorkspaceDraftSession,
): boolean {
  return isProfileDraftDirty(session.draft, session.initialValue);
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

export function RenderProfileEditor({
  onError,
  aiStatus = null,
  onInventoryChange,
}: RenderProfileEditorProps) {
  const toast = useToast();
  const [profiles, setProfiles] = useState<RenderProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [
    workspaceDraftSession,
    setWorkspaceDraftSession,
    clearWorkspaceDraftSession,
  ] = useSessionStorageState<ProfileWorkspaceDraftSession | null>(
    RENDER_PROFILE_WORKSPACE_DRAFT_SESSION_KEY,
    null,
  );
  const [debuggingProfile, setDebuggingProfile, clearDebuggingProfile] =
    useSessionStorageState<RenderProfile | null>(
      RENDER_PROFILE_DEBUGGER_TARGET_KEY,
      null,
    );
  const [isAIGeneratorOpen, setIsAIGeneratorOpen] = useState(false);
  const [isAIDebuggerOpen, setIsAIDebuggerOpen] = useState(false);
  const [debuggerResetSignal, setDebuggerResetSignal] = useState(0);
  const [showJson, setShowJson] = useState(false);
  const generatorHistory = useAIAttemptHistory<RenderProfile>({
    storageKey: `${RENDER_PROFILE_GENERATOR_SESSION_KEY}.history`,
  });
  const debuggerHistory = useAIAttemptHistory<RenderProfile>({
    storageKey: `${RENDER_PROFILE_DEBUGGER_SESSION_KEY}.history`,
  });

  const aiCapability = describeAICapability(
    aiStatus,
    "Create and edit profiles manually below.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;
  const hiddenWorkspaceDraftSession =
    workspaceDraftSession && !workspaceDraftSession.visible
      ? workspaceDraftSession
      : null;
  const hasDirtySettingsDraft = workspaceDraftSession
    ? isProfileWorkspaceDraftSessionDirty(workspaceDraftSession)
    : false;

  useBeforeUnloadPrompt(hasDirtySettingsDraft);

  const loadProfiles = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await getV1RenderProfiles({
        baseUrl: getApiBaseUrl(),
      });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to load profiles"),
        );
      }
      const nextProfiles = response.data?.profiles || [];
      setProfiles(nextProfiles);
      onInventoryChange?.(nextProfiles.length);
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to load profiles");
      setError(message);
      onError?.(message);
    } finally {
      setLoading(false);
    }
  }, [onError, onInventoryChange]);

  useEffect(() => {
    loadProfiles();
  }, [loadProfiles]);

  const hideWorkspaceDraft = useCallback(() => {
    setWorkspaceDraftSession((current) =>
      current ? { ...current, visible: false } : current,
    );
  }, [setWorkspaceDraftSession]);

  const handleSaveNativeEditSession = async (
    session: ProfileWorkspaceDraftSession,
    input: RenderProfileInput,
  ) => {
    if (session.source !== "native") {
      return;
    }

    const isCreate = session.mode === "create";
    const actionLabel = isCreate
      ? input.name
        ? `Creating ${input.name}`
        : "Creating render profile"
      : `Updating ${session.originalName ?? input.name}`;
    const toastId = toast.show({
      tone: "loading",
      title: actionLabel,
      description: isCreate
        ? "Saving the new render profile configuration."
        : "Saving the latest render profile changes.",
    });
    try {
      setError(null);
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
            isCreate ? "Failed to create profile" : "Failed to update profile",
          ),
        );
      }
      await loadProfiles();
      clearWorkspaceDraftSession();
      toast.update(toastId, {
        tone: "success",
        title: isCreate ? "Render profile created" : "Render profile updated",
        description: isCreate
          ? `${input.name} is now available for fetch configuration.`
          : `${session.originalName ?? input.name} now reflects the latest configuration.`,
      });
    } catch (err) {
      const message = getApiErrorMessage(
        err,
        isCreate ? "Failed to create profile" : "Failed to update profile",
      );
      setError(message);
      onError?.(message);
      toast.update(toastId, {
        tone: "error",
        title: isCreate
          ? "Failed to create render profile"
          : "Failed to update render profile",
        description: message,
      });
    }
  };

  const handleDelete = async (name: string) => {
    const confirmed = await toast.confirm({
      title: `Delete ${name}?`,
      description:
        "This removes the saved render profile from local storage. Jobs that reference it will need a different profile.",
      confirmLabel: "Delete profile",
      cancelLabel: "Keep profile",
      tone: "error",
    });
    if (!confirmed) return;

    const toastId = toast.show({
      tone: "loading",
      title: `Deleting ${name}`,
      description: "Removing the saved render profile.",
    });
    try {
      setError(null);
      const response = await deleteV1RenderProfilesByName({
        baseUrl: getApiBaseUrl(),
        path: { name },
      });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to delete profile"),
        );
      }
      await loadProfiles();
      toast.update(toastId, {
        tone: "success",
        title: "Render profile deleted",
        description: `${name} has been removed.`,
      });
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to delete profile");
      setError(message);
      onError?.(message);
      toast.update(toastId, {
        tone: "error",
        title: "Failed to delete render profile",
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
            ? isProfileWorkspaceDraftSessionDirty(workspaceDraftSession)
              ? "This removes the local Settings draft for the current AI attempt. Your unsaved edits will be lost."
              : "This removes the current AI handoff draft from Settings. You can still reopen the AI modal itself if you keep that session."
            : isProfileWorkspaceDraftSessionDirty(workspaceDraftSession)
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
    profile?: RenderProfile;
  }) => {
    const nextOriginalName =
      options.mode === "edit" ? (options.profile?.name ?? null) : null;
    const nextInitialValue =
      options.mode === "edit" && options.profile
        ? toRenderProfileInput(options.profile)
        : createEmptyRenderProfileInput();

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
      isProfileWorkspaceDraftSessionDirty(workspaceDraftSession) &&
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
      draft: createProfileFormDraft(nextInitialValue),
      visible: true,
    });
  };

  const openAttemptInSettings = async (
    source: AISessionSource,
    attempt: AIAttempt<RenderProfile>,
  ) => {
    if (!attempt.artifact) {
      return;
    }

    if (
      workspaceDraftSession &&
      (workspaceDraftSession.source !== source ||
        workspaceDraftSession.attemptId !== attempt.id) &&
      isProfileWorkspaceDraftSessionDirty(workspaceDraftSession) &&
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
        ? (debuggingProfile?.name ?? attempt.artifact.name)
        : null;
    const nextInitialValue = toRenderProfileInput(attempt.artifact);

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
        draft: createProfileFormDraft(nextInitialValue),
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
    (draft: ProfileFormDraft) => {
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
    session: ProfileWorkspaceDraftSession,
    input: RenderProfileInput,
  ) => {
    if (session.source === "native" || !session.attemptId) {
      return;
    }
    const toastId = toast.show({
      tone: "loading",
      title:
        session.mode === "create"
          ? `Creating ${input.name}`
          : `Updating ${input.name}`,
      description:
        "Saving the manually edited render profile and preserving the AI attempt history.",
    });

    try {
      setError(null);
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

      const savedProfile = (response.data ?? input) as RenderProfile;
      const history =
        session.source === "generator" ? generatorHistory : debuggerHistory;
      history.replaceArtifact(session.attemptId, savedProfile, {
        markManualEdit: true,
      });

      if (session.source === "debugger") {
        setDebuggingProfile(savedProfile);
      }

      await loadProfiles();
      toast.update(toastId, {
        tone: "success",
        title: "Manual edits saved",
        description:
          "The AI attempt now uses your saved render profile as the retry baseline.",
      });
      returnToAISession(session.source, { preserveDraft: false });
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to save render profile");
      setError(message);
      onError?.(message);
      toast.update(toastId, {
        tone: "error",
        title: "Failed to save render profile",
        description: message,
      });
    }
  };

  const handleOpenGenerator = () => {
    hideWorkspaceDraft();
    setIsAIGeneratorOpen(true);
  };

  const handleOpenDebugger = async (profile: RenderProfile) => {
    if (
      debuggingProfile &&
      debuggingProfile.name !== profile.name &&
      !(await toast.confirm({
        title: `Start tuning ${profile.name}?`,
        description: `This replaces the in-progress AI tuning session for ${debuggingProfile.name}. Keep the existing session if you still need that candidate or request draft.`,
        confirmLabel: "Start new tuning session",
        cancelLabel: "Keep existing session",
        tone: "warning",
      }))
    ) {
      return;
    }

    hideWorkspaceDraft();

    if (debuggingProfile && debuggingProfile.name !== profile.name) {
      setDebuggerResetSignal((current) => current + 1);
    }

    setDebuggingProfile(profile);
    setIsAIDebuggerOpen(true);
  };

  if (loading) {
    return <div className="p-4 text-center">Loading profiles...</div>;
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
          <h2 style={{ marginBottom: 4 }}>Render Profiles</h2>
          <p style={{ margin: 0, opacity: 0.8 }}>
            Save reusable fetch and browser overrides only for hosts that need a
            repeatable runtime strategy.
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
            Create Profile
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
                  "a new render profile"
                }${
                  isProfileWorkspaceDraftSessionDirty(
                    hiddenWorkspaceDraftSession,
                  )
                    ? " has unsaved edits."
                    : " is still available in this tab."
                }`
              : `AI handoff draft for Attempt ${hiddenWorkspaceDraftSession.attemptId?.replace(
                  "attempt-",
                  "",
                )}${
                  isProfileWorkspaceDraftSessionDirty(
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
          <ProfileForm
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
          <ProfileForm
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

      {profiles.length === 0 && !workspaceDraftSession ? (
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
      ) : null}

      {showJson && profiles.length > 0 ? (
        <div className="max-h-96 overflow-auto rounded bg-gray-900 p-4 text-green-400">
          <pre className="text-sm">{JSON.stringify(profiles, null, 2)}</pre>
        </div>
      ) : null}

      <AIRenderProfileGenerator
        isOpen={isAIGeneratorOpen}
        aiStatus={aiStatus}
        history={generatorHistory}
        onEditInSettings={(attempt) => {
          void openAttemptInSettings("generator", attempt);
        }}
        onClose={() => setIsAIGeneratorOpen(false)}
        storageKey={RENDER_PROFILE_GENERATOR_SESSION_KEY}
        onSaved={() => {
          void loadProfiles();
        }}
      />

      <AIRenderProfileDebugger
        isOpen={isAIDebuggerOpen}
        aiStatus={aiStatus}
        profile={debuggingProfile}
        history={debuggerHistory}
        onEditInSettings={(attempt) => {
          void openAttemptInSettings("debugger", attempt);
        }}
        onClose={() => {
          setIsAIDebuggerOpen(false);
        }}
        onSaved={() => {
          void loadProfiles();
        }}
        storageKey={RENDER_PROFILE_DEBUGGER_SESSION_KEY}
        resetSignal={debuggerResetSignal}
        onSessionCleared={clearDebuggingProfile}
      />

      <div className="space-y-2">
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
                  void handleOpenDebugger(profile);
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
                  void openNativeEditSession({ mode: "edit", profile });
                }}
                className="text-sm text-blue-600 hover:underline"
              >
                Edit
              </button>
              <button
                type="button"
                onClick={() => handleDelete(profile.name)}
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

interface ProfileFormProps {
  profile?: RenderProfile;
  initialValue?: RenderProfileInput;
  draft?: ProfileFormDraft;
  savedValue?: RenderProfileInput;
  lockName?: boolean;
  title?: string;
  contextNotice?: ReactNode;
  cancelLabel?: string;
  discardLabel?: string;
  submitLabel?: string;
  onDraftChange?: (draft: ProfileFormDraft) => void;
  onSubmit: (input: RenderProfileInput) => void;
  onCancel: () => void;
  onDiscard?: () => void;
}

function ProfileForm({
  profile,
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
}: ProfileFormProps) {
  const seed =
    initialValue ??
    (profile ? toRenderProfileInput(profile) : createEmptyRenderProfileInput());
  const seedDraft = draft ?? createProfileFormDraft(seed);

  const [formData, setFormData] = useState<RenderProfileInput>(
    seedDraft.formData,
  );
  const [hostPatternInput, setHostPatternInput] = useState(
    seedDraft.hostPatternInput,
  );
  const [jsHeavyThresholdInput, setJsHeavyThresholdInput] = useState(
    seedDraft.jsHeavyThresholdInput,
  );
  const [rateLimitQPSInput, setRateLimitQPSInput] = useState(
    seedDraft.rateLimitQPSInput,
  );
  const [rateLimitBurstInput, setRateLimitBurstInput] = useState(
    seedDraft.rateLimitBurstInput,
  );
  const [waitJSON, setWaitJSON] = useState(seedDraft.waitJSON);
  const [blockJSON, setBlockJSON] = useState(seedDraft.blockJSON);
  const [timeoutsJSON, setTimeoutsJSON] = useState(seedDraft.timeoutsJSON);
  const [screenshotJSON, setScreenshotJSON] = useState(
    seedDraft.screenshotJSON,
  );
  const [deviceJSON, setDeviceJSON] = useState(seedDraft.deviceJSON);
  const [formError, setFormError] = useState<string | null>(null);

  const currentDraft = useMemo<ProfileFormDraft>(
    () => ({
      formData,
      hostPatternInput,
      jsHeavyThresholdInput,
      rateLimitQPSInput,
      rateLimitBurstInput,
      waitJSON,
      blockJSON,
      timeoutsJSON,
      screenshotJSON,
      deviceJSON,
    }),
    [
      formData,
      hostPatternInput,
      jsHeavyThresholdInput,
      rateLimitQPSInput,
      rateLimitBurstInput,
      waitJSON,
      blockJSON,
      timeoutsJSON,
      screenshotJSON,
      deviceJSON,
    ],
  );

  useEffect(() => {
    onDraftChange?.(currentDraft);
  }, [currentDraft, onDraftChange]);

  const syncState = useMemo<"clean" | "dirty" | null>(() => {
    const baselineValue = savedValue ?? seed;

    try {
      const workingValue = buildRenderProfileInputFromDraft(currentDraft);
      const isDirty = !deepEqual(workingValue, baselineValue);

      if (isDirty) {
        return "dirty";
      }

      return savedValue ? "clean" : null;
    } catch {
      return "dirty";
    }
  }, [currentDraft, savedValue, seed]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    try {
      onSubmit(buildRenderProfileInputFromDraft(currentDraft));
    } catch (error) {
      setFormError(
        error instanceof Error ? error.message : "Invalid render profile input",
      );
    }
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="space-y-4 rounded border bg-gray-50 p-4"
    >
      <h3 className="font-medium">
        {title ?? (profile ? "Edit Profile" : "Create New Profile")}
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

      {formError ? (
        <div className="error" role="alert">
          {formError}
        </div>
      ) : null}

      <div>
        <label
          htmlFor="profile-name"
          className="mb-1 block text-sm font-medium"
        >
          Name
        </label>
        <input
          id="profile-name"
          type="text"
          value={formData.name}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
          className="w-full rounded border px-3 py-2"
          required
          disabled={lockName || !!profile}
        />
      </div>

      <div>
        <label
          htmlFor="host-patterns"
          className="mb-1 block text-sm font-medium"
        >
          Host Patterns (comma-separated)
        </label>
        <input
          id="host-patterns"
          type="text"
          value={hostPatternInput}
          onChange={(e) => setHostPatternInput(e.target.value)}
          placeholder="example.com, *.example.com"
          className="w-full rounded border px-3 py-2"
          required
        />
        <p className="mt-1 text-xs text-gray-500">
          Examples: example.com, *.example.com, *.api.example.com
        </p>
      </div>

      <div>
        <label
          htmlFor="force-engine"
          className="mb-1 block text-sm font-medium"
        >
          Force Engine
        </label>
        <select
          id="force-engine"
          value={formData.forceEngine || ""}
          onChange={(e) =>
            setFormData({
              ...formData,
              forceEngine: e.target.value
                ? (e.target.value as RenderProfileInput["forceEngine"])
                : undefined,
            })
          }
          className="w-full rounded border px-3 py-2"
        >
          <option value="">Auto-detect</option>
          <option value="http">HTTP</option>
          <option value="chromedp">ChromeDP</option>
          <option value="playwright">Playwright</option>
        </select>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <label className="flex items-center space-x-2">
          <input
            type="checkbox"
            checked={formData.preferHeadless || false}
            onChange={(e) =>
              setFormData({ ...formData, preferHeadless: e.target.checked })
            }
          />
          <span className="text-sm">Prefer Headless</span>
        </label>

        <label className="flex items-center space-x-2">
          <input
            type="checkbox"
            checked={formData.neverHeadless || false}
            onChange={(e) =>
              setFormData({ ...formData, neverHeadless: e.target.checked })
            }
          />
          <span className="text-sm">Never Headless</span>
        </label>

        <label className="flex items-center space-x-2">
          <input
            type="checkbox"
            checked={formData.assumeJsHeavy || false}
            onChange={(e) =>
              setFormData({ ...formData, assumeJsHeavy: e.target.checked })
            }
          />
          <span className="text-sm">Assume JS-Heavy</span>
        </label>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <div>
          <label
            htmlFor="js-heavy-threshold"
            className="mb-1 block text-sm font-medium"
          >
            JS-Heavy Threshold
          </label>
          <input
            id="js-heavy-threshold"
            type="number"
            min="0"
            max="1"
            step="0.01"
            value={jsHeavyThresholdInput}
            onChange={(e) => setJsHeavyThresholdInput(e.target.value)}
            className="w-full rounded border px-3 py-2"
            placeholder="0.50"
          />
        </div>
        <div>
          <label
            htmlFor="rate-limit-qps"
            className="mb-1 block text-sm font-medium"
          >
            Rate Limit QPS
          </label>
          <input
            id="rate-limit-qps"
            type="number"
            min="0"
            step="0.1"
            value={rateLimitQPSInput}
            onChange={(e) => setRateLimitQPSInput(e.target.value)}
            className="w-full rounded border px-3 py-2"
            placeholder="0 = global default"
          />
        </div>
        <div>
          <label
            htmlFor="rate-limit-burst"
            className="mb-1 block text-sm font-medium"
          >
            Rate Limit Burst
          </label>
          <input
            id="rate-limit-burst"
            type="number"
            min="0"
            step="1"
            value={rateLimitBurstInput}
            onChange={(e) => setRateLimitBurstInput(e.target.value)}
            className="w-full rounded border px-3 py-2"
            placeholder="0 = global default"
          />
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <JSONTextarea
          id="render-profile-wait-json"
          label="Wait configuration JSON"
          value={waitJSON}
          onChange={setWaitJSON}
          placeholder={`{\n  "mode": "selector",\n  "selector": "main"\n}`}
          helpText="Optional advanced wait configuration. Leave blank to omit."
        />
        <JSONTextarea
          id="render-profile-block-json"
          label="Block configuration JSON"
          value={blockJSON}
          onChange={setBlockJSON}
          placeholder={`{\n  "resourceTypes": ["image", "font"],\n  "urlPatterns": ["*.tracker.com/*"]\n}`}
          helpText="Optional request blocking rules. Leave blank to omit."
        />
        <JSONTextarea
          id="render-profile-timeouts-json"
          label="Timeout configuration JSON"
          value={timeoutsJSON}
          onChange={setTimeoutsJSON}
          placeholder={`{\n  "maxRenderMs": 30000,\n  "navigationMs": 15000\n}`}
          helpText="Optional per-profile timeout overrides. Leave blank to omit."
        />
        <JSONTextarea
          id="render-profile-screenshot-json"
          label="Screenshot configuration JSON"
          value={screenshotJSON}
          onChange={setScreenshotJSON}
          placeholder={`{\n  "enabled": true,\n  "fullPage": true,\n  "format": "png"\n}`}
          helpText="Optional screenshot capture defaults. Leave blank to omit."
        />
      </div>

      <JSONTextarea
        id="render-profile-device-json"
        label="Device configuration JSON"
        value={deviceJSON}
        onChange={setDeviceJSON}
        placeholder={`{\n  "name": "iPhone 14 Pro",\n  "viewportWidth": 393,\n  "viewportHeight": 852,\n  "deviceScaleFactor": 3,\n  "isMobile": true\n}`}
        helpText="Optional device emulation. Leave blank to omit."
      />

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
          {submitLabel ?? (profile ? "Update" : "Create")}
        </button>
      </div>
    </form>
  );
}

interface JSONTextareaProps {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  helpText: string;
}

function JSONTextarea({
  id,
  label,
  value,
  onChange,
  placeholder,
  helpText,
}: JSONTextareaProps) {
  return (
    <div>
      <label htmlFor={id} className="mb-1 block text-sm font-medium">
        {label}
      </label>
      <textarea
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className="w-full rounded border px-3 py-2 font-mono text-sm"
        rows={8}
      />
      <p className="mt-1 text-xs text-gray-500">{helpText}</p>
    </div>
  );
}
