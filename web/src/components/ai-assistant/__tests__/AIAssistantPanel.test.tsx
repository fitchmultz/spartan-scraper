/**
 * Purpose: Verify the shared AI assistant shell exposes width through a CSS custom property instead of an inline width lock.
 * Responsibilities: Mount the panel inside the persisted provider and assert the rendered style supports mobile CSS overrides.
 * Scope: Shared AI assistant shell presentation behavior only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Mobile layouts must be able to override the desktop width preference, so the panel cannot set a fixed inline `width` style.
 */

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AIAssistantPanel } from "../AIAssistantPanel";
import { AIAssistantProvider } from "../AIAssistantProvider";

describe("AIAssistantPanel", () => {
  it("stores panel width in a CSS custom property", () => {
    render(
      <AIAssistantProvider>
        <AIAssistantPanel title="Template assistant" routeLabel="/templates">
          <div>Assistant body</div>
        </AIAssistantPanel>
      </AIAssistantProvider>,
    );

    const panel = screen.getByLabelText("Template assistant panel");

    expect(panel).toHaveStyle({ "--ai-assistant-panel-width": "380px" });
    expect(panel).not.toHaveStyle({ width: "380px" });
  });
});
