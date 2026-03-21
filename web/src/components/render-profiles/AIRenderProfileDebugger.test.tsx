/**
 * Purpose: Verify the AI render-profile debugger modal request, retry, transparency, and save flows.
 * Responsibilities: Assert tuning payload shaping, resolved-goal rendering, retry preservation, and save handoff behavior.
 * Scope: `AIRenderProfileDebugger` tests only.
 * Usage: Run with `pnpm --dir web test`.
 * Invariants/Assumptions: Tuning results must expose the resolved AI goal before operators choose to save, and retry must preserve request-scoped inputs.
 */
import type { ComponentProps } from "react";
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ToastProvider } from "../toast";
import { AIRenderProfileDebugger } from "../AIRenderProfileDebugger";
import * as api from "../../api";

vi.mock("../../api", () => ({
  aiRenderProfileDebug: vi.fn(),
  putV1RenderProfilesByName: vi.fn(),
}));

function renderDebugger(
  props: Partial<ComponentProps<typeof AIRenderProfileDebugger>> = {},
) {
  return render(
    <ToastProvider>
      <AIRenderProfileDebugger
        isOpen={true}
        profile={{
          name: "example-app",
          hostPatterns: ["example.com"],
          wait: { mode: "selector", selector: ".missing" },
        }}
        onClose={vi.fn()}
        onSaved={vi.fn()}
        {...props}
      />
    </ToastProvider>,
  );
}

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
    window.sessionStorage.clear();
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

    renderDebugger({ profile, onClose, onSaved });

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
      await screen.findByRole("button", {
        name: /save selected tuned profile/i,
      }),
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

  it("retains full history, restores non-latest guidance, switches baselines, and saves a restored tuned profile", async () => {
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
    vi.mocked(api.putV1RenderProfilesByName).mockResolvedValue({
      data: {
        name: "example-app",
        hostPatterns: ["example.com"],
        wait: { mode: "selector", selector: "main" },
      },
      error: undefined,
      request: new Request(
        "http://localhost:8741/v1/render-profiles/example-app",
      ),
      response: new Response(),
    });

    renderDebugger({ profile });

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
            profile: expect.objectContaining({
              wait: { mode: "selector", selector: "main" },
            }),
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

    const history = screen.getByRole("region", { name: /attempt history/i });
    expect(within(history).getByText(/attempt 1/i)).toBeInTheDocument();
    expect(within(history).getByText(/attempt 2/i)).toBeInTheDocument();
    expect(within(history).getByText(/attempt 3/i)).toBeInTheDocument();
    expect(within(history).getByText(/route-1/i)).toBeInTheDocument();
    expect(within(history).getByText(/route-2/i)).toBeInTheDocument();
    expect(within(history).getByText(/route-3/i)).toBeInTheDocument();

    fireEvent.click(
      within(history).getByRole("button", {
        name: /restore guidance from attempt 1/i,
      }),
    );
    expect(instructions).toHaveValue("Derived tuning goal v1");

    fireEvent.click(
      within(history).getByRole("button", {
        name: /use attempt 1 as baseline/i,
      }),
    );

    const selectedCandidate = screen.getByRole("region", {
      name: /latest candidate · attempt 3/i,
    });
    expect(
      within(selectedCandidate).getByText("Wait selector"),
    ).toBeInTheDocument();
    expect(within(selectedCandidate).getByText("main")).toBeInTheDocument();
    expect(
      within(selectedCandidate).getByText("#app-root"),
    ).toBeInTheDocument();

    fireEvent.click(
      within(history).getByRole("button", {
        name: /select attempt 1/i,
      }),
    );
    expect(within(history).getByText(/route-3/i)).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: /save selected tuned profile/i }),
    );

    await waitFor(() => {
      expect(api.putV1RenderProfilesByName).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        path: { name: "example-app" },
        body: expect.objectContaining({
          wait: { mode: "selector", selector: "main" },
        }),
      });
    });
  });

  it("resets a multi-attempt tuning session without closing the modal or losing request-scoped inputs", async () => {
    vi.mocked(api.aiRenderProfileDebug)
      .mockResolvedValueOnce({
        data: {
          issues: ["wait.selector matched no elements"],
          resolved_goal: { source: "derived", text: "First pass" },
          suggested_profile: {
            name: "example-app",
            hostPatterns: ["example.com"],
            wait: { mode: "selector", selector: "main" },
          },
          route_id: "route-1",
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
          resolved_goal: { source: "explicit", text: "Second pass" },
          suggested_profile: {
            name: "example-app",
            hostPatterns: ["example.com"],
            wait: { mode: "selector", selector: "#app-root" },
          },
          route_id: "route-2",
        },
        error: undefined,
        request: new Request(
          "http://localhost:8741/v1/ai/render-profile-debug",
        ),
        response: new Response(),
      });

    const closeSpy = vi.fn();

    renderDebugger({ profile, onClose: closeSpy });

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
    fireEvent.click(screen.getByLabelText(/include screenshot context/i));

    fireEvent.click(screen.getByRole("button", { name: /tune profile/i }));
    await screen.findByRole("region", { name: /attempt history/i });

    fireEvent.click(
      screen.getByRole("button", { name: /retry with changes/i }),
    );
    await screen.findByRole("region", { name: /latest candidate/i });

    fireEvent.click(screen.getByRole("button", { name: /reset session/i }));
    fireEvent.click(
      within(screen.getByRole("alertdialog")).getByRole("button", {
        name: /^reset session$/i,
      }),
    );

    await waitFor(() => {
      expect(closeSpy).not.toHaveBeenCalled();
      expect(
        screen.queryByRole("region", { name: /attempt history/i }),
      ).not.toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: /tune profile/i }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/target url/i)).toHaveValue(
      "https://example.com/app",
    );
    expect(screen.getByText("debug-profile.png")).toBeInTheDocument();
    expect(screen.getByLabelText(/use headless browser/i)).toBeChecked();
    expect(screen.getByLabelText(/include screenshot context/i)).toBeChecked();
  });

  it("closes without discarding and only clears the tuning session after explicit discard", async () => {
    vi.mocked(api.aiRenderProfileDebug).mockResolvedValue({
      data: {
        issues: ["wait.selector matched no elements"],
        resolved_goal: { source: "explicit", text: "Use main" },
        suggested_profile: {
          name: "example-app",
          hostPatterns: ["example.com"],
          wait: { mode: "selector", selector: "main" },
        },
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/ai/render-profile-debug"),
      response: new Response(),
    });

    const onClose = vi.fn();
    const onSaved = vi.fn();
    const { rerender } = render(
      <ToastProvider>
        <AIRenderProfileDebugger
          isOpen
          profile={profile}
          onClose={onClose}
          onSaved={onSaved}
        />
      </ToastProvider>,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.change(screen.getByLabelText(/tuning instructions/i), {
      target: { value: "Use main" },
    });
    fireEvent.click(screen.getByRole("button", { name: /tune profile/i }));
    await screen.findByRole("region", { name: /attempt history/i });

    fireEvent.click(screen.getAllByRole("button", { name: /^close$/i })[0]);
    expect(onClose).toHaveBeenCalledTimes(1);

    rerender(
      <ToastProvider>
        <AIRenderProfileDebugger
          isOpen={false}
          profile={profile}
          onClose={onClose}
          onSaved={onSaved}
        />
      </ToastProvider>,
    );
    rerender(
      <ToastProvider>
        <AIRenderProfileDebugger
          isOpen
          profile={profile}
          onClose={onClose}
          onSaved={onSaved}
        />
      </ToastProvider>,
    );

    expect(screen.getByLabelText(/target url/i)).toHaveValue(
      "https://example.com/app",
    );
    expect(screen.getByLabelText(/tuning instructions/i)).toHaveValue(
      "Use main",
    );
    expect(
      screen.getByRole("region", { name: /attempt history/i }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /discard session/i }));
    fireEvent.click(
      within(screen.getByRole("alertdialog")).getByRole("button", {
        name: /^discard session$/i,
      }),
    );
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledTimes(2);
    });

    rerender(
      <ToastProvider>
        <AIRenderProfileDebugger
          isOpen={false}
          profile={profile}
          onClose={onClose}
          onSaved={onSaved}
        />
      </ToastProvider>,
    );
    rerender(
      <ToastProvider>
        <AIRenderProfileDebugger
          isOpen
          profile={profile}
          onClose={onClose}
          onSaved={onSaved}
        />
      </ToastProvider>,
    );

    expect(
      screen.queryByRole("region", { name: /attempt history/i }),
    ).not.toBeInTheDocument();
    expect(screen.getByLabelText(/target url/i)).toHaveValue("");
  });
});
