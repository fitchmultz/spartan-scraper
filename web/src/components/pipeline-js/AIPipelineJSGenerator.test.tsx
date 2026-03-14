import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
        baseUrl: "",
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
});
