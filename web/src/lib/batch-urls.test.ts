import { describe, expect, it } from "vitest";

import { parseBatchUrls, summarizeBatchUrls } from "./batch-urls";

describe("parseBatchUrls", () => {
  it("counts comma-separated and newline-separated URLs", () => {
    expect(
      parseBatchUrls(
        "https://example.com, https://example.org\nhttps://example.net",
      ),
    ).toEqual([
      "https://example.com",
      "https://example.org",
      "https://example.net",
    ]);
  });

  it("ignores whitespace and duplicate separators", () => {
    expect(
      parseBatchUrls(
        "  https://example.com  , ,\n\nhttps://example.org\r\n,   https://example.net   ",
      ),
    ).toEqual([
      "https://example.com",
      "https://example.org",
      "https://example.net",
    ]);
  });

  it("returns an empty list when the field is cleared", () => {
    expect(parseBatchUrls("   \n ,  ")).toEqual([]);
  });
});

describe("summarizeBatchUrls", () => {
  it("returns visible URLs and a remainder count", () => {
    expect(
      summarizeBatchUrls(
        [
          "https://example.com",
          "https://example.org",
          "https://example.net",
          "https://example.edu",
        ],
        3,
      ),
    ).toEqual({
      visible: [
        "https://example.com",
        "https://example.org",
        "https://example.net",
      ],
      remaining: 1,
    });
  });
});
