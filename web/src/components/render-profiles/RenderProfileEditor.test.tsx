/**
 * Purpose: Verify the render-profile Settings editor keeps optional AI helpers non-blocking and supports round-trip AI handoff.
 * Responsibilities: Assert manual profile management remains available, AI assistance disables cleanly, and manual Settings edits preserve AI session history for continued retries.
 * Scope: RenderProfileEditor behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Render profiles stay manually editable even when AI helpers are disabled.
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { ToastProvider } from "../toast";
import { getApiBaseUrl } from "../../lib/api-config";
import { RenderProfileEditor } from "./RenderProfileEditor";
import * as api from "../../api";

vi.mock("../../api", () => ({
  aiRenderProfileDebug: vi.fn(),
  aiRenderProfileGenerate: vi.fn(),
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

  it("preserves AI history after manual Settings edits and retries from the edited profile", async () => {
    vi.mocked(api.getV1RenderProfiles)
      .mockResolvedValueOnce({
        data: {
          profiles: [
            {
              name: "news",
              hostPatterns: ["example.com"],
              wait: { mode: "selector", selector: ".missing" },
            },
          ],
        },
        request: new Request("http://localhost:8741/v1/render-profiles"),
        response: new Response(),
      })
      .mockResolvedValue({
        data: {
          profiles: [
            {
              name: "news",
              hostPatterns: ["example.com"],
              preferHeadless: false,
              wait: { mode: "selector", selector: "main" },
            },
          ],
        },
        request: new Request("http://localhost:8741/v1/render-profiles"),
        response: new Response(),
      });

    vi.mocked(api.aiRenderProfileDebug)
      .mockResolvedValueOnce({
        data: {
          issues: ["wait.selector matched no elements"],
          resolved_goal: {
            source: "explicit",
            text: "Use the visible main shell",
          },
          suggested_profile: {
            name: "news",
            hostPatterns: ["example.com"],
            preferHeadless: true,
            wait: { mode: "selector", selector: "main" },
          },
          route_id: "route-1",
          provider: "openai",
          model: "gpt-5.4",
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
            text: "Keep the edited profile and simplify waits",
          },
          suggested_profile: {
            name: "news",
            hostPatterns: ["example.com"],
            preferHeadless: false,
            wait: { mode: "selector", selector: "main" },
          },
          route_id: "route-2",
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
        name: "news",
        hostPatterns: ["example.com"],
        preferHeadless: false,
        wait: { mode: "selector", selector: "main" },
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/render-profiles/news"),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <RenderProfileEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /tune with ai/i }),
    );
    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.click(screen.getByRole("button", { name: /tune profile/i }));

    const history = await screen.findByRole("region", {
      name: /attempt history/i,
    });
    fireEvent.click(
      within(history).getByRole("button", {
        name: /edit attempt 1 in settings/i,
      }),
    );

    expect(
      await screen.findByRole("heading", {
        name: /edit profile from ai session/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/^name$/i)).toHaveValue("news");
    fireEvent.click(screen.getByLabelText(/prefer headless/i));
    fireEvent.click(screen.getByRole("button", { name: /^update$/i }));

    expect(
      await screen.findAllByText(/manually edited in settings/i),
    ).toHaveLength(2);
    expect(
      screen.getAllByText(/manually edited/i).length,
    ).toBeGreaterThanOrEqual(2);

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
              preferHeadless: false,
              wait: { mode: "selector", selector: "main" },
            }),
          }),
        }),
      );
    });
  });
});
