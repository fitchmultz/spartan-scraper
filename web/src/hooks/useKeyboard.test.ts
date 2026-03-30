/**
 * Purpose: Verify keyboard shortcut hydration and input suppression for the web shell.
 * Responsibilities: Assert persisted shortcuts load immediately and non-modifier shortcuts stay quiet while typing.
 * Scope: Hook-level keyboard shortcut behavior only.
 * Usage: Run via `pnpm run test` or `make test-ci`.
 * Invariants/Assumptions: Shortcut state comes from browser localStorage and input elements should not trigger help toggles.
 */

import { renderHook, act } from "@testing-library/react";
import {
  afterAll,
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";
import { useKeyboard } from "./useKeyboard";

const storage = new Map<string, string>();

beforeAll(() => {
  const localStorageMock = {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      storage.set(key, value);
    },
    removeItem: (key: string) => {
      storage.delete(key);
    },
    clear: () => {
      storage.clear();
    },
  };

  vi.stubGlobal("localStorage", localStorageMock);
  if (typeof window !== "undefined") {
    Object.defineProperty(window, "localStorage", {
      value: localStorageMock,
      configurable: true,
    });
  }
});

afterAll(() => {
  vi.unstubAllGlobals();
});

describe("useKeyboard", () => {
  beforeEach(() => {
    storage.clear();
  });

  afterEach(() => {
    storage.clear();
  });

  it("loads stored shortcuts during initial render", () => {
    localStorage.setItem(
      "spartan-keyboard-shortcuts",
      JSON.stringify({
        help: "h",
        commandPalette: "mod+p",
      }),
    );

    const { result } = renderHook(() => useKeyboard());

    expect(result.current.shortcuts.help).toBe("h");
    expect(result.current.shortcuts.commandPalette).toBe("mod+p");
    expect(result.current.shortcuts.submitForm).toBe("mod+enter");
  });

  it("toggles the help modal with '?'", () => {
    const { result } = renderHook(() => useKeyboard());

    act(() => {
      window.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "?",
          shiftKey: true,
          bubbles: true,
        }),
      );
    });

    expect(result.current.isHelpOpen).toBe(true);

    act(() => {
      window.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "?",
          shiftKey: true,
          bubbles: true,
        }),
      );
    });

    expect(result.current.isHelpOpen).toBe(false);
  });

  it("ignores non-modifier shortcuts while typing in inputs", () => {
    const { result } = renderHook(() => useKeyboard());

    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();

    act(() => {
      input.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "?",
          shiftKey: true,
          bubbles: true,
        }),
      );
    });

    expect(result.current.isHelpOpen).toBe(false);

    input.remove();
  });
});
