/**
 * Tests for shared input parsing helpers.
 *
 * Verifies the common split/trim behavior used by request builders and form
 * adapters.
 */
import { describe, expect, it } from "vitest";

import {
  parseLineSeparatedMap,
  parseOptionalList,
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
