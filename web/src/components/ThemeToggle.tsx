/**
 * Purpose: Render the theme toggle UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import { useState, useRef, useEffect } from "react";
import type { Theme, ResolvedTheme } from "../hooks/useTheme";

export interface ThemeToggleProps {
  /** Current theme setting */
  theme: Theme;
  /** Resolved theme (actual applied theme) */
  resolvedTheme: ResolvedTheme;
  /** Callback when theme is explicitly changed */
  onThemeChange: (theme: Theme) => void;
  /** Callback for quick toggle */
  onToggle: () => void;
}

/**
 * Icon components for theme states.
 */
const icons = {
  light: (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="5" />
      <line x1="12" y1="1" x2="12" y2="3" />
      <line x1="12" y1="21" x2="12" y2="23" />
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
      <line x1="1" y1="12" x2="3" y2="12" />
      <line x1="21" y1="12" x2="23" y2="12" />
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
    </svg>
  ),
  dark: (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      aria-hidden="true"
    >
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  ),
  system: (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      aria-hidden="true"
    >
      <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
      <line x1="8" y1="21" x2="16" y2="21" />
      <line x1="12" y1="17" x2="12" y2="21" />
    </svg>
  ),
};

/**
 * Theme toggle button with dropdown menu.
 *
 * Click to toggle between light/dark, right-click to open dropdown
 * with light/dark/system options.
 *
 * @example
 * ```tsx
 * function Header() {
 *   const { theme, resolvedTheme, setTheme, toggleTheme } = useTheme();
 *
 *   return (
 *     <ThemeToggle
 *       theme={theme}
 *       resolvedTheme={resolvedTheme}
 *       onThemeChange={setTheme}
 *       onToggle={toggleTheme}
 *     />
 *   );
 * }
 * ```
 */
export function ThemeToggle({
  theme,
  resolvedTheme,
  onThemeChange,
  onToggle,
}: ThemeToggleProps) {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    }

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Close dropdown on Escape key
  useEffect(() => {
    function handleEscape(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener("keydown", handleEscape);
      return () => document.removeEventListener("keydown", handleEscape);
    }
  }, [isOpen]);

  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    setIsOpen(!isOpen);
  };

  const handleThemeSelect = (newTheme: Theme) => {
    onThemeChange(newTheme);
    setIsOpen(false);
  };

  return (
    <div ref={dropdownRef} style={{ position: "relative" }}>
      <button
        type="button"
        className="secondary"
        onClick={onToggle}
        onContextMenu={handleContextMenu}
        title={`Theme: ${theme} (click to toggle, right-click for options)`}
        aria-label={`Current theme: ${theme}. Click to toggle, right-click for options.`}
        aria-haspopup="menu"
        aria-expanded={isOpen}
        style={{
          display: "flex",
          alignItems: "center",
          gap: "6px",
          padding: "8px 12px",
        }}
      >
        {icons[resolvedTheme]}
        <span style={{ textTransform: "capitalize" }}>{theme}</span>
      </button>

      {isOpen && (
        <div
          role="menu"
          aria-label="Theme options"
          style={{
            position: "absolute",
            top: "100%",
            right: 0,
            marginTop: "8px",
            background: "var(--panel)",
            border: "1px solid var(--stroke)",
            borderRadius: "12px",
            padding: "8px",
            minWidth: "140px",
            boxShadow: "var(--shadow)",
            zIndex: 100,
          }}
        >
          {(["light", "dark", "system"] as const).map((t) => (
            <button
              key={t}
              type="button"
              role="menuitem"
              onClick={() => handleThemeSelect(t)}
              style={{
                display: "flex",
                alignItems: "center",
                gap: "8px",
                width: "100%",
                padding: "8px 12px",
                border: "none",
                background:
                  theme === t ? "rgba(255, 183, 0, 0.15)" : "transparent",
                color: "var(--text)",
                borderRadius: "8px",
                cursor: "pointer",
                textTransform: "capitalize",
              }}
            >
              {icons[t]}
              {t}
              {theme === t && (
                <span style={{ marginLeft: "auto", color: "var(--accent)" }}>
                  ✓
                </span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
