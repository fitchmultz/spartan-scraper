/**
 * Purpose: Verify the pipeline-JS Settings editor keeps AI helpers optional and supports round-trip AI handoff.
 * Responsibilities: Assert manual authoring remains available, AI-only actions disable cleanly, and manual Settings edits preserve AI session history for continued retries.
 * Scope: PipelineJSEditor behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: AI generation/tuning is optional and should never feel required for first-run Settings workflows.
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
import { PipelineJSEditor } from "./PipelineJSEditor";
import * as api from "../../api";

vi.mock("../../api", () => ({
  aiPipelineJsDebug: vi.fn(),
  aiPipelineJsGenerate: vi.fn(),
  getV1PipelineJs: vi.fn(),
  postV1PipelineJs: vi.fn(),
  putV1PipelineJsByName: vi.fn(),
  deleteV1PipelineJsByName: vi.fn(),
}));

vi.mock("../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

describe("PipelineJSEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.sessionStorage.clear();
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
    expect(api.getV1PipelineJs).toHaveBeenCalledWith({
      baseUrl: getApiBaseUrl(),
    });
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

  it("preserves AI history after manual Settings edits and retries from the edited script", async () => {
    vi.mocked(api.getV1PipelineJs)
      .mockResolvedValueOnce({
        data: {
          scripts: [
            {
              name: "normalize-app-shell",
              hostPatterns: ["example.com"],
              selectors: [".missing"],
            },
          ],
        },
        request: new Request("http://localhost:8741/v1/pipeline-js"),
        response: new Response(),
      })
      .mockResolvedValue({
        data: {
          scripts: [
            {
              name: "normalize-app-shell",
              hostPatterns: ["example.com"],
              selectors: ["#app-root"],
            },
          ],
        },
        request: new Request("http://localhost:8741/v1/pipeline-js"),
        response: new Response(),
      });

    vi.mocked(api.aiPipelineJsDebug)
      .mockResolvedValueOnce({
        data: {
          issues: ["selectors[0] matched no elements"],
          resolved_goal: {
            source: "explicit",
            text: "Use the visible app shell",
          },
          suggested_script: {
            name: "normalize-app-shell",
            hostPatterns: ["example.com"],
            selectors: ["main"],
          },
          route_id: "route-1",
          provider: "openai",
          model: "gpt-5.4",
        },
        error: undefined,
        request: new Request("http://localhost:8741/v1/ai/pipeline-js-debug"),
        response: new Response(),
      })
      .mockResolvedValueOnce({
        data: {
          issues: ["selector verified"],
          resolved_goal: {
            source: "explicit",
            text: "Keep the edited selector and add a scroll reset",
          },
          suggested_script: {
            name: "normalize-app-shell",
            hostPatterns: ["example.com"],
            selectors: ["#app-root"],
            postNav: "window.scrollTo(0, 0);",
          },
          route_id: "route-2",
          provider: "openai",
          model: "gpt-5.4",
        },
        error: undefined,
        request: new Request("http://localhost:8741/v1/ai/pipeline-js-debug"),
        response: new Response(),
      });

    vi.mocked(api.putV1PipelineJsByName).mockResolvedValue({
      data: {
        name: "normalize-app-shell",
        hostPatterns: ["example.com"],
        selectors: ["#app-root"],
      },
      error: undefined,
      request: new Request(
        "http://localhost:8741/v1/pipeline-js/normalize-app-shell",
      ),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <PipelineJSEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /tune with ai/i }),
    );
    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.click(screen.getByRole("button", { name: /tune script/i }));

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
        name: /edit script from ai session/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent(/in sync with saved/i);
    expect(screen.getByLabelText(/^name$/i)).toHaveValue("normalize-app-shell");
    fireEvent.change(screen.getByLabelText(/wait selectors/i), {
      target: { value: "#app-root" },
    });
    expect(screen.getByRole("status")).toHaveTextContent(/unsaved changes/i);
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
      expect(api.aiPipelineJsDebug).toHaveBeenNthCalledWith(
        2,
        expect.objectContaining({
          body: expect.objectContaining({
            url: "https://example.com/app",
            script: expect.objectContaining({
              selectors: ["#app-root"],
            }),
          }),
        }),
      );
    });
  });

  it("keeps the working draft after a save failure so the operator can retry", async () => {
    vi.mocked(api.getV1PipelineJs).mockResolvedValue({
      data: {
        scripts: [
          {
            name: "normalize-app-shell",
            hostPatterns: ["example.com"],
            selectors: [".missing"],
          },
        ],
      },
      request: new Request("http://localhost:8741/v1/pipeline-js"),
      response: new Response(),
    });

    vi.mocked(api.aiPipelineJsDebug).mockResolvedValue({
      data: {
        issues: ["selectors[0] matched no elements"],
        resolved_goal: {
          source: "explicit",
          text: "Use the visible app shell",
        },
        suggested_script: {
          name: "normalize-app-shell",
          hostPatterns: ["example.com"],
          selectors: ["main"],
        },
        route_id: "route-1",
      },
      error: undefined,
      request: new Request("http://localhost:8741/v1/ai/pipeline-js-debug"),
      response: new Response(),
    });

    vi.mocked(api.putV1PipelineJsByName)
      .mockResolvedValueOnce({
        data: undefined,
        error: { error: "save failed" },
        request: new Request(
          "http://localhost:8741/v1/pipeline-js/normalize-app-shell",
        ),
        response: new Response(null, { status: 500 }),
      })
      .mockResolvedValueOnce({
        data: {
          name: "normalize-app-shell",
          hostPatterns: ["example.com"],
          selectors: ["#app-root"],
        },
        error: undefined,
        request: new Request(
          "http://localhost:8741/v1/pipeline-js/normalize-app-shell",
        ),
        response: new Response(),
      });

    render(
      <ToastProvider>
        <PipelineJSEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /tune with ai/i }),
    );
    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });
    fireEvent.click(screen.getByRole("button", { name: /tune script/i }));

    const history = await screen.findByRole("region", {
      name: /attempt history/i,
    });

    fireEvent.click(
      within(history).getByRole("button", {
        name: /edit attempt 1 in settings/i,
      }),
    );

    fireEvent.change(screen.getByLabelText(/wait selectors/i), {
      target: { value: "#app-root" },
    });

    expect(screen.getByRole("status")).toHaveTextContent(/unsaved changes/i);

    fireEvent.click(screen.getByRole("button", { name: /^update$/i }));

    expect(await screen.findAllByText(/save failed/i)).not.toHaveLength(0);
    expect(screen.getByLabelText(/wait selectors/i)).toHaveValue("#app-root");
    expect(screen.getByRole("status")).toHaveTextContent(/unsaved changes/i);

    fireEvent.click(screen.getByRole("button", { name: /^update$/i }));

    await waitFor(() => {
      expect(api.putV1PipelineJsByName).toHaveBeenNthCalledWith(
        2,
        expect.objectContaining({
          body: expect.objectContaining({
            selectors: ["#app-root"],
          }),
        }),
      );
    });
  });

  it("keeps unsaved handoff edits local when returning to AI, while preserving the local draft", async () => {
    vi.mocked(api.getV1PipelineJs).mockResolvedValue({
      data: { scripts: [] },
      request: new Request("http://localhost:8741/v1/pipeline-js"),
      response: new Response(),
    });

    vi.mocked(api.aiPipelineJsGenerate).mockResolvedValue({
      data: {
        script: {
          name: "normalize-app-shell",
          hostPatterns: ["example.com"],
          selectors: ["main"],
        },
        resolved_goal: {
          source: "explicit",
          text: "Generate a stable pipeline script",
        },
        route_id: "route-1",
      },
      request: new Request("http://localhost:8741/v1/ai/pipeline-js-generate"),
      response: new Response(),
    });

    render(
      <ToastProvider>
        <PipelineJSEditor />
      </ToastProvider>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /generate with ai/i }),
    );

    fireEvent.change(screen.getByLabelText(/target url/i), {
      target: { value: "https://example.com/app" },
    });

    fireEvent.click(screen.getByRole("button", { name: /generate script/i }));

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

    fireEvent.change(screen.getByLabelText(/wait selectors/i), {
      target: { value: "main, #app-root" },
    });

    expect(screen.getByRole("status")).toHaveTextContent(/unsaved changes/i);

    fireEvent.click(
      screen.getByRole("button", { name: /back to ai session/i }),
    );

    const latestCandidate = await screen.findByRole("region", {
      name: /latest candidate/i,
    });

    expect(within(latestCandidate).getByText(/"main"/i)).toBeInTheDocument();
    expect(
      within(latestCandidate).queryByText(/"#app-root"/i),
    ).not.toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: /edit attempt 1 in settings/i }),
    );

    expect(await screen.findByLabelText(/wait selectors/i)).toHaveValue(
      "main, #app-root",
    );
  });

  it("restores a closed generator session after the Settings editor remounts", async () => {
    vi.mocked(api.getV1PipelineJs).mockResolvedValue({
      data: { scripts: [] },
      request: new Request("http://localhost:8741/v1/pipeline-js"),
      response: new Response(),
    });
    vi.mocked(api.aiPipelineJsGenerate).mockResolvedValue({
      data: {
        script: {
          name: "normalize-app-shell",
          hostPatterns: ["example.com"],
          selectors: ["main"],
        },
        resolved_goal: {
          source: "explicit",
          text: "Keep the visible app shell",
        },
      },
      request: new Request("http://localhost:8741/v1/ai/pipeline-js-generate"),
      response: new Response(),
    });

    const firstRender = render(
      <ToastProvider>
        <PipelineJSEditor />
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
    fireEvent.click(screen.getByRole("button", { name: /generate script/i }));
    await screen.findByRole("region", { name: /attempt history/i });

    fireEvent.click(screen.getAllByRole("button", { name: /^close$/i })[0]);
    firstRender.unmount();

    render(
      <ToastProvider>
        <PipelineJSEditor />
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
