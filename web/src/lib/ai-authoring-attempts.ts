/**
 * Purpose: Normalize AI generation and debugging responses into one shared attempt-history shape.
 * Responsibilities: Preserve artifacts, trust metadata, diagnostics, and raw responses across all AI authoring modals.
 * Scope: Frontend-only normalization for AI modal session state.
 * Usage: Convert API responses before appending them into `useAIAttemptHistory()`.
 * Invariants/Assumptions: Generator responses must include an artifact, and debugger responses may legitimately omit a suggested artifact.
 */

import type {
  AiPipelineJsDebugResponse,
  AiPipelineJsGenerateResponse,
  AiRenderProfileDebugResponse,
  AiRenderProfileGenerateResponse,
  JsTargetScript,
  RenderProfile,
} from "../api";
import type { AIAttemptDraft } from "../hooks/useAIAttemptHistory";

function createDraft<TArtifact>(input: {
  artifact: TArtifact | null;
  guidanceText?: string;
  resolvedGoal?: AIAttemptDraft<TArtifact>["resolvedGoal"];
  explanation?: string;
  routeId?: string;
  provider?: string;
  model?: string;
  visualContextUsed?: boolean;
  issues?: string[];
  recheckStatus?: number;
  recheckEngine?: string;
  recheckError?: string;
  rawResponse: unknown;
}): AIAttemptDraft<TArtifact> {
  return {
    artifact: input.artifact,
    guidanceText: input.guidanceText ?? "",
    resolvedGoal: input.resolvedGoal ?? null,
    explanation: input.explanation ?? "",
    routeId: input.routeId ?? "",
    provider: input.provider ?? "",
    model: input.model ?? "",
    visualContextUsed: input.visualContextUsed ?? false,
    issues: input.issues ?? [],
    recheckStatus: input.recheckStatus,
    recheckEngine: input.recheckEngine,
    recheckError: input.recheckError,
    rawResponse: input.rawResponse,
  };
}

export function toPipelineJsGenerateAttempt(
  response: AiPipelineJsGenerateResponse,
): AIAttemptDraft<JsTargetScript> {
  if (!response.script) {
    throw new Error("No pipeline JS script was generated. Please try again.");
  }

  return createDraft({
    artifact: response.script,
    guidanceText: response.resolved_goal?.text,
    resolvedGoal: response.resolved_goal ?? null,
    explanation: response.explanation,
    routeId: response.route_id,
    provider: response.provider,
    model: response.model,
    visualContextUsed: response.visual_context_used,
    rawResponse: response,
  });
}

export function toRenderProfileGenerateAttempt(
  response: AiRenderProfileGenerateResponse,
): AIAttemptDraft<RenderProfile> {
  if (!response.profile) {
    throw new Error("No render profile was generated. Please try again.");
  }

  return createDraft({
    artifact: response.profile,
    guidanceText: response.resolved_goal?.text,
    resolvedGoal: response.resolved_goal ?? null,
    explanation: response.explanation,
    routeId: response.route_id,
    provider: response.provider,
    model: response.model,
    visualContextUsed: response.visual_context_used,
    rawResponse: response,
  });
}

export function toPipelineJsDebugAttempt(
  response: AiPipelineJsDebugResponse,
): AIAttemptDraft<JsTargetScript> {
  return createDraft({
    artifact: response.suggested_script ?? null,
    guidanceText: response.resolved_goal?.text,
    resolvedGoal: response.resolved_goal ?? null,
    explanation: response.explanation,
    routeId: response.route_id,
    provider: response.provider,
    model: response.model,
    visualContextUsed: response.visual_context_used,
    issues: response.issues ?? [],
    recheckStatus: response.recheck_status,
    recheckEngine: response.recheck_engine,
    recheckError: response.recheck_error,
    rawResponse: response,
  });
}

export function toRenderProfileDebugAttempt(
  response: AiRenderProfileDebugResponse,
): AIAttemptDraft<RenderProfile> {
  return createDraft({
    artifact: response.suggested_profile ?? null,
    guidanceText: response.resolved_goal?.text,
    resolvedGoal: response.resolved_goal ?? null,
    explanation: response.explanation,
    routeId: response.route_id,
    provider: response.provider,
    model: response.model,
    visualContextUsed: response.visual_context_used,
    issues: response.issues ?? [],
    recheckStatus: response.recheck_status,
    recheckEngine: response.recheck_engine,
    recheckError: response.recheck_error,
    rawResponse: response,
  });
}
