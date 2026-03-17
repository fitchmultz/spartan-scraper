/**
 * Purpose: Verify keyboard shortcut help includes route-specific discoverability guidance.
 * Responsibilities: Assert the route section renders for route-aware help surfaces and keeps job-creation shortcuts visible.
 * Scope: Keyboard shortcut help rendering only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Route-specific shortcut content is sourced from shared onboarding route-help config.
 */

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { KeyboardShortcutsHelp } from "./KeyboardShortcutsHelp";
import type { ShortcutConfig } from "../hooks/useKeyboard";

const shortcuts: ShortcutConfig = {
  commandPalette: "mod+k",
  submitForm: "mod+enter",
  search: "/",
  help: "?",
  escape: "escape",
  navigateJobs: "g j",
  navigateResults: "g r",
  navigateForms: "g f",
};

describe("KeyboardShortcutsHelp", () => {
  it("renders a route-specific shortcut section", () => {
    render(
      <KeyboardShortcutsHelp
        isOpen
        onClose={vi.fn()}
        shortcuts={shortcuts}
        routeKind="new-job"
      />,
    );

    expect(screen.getByText("This Route")).toBeInTheDocument();
    expect(screen.getByText("Submit current job")).toBeInTheDocument();
  });
});
