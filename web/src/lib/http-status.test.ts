/**
 * Purpose: Verify http status behavior with automated regression coverage.
 * Responsibilities: Define focused test cases, fixtures, and assertions for the module under test.
 * Scope: Automated test coverage only; production logic stays in the adjacent source modules.
 * Usage: Run through the repo test entrypoints or the feature-local test command.
 * Invariants/Assumptions: Tests should describe the current contract clearly and remain deterministic under local CI settings.
 */

import { describe, expect, it } from "vitest";

import {
  getDetailedHttpStatusClass,
  getSimpleHttpStatusClass,
} from "./http-status";

describe("getSimpleHttpStatusClass", () => {
  it("maps standard classes", () => {
    expect(getSimpleHttpStatusClass(200)).toBe("success");
    expect(getSimpleHttpStatusClass(302)).toBe("running");
    expect(getSimpleHttpStatusClass(500)).toBe("failed");
  });

  it("supports empty zero-status badges", () => {
    expect(getSimpleHttpStatusClass(0, { emptyWhenZero: true })).toBe("");
  });
});

describe("getDetailedHttpStatusClass", () => {
  it("maps detailed ranges", () => {
    expect(getDetailedHttpStatusClass(undefined)).toBe("unknown");
    expect(getDetailedHttpStatusClass(204)).toBe("success");
    expect(getDetailedHttpStatusClass(302)).toBe("redirect");
    expect(getDetailedHttpStatusClass(404)).toBe("client-error");
    expect(getDetailedHttpStatusClass(500)).toBe("server-error");
  });
});
