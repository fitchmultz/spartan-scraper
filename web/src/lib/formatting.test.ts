/**
 * Tests for shared formatting helpers.
 *
 * Verifies shared date, duration, and truncation behavior used by multiple UI
 * surfaces.
 */
import { describe, expect, it } from "vitest";

import {
  formatDateTime,
  formatDisplayValue,
  formatMillisecondsAsDuration,
  formatSecondsAsApproximateDuration,
  formatSecondsAsDuration,
  truncateEnd,
  truncateMiddle,
} from "./formatting";

describe("formatDateTime", () => {
  it("returns the configured fallback for empty values", () => {
    expect(formatDateTime(undefined, "Never")).toBe("Never");
  });

  it("returns the original string for invalid dates", () => {
    expect(formatDateTime("not-a-date")).toBe("not-a-date");
  });
});

describe("formatSecondsAsDuration", () => {
  it("formats seconds across supported units", () => {
    expect(formatSecondsAsDuration(45)).toBe("45s");
    expect(formatSecondsAsDuration(120)).toBe("2m");
    expect(formatSecondsAsDuration(7200)).toBe("2h");
    expect(formatSecondsAsDuration(172800)).toBe("2d");
  });
});

describe("formatSecondsAsApproximateDuration", () => {
  it("formats seconds with rounded-up units for quick previews", () => {
    expect(formatSecondsAsApproximateDuration(45)).toBe("45s");
    expect(formatSecondsAsApproximateDuration(61)).toBe("~2min");
    expect(formatSecondsAsApproximateDuration(3601)).toBe("~2h");
    expect(formatSecondsAsApproximateDuration(90000)).toBe("~2d");
  });
});

describe("formatMillisecondsAsDuration", () => {
  it("formats missing and small durations safely", () => {
    expect(formatMillisecondsAsDuration(undefined)).toBe("-");
    expect(formatMillisecondsAsDuration(0.2)).toBe("<1ms");
    expect(formatMillisecondsAsDuration(25)).toBe("25ms");
    expect(formatMillisecondsAsDuration(2500)).toBe("2.50s");
  });
});

describe("formatDisplayValue", () => {
  it("uses configurable labels for empty and boolean values", () => {
    expect(
      formatDisplayValue(undefined, {
        emptyLabel: "Not set",
        trueLabel: "Yes",
        falseLabel: "No",
      }),
    ).toBe("Not set");
    expect(
      formatDisplayValue(true, {
        trueLabel: "Yes",
        falseLabel: "No",
      }),
    ).toBe("Yes");
    expect(
      formatDisplayValue(false, {
        trueLabel: "Yes",
        falseLabel: "No",
      }),
    ).toBe("No");
  });

  it("formats arrays, objects, and long strings safely", () => {
    expect(formatDisplayValue(["a", "b"])).toBe("[2 items]");
    expect(formatDisplayValue({ ok: true })).toBe("{...}");
    expect(
      formatDisplayValue("abcdefghijklmnopqrstuvwxyz", { maxLength: 10 }),
    ).toBe("abcdefghij...");
  });

  it("can preserve null and undefined distinctions when needed", () => {
    expect(formatDisplayValue(undefined, { undefinedLabel: "undefined" })).toBe(
      "undefined",
    );
    expect(formatDisplayValue(null, { nullLabel: "null" })).toBe("null");
  });
});

describe("truncate helpers", () => {
  it("truncates long strings while preserving shape", () => {
    expect(truncateMiddle("abcdefghijklmnopqrstuvwxyz", 10)).toBe("abc...xyz");
    expect(truncateEnd("abcdefghijklmnopqrstuvwxyz", 10)).toBe("abcdefghij...");
  });

  it("returns fallbacks for empty values", () => {
    expect(truncateMiddle(undefined)).toBe("-");
    expect(truncateEnd(undefined)).toBe("-");
  });
});
