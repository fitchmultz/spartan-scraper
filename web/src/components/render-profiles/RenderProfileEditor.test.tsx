/**
 * Purpose: Verify the render-profile Settings editor keeps optional AI helpers non-blocking.
 * Responsibilities: Assert manual profile management remains available when create calls fail or AI assistance is intentionally unavailable.
 * Scope: RenderProfileEditor behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Render profiles stay manually editable even when AI helpers are disabled.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ToastProvider } from "../toast";
import { getApiBaseUrl } from "../../lib/api-config";
import { RenderProfileEditor } from "./RenderProfileEditor";
import * as api from "../../api";

vi.mock("../../api", () => ({
  getV1RenderProfiles: vi.fn(),
  postV1RenderProfiles: vi.fn(),
  putV1RenderProfilesByName: vi.fn(),
  deleteV1RenderProfilesByName: vi.fn(),
}));

vi.mock("../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

describe("RenderProfileEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getV1RenderProfiles).mockResolvedValue({
      data: { profiles: [] },
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(),
    });
  });

  it("shows a guided first-run empty state and reports inventory count", async () => {
    const onInventoryChange = vi.fn();

    render(
      <ToastProvider>
        <RenderProfileEditor onInventoryChange={onInventoryChange} />
      </ToastProvider>,
    );

    expect(
      await screen.findByText(/no saved render profiles yet/i),
    ).toBeInTheDocument();
    expect(api.getV1RenderProfiles).toHaveBeenCalledWith({
      baseUrl: getApiBaseUrl(),
    });
    expect(
      screen.getByText(
        /most jobs can use spartan's default runtime selection/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /create your first profile/i }),
    ).toBeEnabled();
    expect(onInventoryChange).toHaveBeenCalledWith(0);
  });

  it("disables AI actions and explains the manual path when AI is unavailable", async () => {
    vi.mocked(api.getV1RenderProfiles).mockResolvedValue({
      data: {
        profiles: [
          {
            name: "news",
            hostPatterns: ["example.com"],
            preferHeadless: true,
          },
        ],
      },
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <RenderProfileEditor
          aiStatus={{
            status: "disabled",
            message: "AI helpers are optional and currently disabled.",
          }}
        />
      </ToastProvider>,
    );

    expect(
      await screen.findByText(
        /create and edit profiles manually below\.? turn ai on later/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /generate with ai/i }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /tune with ai/i }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /create profile/i }),
    ).toBeEnabled();
  });

  it("keeps the create form open and shows the API error when create fails", async () => {
    vi.mocked(api.postV1RenderProfiles).mockResolvedValue({
      data: undefined,
      error: { error: "invalid hostPatterns" },
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(null, { status: 400 }),
    });

    const onError = vi.fn();
    render(
      <ToastProvider>
        <RenderProfileEditor onError={onError} />
      </ToastProvider>,
    );

    await screen.findByRole("button", { name: /create profile/i });

    fireEvent.click(screen.getByRole("button", { name: /create profile/i }));
    fireEvent.change(screen.getByLabelText(/^name$/i), {
      target: { value: "news" },
    });
    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "bad pattern" },
    });
    fireEvent.click(screen.getByRole("button", { name: /^create$/i }));

    await waitFor(() => {
      expect(api.postV1RenderProfiles).toHaveBeenCalledWith({
        baseUrl: getApiBaseUrl(),
        body: expect.objectContaining({
          name: "news",
          hostPatterns: ["bad pattern"],
        }),
      });
    });

    expect(await screen.findAllByText("invalid hostPatterns")).toHaveLength(2);
    expect(
      screen.getByRole("heading", { name: /create new profile/i }),
    ).toBeInTheDocument();
    expect(api.getV1RenderProfiles).toHaveBeenCalledTimes(1);
    expect(onError).toHaveBeenCalledWith("invalid hostPatterns");
  });
});
