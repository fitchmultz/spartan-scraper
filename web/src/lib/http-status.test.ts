/**
 * Tests for shared HTTP status presentation helpers.
 *
 * Verifies consistent class mapping for compact and detailed HTTP status
 * displays.
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
