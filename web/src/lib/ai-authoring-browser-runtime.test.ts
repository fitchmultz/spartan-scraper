/**
 * Purpose: Verify shared AI authoring browser-runtime helpers keep state and payload shaping consistent.
 * Responsibilities: Cover headless-gated toggle rules and request payload assembly for image-backed AI authoring flows.
 * Scope: Unit coverage for `ai-authoring-browser-runtime` only.
 * Usage: Runs with the web Vitest suite.
 * Invariants/Assumptions: Headless disables dependent browser capabilities when turned off, and request payloads omit Playwright when headless is off.
 */

import { describe, expect, it } from "vitest";

import {
  buildAIAuthoringBrowserRuntimePayload,
  createAIAuthoringBrowserRuntimeState,
  updateAIAuthoringHeadlessState,
  updateAIAuthoringPlaywrightState,
  updateAIAuthoringVisualState,
} from "./ai-authoring-browser-runtime";

describe("ai-authoring-browser-runtime", () => {
  it("enables headless automatically for dependent capabilities", () => {
    const initial = createAIAuthoringBrowserRuntimeState();

    expect(updateAIAuthoringPlaywrightState(initial, true)).toEqual({
      headless: true,
      playwright: true,
      visual: false,
    });
    expect(updateAIAuthoringVisualState(initial, true)).toEqual({
      headless: true,
      playwright: false,
      visual: true,
    });
  });

  it("clears dependent capabilities when headless is disabled", () => {
    expect(
      updateAIAuthoringHeadlessState(
        { headless: true, playwright: true, visual: true },
        false,
      ),
    ).toEqual({
      headless: false,
      playwright: false,
      visual: false,
    });
  });

  it("builds request payload fields with optional images and gated playwright", () => {
    expect(
      buildAIAuthoringBrowserRuntimePayload(
        { headless: false, playwright: true, visual: false },
        [],
      ),
    ).toEqual({
      headless: false,
      visual: false,
    });

    expect(
      buildAIAuthoringBrowserRuntimePayload(
        { headless: true, playwright: true, visual: true },
        [
          {
            id: "img-1",
            name: "preview.png",
            mimeType: "image/png",
            size: 4,
            data: "ZmFrZQ==",
          },
        ],
      ),
    ).toEqual({
      images: [{ data: "ZmFrZQ==", mime_type: "image/png" }],
      headless: true,
      playwright: true,
      visual: true,
    });
  });
});
