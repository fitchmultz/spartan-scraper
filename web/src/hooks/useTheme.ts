/**
 * Theme Management Hook
 *
 * Manages application theme state with localStorage persistence and
 * system preference detection. Provides theme toggle functionality
 * and automatic sync with OS-level color scheme preferences.
 *
 * @module useTheme
 */

import { useState, useEffect, useCallback } from "react";

export type Theme = "light" | "dark" | "system";
export type ResolvedTheme = "light" | "dark";

export interface UseThemeReturn {
  /** Current theme setting (may be "system") */
  theme: Theme;
  /** Resolved theme after applying system preference */
  resolvedTheme: ResolvedTheme;
  /** Explicitly set the theme */
  setTheme: (theme: Theme) => void;
  /** Toggle between light and dark (ignores system) */
  toggleTheme: () => void;
}

const STORAGE_KEY = "spartan-theme";

/**
 * Get the system color scheme preference.
 * Defaults to "dark" if window is undefined (SSR).
 */
function getSystemPreference(): ResolvedTheme {
  if (typeof window === "undefined") return "dark";
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

/**
 * Get the stored theme from localStorage.
 * Returns null if no theme is stored or if localStorage is unavailable.
 */
function getStoredTheme(): Theme | null {
  if (typeof window === "undefined") return null;
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "light" || stored === "dark" || stored === "system") {
      return stored;
    }
  } catch {
    // localStorage may be unavailable (private browsing, etc.)
  }
  return null;
}

/**
 * Store the theme in localStorage.
 * Silently fails if localStorage is unavailable.
 */
function storeTheme(theme: Theme): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, theme);
  } catch {
    // localStorage may be unavailable
  }
}

/**
 * Apply the resolved theme to the document root.
 * Sets the data-theme attribute for CSS scoping.
 */
function applyThemeToDocument(resolvedTheme: ResolvedTheme): void {
  if (typeof document === "undefined") return;

  const root = document.documentElement;
  if (resolvedTheme === "dark") {
    root.setAttribute("data-theme", "dark");
  } else {
    root.setAttribute("data-theme", "light");
  }
}

/**
 * React hook for theme management.
 *
 * Features:
 * - localStorage persistence
 * - System preference detection
 * - Automatic sync with OS theme changes (when theme is "system")
 * - Manual toggle between light/dark
 *
 * @example
 * ```tsx
 * function App() {
 *   const { theme, resolvedTheme, setTheme, toggleTheme } = useTheme();
 *
 *   return (
 *     <div>
 *       <button onClick={toggleTheme}>
 *         Toggle theme (currently {resolvedTheme})
 *       </button>
 *       <select value={theme} onChange={e => setTheme(e.target.value as Theme)}>
 *         <option value="system">System</option>
 *         <option value="light">Light</option>
 *         <option value="dark">Dark</option>
 *       </select>
 *     </div>
 *   );
 * }
 * ```
 */
export function useTheme(): UseThemeReturn {
  const [theme, setThemeState] = useState<Theme>("system");
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>("dark");

  // Apply theme to document and update resolved theme
  const applyTheme = useCallback((newTheme: Theme) => {
    const resolved = newTheme === "system" ? getSystemPreference() : newTheme;
    setResolvedTheme(resolved);
    applyThemeToDocument(resolved);
  }, []);

  // Initialize theme from storage on mount
  useEffect(() => {
    const stored = getStoredTheme();
    if (stored) {
      setThemeState(stored);
      applyTheme(stored);
    } else {
      // Default to system preference
      applyTheme("system");
    }
  }, [applyTheme]);

  // Listen for system preference changes
  useEffect(() => {
    if (typeof window === "undefined") return;

    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handleChange = () => {
      if (theme === "system") {
        applyTheme("system");
      }
    };

    mediaQuery.addEventListener("change", handleChange);
    return () => mediaQuery.removeEventListener("change", handleChange);
  }, [theme, applyTheme]);

  /**
   * Explicitly set the theme.
   */
  const setTheme = useCallback(
    (newTheme: Theme) => {
      setThemeState(newTheme);
      storeTheme(newTheme);
      applyTheme(newTheme);
    },
    [applyTheme],
  );

  /**
   * Toggle between light and dark themes.
   * If currently "system", resolves to the opposite of current system preference.
   */
  const toggleTheme = useCallback(() => {
    const newTheme: ResolvedTheme = resolvedTheme === "dark" ? "light" : "dark";
    setTheme(newTheme);
  }, [resolvedTheme, setTheme]);

  return { theme, resolvedTheme, setTheme, toggleTheme };
}
