/**
 * Purpose: Verify the AI render-profile debugger modal request, transparency, and save flows.
 * Responsibilities: Assert tuning payload shaping, resolved-goal rendering, and save handoff behavior.
 * Scope: `AIRenderProfileDebugger` tests only.
 * Usage: Run with `pnpm --dir web test`.
 * Invariants/Assumptions: Tuning results must expose the resolved AI goal before operators choose to save.
 */
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AIRenderProfileDebugger } from "../AIRenderProfileDebugger";
import * as api from "../../api";

vi.mock("../../api", () => ({
  aiRenderProfileDebug: vi.fn(),
  putV1RenderProfilesByName: vi.fn(),
}));

describe("AIRenderProfileDebugger", () => {
  const onClose = vi.fn();
  const onSaved = vi.fn();
  const profile = {
    name: "example-app",
    hostPatterns: ["example.com"],
    wait: { mode: "selector" as const, selector: ".missing" },
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls aiRenderProfileDebug and saves the suggested profile", async () => {
    vi.mocked(api.aiRenderProfileDebug).mockResolvedValue({
      data: {
        issues: ["wait.selector matched no elements"],
        resolved_goal: {
          source: "explicit",
          text: 'Tune the render profile named "example-app" for the supplied page while preserving its purpose and keeping changes minimal, deterministic, and operationally useful. Operator guidance: Prefer the visible main shell',
        },
        explanation: "Use the visible main shell.",
        suggested_profile: {
          name: "example-app",
          hostPatterns: ["example.com"],
          preferHeadless: true,
          wait: { mode: "selector", selector: "main" },
        },
        route_id: "openai/gpt-5.4",
        provider: "openai",
        model: "gpt-5.4",
        visual_context_used: true,
        recheck_status: 200,
        recheck_engine: "http",
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/ai/render-profile-debug"),
      response: new Response(),
    });
    vi.mocked(api.putV1RenderProfilesByName).mockResolvedValue({
      data: {
        name: "example-app",
        hostPatterns: ["example.com"],
        preferHeadless: true,
        wait: { mode: "selector", selector: "main" },
      },
      error: undefined,
      request: new Request(
        "http://localhost:8741/v1/render-profiles/example-app",
      ),
      response: new Response(),
    });

    render(
      <AIRenderProfileDebugger
        isOpen={true}
        profile={profile}
        onClose={onClose}
        onSaved={onSaved}
      />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.change(screen.getByLabelText(/tuning instructions/i), {
      target: { value: "Prefer the visible main shell" },
    });
    fireEvent.click(screen.getByLabelText(/use headless browser/i));
    fireEvent.click(screen.getByLabelText(/include screenshot context/i));
    fireEvent.click(screen.getByRole("button", { name: /tune profile/i }));

    await waitFor(() => {
      expect(api.aiRenderProfileDebug).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        body: {
          url: "https://example.com/app",
          profile,
          instructions: "Prefer the visible main shell",
          headless: true,
          playwright: false,
          visual: true,
        },
      });
    });

    expect(screen.getByText(/detected issues/i)).toBeInTheDocument();
    expect(
      screen.getByText(/wait.selector matched no elements/i),
    ).toBeInTheDocument();
    expect(await screen.findByText("Explicit")).toBeInTheDocument();
    expect(
      screen.getByText(/operator guidance: prefer the visible main shell/i),
    ).toBeInTheDocument();

    fireEvent.click(
      await screen.findByRole("button", { name: /save tuned profile/i }),
    );

    await waitFor(() => {
      expect(api.putV1RenderProfilesByName).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        path: { name: "example-app" },
        body: {
          name: "example-app",
          hostPatterns: ["example.com"],
          preferHeadless: true,
          wait: { mode: "selector", selector: "main" },
        },
      });
    });

    expect(onSaved).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
