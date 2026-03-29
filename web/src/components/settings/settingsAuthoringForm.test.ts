/**
 * Purpose: Verify shared Settings authoring field helpers preserve honest codec semantics and draft sync state.
 * Responsibilities: Assert optional JSON formatting keeps valid falsy values and sync-state calculation treats explicit saved values distinctly from unsaved drafts.
 * Scope: `settingsAuthoringForm.tsx` helpers only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Shared Settings authoring helpers must stay safe to reuse outside the current object-only editor surfaces.
 */

import { describe, expect, it } from "vitest";

import {
  formatOptionalJSON,
  getSettingsDraftSyncState,
  parseOptionalJSONObject,
  parseOptionalNumber,
} from "./settingsAuthoringForm";

describe("settingsAuthoringForm helpers", () => {
  it("preserves valid falsy JSON values when formatting optional JSON", () => {
    expect(formatOptionalJSON(false)).toBe("false");
    expect(formatOptionalJSON(0)).toBe("0");
    expect(formatOptionalJSON(null)).toBe("");
  });

  it("treats explicit falsy saved values as in sync", () => {
    expect(
      getSettingsDraftSyncState({
        draft: { next: 0 },
        initialValue: 1,
        savedValue: 0,
        buildValue: (draft) => draft.next,
      }),
    ).toBe("clean");
  });

  it("rejects non-object JSON when an object-only field is expected", () => {
    expect(() =>
      parseOptionalJSONObject("Wait configuration", "false"),
    ).toThrow("Wait configuration must be a JSON object");
    expect(() => parseOptionalJSONObject("Wait configuration", "[]")).toThrow(
      "Wait configuration must be a JSON object",
    );
  });

  it("rejects invalid optional numeric input instead of silently clearing it", () => {
    expect(parseOptionalNumber("Rate Limit QPS", "  ")).toBeUndefined();
    expect(() => parseOptionalNumber("Rate Limit QPS", "abc")).toThrow(
      "Rate Limit QPS must be a valid number",
    );
  });
});
