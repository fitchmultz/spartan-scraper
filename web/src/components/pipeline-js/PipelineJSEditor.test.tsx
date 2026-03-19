/**
 * Purpose: Verify the pipeline-JS Settings editor keeps AI helpers optional and non-blocking.
 * Responsibilities: Assert manual authoring remains available and AI-only actions disable cleanly when AI is unavailable.
 * Scope: PipelineJSEditor behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: AI generation/tuning is optional and should never feel required for first-run Settings workflows.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ToastProvider } from "../toast";
import { PipelineJSEditor } from "./PipelineJSEditor";
import * as api from "../../api";

vi.mock("../../api", () => ({
  getV1PipelineJs: vi.fn(),
  postV1PipelineJs: vi.fn(),
  putV1PipelineJsByName: vi.fn(),
  deleteV1PipelineJsByName: vi.fn(),
}));

describe("PipelineJSEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getV1PipelineJs).mockResolvedValue({
      data: {
        scripts: [
          {
            name: "normalize-app-shell",
            hostPatterns: ["example.com"],
            selectors: ["main"],
            postNav: "window.scrollTo(0, 0);",
          },
        ],
      },
      request: new Request("http://localhost:8741/v1/pipeline-js"),
      response: new Response(),
    });
  });

  it("shows a guided first-run empty state and reports inventory count", async () => {
    vi.mocked(api.getV1PipelineJs).mockResolvedValue({
      data: { scripts: [] },
      request: new Request("http://localhost:8741/v1/pipeline-js"),
      response: new Response(),
    });

    const onInventoryChange = vi.fn();

    render(
      <ToastProvider>
        <PipelineJSEditor onInventoryChange={onInventoryChange} />
      </ToastProvider>,
    );

    expect(
      await screen.findByText(/no pipeline scripts yet/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /most sites do not need custom javascript in the fetch pipeline/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /create your first script/i }),
    ).toBeEnabled();
    expect(onInventoryChange).toHaveBeenCalledWith(0);
  });

  it("disables AI actions and explains the manual path when AI is unavailable", async () => {
    render(
      <ToastProvider>
        <PipelineJSEditor
          aiStatus={{
            status: "disabled",
            message: "AI helpers are optional and currently disabled.",
          }}
        />
      </ToastProvider>,
    );

    expect(
      await screen.findByText(
        /create and edit scripts manually below\.? turn ai on later/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /generate with ai/i }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /tune with ai/i }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /create script/i }),
    ).toBeEnabled();
  });
});
