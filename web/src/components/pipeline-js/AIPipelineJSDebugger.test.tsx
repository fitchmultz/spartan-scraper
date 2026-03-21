/**
 * Purpose: Verify the AI pipeline-JS debugger modal request, retry, transparency, and save flows.
 * Responsibilities: Assert tuning payload shaping, resolved-goal rendering, retry preservation, and save handoff behavior.
 * Scope: `AIPipelineJSDebugger` tests only.
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

import { AIPipelineJSDebugger } from "../AIPipelineJSDebugger";
import * as api from "../../api";

vi.mock("../../api", () => ({
  aiPipelineJsDebug: vi.fn(),
  putV1PipelineJsByName: vi.fn(),
}));

describe("AIPipelineJSDebugger", () => {
  const onClose = vi.fn();
  const onSaved = vi.fn();
  const script = {
    name: "example-app",
    hostPatterns: ["example.com"],
    selectors: [".missing"],
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls aiPipelineJsDebug and saves the suggested script", async () => {
    vi.mocked(api.aiPipelineJsDebug).mockResolvedValue({
      data: {
        issues: ["selectors[0] matched no elements"],
        resolved_goal: {
          source: "explicit",
          text: 'Tune the pipeline JS script named "example-app" for the supplied page while preserving its purpose and keeping changes minimal, deterministic, and operationally useful. Operator guidance: Prefer selector waits over custom JavaScript',
        },
        explanation: "Use the visible main shell selector.",
        suggested_script: {
          name: "example-app",
          hostPatterns: ["example.com"],
          selectors: ["main"],
          postNav: "window.scrollTo(0, 0);",
        },
        route_id: "openai/gpt-5.4",
        provider: "openai",
        model: "gpt-5.4",
        recheck_status: 200,
        recheck_engine: "playwright",
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/ai/pipeline-js-debug"),
      response: new Response(),
    });
    vi.mocked(api.putV1PipelineJsByName).mockResolvedValue({
      data: {
        name: "example-app",
        hostPatterns: ["example.com"],
        selectors: ["main"],
        postNav: "window.scrollTo(0, 0);",
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/pipeline-js/example-app"),
      response: new Response(),
    });

    render(
      <AIPipelineJSDebugger
        isOpen={true}
        script={script}
        onClose={onClose}
        onSaved={onSaved}
      />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.change(screen.getByLabelText(/tuning instructions/i), {
      target: { value: "Prefer selector waits over custom JavaScript" },
    });
    fireEvent.click(screen.getByLabelText(/use playwright/i));
    fireEvent.click(screen.getByRole("button", { name: /tune script/i }));

    await waitFor(() => {
      expect(api.aiPipelineJsDebug).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        body: {
          url: "https://example.com/app",
          script,
          instructions: "Prefer selector waits over custom JavaScript",
          headless: true,
          playwright: true,
          visual: false,
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
      within(latestCandidate).getByText(/selectors\[0\] matched no elements/i),
    ).toBeInTheDocument();
    const resolvedGoal = within(latestCandidate).getByRole("region", {
      name: /resolved goal/i,
    });
    expect(within(resolvedGoal).getByText("Explicit")).toBeInTheDocument();
    expect(
      within(resolvedGoal).getByText(
        /operator guidance: prefer selector waits over custom javascript/i,
      ),
    ).toBeInTheDocument();

    fireEvent.click(
      await screen.findByRole("button", {
        name: /save selected tuned script/i,
      }),
    );

    await waitFor(() => {
      expect(api.putV1PipelineJsByName).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        path: { name: "example-app" },
        body: {
          name: "example-app",
          hostPatterns: ["example.com"],
          selectors: ["main"],
          postNav: "window.scrollTo(0, 0);",
        },
      });
    });

    expect(onSaved).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("retains full history, restores non-latest guidance, switches baselines, and saves a restored tuned script", async () => {
    vi.mocked(api.aiPipelineJsDebug)
      .mockResolvedValueOnce({
        data: {
          issues: ["selectors[0] matched no elements"],
          resolved_goal: { source: "derived", text: "Derived tuning goal v1" },
          suggested_script: {
            name: "example-app",
            hostPatterns: ["example.com"],
            selectors: ["main"],
          },
          route_id: "route-1",
          provider: "openai",
          model: "gpt-5.4",
          visual_context_used: true,
        },
        error: undefined,
        request: new Request("http://localhost:8741/v1/ai/pipeline-js-debug"),
        response: new Response(),
      })
      .mockResolvedValueOnce({
        data: {
          issues: ["selector verified"],
          resolved_goal: {
            source: "explicit",
            text: "Prefer #app-root over main",
          },
          suggested_script: {
            name: "example-app",
            hostPatterns: ["example.com"],
            selectors: ["#app-root"],
          },
          route_id: "route-2",
          provider: "anthropic",
          model: "claude-sonnet-4-5",
        },
        error: undefined,
        request: new Request("http://localhost:8741/v1/ai/pipeline-js-debug"),
        response: new Response(),
      })
      .mockResolvedValueOnce({
        data: {
          issues: ["script stabilized"],
          resolved_goal: {
            source: "explicit",
            text: "Wait for #app-root",
          },
          suggested_script: {
            name: "example-app",
            hostPatterns: ["example.com"],
            selectors: ["#app-root"],
            postNav: "window.scrollTo(0, 0);",
          },
          route_id: "route-3",
          provider: "openai",
          model: "gpt-5.4",
        },
        error: undefined,
        request: new Request("http://localhost:8741/v1/ai/pipeline-js-debug"),
        response: new Response(),
      });
    vi.mocked(api.putV1PipelineJsByName).mockResolvedValue({
      data: {
        name: "example-app",
        hostPatterns: ["example.com"],
        selectors: ["main"],
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/pipeline-js/example-app"),
      response: new Response(),
    });

    render(
      <AIPipelineJSDebugger
        isOpen={true}
        script={script}
        onClose={vi.fn()}
        onSaved={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    const image = new File(["fake"], "debug-script.png", {
      type: "image/png",
    });
    fireEvent.change(screen.getByLabelText(/upload images/i), {
      target: { files: [image] },
    });
    await screen.findByText("debug-script.png");
    fireEvent.click(screen.getByLabelText(/use headless browser/i));
    fireEvent.click(screen.getByLabelText(/use playwright/i));
    fireEvent.click(screen.getByLabelText(/include screenshot context/i));
    fireEvent.click(screen.getByRole("button", { name: /tune script/i }));

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
      expect(api.aiPipelineJsDebug).toHaveBeenNthCalledWith(
        2,
        expect.objectContaining({
          body: expect.objectContaining({
            url: "https://example.com/app",
            script: expect.objectContaining({
              selectors: ["main"],
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
      expect(api.aiPipelineJsDebug).toHaveBeenNthCalledWith(
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
      within(selectedCandidate).getByText("Wait selectors"),
    ).toBeInTheDocument();
    expect(within(selectedCandidate).getByText(/"main"/)).toBeInTheDocument();
    expect(
      within(selectedCandidate).getByText(/"#app-root"/),
    ).toBeInTheDocument();

    fireEvent.click(
      within(history).getByRole("button", {
        name: /select attempt 1/i,
      }),
    );
    expect(within(history).getByText(/route-3/i)).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: /save selected tuned script/i }),
    );

    await waitFor(() => {
      expect(api.putV1PipelineJsByName).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        path: { name: "example-app" },
        body: expect.objectContaining({
          selectors: ["main"],
        }),
      });
    });
  });

  it("resets a multi-attempt tuning session without closing the modal or losing request-scoped inputs", async () => {
    vi.mocked(api.aiPipelineJsDebug)
      .mockResolvedValueOnce({
        data: {
          issues: ["selectors[0] matched no elements"],
          resolved_goal: { source: "derived", text: "First pass" },
          suggested_script: {
            name: "example-app",
            hostPatterns: ["example.com"],
            selectors: ["main"],
          },
          route_id: "route-1",
        },
        error: undefined,
        request: new Request("http://localhost:8741/v1/ai/pipeline-js-debug"),
        response: new Response(),
      })
      .mockResolvedValueOnce({
        data: {
          issues: ["selector verified"],
          resolved_goal: { source: "explicit", text: "Second pass" },
          suggested_script: {
            name: "example-app",
            hostPatterns: ["example.com"],
            selectors: ["#app-root"],
          },
          route_id: "route-2",
        },
        error: undefined,
        request: new Request("http://localhost:8741/v1/ai/pipeline-js-debug"),
        response: new Response(),
      });

    const closeSpy = vi.fn();

    render(
      <AIPipelineJSDebugger
        isOpen={true}
        script={script}
        onClose={closeSpy}
        onSaved={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });

    const image = new File(["fake"], "debug-script.png", {
      type: "image/png",
    });
    fireEvent.change(screen.getByLabelText(/upload images/i), {
      target: { files: [image] },
    });
    await screen.findByText("debug-script.png");

    fireEvent.click(screen.getByLabelText(/use headless browser/i));
    fireEvent.click(screen.getByLabelText(/include screenshot context/i));

    fireEvent.click(screen.getByRole("button", { name: /tune script/i }));
    await screen.findByRole("region", { name: /attempt history/i });

    fireEvent.click(
      screen.getByRole("button", { name: /retry with changes/i }),
    );
    await screen.findByRole("region", { name: /latest candidate/i });

    fireEvent.click(screen.getByRole("button", { name: /reset session/i }));

    expect(closeSpy).not.toHaveBeenCalled();
    expect(
      screen.queryByRole("region", { name: /attempt history/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /tune script/i }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/target url/i)).toHaveValue(
      "https://example.com/app",
    );
    expect(screen.getByText("debug-script.png")).toBeInTheDocument();
    expect(screen.getByLabelText(/use headless browser/i)).toBeChecked();
    expect(screen.getByLabelText(/include screenshot context/i)).toBeChecked();
  });
});
