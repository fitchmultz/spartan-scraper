/**
 * Purpose: Render the shortcut hint UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import { formatShortcut, isMacPlatform } from "../lib/keyboard-shortcuts";

/**
 * Props for the ShortcutHint component.
 */
export type ShortcutHintProps = {
  /** Shortcut string (e.g., "mod+k", "ctrl+enter") */
  shortcut: string;
  /** Additional CSS class names */
  className?: string;
  /** Platform indicator - defaults to auto-detect */
  isMac?: boolean;
};

/**
 * Shortcut Hint Component
 *
 * Displays a keyboard shortcut in a styled kbd element.
 * Automatically detects platform for proper modifier key display.
 *
 * @example
 * ```tsx
 * <button>
 *   Submit <ShortcutHint shortcut="mod+enter" />
 * </button>
 * ```
 */
export function ShortcutHint({
  shortcut,
  className = "",
  isMac: isMacProp,
}: ShortcutHintProps) {
  const isMac = isMacProp ?? isMacPlatform();
  const formatted = formatShortcut(shortcut, isMac);

  return <kbd className={`shortcut-hint ${className}`}>{formatted}</kbd>;
}
