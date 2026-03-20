/**
 * Purpose: Verify the command palette stays focus-safe and reaches every major operator surface.
 * Responsibilities: Assert the search input autofocuses on open and route/job commands navigate to concrete paths.
 * Scope: CommandPalette interaction tests only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Recent-job entries must drill into `/jobs/:id`, and palette commands should navigate by path rather than a lossy view abstraction.
 */

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
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
        onNavigateToPath={vi.fn()}
        onSubmitForm={vi.fn()}
        onCancelJob={vi.fn()}
      />,
    );

    const input = screen.getByLabelText("Search commands");

    await waitFor(() => {
      expect(document.activeElement).toBe(input);
    });
  });

  it("navigates to major routes and direct job detail routes", async () => {
    const user = userEvent.setup();
    const onNavigateToPath = vi.fn();

    render(
      <CommandPalette
        isOpen
        onClose={vi.fn()}
        jobs={[
          {
            id: "job-12345678",
            kind: "scrape",
            status: "succeeded",
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            specVersion: 1,
            spec: {},
            run: { waitMs: 0, runMs: 0, totalMs: 0 },
          },
        ]}
        onNavigateToPath={onNavigateToPath}
        onSubmitForm={vi.fn()}
        onCancelJob={vi.fn()}
        presets={[]}
      />,
    );

    await user.click(screen.getByText("Open Templates"));
    expect(onNavigateToPath).toHaveBeenCalledWith("/templates");

    await user.click(screen.getByRole("option", { name: /scrape: job-1234/i }));
    expect(onNavigateToPath).toHaveBeenCalledWith("/jobs/job-12345678");
  });
});
