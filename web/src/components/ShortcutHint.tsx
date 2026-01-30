/**
 * Shortcut Hint Component
 *
 * Reusable component for displaying keyboard shortcut hints inline.
 * Shows platform-appropriate shortcuts (⌘ for Mac, Ctrl for others).
 *
 * @module ShortcutHint
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
