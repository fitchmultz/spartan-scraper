/**
 * Tests for useKeyboard hook.
 *
 * @module useKeyboard.test
 */

import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useKeyboard } from "./useKeyboard";

describe("useKeyboard", () => {
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
