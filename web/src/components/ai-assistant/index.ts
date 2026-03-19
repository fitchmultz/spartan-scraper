/**
 * Purpose: Re-export the integrated AI assistant provider, shell, adapters, and hook from one module boundary.
 * Responsibilities: Keep route surfaces and tests importing a stable assistant module path.
 * Scope: AI assistant barrel exports only.
 * Usage: Import from `../ai-assistant` or `./ai-assistant` in web components.
 * Invariants/Assumptions: Route adapters remain the supported integration points for route-aware AI behavior.
 */

export * from "./AIAssistantProvider";
export * from "./AIAssistantPanel";
export * from "./AIUnavailableNotice";
export * from "./aiCapability";
export * from "./JobSubmissionAssistantSection";
export * from "./TemplateAssistantSection";
export * from "./ResultsAssistantSection";
export * from "./useAIAssistant";
