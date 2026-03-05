/**
 * Keyboard Shortcuts Help Modal
 *
 * Displays a cheatsheet of all available keyboard shortcuts.
 * Organized by category: Global, Navigation, Forms, and Modal shortcuts.
 *
 * @module KeyboardShortcutsHelp
 */

import type { ShortcutConfig } from "../hooks/useKeyboard";
import { formatShortcut } from "../lib/keyboard-shortcuts";

export type KeyboardShortcutsHelpProps = {
  /** Whether the modal is visible */
  isOpen: boolean;
  /** Callback when modal should close */
  onClose: () => void;
  /** Current shortcut configuration */
  shortcuts: ShortcutConfig;
  /** Platform indicator for shortcut display */
  isMac?: boolean;
};

type ShortcutSection = {
  title: string;
  items: { label: string; shortcut: string }[];
};

/**
 * Keyboard Shortcuts Help Modal
 *
 * A modal dialog displaying all available keyboard shortcuts
 * organized by category for easy reference.
 */
export function KeyboardShortcutsHelp({
  isOpen,
  onClose,
  shortcuts,
  isMac = false,
}: KeyboardShortcutsHelpProps) {
  if (!isOpen) return null;

  const sections: ShortcutSection[] = [
    {
      title: "Global",
      items: [
        { label: "Command Palette", shortcut: shortcuts.commandPalette },
        { label: "Help / Cheatsheet", shortcut: shortcuts.help },
        { label: "Focus Search", shortcut: shortcuts.search },
        { label: "Close Modal", shortcut: shortcuts.escape },
      ],
    },
    {
      title: "Navigation",
      items: [
        { label: "Go to Jobs", shortcut: shortcuts.navigateJobs },
        { label: "Go to Results", shortcut: shortcuts.navigateResults },
        { label: "Go to Forms", shortcut: shortcuts.navigateForms },
      ],
    },
    {
      title: "Forms",
      items: [{ label: "Submit Form", shortcut: shortcuts.submitForm }],
    },
  ];

  return (
    <div
      className="shortcuts-help-overlay"
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          onClose();
        }
      }}
      onKeyDown={(e) => {
        if (e.key === "Escape") {
          e.preventDefault();
          onClose();
        }
      }}
      role="dialog"
      aria-modal="true"
      aria-label="Keyboard shortcuts help"
    >
      <div className="shortcuts-help-modal">
        <div className="shortcuts-help-header">
          <h2>Keyboard Shortcuts</h2>
          <button
            type="button"
            className="shortcuts-help-close"
            onClick={onClose}
            aria-label="Close help"
          >
            ×
          </button>
        </div>

        <div className="shortcuts-help-content">
          {sections.map((section) => (
            <div key={section.title} className="shortcuts-help-section">
              <h3 className="shortcuts-help-section-title">{section.title}</h3>
              <div className="shortcuts-help-list">
                {section.items.map((item) => (
                  <div key={item.label} className="shortcuts-help-item">
                    <span className="shortcuts-help-label">{item.label}</span>
                    <kbd className="shortcuts-help-key">
                      {formatShortcut(item.shortcut, isMac)}
                    </kbd>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        <div className="shortcuts-help-footer">
          <span className="shortcuts-help-hint">
            Press <kbd>?</kbd> to open this help when focus is outside text
            inputs
          </span>
        </div>
      </div>
    </div>
  );
}
