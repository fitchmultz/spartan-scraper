/**
 * Purpose: Render and encode the Settings form for saved render profiles.
 * Responsibilities: Own render-profile draft field state, convert between form inputs and API payloads, surface validation errors, and notify the parent editor when the working draft changes.
 * Scope: Render-profile authoring fields only; inventory loading, AI handoff, and persistence stay with the parent Settings editor.
 * Usage: Mounted by `RenderProfileEditor` for native and AI-backed Settings drafts.
 * Invariants/Assumptions: Name locking is handled by props, optional JSON sections may be blank, and invalid structured, ranged, or integer-only numeric draft input should remain visible and block submit with a field-level error.
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type FormEvent,
  type ReactNode,
  type SetStateAction,
} from "react";

import type { RenderProfile, RenderProfileInput } from "../../api";
import {
  parseOptionalNonNegativeInteger,
  parseOptionalNumberInRange,
} from "../../lib/input-parsing";
import {
  formatCommaSeparatedList,
  formatOptionalJSON,
  getSettingsDraftSyncState,
  parseCommaSeparatedList,
  parseOptionalJSONObject,
  SettingsDraftForm,
} from "../settings/settingsAuthoringForm";

export interface ProfileFormDraft {
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

interface RenderProfileFormProps {
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

function useStateWithErrorReset<T>(
  initialValue: T,
  clearError: () => void,
): [T, (next: SetStateAction<T>) => void] {
  const [value, setValue] = useState(initialValue);

  const setValueAndClearError = useCallback(
    (next: SetStateAction<T>) => {
      setValue(next);
      clearError();
    },
    [clearError],
  );

  return [value, setValueAndClearError];
}

export function createEmptyRenderProfileInput(): RenderProfileInput {
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

export function toRenderProfileInput(
  profile: RenderProfile,
): RenderProfileInput {
  return {
    name: profile.name,
    hostPatterns: [...profile.hostPatterns],
    forceEngine: profile.forceEngine,
    preferHeadless: profile.preferHeadless ? true : undefined,
    neverHeadless: profile.neverHeadless ? true : undefined,
    assumeJsHeavy: profile.assumeJsHeavy ? true : undefined,
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

export function createProfileFormDraft(
  seed: RenderProfileInput,
): ProfileFormDraft {
  return {
    formData: seed,
    hostPatternInput: formatCommaSeparatedList(seed.hostPatterns),
    jsHeavyThresholdInput: seed.jsHeavyThreshold?.toString() || "",
    rateLimitQPSInput: seed.rateLimitQPS?.toString() || "",
    rateLimitBurstInput: seed.rateLimitBurst?.toString() || "",
    waitJSON: formatOptionalJSON(seed.wait),
    blockJSON: formatOptionalJSON(seed.block),
    timeoutsJSON: formatOptionalJSON(seed.timeouts),
    screenshotJSON: formatOptionalJSON(seed.screenshot),
    deviceJSON: formatOptionalJSON(seed.device),
  };
}

export function buildRenderProfileInputFromDraft(
  draft: ProfileFormDraft,
): RenderProfileInput {
  const hostPatterns = parseCommaSeparatedList(draft.hostPatternInput);

  const jsHeavyThreshold = parseOptionalNumberInRange(
    "JS-Heavy Threshold",
    draft.jsHeavyThresholdInput,
    0,
    1,
  );

  const rateLimitQPS = parseOptionalNonNegativeInteger(
    "Rate Limit QPS",
    draft.rateLimitQPSInput,
  );

  const rateLimitBurst = parseOptionalNonNegativeInteger(
    "Rate Limit Burst",
    draft.rateLimitBurstInput,
  );

  return {
    ...draft.formData,
    hostPatterns,
    forceEngine: draft.formData.forceEngine || undefined,
    preferHeadless: draft.formData.preferHeadless ? true : undefined,
    neverHeadless: draft.formData.neverHeadless ? true : undefined,
    assumeJsHeavy: draft.formData.assumeJsHeavy ? true : undefined,
    jsHeavyThreshold,
    rateLimitQPS,
    rateLimitBurst,
    wait: parseOptionalJSONObject<NonNullable<RenderProfileInput["wait"]>>(
      "Wait configuration",
      draft.waitJSON,
    ),
    block: parseOptionalJSONObject<NonNullable<RenderProfileInput["block"]>>(
      "Block configuration",
      draft.blockJSON,
    ),
    timeouts: parseOptionalJSONObject<
      NonNullable<RenderProfileInput["timeouts"]>
    >("Timeout configuration", draft.timeoutsJSON),
    screenshot: parseOptionalJSONObject<
      NonNullable<RenderProfileInput["screenshot"]>
    >("Screenshot configuration", draft.screenshotJSON),
    device: parseOptionalJSONObject<NonNullable<RenderProfileInput["device"]>>(
      "Device configuration",
      draft.deviceJSON,
    ),
  };
}

export function isProfileDraftDirty(
  draft: ProfileFormDraft,
  initialValue: RenderProfileInput,
): boolean {
  return (
    getSettingsDraftSyncState({
      draft,
      initialValue,
      buildValue: buildRenderProfileInputFromDraft,
    }) === "dirty"
  );
}

const PROFILE_FIELD_ERROR_LABELS = [
  "JS-Heavy Threshold",
  "Rate Limit QPS",
  "Rate Limit Burst",
  "Wait configuration",
  "Block configuration",
  "Timeout configuration",
  "Screenshot configuration",
  "Device configuration",
] as const;

type ProfileFieldErrorLabel = (typeof PROFILE_FIELD_ERROR_LABELS)[number];

function getProfileFieldError(
  error: string | null,
  label: ProfileFieldErrorLabel,
): string | null {
  if (!error?.startsWith(`${label} `)) {
    return null;
  }

  return error;
}

function isProfileFieldError(error: string | null): boolean {
  if (!error) {
    return false;
  }

  return PROFILE_FIELD_ERROR_LABELS.some((label) =>
    error.startsWith(`${label} `),
  );
}

export function RenderProfileForm({
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
}: RenderProfileFormProps) {
  const seed = useMemo(
    () =>
      initialValue ??
      (profile
        ? toRenderProfileInput(profile)
        : createEmptyRenderProfileInput()),
    [initialValue, profile],
  );
  const seedDraft = useMemo(
    () => draft ?? createProfileFormDraft(seed),
    [draft, seed],
  );

  const [formError, setFormError] = useState<string | null>(null);
  const clearFormError = useCallback(() => {
    setFormError(null);
  }, []);

  const [formData, setFormData] = useStateWithErrorReset(
    seedDraft.formData,
    clearFormError,
  );
  const [hostPatternInput, setHostPatternInput] = useStateWithErrorReset(
    seedDraft.hostPatternInput,
    clearFormError,
  );
  const [jsHeavyThresholdInput, setJsHeavyThresholdInput] =
    useStateWithErrorReset(seedDraft.jsHeavyThresholdInput, clearFormError);
  const [rateLimitQPSInput, setRateLimitQPSInput] = useStateWithErrorReset(
    seedDraft.rateLimitQPSInput,
    clearFormError,
  );
  const [rateLimitBurstInput, setRateLimitBurstInput] = useStateWithErrorReset(
    seedDraft.rateLimitBurstInput,
    clearFormError,
  );
  const [waitJSON, setWaitJSON] = useStateWithErrorReset(
    seedDraft.waitJSON,
    clearFormError,
  );
  const [blockJSON, setBlockJSON] = useStateWithErrorReset(
    seedDraft.blockJSON,
    clearFormError,
  );
  const [timeoutsJSON, setTimeoutsJSON] = useStateWithErrorReset(
    seedDraft.timeoutsJSON,
    clearFormError,
  );
  const [screenshotJSON, setScreenshotJSON] = useStateWithErrorReset(
    seedDraft.screenshotJSON,
    clearFormError,
  );
  const [deviceJSON, setDeviceJSON] = useStateWithErrorReset(
    seedDraft.deviceJSON,
    clearFormError,
  );
  const jsHeavyThresholdError = getProfileFieldError(
    formError,
    "JS-Heavy Threshold",
  );
  const rateLimitQpsError = getProfileFieldError(formError, "Rate Limit QPS");
  const rateLimitBurstError = getProfileFieldError(
    formError,
    "Rate Limit Burst",
  );
  const waitError = getProfileFieldError(formError, "Wait configuration");
  const blockError = getProfileFieldError(formError, "Block configuration");
  const timeoutsError = getProfileFieldError(
    formError,
    "Timeout configuration",
  );
  const screenshotError = getProfileFieldError(
    formError,
    "Screenshot configuration",
  );
  const deviceError = getProfileFieldError(formError, "Device configuration");
  const topLevelError = isProfileFieldError(formError) ? null : formError;

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
      blockJSON,
      deviceJSON,
      formData,
      hostPatternInput,
      jsHeavyThresholdInput,
      rateLimitBurstInput,
      rateLimitQPSInput,
      screenshotJSON,
      timeoutsJSON,
      waitJSON,
    ],
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
        buildValue: buildRenderProfileInputFromDraft,
      }),
    [currentDraft, savedValue, seed],
  );

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
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
    <SettingsDraftForm
      title={title ?? (profile ? "Edit Profile" : "Create New Profile")}
      syncState={syncState}
      contextNotice={contextNotice}
      error={topLevelError}
      cancelLabel={cancelLabel}
      discardLabel={discardLabel}
      submitLabel={submitLabel ?? (profile ? "Update" : "Create")}
      onSubmit={handleSubmit}
      onCancel={onCancel}
      onDiscard={onDiscard}
    >
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
          onChange={(event) =>
            setFormData({ ...formData, name: event.target.value })
          }
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
          onChange={(event) => setHostPatternInput(event.target.value)}
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
          onChange={(event) =>
            setFormData({
              ...formData,
              forceEngine: event.target.value
                ? (event.target.value as RenderProfileInput["forceEngine"])
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
            onChange={(event) =>
              setFormData({
                ...formData,
                preferHeadless: event.target.checked,
              })
            }
          />
          <span className="text-sm">Prefer Headless</span>
        </label>

        <label className="flex items-center space-x-2">
          <input
            type="checkbox"
            checked={formData.neverHeadless || false}
            onChange={(event) =>
              setFormData({
                ...formData,
                neverHeadless: event.target.checked,
              })
            }
          />
          <span className="text-sm">Never Headless</span>
        </label>

        <label className="flex items-center space-x-2">
          <input
            type="checkbox"
            checked={formData.assumeJsHeavy || false}
            onChange={(event) =>
              setFormData({
                ...formData,
                assumeJsHeavy: event.target.checked,
              })
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
            type="text"
            inputMode="decimal"
            value={jsHeavyThresholdInput}
            onChange={(event) => setJsHeavyThresholdInput(event.target.value)}
            className="w-full rounded border px-3 py-2"
            placeholder="0.50"
            aria-invalid={Boolean(jsHeavyThresholdError)}
            aria-describedby={
              jsHeavyThresholdError ? "js-heavy-threshold-error" : undefined
            }
          />
          {jsHeavyThresholdError ? (
            <p
              id="js-heavy-threshold-error"
              className="mt-1 text-xs text-red-600"
              role="alert"
            >
              {jsHeavyThresholdError}
            </p>
          ) : null}
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
            type="text"
            inputMode="numeric"
            value={rateLimitQPSInput}
            onChange={(event) => setRateLimitQPSInput(event.target.value)}
            className="w-full rounded border px-3 py-2"
            placeholder="0 = global default"
            aria-invalid={Boolean(rateLimitQpsError)}
            aria-describedby={
              rateLimitQpsError ? "rate-limit-qps-error" : undefined
            }
          />
          {rateLimitQpsError ? (
            <p
              id="rate-limit-qps-error"
              className="mt-1 text-xs text-red-600"
              role="alert"
            >
              {rateLimitQpsError}
            </p>
          ) : null}
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
            type="text"
            inputMode="numeric"
            value={rateLimitBurstInput}
            onChange={(event) => setRateLimitBurstInput(event.target.value)}
            className="w-full rounded border px-3 py-2"
            placeholder="0 = global default"
            aria-invalid={Boolean(rateLimitBurstError)}
            aria-describedby={
              rateLimitBurstError ? "rate-limit-burst-error" : undefined
            }
          />
          {rateLimitBurstError ? (
            <p
              id="rate-limit-burst-error"
              className="mt-1 text-xs text-red-600"
              role="alert"
            >
              {rateLimitBurstError}
            </p>
          ) : null}
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <JSONTextarea
          id="render-profile-wait-json"
          label="Wait configuration JSON"
          value={waitJSON}
          onChange={setWaitJSON}
          error={waitError}
          placeholder={`{\n  "mode": "selector",\n  "selector": "main"\n}`}
          helpText="Optional advanced wait configuration. Leave blank to omit."
        />
        <JSONTextarea
          id="render-profile-block-json"
          label="Block configuration JSON"
          value={blockJSON}
          onChange={setBlockJSON}
          error={blockError}
          placeholder={`{\n  "resourceTypes": ["image", "font"],\n  "urlPatterns": ["*.tracker.com/*"]\n}`}
          helpText="Optional request blocking rules. Leave blank to omit."
        />
        <JSONTextarea
          id="render-profile-timeouts-json"
          label="Timeout configuration JSON"
          value={timeoutsJSON}
          onChange={setTimeoutsJSON}
          error={timeoutsError}
          placeholder={`{\n  "maxRenderMs": 30000,\n  "navigationMs": 15000\n}`}
          helpText="Optional per-profile timeout overrides. Leave blank to omit."
        />
        <JSONTextarea
          id="render-profile-screenshot-json"
          label="Screenshot configuration JSON"
          value={screenshotJSON}
          onChange={setScreenshotJSON}
          error={screenshotError}
          placeholder={`{\n  "enabled": true,\n  "fullPage": true,\n  "format": "png"\n}`}
          helpText="Optional screenshot capture defaults. Leave blank to omit."
        />
      </div>

      <JSONTextarea
        id="render-profile-device-json"
        label="Device configuration JSON"
        value={deviceJSON}
        onChange={setDeviceJSON}
        error={deviceError}
        placeholder={`{\n  "name": "iPhone 14 Pro",\n  "viewportWidth": 393,\n  "viewportHeight": 852,\n  "deviceScaleFactor": 3,\n  "isMobile": true\n}`}
        helpText="Optional device emulation. Leave blank to omit."
      />
    </SettingsDraftForm>
  );
}

interface JSONTextareaProps {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  helpText: string;
  error?: string | null;
}

function JSONTextarea({
  id,
  label,
  value,
  onChange,
  placeholder,
  helpText,
  error,
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
        aria-invalid={Boolean(error)}
        aria-describedby={error ? `${id}-error` : undefined}
      />
      {error ? (
        <p
          id={`${id}-error`}
          className="mt-1 text-xs text-red-600"
          role="alert"
        >
          {error}
        </p>
      ) : null}
      <p className="mt-1 text-xs text-gray-500">{helpText}</p>
    </div>
  );
}
