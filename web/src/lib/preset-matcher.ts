/**
 * Purpose: Provide reusable preset matcher helpers for the web app.
 * Responsibilities: Define pure helpers, adapters, and small utility contracts shared across feature modules.
 * Scope: Shared helper logic only; route rendering and persistence stay elsewhere.
 * Usage: Import from adjacent modules that need the helper behavior defined here.
 * Invariants/Assumptions: Helpers should stay side-effect-light and reflect the current product contracts.
 */

import type { JobPreset } from "../types/presets";

/**
 * Checks if a URL matches a preset's URL patterns.
 *
 * @param url - The URL to check
 * @param patterns - Array of regex pattern strings
 * @returns True if any pattern matches
 */
export function matchesUrlPatterns(url: string, patterns: string[]): boolean {
  if (!patterns || patterns.length === 0) {
    return false;
  }

  for (const pattern of patterns) {
    try {
      const regex = new RegExp(pattern, "i");
      if (regex.test(url)) {
        return true;
      }
    } catch {
      // Invalid regex, skip this pattern
    }
  }

  return false;
}

/**
 * Calculate pattern specificity score.
 * More specific patterns (longer, more literal characters) score higher.
 *
 * @param pattern - The regex pattern string
 * @returns Specificity score (higher = more specific)
 */
function getPatternSpecificity(pattern: string): number {
  // Remove regex special characters to count literal characters
  const literalChars = pattern.replace(/[.*+?^${}()|[\]\\]/g, "").length;
  // Longer patterns with more literal characters are more specific
  return literalChars * 2 + pattern.length;
}

/**
 * Get the maximum specificity score for a preset's URL patterns.
 *
 * @param preset - The preset to evaluate
 * @returns Maximum specificity score
 */
function getPresetSpecificity(preset: JobPreset): number {
  if (!preset.urlPatterns || preset.urlPatterns.length === 0) {
    return 0;
  }

  return Math.max(...preset.urlPatterns.map(getPatternSpecificity));
}

/**
 * Detects applicable presets for a given URL based on pattern matching.
 * Returns presets sorted by specificity (most specific first).
 *
 * @param url - The URL to match
 * @param presets - Array of presets to check
 * @returns Array of matching presets, sorted by specificity
 */
export function detectPresetsForUrl(
  url: string,
  presets: JobPreset[],
): JobPreset[] {
  if (!url) {
    return [];
  }

  const matching = presets.filter((preset) => {
    if (!preset.urlPatterns || preset.urlPatterns.length === 0) {
      return false;
    }
    return matchesUrlPatterns(url, preset.urlPatterns);
  });

  // Sort by specificity (most specific patterns first)
  return matching.sort((a, b) => {
    const specificityA = getPresetSpecificity(a);
    const specificityB = getPresetSpecificity(b);
    return specificityB - specificityA;
  });
}

/**
 * Get a human-readable description of estimated resource usage.
 *
 * @param resources - The preset resources
 * @returns Human-readable description
 */
export function getResourceDescription(
  resources: JobPreset["resources"],
): string {
  const parts: string[] = [];

  // Time description
  if (resources.timeSeconds < 60) {
    parts.push(`${resources.timeSeconds}s`);
  } else {
    const minutes = Math.ceil(resources.timeSeconds / 60);
    parts.push(`~${minutes}min`);
  }

  // CPU/Memory indicators
  if (resources.cpu === "high" || resources.memory === "high") {
    parts.push("intensive");
  } else if (resources.cpu === "low" && resources.memory === "low") {
    parts.push("lightweight");
  } else {
    parts.push("moderate");
  }

  return parts.join(", ");
}

/**
 * Get color indicator for resource level.
 *
 * @param level - The resource level
 * @returns CSS color variable or color value
 */
export function getResourceColor(level: "low" | "medium" | "high"): string {
  switch (level) {
    case "low":
      return "var(--success, #22c55e)";
    case "medium":
      return "var(--warning, #f59e0b)";
    case "high":
      return "var(--error, #ef4444)";
    default:
      return "var(--muted)";
  }
}
