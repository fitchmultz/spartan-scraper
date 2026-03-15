/**
 * Watch utility tests.
 *
 * Covers watch artifact lookup and URL resolution helpers so the watch UI keeps
 * using explicit artifact download URLs instead of raw filesystem paths.
 */

import { describe, expect, it } from "vitest";
import type { WatchCheckResult } from "../api";
import {
  getWatchArtifact,
  getWatchArtifactLabel,
  getWatchArtifactUrl,
} from "./watch-utils";

describe("watch artifact helpers", () => {
  const result: Pick<WatchCheckResult, "artifacts"> = {
    artifacts: [
      {
        kind: "current-screenshot",
        filename: "current.png",
        contentType: "image/png",
        downloadUrl: "/v1/watch/watch-123/artifacts/current-screenshot",
      },
      {
        kind: "visual-diff",
        filename: "diff.png",
        contentType: "image/png",
        downloadUrl: "/v1/watch/watch-123/artifacts/visual-diff",
      },
    ],
  };

  it("finds a watch artifact by kind", () => {
    expect(getWatchArtifact(result, "current-screenshot")?.filename).toBe(
      "current.png",
    );
    expect(getWatchArtifact(result, "previous-screenshot")).toBeUndefined();
  });

  it("builds a browser-safe artifact URL from the API path", () => {
    expect(getWatchArtifactUrl(result.artifacts?.[0])).toBe(
      "/v1/watch/watch-123/artifacts/current-screenshot",
    );
    expect(getWatchArtifactUrl(undefined)).toBe("");
  });

  it("returns human-friendly artifact labels", () => {
    expect(getWatchArtifactLabel("current-screenshot")).toBe(
      "Current Screenshot",
    );
    expect(getWatchArtifactLabel("previous-screenshot")).toBe(
      "Previous Screenshot",
    );
    expect(getWatchArtifactLabel("visual-diff")).toBe("Visual Diff");
  });
});
