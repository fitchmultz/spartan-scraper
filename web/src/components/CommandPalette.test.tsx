/**
 * CommandPalette focus behavior tests.
 *
 * @module CommandPalette.test
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { CommandPalette } from "./CommandPalette";

describe("CommandPalette", () => {
  beforeEach(() => {
    vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
      cb(0);
      return 1;
    });
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
    vi.stubGlobal(
      "ResizeObserver",
      class {
        observe() {}
        unobserve() {}
        disconnect() {}
      },
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("focuses the command input when opened", async () => {
    render(
      <CommandPalette
        isOpen
        onClose={vi.fn()}
        jobs={[]}
        onNavigate={vi.fn()}
        onSubmitForm={vi.fn()}
        onCancelJob={vi.fn()}
      />,
    );

    const input = screen.getByLabelText("Search commands");

    await waitFor(() => {
      expect(document.activeElement).toBe(input);
    });
  });
});
