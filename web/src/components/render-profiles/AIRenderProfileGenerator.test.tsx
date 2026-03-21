/**
 * Purpose: Verify the AI render-profile generator modal request, retry, and save flows.
 * Responsibilities: Assert guided and instructionless submissions, resolved-goal rendering, retry preservation, and save handoff behavior.
 * Scope: `AIRenderProfileGenerator` tests only.
 * Usage: Run with `pnpm --dir web test`.
 * Invariants/Assumptions: URL remains required, retry keeps request-scoped inputs intact, and generated profiles are only persisted after explicit save.
 */
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { AIRenderProfileGenerator } from "../AIRenderProfileGenerator";
import * as api from "../../api";

vi.mock("../../api", () => ({
  aiRenderProfileGenerate: vi.fn(),
  postV1RenderProfiles: vi.fn(),
}));

describe("AIRenderProfileGenerator", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls aiRenderProfileGenerate with URL mode options and saves the result", async () => {
    vi.mocked(api.aiRenderProfileGenerate).mockResolvedValue({
      data: {
        profile: {
          name: "example-app",
          hostPatterns: ["example.com"],
          preferHeadless: true,
        },
        resolved_goal: {
          source: "explicit",
          text: "Wait for the dashboard shell and prefer headless mode",
        },
        explanation: "Use headless mode for the app shell.",
        route_id: "openai/gpt-5.4",
        provider: "openai",
        model: "gpt-5.4",
        visual_context_used: true,
      },
      request: new Request(
        "http://localhost:8741/v1/ai/render-profile-generate",
      ),
      response: new Response(),
    });
    vi.mocked(api.postV1RenderProfiles).mockResolvedValue({
      data: {
        name: "example-app",
        hostPatterns: ["example.com"],
        preferHeadless: true,
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(),
    });

    const onSaved = vi.fn();
    const onClose = vi.fn();
    render(
      <AIRenderProfileGenerator isOpen onClose={onClose} onSaved={onSaved} />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.change(screen.getByLabelText(/^profile name$/i), {
      target: { value: "example-app" },
    });
    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "example.com, *.example.com" },
    });
    fireEvent.change(screen.getByLabelText(/instructions/i), {
      target: {
        value: "Wait for the dashboard shell and prefer headless mode",
      },
    });
    const image = new File(["fake"], "profile.png", { type: "image/png" });
    fireEvent.change(screen.getByLabelText(/upload images/i), {
      target: { files: [image] },
    });
    await screen.findByText("profile.png");
    fireEvent.click(screen.getByLabelText(/fetch headless/i));
    fireEvent.click(screen.getByLabelText(/use playwright/i));
    fireEvent.click(screen.getByLabelText(/include screenshot context/i));
    fireEvent.click(screen.getByRole("button", { name: /generate profile/i }));

    await waitFor(() => {
      expect(api.aiRenderProfileGenerate).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        body: {
          url: "https://example.com/app",
          name: "example-app",
          host_patterns: ["example.com", "*.example.com"],
          instructions: "Wait for the dashboard shell and prefer headless mode",
          images: [{ data: "ZmFrZQ==", mime_type: "image/png" }],
          headless: true,
          playwright: true,
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
        "Wait for the dashboard shell and prefer headless mode",
      ),
    ).toBeInTheDocument();

    fireEvent.click(
      await screen.findByRole("button", { name: /save selected profile/i }),
    );

    await waitFor(() => {
      expect(api.postV1RenderProfiles).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        body: {
          name: "example-app",
          hostPatterns: ["example.com"],
          preferHeadless: true,
        },
      });
    });

    expect(onSaved).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("allows render-profile generation without instructions", async () => {
    vi.mocked(api.aiRenderProfileGenerate).mockResolvedValue({
      data: {
        profile: {
          name: "example-app",
          hostPatterns: ["example.com"],
          preferHeadless: true,
        },
        resolved_goal: {
          source: "derived",
          text: 'Generate a render profile for "example-app" on example.com.',
        },
      },
      request: new Request(
        "http://localhost:8741/v1/ai/render-profile-generate",
      ),
      response: new Response(),
    });

    render(
      <AIRenderProfileGenerator isOpen onClose={vi.fn()} onSaved={vi.fn()} />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.click(screen.getByRole("button", { name: /generate profile/i }));

    await waitFor(() => {
      expect(api.aiRenderProfileGenerate).toHaveBeenCalledWith({
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
        'Generate a render profile for "example-app" on example.com.',
      ),
    ).toBeInTheDocument();
  });

  it("retains full history, restores non-latest guidance, switches baselines, and saves a restored attempt", async () => {
    vi.mocked(api.aiRenderProfileGenerate)
      .mockResolvedValueOnce({
        data: {
          profile: {
            name: "example-app",
            hostPatterns: ["example.com"],
            wait: { mode: "selector", selector: "main" },
          },
          resolved_goal: { source: "derived", text: "Derived goal v1" },
          route_id: "route-1",
          provider: "openai",
          model: "gpt-5.4",
          visual_context_used: true,
        },
        request: new Request(
          "http://localhost:8741/v1/ai/render-profile-generate",
        ),
        response: new Response(),
      })
      .mockResolvedValueOnce({
        data: {
          profile: {
            name: "example-app",
            hostPatterns: ["example.com"],
            preferHeadless: true,
            wait: { mode: "selector", selector: "#app-root" },
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
          "http://localhost:8741/v1/ai/render-profile-generate",
        ),
        response: new Response(),
      })
      .mockResolvedValueOnce({
        data: {
          profile: {
            name: "example-app",
            hostPatterns: ["example.com"],
            wait: { mode: "selector", selector: "#app-root" },
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
          "http://localhost:8741/v1/ai/render-profile-generate",
        ),
        response: new Response(),
      });
    vi.mocked(api.postV1RenderProfiles).mockResolvedValue({
      data: {
        name: "example-app",
        hostPatterns: ["example.com"],
        wait: { mode: "selector", selector: "main" },
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(),
    });

    render(
      <AIRenderProfileGenerator isOpen onClose={vi.fn()} onSaved={vi.fn()} />,
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.change(screen.getByLabelText(/^profile name$/i), {
      target: { value: "example-app" },
    });
    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "example.com" },
    });
    const image = new File(["fake"], "retry-profile.png", {
      type: "image/png",
    });
    fireEvent.change(screen.getByLabelText(/upload images/i), {
      target: { files: [image] },
    });
    await screen.findByText("retry-profile.png");
    fireEvent.click(screen.getByLabelText(/fetch headless/i));
    fireEvent.click(screen.getByLabelText(/use playwright/i));
    fireEvent.click(screen.getByLabelText(/include screenshot context/i));
    fireEvent.click(screen.getByRole("button", { name: /generate profile/i }));

    const instructions = screen.getByLabelText(/instructions/i);
    await waitFor(() => {
      expect(instructions).toHaveValue("Derived goal v1");
    });

    fireEvent.change(instructions, {
      target: { value: "Use the visible app shell" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: /retry with changes/i }),
    );

    await waitFor(() => {
      expect(api.aiRenderProfileGenerate).toHaveBeenNthCalledWith(
        2,
        expect.objectContaining({
          body: expect.objectContaining({
            url: "https://example.com/app",
            name: "example-app",
            host_patterns: ["example.com"],
            instructions: "Use the visible app shell",
            images: [{ data: "ZmFrZQ==", mime_type: "image/png" }],
            headless: true,
            playwright: true,
            visual: true,
          }),
        }),
      );
    });
    await waitFor(() => {
      expect(instructions).toHaveValue("Use the visible app shell");
    });

    fireEvent.change(instructions, {
      target: { value: "Wait for #app-root" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: /retry with changes/i }),
    );

    await waitFor(() => {
      expect(api.aiRenderProfileGenerate).toHaveBeenNthCalledWith(
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
    expect(instructions).toHaveValue("Derived goal v1");

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
      screen.getByRole("button", { name: /save selected profile/i }),
    );

    await waitFor(() => {
      expect(api.postV1RenderProfiles).toHaveBeenCalledWith({
        baseUrl: expect.any(String),
        body: expect.objectContaining({
          wait: { mode: "selector", selector: "main" },
        }),
      });
    });
  });
});
