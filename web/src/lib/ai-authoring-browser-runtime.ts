/**
 * Purpose: Centralize browser-runtime state rules and API payload shaping for AI authoring flows.
 * Responsibilities: Define the shared headless/playwright/visual state shape, enforce consistent toggle interactions, and build request-ready payload fields including optional image attachments.
 * Scope: AI authoring browser-runtime helpers only; broader form and batch request shaping lives in sibling utilities.
 * Usage: Import into AI authoring components to initialize local state, update runtime toggles, and assemble API request fields.
 * Invariants/Assumptions: Playwright and visual screenshot context are headless-gated capabilities, disabling headless clears those dependent options, and omitted image payloads stay out of request bodies.
 */

import { toAIImagePayloads, type AttachedAIImage } from "./ai-image-utils";

export interface AIAuthoringBrowserRuntimeState {
  headless: boolean;
  playwright: boolean;
  visual: boolean;
}

export function createAIAuthoringBrowserRuntimeState(): AIAuthoringBrowserRuntimeState {
  return {
    headless: false,
    playwright: false,
    visual: false,
  };
}

export function hasAIAuthoringBrowserRuntimeDraft(
  state: AIAuthoringBrowserRuntimeState,
): boolean {
  return state.headless || state.playwright || state.visual;
}

export function updateAIAuthoringHeadlessState(
  state: AIAuthoringBrowserRuntimeState,
  enabled: boolean,
): AIAuthoringBrowserRuntimeState {
  return {
    ...state,
    headless: enabled,
    playwright: enabled ? state.playwright : false,
    visual: enabled ? state.visual : false,
  };
}

export function updateAIAuthoringPlaywrightState(
  state: AIAuthoringBrowserRuntimeState,
  enabled: boolean,
): AIAuthoringBrowserRuntimeState {
  return {
    ...state,
    headless: enabled ? true : state.headless,
    playwright: enabled,
  };
}

export function updateAIAuthoringVisualState(
  state: AIAuthoringBrowserRuntimeState,
  enabled: boolean,
): AIAuthoringBrowserRuntimeState {
  return {
    ...state,
    headless: enabled ? true : state.headless,
    visual: enabled,
  };
}

export function buildAIAuthoringBrowserRuntimeFields(
  state: AIAuthoringBrowserRuntimeState,
) {
  return {
    headless: state.headless,
    ...(state.headless ? { playwright: state.playwright } : {}),
    visual: state.visual,
  };
}

export function buildAIAuthoringBrowserRuntimePayload(
  state: AIAuthoringBrowserRuntimeState,
  images: AttachedAIImage[],
) {
  return {
    ...(images.length > 0 ? { images: toAIImagePayloads(images) } : {}),
    ...buildAIAuthoringBrowserRuntimeFields(state),
  };
}
