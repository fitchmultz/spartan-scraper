/**
 * Keyboard Shortcuts Management Hook
 *
 * Manages global keyboard shortcuts with localStorage persistence for user preferences.
 * Provides a command palette trigger, form submission shortcuts, navigation, and help.
 * Normalizes shortcuts across platforms (Mac/Windows/Linux).
 *
 * @module useKeyboard
 */

import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import {
  isMacPlatform,
  formatShortcut,
  normalizeShortcut,
} from "../lib/keyboard-shortcuts";

export type ShortcutConfig = {
  /** Open command palette (default: mod+k) */
  commandPalette: string;
  /** Submit current form (default: mod+enter) */
  submitForm: string;
  /** Focus search/filter (default: /) */
  search: string;
  /** Open help/cheatsheet (default: ?) */
  help: string;
  /** Close modal/palette (default: escape) */
  escape: string;
  /** Navigate to jobs section (default: g j) */
  navigateJobs: string;
  /** Navigate to results section (default: g r) */
  navigateResults: string;
  /** Navigate to forms section (default: g f) */
  navigateForms: string;
};

export type KeyboardState = {
  /** Current shortcut configuration */
  shortcuts: ShortcutConfig;
  /** Whether command palette is open */
  isCommandPaletteOpen: boolean;
  /** Whether help modal is open */
  isHelpOpen: boolean;
  /** Update a specific shortcut */
  updateShortcut: (key: keyof ShortcutConfig, value: string) => void;
  /** Reset all shortcuts to defaults */
  resetShortcuts: () => void;
  /** Open command palette */
  openCommandPalette: () => void;
  /** Close command palette */
  closeCommandPalette: () => void;
  /** Open help modal */
  openHelp: () => void;
  /** Close help modal */
  closeHelp: () => void;
  /** Toggle help modal */
  toggleHelp: () => void;
  /** Check if user is on Mac */
  isMac: boolean;
  /** Format shortcut for display (e.g., "mod+k" -> "⌘K" or "Ctrl+K") */
  formatShortcut: (shortcut: string) => string;
};

const STORAGE_KEY = "spartan-keyboard-shortcuts";

const DEFAULT_SHORTCUTS: ShortcutConfig = {
  commandPalette: "mod+k",
  submitForm: "mod+enter",
  search: "/",
  help: "?",
  escape: "escape",
  navigateJobs: "g j",
  navigateResults: "g r",
  navigateForms: "g f",
};

// Re-export for convenience
export { isMacPlatform, formatShortcut, normalizeShortcut };

/**
 * Get stored shortcuts from localStorage.
 */
function getStoredShortcuts(): Partial<ShortcutConfig> | null {
  if (typeof window === "undefined") return null;
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored);
      // Validate that parsed object has expected keys
      if (typeof parsed === "object" && parsed !== null) {
        return parsed as Partial<ShortcutConfig>;
      }
    }
  } catch {
    // localStorage may be unavailable or data corrupted
  }
  return null;
}

/**
 * Store shortcuts in localStorage.
 */
function storeShortcuts(shortcuts: ShortcutConfig): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(shortcuts));
  } catch {
    // localStorage may be unavailable
  }
}

/**
 * Check if an element is an input, textarea, or contenteditable.
 */
function isInputElement(element: EventTarget | null): boolean {
  if (!(element instanceof HTMLElement)) return false;
  const tagName = element.tagName.toLowerCase();
  const isContentEditable = element.isContentEditable;
  return tagName === "input" || tagName === "textarea" || isContentEditable;
}

/**
 * Match a keyboard event against a shortcut string.
 */
function matchesShortcut(
  event: KeyboardEvent,
  shortcut: string,
  isMac: boolean,
): boolean {
  const normalized = normalizeShortcut(shortcut, isMac);
  const parts = normalized.split(/[\s+]+/);

  // Check for sequence shortcuts (e.g., "g j")
  if (parts.length === 2 && parts.every((p) => p.length === 1)) {
    // Sequence shortcuts are handled separately via state
    return false;
  }

  const key = parts[parts.length - 1];
  const modifiers = parts.slice(0, -1);

  // Check if key matches
  const keyMatches =
    event.key.toLowerCase() === key ||
    (key === "meta" && event.key === "Meta") ||
    (key === "ctrl" && event.key === "Control") ||
    (key === "alt" && event.key === "Alt") ||
    (key === "shift" && event.key === "Shift") ||
    (key === "enter" && event.key === "Enter") ||
    (key === "escape" && event.key === "Escape") ||
    (key === "?" && event.key === "?") ||
    (key === "/" && event.key === "/");

  if (!keyMatches) return false;

  // Check modifiers
  const hasMeta = modifiers.includes("meta");
  const hasCtrl = modifiers.includes("ctrl");
  const hasAlt = modifiers.includes("alt");
  const hasShift = modifiers.includes("shift");

  if (hasMeta !== event.metaKey) return false;
  if (hasCtrl !== event.ctrlKey) return false;
  if (hasAlt !== event.altKey) return false;

  // `?` requires Shift on many keyboard layouts even when the
  // logical shortcut should be represented as a single key.
  const allowsImplicitShift =
    key === "?" && !hasShift && !hasMeta && !hasCtrl && !hasAlt;

  if (!allowsImplicitShift && hasShift !== event.shiftKey) return false;

  return true;
}

/**
 * React hook for global keyboard shortcut management.
 *
 * Features:
 * - localStorage persistence for custom shortcuts
 * - Platform-aware shortcut normalization (mod = Cmd on Mac, Ctrl elsewhere)
 * - Sequence shortcuts (e.g., "g j" for "go to jobs")
 * - Input field detection to avoid triggering shortcuts while typing
 * - Prevent default for browser-conflicting shortcuts
 *
 * @example
 * ```tsx
 * function App() {
 *   const {
 *     isCommandPaletteOpen,
 *     openCommandPalette,
 *     closeCommandPalette,
 *     shortcuts,
 *     formatShortcut
 *   } = useKeyboard();
 *
 *   return (
 *     <>
 *       <button onClick={openCommandPalette}>
 *         Command Palette {formatShortcut(shortcuts.commandPalette)}
 *       </button>
 *       {isCommandPaletteOpen && <CommandPalette onClose={closeCommandPalette} />}
 *     </>
 *   );
 * }
 * ```
 */
export function useKeyboard(): KeyboardState {
  const isMac = useMemo(() => isMacPlatform(), []);
  const [shortcuts, setShortcuts] = useState<ShortcutConfig>(DEFAULT_SHORTCUTS);
  const [isCommandPaletteOpen, setIsCommandPaletteOpen] = useState(false);
  const [isHelpOpen, setIsHelpOpen] = useState(false);

  // Sequence shortcut state (for "g j" style shortcuts)
  const sequenceBuffer = useRef<string>("");
  const sequenceTimeout = useRef<number | null>(null);

  // Load stored shortcuts on mount
  useEffect(() => {
    const stored = getStoredShortcuts();
    if (stored) {
      setShortcuts((prev) => ({ ...prev, ...stored }));
    }
  }, []);

  // Persist shortcuts when they change
  useEffect(() => {
    storeShortcuts(shortcuts);
  }, [shortcuts]);

  /**
   * Update a specific shortcut.
   */
  const updateShortcut = useCallback(
    (key: keyof ShortcutConfig, value: string) => {
      setShortcuts((prev) => ({ ...prev, [key]: value }));
    },
    [],
  );

  /**
   * Reset all shortcuts to defaults.
   */
  const resetShortcuts = useCallback(() => {
    setShortcuts(DEFAULT_SHORTCUTS);
  }, []);

  /**
   * Open command palette.
   */
  const openCommandPalette = useCallback(() => {
    setIsCommandPaletteOpen(true);
  }, []);

  /**
   * Close command palette.
   */
  const closeCommandPalette = useCallback(() => {
    setIsCommandPaletteOpen(false);
  }, []);

  /**
   * Open help modal.
   */
  const openHelp = useCallback(() => {
    setIsHelpOpen(true);
  }, []);

  /**
   * Close help modal.
   */
  const closeHelp = useCallback(() => {
    setIsHelpOpen(false);
  }, []);

  /**
   * Toggle help modal.
   */
  const toggleHelp = useCallback(() => {
    setIsHelpOpen((prev) => !prev);
  }, []);

  /**
   * Format shortcut for display.
   */
  const formatShortcutForDisplay = useCallback(
    (shortcut: string) => formatShortcut(shortcut, isMac),
    [isMac],
  );

  // Global keyboard event handler
  useEffect(() => {
    if (typeof window === "undefined") return;

    const handleKeyDown = (event: KeyboardEvent) => {
      // Don't trigger shortcuts when typing in inputs (except for Escape and mod+ shortcuts)
      const inInput = isInputElement(event.target as HTMLElement);
      const isEscape = event.key === "Escape";
      const isModShortcut = event.metaKey || event.ctrlKey;

      if (inInput && !isEscape && !isModShortcut) {
        return;
      }

      // Handle Escape first (highest priority)
      if (matchesShortcut(event, shortcuts.escape, isMac)) {
        if (isCommandPaletteOpen) {
          event.preventDefault();
          closeCommandPalette();
          return;
        }
        if (isHelpOpen) {
          event.preventDefault();
          closeHelp();
          return;
        }
      }

      // Handle Command Palette (mod+k)
      if (matchesShortcut(event, shortcuts.commandPalette, isMac)) {
        event.preventDefault();
        if (isCommandPaletteOpen) {
          closeCommandPalette();
        } else {
          openCommandPalette();
        }
        return;
      }

      // Handle Help (?)
      if (matchesShortcut(event, shortcuts.help, isMac)) {
        event.preventDefault();
        toggleHelp();
        return;
      }

      // Handle Search (/) - only when not in input
      if (matchesShortcut(event, shortcuts.search, isMac) && !inInput) {
        event.preventDefault();
        // Focus search is handled by the component using this hook
        return;
      }

      // Handle sequence shortcuts (e.g., "g j")
      const key = event.key.toLowerCase();
      if (key.length === 1 && !inInput) {
        // Clear buffer after timeout
        if (sequenceTimeout.current) {
          window.clearTimeout(sequenceTimeout.current);
        }

        sequenceBuffer.current += key;

        // Check for matching sequences
        const sequences: [string, keyof ShortcutConfig][] = [
          ["gj", "navigateJobs"],
          ["gr", "navigateResults"],
          ["gf", "navigateForms"],
        ];

        for (const [seq, shortcutKey] of sequences) {
          if (sequenceBuffer.current === seq) {
            event.preventDefault();
            sequenceBuffer.current = "";
            // Dispatch custom event for navigation
            window.dispatchEvent(
              new CustomEvent("keyboard-navigate", {
                detail: { destination: shortcutKey },
              }),
            );
            return;
          }
        }

        // Reset buffer after delay
        sequenceTimeout.current = window.setTimeout(() => {
          sequenceBuffer.current = "";
        }, 500);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      if (sequenceTimeout.current) {
        window.clearTimeout(sequenceTimeout.current);
      }
    };
  }, [
    shortcuts,
    isMac,
    isCommandPaletteOpen,
    isHelpOpen,
    openCommandPalette,
    closeCommandPalette,
    closeHelp,
    toggleHelp,
  ]);

  return {
    shortcuts,
    isCommandPaletteOpen,
    isHelpOpen,
    updateShortcut,
    resetShortcuts,
    openCommandPalette,
    closeCommandPalette,
    openHelp,
    closeHelp,
    toggleHelp,
    isMac,
    formatShortcut: formatShortcutForDisplay,
  };
}
