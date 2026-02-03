/**
 * Build chunking configuration for Vite/Rollup.
 *
 * This module provides the manualChunks function that determines
 * how modules are split into separate chunks during the build.
 * The goal is to minimize initial bundle size while maintaining
 * reasonable chunk granularity.
 *
 * Chunking strategy:
 * - vendor-react: React core libraries (always needed, loaded first)
 * - vendor-ui: UI primitives (cmdk, radix-ui) - needed for interactions
 * - vendor-onboarding: react-joyride and deps - only needed for first-time users
 * - feature-*: Major feature areas split by directory for code splitting
 * - api: Generated API client code (large type definitions)
 *
 * @module lib/build-chunks
 */

/**
 * Determines the chunk name for a given module ID.
 * Normalizes paths for cross-platform consistency (Windows/Unix).
 *
 * This function is pure and has no side effects, making it safe to
 * use in both the build process and unit tests.
 *
 * @param id - The module ID (typically a file path from Rollup)
 * @returns The chunk name, or undefined to let Rollup use default chunking
 *
 * @example
 * ```ts
 * chunkNameForModuleId("/node_modules/react/index.js")
 * // returns "vendor-react"
 *
 * chunkNameForModuleId("/src/components/results/Explorer.tsx")
 * // returns "feature-results"
 * ```
 */
export function chunkNameForModuleId(id: string): string | undefined {
  // Normalize path separators for cross-platform consistency
  // Windows uses backslashes, Unix uses forward slashes
  const normalizedId = id.replaceAll("\\", "/");

  // Vendor chunks - third-party dependencies from node_modules
  if (normalizedId.includes("node_modules")) {
    // Onboarding library and its dependencies (check first - before generic react check)
    // These are large and only needed for first-time user onboarding
    if (
      normalizedId.includes("react-joyride") ||
      normalizedId.includes("react-floater") ||
      normalizedId.includes("react-focus-lock") ||
      normalizedId.includes("focus-lock") ||
      normalizedId.includes("tabbable")
    ) {
      return "vendor-onboarding";
    }

    // UI primitives for command palette and dialogs (check before generic react)
    if (normalizedId.includes("cmdk") || normalizedId.includes("@radix-ui")) {
      return "vendor-ui";
    }

    // React ecosystem - core framework (always loaded)
    // Check for exact package names to avoid matching react-joyride, etc.
    if (
      normalizedId.includes("/react/") ||
      normalizedId.includes("/react-dom/") ||
      normalizedId.includes("/react/jsx") ||
      normalizedId.includes("/react-dom/")
    ) {
      return "vendor-react";
    }

    // Other third-party libs - let Rollup handle with default heuristics
    // This avoids creating too many small chunks
    return undefined;
  }

  // Internal source chunks - organize by feature directory
  if (normalizedId.includes("/src/")) {
    // API client - contains large generated type definitions
    if (normalizedId.includes("/src/api/")) {
      return "feature-api";
    }

    // Results feature (includes explorer, viewer, etc)
    if (normalizedId.includes("/src/components/results/")) {
      return "feature-results";
    }

    // Template-related components (heavy components used in multiple places)
    if (
      normalizedId.includes("/src/components/templates/") ||
      normalizedId.includes("/src/components/Template") ||
      normalizedId.includes("/src/components/VisualSelectorBuilder")
    ) {
      return "feature-templates";
    }

    // Feed management
    if (normalizedId.includes("/src/components/feeds/")) {
      return "feature-feeds";
    }

    // Watch management
    if (normalizedId.includes("/src/components/watches/")) {
      return "feature-watches";
    }

    // Export schedules
    if (normalizedId.includes("/src/components/export-schedules/")) {
      return "feature-export-schedules";
    }

    // Webhooks
    if (normalizedId.includes("/src/components/webhooks/")) {
      return "feature-webhooks";
    }

    // Chains
    if (normalizedId.includes("/src/components/chains/")) {
      return "feature-chains";
    }

    // Batches
    if (normalizedId.includes("/src/components/batches/")) {
      return "feature-batches";
    }

    // Presets
    if (normalizedId.includes("/src/components/presets/")) {
      return "feature-presets";
    }

    // Jobs (forms, submission)
    if (normalizedId.includes("/src/components/jobs/")) {
      return "feature-jobs";
    }
  }

  // Default: let Rollup decide based on its heuristics
  return undefined;
}
