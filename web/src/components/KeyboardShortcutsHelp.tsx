/**
 * Purpose: Render the keyboard shortcut help overlay with global, navigation, and route-specific guidance.
 * Responsibilities: Present shortcut groups, format keys for the current platform, and expose route-relevant actions alongside global shell affordances.
 * Scope: Keyboard shortcut help presentation only.
 * Usage: Mount from `App.tsx` and pass the current shortcut config plus optional route kind.
 * Invariants/Assumptions: The overlay closes on backdrop click or Escape, and route-specific content must stay aligned with the onboarding route-help model.
 */

import type { ShortcutConfig } from "../hooks/useKeyboard";
import { formatShortcut } from "../lib/keyboard-shortcuts";
import { ROUTE_HELP_CONTENT, type OnboardingRouteKey } from "../lib/onboarding";

export type KeyboardShortcutsHelpProps = {
  isOpen: boolean;
  onClose: () => void;
  shortcuts: ShortcutConfig;
  isMac?: boolean;
  routeKind?: OnboardingRouteKey;
};

type ShortcutSection = {
  title: string;
  items: { label: string; shortcut: string }[];
};

export function KeyboardShortcutsHelp({
  isOpen,
  onClose,
  shortcuts,
  isMac = false,
  routeKind,
}: KeyboardShortcutsHelpProps) {
  if (!isOpen) {
    return null;
  }

  const routeSection: ShortcutSection | null = routeKind
    ? {
        title: "This Route",
        items: ROUTE_HELP_CONTENT[routeKind].shortcuts.map((item) => ({
          label: item.label,
          shortcut: shortcuts[item.shortcut],
        })),
      }
    : null;

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
    ...(routeSection ? [routeSection] : []),
  ];

  if (routeKind === "new-job") {
    sections.push({
      title: "Job Creation",
      items: [{ label: "Submit current job", shortcut: shortcuts.submitForm }],
    });
  }

  return (
    <div
      className="shortcuts-help-overlay"
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
      onKeyDown={(event) => {
        if (event.key === "Escape") {
          event.preventDefault();
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
            Press <kbd>?</kbd> when focus is outside text inputs to reopen this
            help
          </span>
        </div>
      </div>
    </div>
  );
}
