import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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

    expect(screen.getByText(/detected issues/i)).toBeInTheDocument();
    expect(
      screen.getByText(/selectors\[0\] matched no elements/i),
    ).toBeInTheDocument();

    fireEvent.click(
      await screen.findByRole("button", { name: /save tuned script/i }),
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
});
