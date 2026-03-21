/**
 * Purpose: Verify the AI render-profile debugger modal request, retry, transparency, and save flows.
 * Responsibilities: Assert tuning payload shaping, resolved-goal rendering, retry preservation, and save handoff behavior.
 * Scope: `AIRenderProfileDebugger` tests only.
 * Usage: Run with `pnpm --dir web test`.
 * Invariants/Assumptions: Tuning results must expose the resolved AI goal before operators choose to save, and retry must preserve request-scoped inputs.
 */
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
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

    const latestCandidate = await screen.findByRole("region", {
      name: /latest candidate/i,
    });
    expect(
      within(latestCandidate).getByText(/detected issues/i),
    ).toBeInTheDocument();
    expect(
      within(latestCandidate).getByText(/wait.selector matched no elements/i),
    ).toBeInTheDocument();
    const resolvedGoal = within(latestCandidate).getByRole("region", {
      name: /resolved goal/i,
    });
    expect(within(resolvedGoal).getByText("Explicit")).toBeInTheDocument();
    expect(
      within(resolvedGoal).getByText(
        /operator guidance: prefer the visible main shell/i,
      ),
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

  it("preserves debugger inputs across retries and replaces stale comparison metadata", async () => {
    vi.mocked(api.aiRenderProfileDebug)
      .mockResolvedValueOnce({
        data: {
          issues: ["wait.selector matched no elements"],
          resolved_goal: { source: "derived", text: "Derived tuning goal v1" },
          suggested_profile: {
            name: "example-app",
            hostPatterns: ["example.com"],
            wait: { mode: "selector", selector: "main" },
          },
          route_id: "route-1",
          provider: "openai",
          model: "gpt-5.4",
          visual_context_used: true,
        },
        error: undefined,
        request: new Request(
          "http://localhost:8741/v1/ai/render-profile-debug",
        ),
        response: new Response(),
      })
      .mockResolvedValueOnce({
        data: {
          issues: ["timing stabilized"],
          resolved_goal: {
            source: "explicit",
            text: "Prefer #app-root over main",
          },
          suggested_profile: {
            name: "example-app",
            hostPatterns: ["example.com"],
            wait: { mode: "selector", selector: "#app-root" },
          },
          route_id: "route-2",
          provider: "anthropic",
          model: "claude-sonnet-4-5",
        },
        error: undefined,
        request: new Request(
          "http://localhost:8741/v1/ai/render-profile-debug",
        ),
        response: new Response(),
      })
      .mockResolvedValueOnce({
        data: {
          issues: ["selector verified"],
          resolved_goal: {
            source: "explicit",
            text: "Wait for #app-root",
          },
          suggested_profile: {
            name: "example-app",
            hostPatterns: ["example.com"],
            wait: { mode: "selector", selector: "#app-root" },
            preferHeadless: true,
          },
          route_id: "route-3",
          provider: "openai",
          model: "gpt-5.4",
        },
        error: undefined,
        request: new Request(
          "http://localhost:8741/v1/ai/render-profile-debug",
        ),
        response: new Response(),
      });

    render(
      <AIRenderProfileDebugger
        isOpen={true}
        profile={profile}
        onClose={vi.fn()}
        onSaved={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    const image = new File(["fake"], "debug-profile.png", {
      type: "image/png",
    });
    fireEvent.change(screen.getByLabelText(/upload images/i), {
      target: { files: [image] },
    });
    await screen.findByText("debug-profile.png");
    fireEvent.click(screen.getByLabelText(/use headless browser/i));
    fireEvent.click(screen.getByLabelText(/use playwright/i));
    fireEvent.click(screen.getByLabelText(/include screenshot context/i));
    fireEvent.click(screen.getByRole("button", { name: /tune profile/i }));

    const instructions = screen.getByLabelText(/tuning instructions/i);
    await waitFor(() => {
      expect(instructions).toHaveValue("Derived tuning goal v1");
    });

    fireEvent.change(instructions, {
      target: { value: "Prefer #app-root over main" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: /retry with changes/i }),
    );

    await waitFor(() => {
      expect(api.aiRenderProfileDebug).toHaveBeenNthCalledWith(
        2,
        expect.objectContaining({
          body: expect.objectContaining({
            url: "https://example.com/app",
            profile,
            instructions: "Prefer #app-root over main",
            images: [{ data: "ZmFrZQ==", mime_type: "image/png" }],
            headless: true,
            playwright: true,
            visual: true,
          }),
        }),
      );
    });
    await waitFor(() => {
      expect(instructions).toHaveValue("Prefer #app-root over main");
    });

    const previousCandidate = await screen.findByRole("region", {
      name: /previous candidate/i,
    });
    const latestCandidate = await screen.findByRole("region", {
      name: /latest candidate/i,
    });
    expect(
      within(previousCandidate).getByText(/route: route-1/i),
    ).toBeInTheDocument();
    expect(
      within(latestCandidate).getByText(/route: route-2/i),
    ).toBeInTheDocument();
    expect(
      within(latestCandidate).getByText("Wait selector"),
    ).toBeInTheDocument();
    expect(
      within(latestCandidate).queryByText("Host patterns"),
    ).not.toBeInTheDocument();
    expect(
      within(latestCandidate).getByRole("button", { name: /show raw json/i }),
    ).toBeInTheDocument();

    fireEvent.change(instructions, {
      target: { value: "Wait for #app-root" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: /retry with changes/i }),
    );

    await waitFor(() => {
      expect(api.aiRenderProfileDebug).toHaveBeenNthCalledWith(
        3,
        expect.objectContaining({
          body: expect.objectContaining({
            instructions: "Wait for #app-root",
            images: [{ data: "ZmFrZQ==", mime_type: "image/png" }],
            headless: true,
            playwright: true,
            visual: true,
          }),
        }),
      );
    });
    await waitFor(() => {
      expect(instructions).toHaveValue("Wait for #app-root");
    });

    expect(
      within(previousCandidate).queryByText(/route: route-1/i),
    ).not.toBeInTheDocument();
    expect(
      within(previousCandidate).getByText(/route: route-2/i),
    ).toBeInTheDocument();
    expect(
      within(latestCandidate).getByText(/route: route-3/i),
    ).toBeInTheDocument();
  });
});
