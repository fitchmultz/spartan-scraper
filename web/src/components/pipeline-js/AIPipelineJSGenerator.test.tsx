/**
 * Purpose: Verify the AI pipeline-JS generator modal request, retry, and save flows.
 * Responsibilities: Assert guided and instructionless submissions, resolved-goal rendering, retry preservation, and save handoff behavior.
 * Scope: `AIPipelineJSGenerator` tests only.
 * Usage: Run with `pnpm --dir web test`.
 * Invariants/Assumptions: URL remains required, retry keeps request-scoped inputs intact, and generated scripts are only persisted after explicit save.
 */
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { AIPipelineJSGenerator } from "../AIPipelineJSGenerator";
import * as api from "../../api";

vi.mock("../../api", () => ({
  aiPipelineJsGenerate: vi.fn(),
  postV1PipelineJs: vi.fn(),
}));

describe("AIPipelineJSGenerator", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls aiPipelineJsGenerate with URL mode options and saves the result", async () => {
    vi.mocked(api.aiPipelineJsGenerate).mockResolvedValue({
      data: {
        script: {
          name: "example-app",
          hostPatterns: ["example.com"],
          selectors: ["main"],
          postNav: "window.scrollTo(0, 0);",
        },
        resolved_goal: {
          source: "explicit",
          text: "Wait for the main dashboard and reset scroll position",
        },
        explanation: "Wait for the main app shell and normalize scroll.",
        route_id: "openai/gpt-5.4",
        provider: "openai",
        model: "gpt-5.4",
        visual_context_used: true,
      },
      request: new Request("http://localhost:8741/v1/ai/pipeline-js-generate"),
      response: new Response(),
    });
    vi.mocked(api.postV1PipelineJs).mockResolvedValue({
      data: {
        name: "example-app",
        hostPatterns: ["example.com"],
        selectors: ["main"],
        postNav: "window.scrollTo(0, 0);",
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/pipeline-js"),
      response: new Response(),
    });

    const onSaved = vi.fn();
    const onClose = vi.fn();
    render(
      <AIPipelineJSGenerator isOpen onClose={onClose} onSaved={onSaved} />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.change(screen.getByLabelText(/^script name$/i), {
      target: { value: "example-app" },
    });
    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "example.com" },
    });
    fireEvent.change(screen.getByLabelText(/instructions/i), {
      target: {
        value: "Wait for the main dashboard and reset scroll position",
      },
    });
    const image = new File(["fake"], "script.png", { type: "image/png" });
    fireEvent.change(screen.getByLabelText(/upload images/i), {
      target: { files: [image] },
    });
    await screen.findByText("script.png");
    fireEvent.click(screen.getByLabelText(/include screenshot context/i));
    fireEvent.click(screen.getByRole("button", { name: /generate script/i }));

    await waitFor(() => {
      expect(api.aiPipelineJsGenerate).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        body: {
          url: "https://example.com/app",
          name: "example-app",
          host_patterns: ["example.com"],
          instructions: "Wait for the main dashboard and reset scroll position",
          images: [{ data: "ZmFrZQ==", mime_type: "image/png" }],
          headless: true,
          playwright: false,
          visual: true,
        },
      });
    });

    const latestCandidate = await screen.findByRole("region", {
      name: /latest candidate/i,
    });
    const resolvedGoal = within(latestCandidate).getByRole("region", {
      name: /resolved goal/i,
    });
    expect(within(resolvedGoal).getByText("Explicit")).toBeInTheDocument();
    expect(
      within(resolvedGoal).getByText(
        "Wait for the main dashboard and reset scroll position",
      ),
    ).toBeInTheDocument();

    fireEvent.click(
      await screen.findByRole("button", { name: /save script/i }),
    );

    await waitFor(() => {
      expect(api.postV1PipelineJs).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
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

  it("allows pipeline-JS generation without instructions", async () => {
    vi.mocked(api.aiPipelineJsGenerate).mockResolvedValue({
      data: {
        script: {
          name: "example-app",
          hostPatterns: ["example.com"],
          selectors: ["main"],
        },
        resolved_goal: {
          source: "derived",
          text: 'Generate the minimal deterministic pipeline JS needed for "example-app" on example.com.',
        },
      },
      request: new Request("http://localhost:8741/v1/ai/pipeline-js-generate"),
      response: new Response(),
    });

    render(
      <AIPipelineJSGenerator isOpen onClose={vi.fn()} onSaved={vi.fn()} />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.click(screen.getByRole("button", { name: /generate script/i }));

    await waitFor(() => {
      expect(api.aiPipelineJsGenerate).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        body: {
          url: "https://example.com/app",
          headless: false,
          visual: false,
        },
      });
    });

    const latestCandidate = await screen.findByRole("region", {
      name: /latest candidate/i,
    });
    const resolvedGoal = within(latestCandidate).getByRole("region", {
      name: /resolved goal/i,
    });
    expect(
      within(resolvedGoal).getByText("System-derived"),
    ).toBeInTheDocument();
    expect(
      within(resolvedGoal).getByText(
        'Generate the minimal deterministic pipeline JS needed for "example-app" on example.com.',
      ),
    ).toBeInTheDocument();
  });

  it("retries pipeline generation with preserved images, toggles, and clean comparison metadata", async () => {
    vi.mocked(api.aiPipelineJsGenerate)
      .mockResolvedValueOnce({
        data: {
          script: {
            name: "example-app",
            hostPatterns: ["example.com"],
            selectors: ["main"],
          },
          resolved_goal: {
            source: "derived",
            text: "Derived pipeline goal v1",
          },
          route_id: "route-1",
          provider: "openai",
          model: "gpt-5.4",
          visual_context_used: true,
        },
        request: new Request(
          "http://localhost:8741/v1/ai/pipeline-js-generate",
        ),
        response: new Response(),
      })
      .mockResolvedValueOnce({
        data: {
          script: {
            name: "example-app",
            hostPatterns: ["example.com"],
            selectors: ["#app-root"],
            postNav: "window.scrollTo(0, 0);",
          },
          resolved_goal: {
            source: "explicit",
            text: "Use the visible app shell",
          },
          route_id: "route-2",
          provider: "anthropic",
          model: "claude-sonnet-4-5",
        },
        request: new Request(
          "http://localhost:8741/v1/ai/pipeline-js-generate",
        ),
        response: new Response(),
      })
      .mockResolvedValueOnce({
        data: {
          script: {
            name: "example-app",
            hostPatterns: ["example.com"],
            selectors: ["#app-root"],
          },
          resolved_goal: {
            source: "explicit",
            text: "Wait for #app-root",
          },
          route_id: "route-3",
          provider: "openai",
          model: "gpt-5.4",
        },
        request: new Request(
          "http://localhost:8741/v1/ai/pipeline-js-generate",
        ),
        response: new Response(),
      });

    render(
      <AIPipelineJSGenerator isOpen onClose={vi.fn()} onSaved={vi.fn()} />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.change(screen.getByLabelText(/^script name$/i), {
      target: { value: "example-app" },
    });
    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "example.com" },
    });
    const image = new File(["fake"], "retry-script.png", {
      type: "image/png",
    });
    fireEvent.change(screen.getByLabelText(/upload images/i), {
      target: { files: [image] },
    });
    await screen.findByText("retry-script.png");
    fireEvent.click(screen.getByLabelText(/include screenshot context/i));
    fireEvent.click(screen.getByRole("button", { name: /generate script/i }));

    const instructions = screen.getByLabelText(/instructions/i);
    await waitFor(() => {
      expect(instructions).toHaveValue("Derived pipeline goal v1");
    });

    fireEvent.change(instructions, {
      target: { value: "Use the visible app shell" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: /retry with changes/i }),
    );

    await waitFor(() => {
      expect(api.aiPipelineJsGenerate).toHaveBeenNthCalledWith(
        2,
        expect.objectContaining({
          body: expect.objectContaining({
            url: "https://example.com/app",
            name: "example-app",
            host_patterns: ["example.com"],
            instructions: "Use the visible app shell",
            images: [{ data: "ZmFrZQ==", mime_type: "image/png" }],
            headless: true,
            playwright: false,
            visual: true,
          }),
        }),
      );
    });
    await waitFor(() => {
      expect(instructions).toHaveValue("Use the visible app shell");
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

    fireEvent.change(instructions, {
      target: { value: "Wait for #app-root" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: /retry with changes/i }),
    );

    await waitFor(() => {
      expect(api.aiPipelineJsGenerate).toHaveBeenNthCalledWith(
        3,
        expect.objectContaining({
          body: expect.objectContaining({
            instructions: "Wait for #app-root",
            images: [{ data: "ZmFrZQ==", mime_type: "image/png" }],
            headless: true,
            playwright: false,
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
