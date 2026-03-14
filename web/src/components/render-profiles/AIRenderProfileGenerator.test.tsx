import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
});
