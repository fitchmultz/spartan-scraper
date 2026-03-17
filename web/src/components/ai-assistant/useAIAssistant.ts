/**
 * Purpose: Provide ergonomic access to the shared integrated AI assistant controller.
 * Responsibilities: Read the provider context and fail fast when the assistant hook is used outside the provider tree.
 * Scope: Hook access to `AIAssistantProvider` state only.
 * Usage: Call `useAIAssistant()` from route surfaces, assistant adapters, and assistant launch buttons.
 * Invariants/Assumptions: Consumers always live beneath `AIAssistantProvider` in the render tree.
 */

import { useContext } from "react";
import { AIAssistantContext } from "./AIAssistantProvider";

export function useAIAssistant() {
  const value = useContext(AIAssistantContext);

  if (!value) {
    throw new Error("useAIAssistant must be used within AIAssistantProvider.");
  }

  return value;
}
