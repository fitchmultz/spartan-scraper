/**
 * PipelineJSEditor - Web UI component for managing pipeline JavaScript scripts
 *
 * Responsibilities:
 * - Display list of pipeline JS scripts with create/edit/delete capabilities
 * - Form for creating/editing JS script configuration
 * - JSON preview for advanced users
 * - Empty state guidance
 *
 * This component does NOT:
 * - Execute JavaScript code
 * - Handle script matching at runtime
 */

import { useState, useEffect, useCallback } from "react";
import {
  getV1PipelineJs,
  postV1PipelineJs,
  putV1PipelineJsByName,
  deleteV1PipelineJsByName,
  type JsTargetScript,
  type PipelineJsInput,
} from "../../api";
import { getApiErrorMessage } from "../../lib/api-errors";

interface PipelineJSEditorProps {
  onError?: (error: string) => void;
}

export function PipelineJSEditor({ onError }: PipelineJSEditorProps) {
  const [scripts, setScripts] = useState<JsTargetScript[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingScript, setEditingScript] = useState<JsTargetScript | null>(
    null,
  );
  const [isCreating, setIsCreating] = useState(false);
  const [showJson, setShowJson] = useState(false);

  const loadScripts = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await getV1PipelineJs();
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to load scripts"),
        );
      }
      setScripts(response.data?.scripts || []);
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to load scripts");
      setError(message);
      onError?.(message);
    } finally {
      setLoading(false);
    }
  }, [onError]);

  useEffect(() => {
    loadScripts();
  }, [loadScripts]);

  const handleCreate = async (input: PipelineJsInput) => {
    try {
      setError(null);
      const response = await postV1PipelineJs({ body: input });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to create script"),
        );
      }
      await loadScripts();
      setIsCreating(false);
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to create script");
      setError(message);
      onError?.(message);
    }
  };

  const handleUpdate = async (name: string, input: PipelineJsInput) => {
    try {
      setError(null);
      const response = await putV1PipelineJsByName({
        path: { name },
        body: input,
      });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to update script"),
        );
      }
      await loadScripts();
      setEditingScript(null);
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to update script");
      setError(message);
      onError?.(message);
    }
  };

  const handleDelete = async (name: string) => {
    if (!confirm(`Delete script "${name}"?`)) return;
    try {
      setError(null);
      const response = await deleteV1PipelineJsByName({ path: { name } });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to delete script"),
        );
      }
      await loadScripts();
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to delete script");
      setError(message);
      onError?.(message);
    }
  };

  if (loading) {
    return <div className="p-4 text-center">Loading scripts...</div>;
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-semibold">Pipeline JavaScript</h2>
        <div className="space-x-2">
          <button
            type="button"
            onClick={() => setShowJson(!showJson)}
            className="px-3 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded"
          >
            {showJson ? "Hide JSON" : "Show JSON"}
          </button>
          <button
            type="button"
            onClick={() => setIsCreating(true)}
            className="px-3 py-1 text-sm bg-blue-600 text-white hover:bg-blue-700 rounded"
          >
            Create Script
          </button>
        </div>
      </div>

      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}

      {scripts.length === 0 && !isCreating && (
        <div className="p-8 text-center bg-gray-50 rounded-lg border-2 border-dashed">
          <p className="text-gray-600 mb-4">
            No pipeline JavaScript scripts configured
          </p>
          <p className="text-sm text-gray-500 mb-4">
            Pipeline JS scripts run custom JavaScript on matching pages before
            or after navigation
          </p>
          <button
            type="button"
            onClick={() => setIsCreating(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            Create Your First Script
          </button>
        </div>
      )}

      {isCreating && (
        <ScriptForm
          onSubmit={handleCreate}
          onCancel={() => setIsCreating(false)}
        />
      )}

      {editingScript && (
        <ScriptForm
          script={editingScript}
          onSubmit={(input) => handleUpdate(editingScript.name, input)}
          onCancel={() => setEditingScript(null)}
        />
      )}

      {showJson && scripts.length > 0 && (
        <div className="bg-gray-900 text-green-400 p-4 rounded overflow-auto max-h-96">
          <pre className="text-sm">{JSON.stringify(scripts, null, 2)}</pre>
        </div>
      )}

      <div className="space-y-2">
        {scripts.map((script) => (
          <div
            key={script.name}
            className="p-4 border rounded hover:bg-gray-50 flex justify-between items-start"
          >
            <div className="flex-1">
              <h3 className="font-medium">{script.name}</h3>
              <p className="text-sm text-gray-600">
                Hosts: {script.hostPatterns.join(", ")}
              </p>
              {script.engine && (
                <span className="text-xs bg-purple-100 text-purple-800 px-2 py-0.5 rounded mr-2">
                  {script.engine}
                </span>
              )}
              {script.preNav && (
                <span className="text-xs bg-green-100 text-green-800 px-2 py-0.5 rounded mr-2">
                  pre-nav
                </span>
              )}
              {script.postNav && (
                <span className="text-xs bg-blue-100 text-blue-800 px-2 py-0.5 rounded mr-2">
                  post-nav
                </span>
              )}
              {script.selectors && script.selectors.length > 0 && (
                <span className="text-xs bg-orange-100 text-orange-800 px-2 py-0.5 rounded">
                  {script.selectors.length} selector
                  {script.selectors.length !== 1 ? "s" : ""}
                </span>
              )}
            </div>
            <div className="space-x-2">
              <button
                type="button"
                onClick={() => setEditingScript(script)}
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
  onSubmit: (input: PipelineJsInput) => void;
  onCancel: () => void;
}

function ScriptForm({ script, onSubmit, onCancel }: ScriptFormProps) {
  const [formData, setFormData] = useState<PipelineJsInput>({
    name: script?.name || "",
    hostPatterns: script?.hostPatterns || [],
    engine: script?.engine,
    preNav: script?.preNav,
    postNav: script?.postNav,
    selectors: script?.selectors,
  });

  const [hostPatternInput, setHostPatternInput] = useState(
    script?.hostPatterns.join(", ") || "",
  );
  const [selectorInput, setSelectorInput] = useState(
    script?.selectors?.join(", ") || "",
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const patterns = hostPatternInput
      .split(",")
      .map((p: string) => p.trim())
      .filter((p) => p.length > 0);
    const selectors = selectorInput
      .split(",")
      .map((s: string) => s.trim())
      .filter((s) => s.length > 0);
    onSubmit({
      ...formData,
      hostPatterns: patterns,
      selectors: selectors.length > 0 ? selectors : undefined,
    });
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="p-4 border rounded bg-gray-50 space-y-4"
    >
      <h3 className="font-medium">
        {script ? "Edit Script" : "Create New Script"}
      </h3>

      <div>
        <label htmlFor="script-name" className="block text-sm font-medium mb-1">
          Name
        </label>
        <input
          id="script-name"
          type="text"
          value={formData.name}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
          className="w-full px-3 py-2 border rounded"
          required
          disabled={!!script}
        />
      </div>

      <div>
        <label
          htmlFor="script-host-patterns"
          className="block text-sm font-medium mb-1"
        >
          Host Patterns (comma-separated)
        </label>
        <input
          id="script-host-patterns"
          type="text"
          value={hostPatternInput}
          onChange={(e) => setHostPatternInput(e.target.value)}
          placeholder="example.com, *.example.com"
          className="w-full px-3 py-2 border rounded"
          required
        />
        <p className="text-xs text-gray-500 mt-1">
          Examples: example.com, *.example.com
        </p>
      </div>

      <div>
        <label
          htmlFor="script-engine"
          className="block text-sm font-medium mb-1"
        >
          Engine
        </label>
        <select
          id="script-engine"
          value={formData.engine || ""}
          onChange={(e) =>
            setFormData({
              ...formData,
              engine: e.target.value as PipelineJsInput["engine"],
            })
          }
          className="w-full px-3 py-2 border rounded"
        >
          <option value="">Any</option>
          <option value="chromedp">ChromeDP</option>
          <option value="playwright">Playwright</option>
        </select>
      </div>

      <div>
        <label
          htmlFor="script-pre-nav"
          className="block text-sm font-medium mb-1"
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
          className="w-full px-3 py-2 border rounded font-mono text-sm"
          rows={4}
        />
      </div>

      <div>
        <label
          htmlFor="script-post-nav"
          className="block text-sm font-medium mb-1"
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
          className="w-full px-3 py-2 border rounded font-mono text-sm"
          rows={4}
        />
      </div>

      <div>
        <label
          htmlFor="script-selectors"
          className="block text-sm font-medium mb-1"
        >
          Wait Selectors (comma-separated)
        </label>
        <input
          id="script-selectors"
          type="text"
          value={selectorInput}
          onChange={(e) => setSelectorInput(e.target.value)}
          placeholder="#content, .article, [data-loaded]"
          className="w-full px-3 py-2 border rounded"
        />
        <p className="text-xs text-gray-500 mt-1">
          CSS selectors to wait for before considering page loaded
        </p>
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
          {script ? "Update" : "Create"}
        </button>
      </div>
    </form>
  );
}
