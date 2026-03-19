/**
 * Purpose: Verify the first-run onboarding nudge keeps optional capabilities calm and non-blocking.
 * Responsibilities: Assert the nudge frames AI, proxy pooling, and retention as optional while keeping the primary first-run action obvious.
 * Scope: OnboardingNudge rendering only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: A healthy first run should not imply optional capabilities are prerequisites.
 */

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { OnboardingNudge } from "./OnboardingNudge";

describe("OnboardingNudge", () => {
  it("explains that optional capabilities can wait on first run", () => {
    render(
      <OnboardingNudge
        isVisible
        onStartTour={vi.fn()}
        onOpenHelp={vi.fn()}
        onDismiss={vi.fn()}
        onCreateJob={vi.fn()}
        health={{
          status: "ok",
          version: "test",
          components: {
            browser: { status: "ok", message: "Browser automation is ready." },
            ai: {
              status: "disabled",
              message: "AI helpers are optional and currently disabled.",
            },
          },
          notices: [],
        }}
        hasTemplates={false}
      />,
    );

    expect(
      screen.getByText(
        /no ai, proxy pool, or retention setup is required for the first run/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /ai helpers stay off by default and can be enabled later/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /templates can be added later without changing the first-run path/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /create first job/i }),
    ).toBeEnabled();
  });
});
