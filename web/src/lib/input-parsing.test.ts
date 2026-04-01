/**
 * Purpose: Verify input parsing behavior with automated regression coverage.
 * Responsibilities: Define focused test cases, fixtures, and assertions for the module under test.
 * Scope: Automated test coverage only; production logic stays in the adjacent source modules.
 * Usage: Run through the repo test entrypoints or the feature-local test command.
 * Invariants/Assumptions: Tests should describe the current contract clearly and remain deterministic under local CI settings.
 */

import { describe, expect, it } from "vitest";

import {
  parseLineSeparatedMap,
  parseOptionalList,
  parseOptionalNonNegativeInteger,
  parseOptionalNumber,
  parseOptionalNumberInRange,
  parseOptionalWholeNumber,
  splitAndTrim,
} from "./input-parsing";

describe("splitAndTrim", () => {
  it("splits and trims values while dropping blanks", () => {
    expect(splitAndTrim(" one, two , , three ", ",")).toEqual([
      "one",
      "two",
      "three",
    ]);
  });
});

describe("parseOptionalNumber", () => {
  it("returns undefined for empty input and parses finite numbers", () => {
    expect(parseOptionalNumber("Rate Limit QPS", "   ")).toBeUndefined();
    expect(parseOptionalNumber("Rate Limit QPS", " 2.5 ")).toBe(2.5);
  });

  it("rejects invalid input", () => {
    expect(() => parseOptionalNumber("Rate Limit QPS", "abc")).toThrow(
      "Rate Limit QPS must be a valid number",
    );
  });
});

describe("parseOptionalWholeNumber", () => {
  it("returns undefined for empty input and accepts integers", () => {
    expect(parseOptionalWholeNumber("Rate Limit Burst", "")).toBeUndefined();
    expect(parseOptionalWholeNumber("Rate Limit Burst", "-3")).toBe(-3);
  });

  it("rejects fractional input", () => {
    expect(() => parseOptionalWholeNumber("Rate Limit Burst", "1.25")).toThrow(
      "Rate Limit Burst must be a whole number",
    );
  });
});

describe("parseOptionalNonNegativeInteger", () => {
  it("returns undefined for empty input and accepts zero or positive integers", () => {
    expect(
      parseOptionalNonNegativeInteger("Min Change Size", "  "),
    ).toBeUndefined();
    expect(parseOptionalNonNegativeInteger("Min Change Size", "0")).toBe(0);
    expect(parseOptionalNonNegativeInteger("Min Change Size", "7")).toBe(7);
  });

  it("rejects negative integers", () => {
    expect(() =>
      parseOptionalNonNegativeInteger("Min Change Size", "-1"),
    ).toThrow("Min Change Size must be non-negative");
  });
});

describe("parseOptionalNumberInRange", () => {
  it("returns undefined for empty input and parses in-range numbers", () => {
    expect(
      parseOptionalNumberInRange("Diff Threshold", "   ", 0, 1),
    ).toBeUndefined();
    expect(parseOptionalNumberInRange("Diff Threshold", "0.5", 0, 1)).toBe(0.5);
  });

  it("rejects out-of-range numbers", () => {
    expect(() =>
      parseOptionalNumberInRange("Diff Threshold", "1.5", 0, 1),
    ).toThrow("Diff Threshold must be between 0 and 1");
  });
});

describe("parseOptionalList", () => {
  it("returns undefined for empty input", () => {
    expect(parseOptionalList("   ", ",")).toBeUndefined();
  });

  it("parses multi-delimiter lists", () => {
    expect(parseOptionalList("a\nb, c", /[\n,]/)).toEqual(["a", "b", "c"]);
  });
});

describe("parseLineSeparatedMap", () => {
  it("parses line separated headers", () => {
    expect(parseLineSeparatedMap("A: 1\nB: 2", ":")).toEqual({
      A: "1",
      B: "2",
    });
  });

  it("ignores malformed rows and returns undefined when nothing is valid", () => {
    expect(parseLineSeparatedMap("missing-separator", ":")).toBeUndefined();
  });
});
