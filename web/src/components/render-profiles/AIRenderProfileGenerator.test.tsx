/**
 * Purpose: Verify the AI render-profile generator modal request and save flows.
 * Responsibilities: Assert guided and instructionless submissions, payload shaping, and save handoff behavior.
 * Scope: `AIRenderProfileGenerator` tests only.
 * Usage: Run with `pnpm --dir web test`.
 * Invariants/Assumptions: URL remains required, operator guidance is optional, and generated profiles are only persisted after explicit save.
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

    const resolvedGoal = await screen.findByRole("region", {
      name: /resolved goal/i,
    });
    expect(within(resolvedGoal).getByText("Explicit")).toBeInTheDocument();
    expect(
      within(resolvedGoal).getByText(
        "Wait for the dashboard shell and prefer headless mode",
      ),
    ).toBeInTheDocument();

    fireEvent.click(
      await screen.findByRole("button", { name: /save profile/i }),
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

    expect(await screen.findByText("System-derived")).toBeInTheDocument();
    expect(
      screen.getByText(
        'Generate a render profile for "example-app" on example.com.',
      ),
    ).toBeInTheDocument();
  });
});
