/**
 * Purpose: Verify the shared theme hook keeps browser theme state synchronized with storage and media preferences.
 * Responsibilities: Assert initialization, toggling, system-theme resolution, and persisted theme writes.
 * Scope: `useTheme` hook behavior only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: `localStorage` and `matchMedia` are mocked, and DOM dataset/class updates happen synchronously during hook actions.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useTheme, type Theme, type ResolvedTheme } from "./useTheme";

describe("useTheme", () => {
  const localStorageMock = {
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn(),
  };

  const matchMediaMock = vi.fn();

  beforeEach(() => {
    vi.stubGlobal("localStorage", localStorageMock);
    vi.stubGlobal("matchMedia", matchMediaMock);

    // Default to dark mode
    matchMediaMock.mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });

    // Clear document attribute (only in browser environment)
    if (typeof document !== "undefined") {
      document.documentElement.removeAttribute("data-theme");
    }
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("should initialize with system theme by default", () => {
    localStorageMock.getItem.mockReturnValue(null);

    const { result } = renderHook(() => useTheme());

    expect(result.current.theme).toBe("system");
    expect(result.current.resolvedTheme).toBe("light");
  });

  it("should load stored theme from localStorage", () => {
    localStorageMock.getItem.mockReturnValue("dark");

    const { result } = renderHook(() => useTheme());

    expect(result.current.theme).toBe("dark");
    expect(result.current.resolvedTheme).toBe("dark");
  });

  it("should set theme and persist to localStorage", () => {
    localStorageMock.getItem.mockReturnValue(null);

    const { result } = renderHook(() => useTheme());

    act(() => {
      result.current.setTheme("dark");
    });

    expect(result.current.theme).toBe("dark");
    expect(result.current.resolvedTheme).toBe("dark");
    expect(localStorageMock.setItem).toHaveBeenCalledWith(
      "spartan-theme",
      "dark",
    );
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
  });

  it("should toggle between light and dark", () => {
    localStorageMock.getItem.mockReturnValue("dark");

    const { result } = renderHook(() => useTheme());

    expect(result.current.resolvedTheme).toBe("dark");

    act(() => {
      result.current.toggleTheme();
    });

    expect(result.current.theme).toBe("light");
    expect(result.current.resolvedTheme).toBe("light");
  });

  it("should resolve system preference to dark when preferred", () => {
    matchMediaMock.mockReturnValue({
      matches: true, // prefers dark
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });
    localStorageMock.getItem.mockReturnValue("system");

    const { result } = renderHook(() => useTheme());

    expect(result.current.theme).toBe("system");
    expect(result.current.resolvedTheme).toBe("dark");
  });

  it("should handle localStorage errors gracefully", () => {
    localStorageMock.getItem.mockImplementation(() => {
      throw new Error("localStorage unavailable");
    });
    localStorageMock.setItem.mockImplementation(() => {
      throw new Error("localStorage unavailable");
    });

    const { result } = renderHook(() => useTheme());

    // Should not throw and should use default
    expect(result.current.theme).toBe("system");

    // Should not throw when setting theme
    act(() => {
      result.current.setTheme("light");
    });

    expect(result.current.theme).toBe("light");
  });

  it("should apply theme to document root", () => {
    localStorageMock.getItem.mockReturnValue(null);

    const { result } = renderHook(() => useTheme());

    act(() => {
      result.current.setTheme("light");
    });

    expect(document.documentElement.getAttribute("data-theme")).toBe("light");

    act(() => {
      result.current.setTheme("dark");
    });

    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
  });

  it("should support all theme values", () => {
    localStorageMock.getItem.mockReturnValue(null);

    const { result } = renderHook(() => useTheme());

    const themes: Theme[] = ["light", "dark", "system"];

    for (const theme of themes) {
      act(() => {
        result.current.setTheme(theme);
      });

      expect(result.current.theme).toBe(theme);
    }
  });

  it("should export correct types", () => {
    // Type-only check - this will fail at compile time if types are wrong
    const theme: Theme = "light";
    const resolved: ResolvedTheme = "dark";

    expect(theme).toBe("light");
    expect(resolved).toBe("dark");
  });
});
