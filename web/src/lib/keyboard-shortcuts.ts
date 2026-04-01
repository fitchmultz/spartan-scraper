/**
 * Purpose: Provide reusable keyboard shortcuts helpers for the web app.
 * Responsibilities: Define pure helpers, adapters, and small utility contracts shared across feature modules.
 * Scope: Shared helper logic only; route rendering and persistence stay elsewhere.
 * Usage: Import from adjacent modules that need the helper behavior defined here.
 * Invariants/Assumptions: Helpers should stay side-effect-light and reflect the current product contracts.
 */

/**
 * Detect if user is on macOS.
 */
export function isMacPlatform(): boolean {
  if (typeof navigator === "undefined") return false;
  return navigator.platform.toLowerCase().includes("mac");
}

/**
 * Format shortcut for display.
 * Shows ⌘ for Mac, Ctrl for others.
 *
 * @param shortcut - Shortcut string (e.g., "mod+k", "ctrl+enter", "g j")
 * @param isMac - Whether user is on macOS
 * @returns Formatted shortcut string
 *
 * @example
 * ```ts
 * formatShortcut("mod+k", true)  // "⌘K"
 * formatShortcut("mod+k", false) // "Ctrl+K"
 * formatShortcut("g j", true)    // "G J"
 * ```
 */
export function formatShortcut(shortcut: string, isMac: boolean): string {
  const parts = shortcut.toLowerCase().split(/[\s+]+/);
  return parts
    .map((part) => {
      if (part === "mod") return isMac ? "⌘" : "Ctrl";
      if (part === "meta") return "⌘";
      if (part === "ctrl") return "Ctrl";
      if (part === "alt") return isMac ? "⌥" : "Alt";
      if (part === "shift") return isMac ? "⇧" : "Shift";
      if (part === "enter") return "↵";
      if (part === "escape") return "Esc";
      if (part === " ") return "Space";
      return part.toUpperCase();
    })
    .join(isMac ? "" : "+");
}

/**
 * Normalize shortcut string for comparison.
 * Converts "mod" to platform-specific modifier.
 *
 * @param shortcut - Shortcut string to normalize
 * @param isMac - Whether user is on macOS
 * @returns Normalized shortcut string
 */
export function normalizeShortcut(shortcut: string, isMac: boolean): string {
  return shortcut
    .toLowerCase()
    .replace(/\s+/g, " ")
    .trim()
    .replace(/\bmod\b/g, isMac ? "meta" : "ctrl");
}
