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
    window.sessionStorage.clear();
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

  it("restores a closed native profile draft after the Settings editor remounts and lets operators discard it intentionally", async () => {
    const firstRender = render(
      <ToastProvider>
        <RenderProfileEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /create profile/i }),
    );
    fireEvent.change(screen.getByLabelText(/^name$/i), {
      target: { value: "draft-profile" },
    });
    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "example.com" },
    });

    expect(screen.getByRole("status")).toHaveTextContent(/unsaved changes/i);

    fireEvent.click(screen.getByRole("button", { name: /^close$/i }));

    expect(
      await screen.findByRole("button", { name: /resume settings draft/i }),
    ).toBeInTheDocument();

    firstRender.unmount();

    render(
      <ToastProvider>
        <RenderProfileEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /resume settings draft/i }),
    );

    expect(screen.getByLabelText(/^name$/i)).toHaveValue("draft-profile");
    expect(screen.getByLabelText(/host patterns/i)).toHaveValue("example.com");

    fireEvent.click(screen.getByRole("button", { name: /discard draft/i }));

    const confirmDialog = await screen.findByRole("alertdialog");
    fireEvent.click(
      within(confirmDialog).getByRole("button", { name: /discard draft/i }),
    );

    await waitFor(() => {
      expect(
        screen.queryByRole("button", { name: /resume settings draft/i }),
      ).not.toBeInTheDocument();
    });
  });

  it("warns before replacing a dirty native profile draft when switching to another saved profile", async () => {
    vi.mocked(api.getV1RenderProfiles).mockResolvedValue({
      data: {
        profiles: [
          {
            name: "news",
            hostPatterns: ["example.com"],
            preferHeadless: true,
          },
          {
            name: "docs",
            hostPatterns: ["example.org"],
            preferHeadless: false,
          },
        ],
      },
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <RenderProfileEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      (await screen.findAllByRole("button", { name: /edit/i }))[0],
    );

    expect(
      await screen.findByRole("heading", { name: /edit saved profile/i }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByLabelText(/prefer headless/i));
    expect(screen.getByRole("status")).toHaveTextContent(/unsaved changes/i);

    fireEvent.click(screen.getAllByRole("button", { name: /edit/i })[1]);

    let confirmDialog = await screen.findByRole("alertdialog");
    expect(confirmDialog).toHaveTextContent(
      /replace the current settings draft/i,
    );
    fireEvent.click(
      within(confirmDialog).getByRole("button", { name: /keep draft/i }),
    );

    expect(screen.getByLabelText(/^name$/i)).toHaveValue("news");
    expect(screen.getByLabelText(/prefer headless/i)).not.toBeChecked();

    fireEvent.click(screen.getAllByRole("button", { name: /edit/i })[1]);

    confirmDialog = await screen.findByRole("alertdialog");
    fireEvent.click(
      within(confirmDialog).getByRole("button", { name: /discard draft/i }),
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/^name$/i)).toHaveValue("docs");
    });
    expect(screen.getByLabelText(/prefer headless/i)).not.toBeChecked();
  });

  it("keeps the generator and debugger modals mutually exclusive", async () => {
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
    vi.mocked(api.aiRenderProfileDebug).mockResolvedValue({
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
      request: new Request("http://localhost:8741/v1/ai/render-profile-debug"),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <RenderProfileEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /generate with ai/i }),
    );

    expect(
      await screen.findByRole("heading", {
        name: /generate render profile with ai/i,
      }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /tune with ai/i }));

    expect(
      await screen.findByRole("heading", {
        name: /tune render profile with ai/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", {
        name: /generate render profile with ai/i,
      }),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /generate with ai/i }));

    expect(
      await screen.findByRole("heading", {
        name: /generate render profile with ai/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: /tune render profile with ai/i }),
    ).not.toBeInTheDocument();
  });

  it("warns before replacing a hidden local Settings draft with an AI handoff draft", async () => {
    vi.mocked(api.getV1RenderProfiles).mockResolvedValue({
      data: { profiles: [] },
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(),
    });
    vi.mocked(api.aiRenderProfileGenerate).mockResolvedValue({
      data: {
        profile: {
          name: "generated-profile",
          hostPatterns: ["example.com"],
          wait: { mode: "selector", selector: "main" },
        },
        resolved_goal: {
          source: "explicit",
          text: "Generate a stable render profile",
        },
        route_id: "route-1",
      },
      request: new Request(
        "http://localhost:8741/v1/ai/render-profile-generate",
      ),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <RenderProfileEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /create profile/i }),
    );
    fireEvent.change(screen.getByLabelText(/^name$/i), {
      target: { value: "draft-profile" },
    });
    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /^close$/i }));

    fireEvent.click(screen.getByRole("button", { name: /generate with ai/i }));
    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.click(screen.getByRole("button", { name: /generate profile/i }));

    const history = await screen.findByRole("region", {
      name: /attempt history/i,
    });
    fireEvent.click(
      within(history).getByRole("button", {
        name: /edit attempt 1 in settings/i,
      }),
    );

    let confirmDialog = await screen.findByRole("alertdialog");
    expect(confirmDialog).toHaveTextContent(
      /replace the current settings draft/i,
    );
    fireEvent.click(
      within(confirmDialog).getByRole("button", { name: /keep draft/i }),
    );

    expect(
      screen.queryByRole("heading", {
        name: /create profile from ai session/i,
      }),
    ).not.toBeInTheDocument();

    fireEvent.click(
      within(history).getByRole("button", {
        name: /edit attempt 1 in settings/i,
      }),
    );

    confirmDialog = await screen.findByRole("alertdialog");
    fireEvent.click(
      within(confirmDialog).getByRole("button", { name: /discard draft/i }),
    );

    expect(
      await screen.findByRole("heading", {
        name: /create profile from ai session/i,
      }),
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: /back to ai session/i }),
    );

    expect(
      await screen.findByRole("button", { name: /resume ai handoff draft/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /resume settings draft/i }),
    ).not.toBeInTheDocument();
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

  it("clears the local JSON validation error once the draft changes", async () => {
    render(
      <ToastProvider>
        <RenderProfileEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /create profile/i }),
    );
    fireEvent.change(screen.getByLabelText(/^name$/i), {
      target: { value: "news" },
    });
    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "example.com" },
    });
    fireEvent.change(screen.getByLabelText(/wait configuration json/i), {
      target: { value: "{" },
    });

    fireEvent.click(screen.getByRole("button", { name: /^create$/i }));

    expect(
      await screen.findByText(/wait configuration must be valid json/i),
    ).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/wait configuration json/i), {
      target: { value: '{"mode":"selector","selector":"main"}' },
    });

    await waitFor(() => {
      expect(
        screen.queryByText(/wait configuration must be valid json/i),
      ).not.toBeInTheDocument();
    });
  });

  it("rejects non-object JSON for object-only render profile fields", async () => {
    render(
      <ToastProvider>
        <RenderProfileEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /create profile/i }),
    );
    fireEvent.change(screen.getByLabelText(/^name$/i), {
      target: { value: "news" },
    });
    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "example.com" },
    });
    fireEvent.change(screen.getByLabelText(/wait configuration json/i), {
      target: { value: "false" },
    });

    fireEvent.click(screen.getByRole("button", { name: /^create$/i }));

    expect(
      await screen.findByText(/wait configuration must be a json object/i),
    ).toBeInTheDocument();
    expect(api.postV1RenderProfiles).not.toHaveBeenCalled();
  });

  it("warns honestly when saving succeeds but the inventory refresh fails", async () => {
    vi.mocked(api.getV1RenderProfiles)
      .mockResolvedValueOnce({
        data: { profiles: [] },
        request: new Request("http://localhost:8741/v1/render-profiles"),
        response: new Response(),
      })
      .mockResolvedValue({
        data: undefined,
        error: { error: "refresh failed" },
        request: new Request("http://localhost:8741/v1/render-profiles"),
        response: new Response(null, { status: 500 }),
      });
    vi.mocked(api.postV1RenderProfiles).mockResolvedValue({
      data: {
        name: "news",
        hostPatterns: ["example.com"],
      },
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(),
    });

    const onError = vi.fn();
    render(
      <ToastProvider>
        <RenderProfileEditor onError={onError} />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /create profile/i }),
    );
    fireEvent.change(screen.getByLabelText(/^name$/i), {
      target: { value: "news" },
    });
    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /^create$/i }));

    await waitFor(() => {
      expect(api.postV1RenderProfiles).toHaveBeenCalled();
    });

    expect(
      screen.queryByRole("heading", { name: /create new profile/i }),
    ).not.toBeInTheDocument();
    expect(
      await screen.findByText(
        /saved, but the latest inventory refresh failed/i,
      ),
    ).toBeInTheDocument();
    expect(onError).toHaveBeenCalledWith("refresh failed");
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
    expect(screen.getByRole("status")).toHaveTextContent(/in sync with saved/i);
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

  it("keeps unsaved handoff edits local when returning to AI, while preserving the local draft", async () => {
    vi.mocked(api.getV1RenderProfiles).mockResolvedValue({
      data: { profiles: [] },
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(),
    });

    vi.mocked(api.aiRenderProfileGenerate).mockResolvedValue({
      data: {
        profile: {
          name: "news",
          hostPatterns: ["example.com"],
          wait: { mode: "selector", selector: "main" },
        },
        resolved_goal: {
          source: "explicit",
          text: "Generate a stable render profile",
        },
        route_id: "route-1",
      },
      request: new Request(
        "http://localhost:8741/v1/ai/render-profile-generate",
      ),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <RenderProfileEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /generate with ai/i }),
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });

    fireEvent.click(screen.getByRole("button", { name: /generate profile/i }));

    const history = await screen.findByRole("region", {
      name: /attempt history/i,
    });

    fireEvent.click(
      within(history).getByRole("button", {
        name: /edit attempt 1 in settings/i,
      }),
    );

    expect(
      await screen.findByText(/back to ai session returns to the modal/i),
    ).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent(/in sync with saved/i);

    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "example.com, *.example.com" },
    });

    expect(screen.getByRole("status")).toHaveTextContent(/unsaved changes/i);

    fireEvent.click(
      screen.getByRole("button", { name: /back to ai session/i }),
    );

    const latestCandidate = await screen.findByRole("region", {
      name: /latest candidate/i,
    });

    expect(
      within(latestCandidate).getByText(/example\.com/i),
    ).toBeInTheDocument();
    expect(
      within(latestCandidate).queryByText(/\*\.example\.com/i),
    ).not.toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: /edit attempt 1 in settings/i }),
    );

    expect(await screen.findByLabelText(/host patterns/i)).toHaveValue(
      "example.com, *.example.com",
    );
  });

  it("keeps the working profile draft after a save failure so the operator can retry", async () => {
    vi.mocked(api.getV1RenderProfiles).mockResolvedValue({
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
    });

    vi.mocked(api.aiRenderProfileDebug).mockResolvedValue({
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
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/ai/render-profile-debug"),
      response: new Response(),
    });

    vi.mocked(api.putV1RenderProfilesByName)
      .mockResolvedValueOnce({
        data: undefined,
        error: { error: "save failed" },
        request: new Request("http://localhost:8741/v1/render-profiles/news"),
        response: new Response(null, { status: 500 }),
      })
      .mockResolvedValueOnce({
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

    fireEvent.click(screen.getByLabelText(/prefer headless/i));

    expect(screen.getByRole("status")).toHaveTextContent(/unsaved changes/i);

    fireEvent.click(screen.getByRole("button", { name: /^update$/i }));

    expect(await screen.findAllByText(/save failed/i)).not.toHaveLength(0);
    expect(screen.getByLabelText(/prefer headless/i)).not.toBeChecked();
    expect(screen.getByRole("status")).toHaveTextContent(/unsaved changes/i);

    fireEvent.click(screen.getByRole("button", { name: /^update$/i }));

    await waitFor(() => {
      expect(api.putV1RenderProfilesByName).toHaveBeenNthCalledWith(
        2,
        expect.objectContaining({
          path: { name: "news" },
          body: expect.objectContaining({
            name: "news",
            hostPatterns: ["example.com"],
            wait: { mode: "selector", selector: "main" },
            preferHeadless: undefined,
          }),
        }),
      );
    });
  });

  it("restores a closed generator session after the Settings editor remounts", async () => {
    vi.mocked(api.getV1RenderProfiles).mockResolvedValue({
      data: { profiles: [] },
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(),
    });
    vi.mocked(api.aiRenderProfileGenerate).mockResolvedValue({
      data: {
        profile: {
          name: "news",
          hostPatterns: ["example.com"],
          wait: { mode: "selector", selector: "main" },
        },
        resolved_goal: {
          source: "explicit",
          text: "Keep the visible app shell",
        },
      },
      request: new Request(
        "http://localhost:8741/v1/ai/render-profile-generate",
      ),
      response: new Response(),
    });

    const firstRender = render(
      <ToastProvider>
        <RenderProfileEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /generate with ai/i }),
    );
    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.change(screen.getByLabelText(/instructions/i), {
      target: { value: "Keep the visible app shell" },
    });
    fireEvent.click(screen.getByRole("button", { name: /generate profile/i }));
    await screen.findByRole("region", { name: /attempt history/i });

    fireEvent.click(screen.getAllByRole("button", { name: /^close$/i })[0]);
    firstRender.unmount();

    render(
      <ToastProvider>
        <RenderProfileEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /generate with ai/i }),
    );

    expect(screen.getByLabelText(/target url/i)).toHaveValue(
      "https://example.com/app",
    );
    expect(screen.getByLabelText(/instructions/i)).toHaveValue(
      "Keep the visible app shell",
    );
    expect(
      screen.getByRole("region", { name: /attempt history/i }),
    ).toBeInTheDocument();
  });
});
