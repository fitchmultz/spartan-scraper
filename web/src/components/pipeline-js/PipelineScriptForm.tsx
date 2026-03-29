/**
 * Purpose: Render and encode the Settings form for saved pipeline JavaScript scripts.
 * Responsibilities: Own script draft field state, convert between form inputs and API payloads, and notify the parent editor when the working Settings draft changes.
 * Scope: Pipeline-script authoring fields only; inventory loading, AI handoff, and persistence stay with the parent Settings editor.
 * Usage: Mounted by `PipelineJSEditor` for native and AI-backed Settings drafts.
 * Invariants/Assumptions: Name locking is controlled by props, selector and host-pattern inputs stay comma-separated, and submit should emit a normalized API payload.
 */

import {
  useEffect,
  useMemo,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";

import type { JsTargetScript, PipelineJsInput } from "../../api";
import {
  formatCommaSeparatedList,
  getSettingsDraftSyncState,
  parseCommaSeparatedList,
  SettingsDraftForm,
} from "../settings/settingsAuthoringForm";

export interface ScriptFormDraft {
  formData: PipelineJsInput;
  hostPatternInput: string;
  selectorInput: string;
}

interface PipelineScriptFormProps {
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

export function createEmptyPipelineJsInput(): PipelineJsInput {
  return {
    name: "",
    hostPatterns: [],
    engine: undefined,
    preNav: undefined,
    postNav: undefined,
    selectors: undefined,
  };
}

export function toPipelineJsInput(script: JsTargetScript): PipelineJsInput {
  return {
    name: script.name,
    hostPatterns: [...script.hostPatterns],
    engine: script.engine,
    preNav: script.preNav,
    postNav: script.postNav,
    selectors: script.selectors ? [...script.selectors] : undefined,
  };
}

export function createScriptFormDraft(seed: PipelineJsInput): ScriptFormDraft {
  return {
    formData: seed,
    hostPatternInput: formatCommaSeparatedList(seed.hostPatterns),
    selectorInput: formatCommaSeparatedList(seed.selectors),
  };
}

export function buildPipelineJsInputFromDraft(
  draft: ScriptFormDraft,
): PipelineJsInput {
  const hostPatterns = parseCommaSeparatedList(draft.hostPatternInput);
  const selectors = parseCommaSeparatedList(draft.selectorInput);

  return {
    ...draft.formData,
    engine: draft.formData.engine || undefined,
    hostPatterns,
    selectors: selectors.length > 0 ? selectors : undefined,
  };
}

export function isScriptDraftDirty(
  draft: ScriptFormDraft,
  initialValue: PipelineJsInput,
): boolean {
  return (
    getSettingsDraftSyncState({
      draft,
      initialValue,
      buildValue: buildPipelineJsInputFromDraft,
    }) === "dirty"
  );
}

export function PipelineScriptForm({
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
}: PipelineScriptFormProps) {
  const seed = useMemo(
    () =>
      initialValue ??
      (script ? toPipelineJsInput(script) : createEmptyPipelineJsInput()),
    [initialValue, script],
  );
  const seedDraft = useMemo(
    () => draft ?? createScriptFormDraft(seed),
    [draft, seed],
  );

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

  const syncState = useMemo(
    () =>
      getSettingsDraftSyncState({
        draft: currentDraft,
        initialValue: seed,
        savedValue,
        buildValue: buildPipelineJsInputFromDraft,
      }),
    [currentDraft, savedValue, seed],
  );

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onSubmit(buildPipelineJsInputFromDraft(currentDraft));
  };

  return (
    <SettingsDraftForm
      title={title ?? (script ? "Edit Script" : "Create New Script")}
      syncState={syncState}
      contextNotice={contextNotice}
      cancelLabel={cancelLabel}
      discardLabel={discardLabel}
      submitLabel={submitLabel ?? (script ? "Update" : "Create")}
      onSubmit={handleSubmit}
      onCancel={onCancel}
      onDiscard={onDiscard}
    >
      <div>
        <label htmlFor="script-name" className="mb-1 block text-sm font-medium">
          Name
        </label>
        <input
          id="script-name"
          type="text"
          value={formData.name}
          onChange={(event) =>
            setFormData({ ...formData, name: event.target.value })
          }
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
          onChange={(event) => setHostPatternInput(event.target.value)}
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
          onChange={(event) =>
            setFormData({
              ...formData,
              engine: event.target.value
                ? (event.target.value as PipelineJsInput["engine"])
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
          onChange={(event) =>
            setFormData({
              ...formData,
              preNav: event.target.value || undefined,
            })
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
          onChange={(event) =>
            setFormData({
              ...formData,
              postNav: event.target.value || undefined,
            })
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
          onChange={(event) => setSelectorInput(event.target.value)}
          placeholder="#content, .article, [data-loaded]"
          className="w-full rounded border px-3 py-2"
        />
        <p className="mt-1 text-xs text-gray-500">
          CSS selectors to wait for before considering page loaded
        </p>
      </div>
    </SettingsDraftForm>
  );
}
