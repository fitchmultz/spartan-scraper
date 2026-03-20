/**
 * Purpose: Provide the Settings-route editor for saved render profiles.
 * Responsibilities: Load the render-profile inventory, coordinate create/edit/delete flows, expose JSON inspection and AI helpers, and route transient operator feedback through the shared toast system.
 * Scope: Browser-side render-profile management only; fetch execution and runtime matching stay outside this component.
 * Usage: Render inside the Settings route with the app-level toast provider already mounted.
 * Invariants/Assumptions: Profiles are persisted through the generated API client, destructive actions require the shared confirmation dialog, and API errors should remain visible in both inline state and toasts.
 */

import { useState, useEffect, useCallback } from "react";
import {
  getV1RenderProfiles,
  postV1RenderProfiles,
  putV1RenderProfilesByName,
  deleteV1RenderProfilesByName,
  type ComponentStatus,
  type RenderProfile,
  type RenderProfileInput,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { AIRenderProfileDebugger } from "../AIRenderProfileDebugger";
import { AIRenderProfileGenerator } from "../AIRenderProfileGenerator";
import { ActionEmptyState } from "../ActionEmptyState";
import { AIUnavailableNotice, describeAICapability } from "../ai-assistant";
import { useToast } from "../toast";

interface RenderProfileEditorProps {
  onError?: (error: string) => void;
  aiStatus?: ComponentStatus | null;
  onInventoryChange?: (count: number) => void;
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
  const [editingProfile, setEditingProfile] = useState<RenderProfile | null>(
    null,
  );
  const [debuggingProfile, setDebuggingProfile] =
    useState<RenderProfile | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [isAIGeneratorOpen, setIsAIGeneratorOpen] = useState(false);
  const [showJson, setShowJson] = useState(false);

  const aiCapability = describeAICapability(
    aiStatus,
    "Create and edit profiles manually below.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;

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

  const handleCreate = async (input: RenderProfileInput) => {
    const toastId = toast.show({
      tone: "loading",
      title: input.name ? `Creating ${input.name}` : "Creating render profile",
      description: "Saving the new render profile configuration.",
    });
    try {
      setError(null);
      const response = await postV1RenderProfiles({
        baseUrl: getApiBaseUrl(),
        body: input,
      });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to create profile"),
        );
      }
      await loadProfiles();
      setIsCreating(false);
      toast.update(toastId, {
        tone: "success",
        title: "Render profile created",
        description: `${input.name} is now available for fetch configuration.`,
      });
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to create profile");
      setError(message);
      onError?.(message);
      toast.update(toastId, {
        tone: "error",
        title: "Failed to create render profile",
        description: message,
      });
    }
  };

  const handleUpdate = async (name: string, input: RenderProfileInput) => {
    const toastId = toast.show({
      tone: "loading",
      title: `Updating ${name}`,
      description: "Saving the latest render profile changes.",
    });
    try {
      setError(null);
      const response = await putV1RenderProfilesByName({
        baseUrl: getApiBaseUrl(),
        path: { name },
        body: input,
      });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to update profile"),
        );
      }
      await loadProfiles();
      setEditingProfile(null);
      toast.update(toastId, {
        tone: "success",
        title: "Render profile updated",
        description: `${name} now reflects the latest configuration.`,
      });
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to update profile");
      setError(message);
      onError?.(message);
      toast.update(toastId, {
        tone: "error",
        title: "Failed to update render profile",
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
            onClick={() => setIsAIGeneratorOpen(true)}
            disabled={aiUnavailable}
            title={aiUnavailableMessage ?? undefined}
            className={aiUnavailable ? "secondary" : undefined}
          >
            Generate with AI
          </button>
          <button type="button" onClick={() => setIsCreating(true)}>
            Create Profile
          </button>
        </div>
      </div>

      {aiUnavailableMessage ? (
        <AIUnavailableNotice message={aiUnavailableMessage} />
      ) : null}

      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}

      {profiles.length === 0 && !isCreating ? (
        <ActionEmptyState
          eyebrow="Optional runtime override"
          title="No saved render profiles yet"
          description="Most jobs can use Spartan's default runtime selection. Add a render profile only when a host needs a stable override for browser usage, engine choice, or rate limits."
          actions={[
            {
              label: "Create your first profile",
              onClick: () => setIsCreating(true),
            },
          ]}
        />
      ) : null}

      {isCreating && (
        <ProfileForm
          onSubmit={handleCreate}
          onCancel={() => setIsCreating(false)}
        />
      )}

      {editingProfile && (
        <ProfileForm
          profile={editingProfile}
          onSubmit={(input) => handleUpdate(editingProfile.name, input)}
          onCancel={() => setEditingProfile(null)}
        />
      )}

      {showJson && profiles.length > 0 && (
        <div className="bg-gray-900 text-green-400 p-4 rounded overflow-auto max-h-96">
          <pre className="text-sm">{JSON.stringify(profiles, null, 2)}</pre>
        </div>
      )}

      <AIRenderProfileGenerator
        isOpen={isAIGeneratorOpen}
        aiStatus={aiStatus}
        onClose={() => setIsAIGeneratorOpen(false)}
        onSaved={() => {
          void loadProfiles();
        }}
      />

      <AIRenderProfileDebugger
        isOpen={debuggingProfile !== null}
        aiStatus={aiStatus}
        profile={debuggingProfile}
        onClose={() => setDebuggingProfile(null)}
        onSaved={() => {
          void loadProfiles();
        }}
      />

      <div className="space-y-2">
        {profiles.map((profile) => (
          <div
            key={profile.name}
            className="p-4 border rounded hover:bg-gray-50 flex justify-between items-start"
          >
            <div>
              <h3 className="font-medium">{profile.name}</h3>
              <p className="text-sm text-gray-600">
                Hosts: {profile.hostPatterns.join(", ")}
              </p>
              {profile.forceEngine && (
                <span className="text-xs bg-blue-100 text-blue-800 px-2 py-0.5 rounded">
                  {profile.forceEngine}
                </span>
              )}
            </div>
            <div className="space-x-2">
              <button
                type="button"
                onClick={() => setDebuggingProfile(profile)}
                disabled={aiUnavailable}
                title={aiUnavailableMessage ?? undefined}
                className={`text-sm ${
                  aiUnavailable
                    ? "text-gray-400 cursor-not-allowed"
                    : "text-purple-600 hover:underline"
                }`}
              >
                Tune with AI
              </button>
              <button
                type="button"
                onClick={() => setEditingProfile(profile)}
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
  onSubmit: (input: RenderProfileInput) => void;
  onCancel: () => void;
}

function ProfileForm({ profile, onSubmit, onCancel }: ProfileFormProps) {
  const [formData, setFormData] = useState<RenderProfileInput>({
    name: profile?.name || "",
    hostPatterns: profile?.hostPatterns || [],
    forceEngine: profile?.forceEngine,
    preferHeadless: profile?.preferHeadless,
    neverHeadless: profile?.neverHeadless,
    assumeJsHeavy: profile?.assumeJsHeavy,
    jsHeavyThreshold: profile?.jsHeavyThreshold,
    rateLimitQPS: profile?.rateLimitQPS,
    rateLimitBurst: profile?.rateLimitBurst,
  });

  const [hostPatternInput, setHostPatternInput] = useState(
    profile?.hostPatterns.join(", ") || "",
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const patterns = hostPatternInput
      .split(",")
      .map((p) => p.trim())
      .filter((p) => p.length > 0);
    onSubmit({ ...formData, hostPatterns: patterns });
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="p-4 border rounded bg-gray-50 space-y-4"
    >
      <h3 className="font-medium">
        {profile ? "Edit Profile" : "Create New Profile"}
      </h3>

      <div>
        <label
          htmlFor="profile-name"
          className="block text-sm font-medium mb-1"
        >
          Name
        </label>
        <input
          id="profile-name"
          type="text"
          value={formData.name}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
          className="w-full px-3 py-2 border rounded"
          required
          disabled={!!profile}
        />
      </div>

      <div>
        <label
          htmlFor="host-patterns"
          className="block text-sm font-medium mb-1"
        >
          Host Patterns (comma-separated)
        </label>
        <input
          id="host-patterns"
          type="text"
          value={hostPatternInput}
          onChange={(e) => setHostPatternInput(e.target.value)}
          placeholder="example.com, *.example.com"
          className="w-full px-3 py-2 border rounded"
          required
        />
        <p className="text-xs text-gray-500 mt-1">
          Examples: example.com, *.example.com, *.api.example.com
        </p>
      </div>

      <div>
        <label
          htmlFor="force-engine"
          className="block text-sm font-medium mb-1"
        >
          Force Engine
        </label>
        <select
          id="force-engine"
          value={formData.forceEngine || ""}
          onChange={(e) =>
            setFormData({
              ...formData,
              forceEngine: e.target.value as RenderProfileInput["forceEngine"],
            })
          }
          className="w-full px-3 py-2 border rounded"
        >
          <option value="">Auto-detect</option>
          <option value="http">HTTP</option>
          <option value="chromedp">ChromeDP</option>
          <option value="playwright">Playwright</option>
        </select>
      </div>

      <div className="grid grid-cols-2 gap-4">
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

      <div className="flex justify-end space-x-2">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 border rounded hover:bg-gray-100"
        >
          Cancel
        </button>
        <button
          type="submit"
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          {profile ? "Update" : "Create"}
        </button>
      </div>
    </form>
  );
}
