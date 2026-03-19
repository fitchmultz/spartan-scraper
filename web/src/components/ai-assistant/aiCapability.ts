/**
 * Purpose: Normalize AI capability messaging for web surfaces that offer optional AI assistance.
 * Responsibilities: Distinguish intentional AI-disabled states from degraded AI runtime states and produce consistent UI-facing guidance.
 * Scope: Presentation-facing AI capability interpretation only.
 * Usage: Import from components that expose optional AI actions and need consistent disabled/degraded copy.
 * Invariants/Assumptions: Manual workflows remain available even when AI assistance is unavailable.
 */

import type { ComponentStatus } from "../../api";

export interface AICapabilityView {
  unavailable: boolean;
  disabledByChoice: boolean;
  message: string | null;
}

export function describeAICapability(
  aiStatus: ComponentStatus | null | undefined,
  manualFallback: string,
): AICapabilityView {
  if (!aiStatus || aiStatus.status === "ok") {
    return {
      unavailable: false,
      disabledByChoice: false,
      message: null,
    };
  }

  if (aiStatus.status === "disabled") {
    return {
      unavailable: true,
      disabledByChoice: true,
      message: `${aiStatus.message || "AI helpers are optional and currently disabled."} ${manualFallback} Enable AI later only when you want assisted generation or tuning.`,
    };
  }

  return {
    unavailable: true,
    disabledByChoice: false,
    message: `${aiStatus.message || "AI helpers currently need attention."} ${manualFallback} Fix AI when you want assisted generation or tuning again.`,
  };
}
