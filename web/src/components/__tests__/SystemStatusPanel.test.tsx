/**
 * Purpose: Verify setup and degraded diagnostics remain actionable in the web UI.
 * Responsibilities: Assert copy-ready commands, routed actions, external links, and one-click diagnostic checks render and execute correctly.
 * Scope: `SystemStatusPanel` behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Diagnostic actions are read-only, clipboard copy stays operator-friendly, and successful one-click checks refresh health state.
 */

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  postV1DiagnosticsAiCheck,
  postV1DiagnosticsBrowserCheck,
  postV1DiagnosticsProxyPoolCheck,
  type HealthResponse,
} from "../../api";
import { SystemStatusPanel } from "../SystemStatusPanel";

vi.mock("../../api", () => ({
  postV1DiagnosticsAiCheck: vi.fn(),
  postV1DiagnosticsBrowserCheck: vi.fn(),
  postV1DiagnosticsProxyPoolCheck: vi.fn(),
}));

vi.mock("../../lib/api-config", () => ({
  getApiBaseUrl: () => "http://127.0.0.1:8741",
}));

describe("SystemStatusPanel", () => {
  beforeEach(() => {
    const request = new Request("http://127.0.0.1:8741/healthz");
    const response = new Response(null, { status: 200 });

    vi.mocked(postV1DiagnosticsAiCheck).mockResolvedValue({
      data: {
        status: "disabled",
        message: "AI is disabled.",
      },
      request,
      response,
    } as never);
    vi.mocked(postV1DiagnosticsBrowserCheck).mockResolvedValue({
      data: {
        status: "ok",
        title: "Browser automation is ready",
        message: "Detected browser tooling at /usr/bin/chromium.",
      },
      request,
      response,
    } as never);
    vi.mocked(postV1DiagnosticsProxyPoolCheck).mockResolvedValue({
      data: {
        status: "disabled",
        message: "Proxy pool is disabled.",
      },
      request,
      response,
    } as never);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.clearAllMocks();
  });

  it("falls back when clipboard permissions are denied", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockRejectedValue(new Error("denied"));
    const execCommand = vi.fn().mockReturnValue(true);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });
    Object.defineProperty(document, "execCommand", {
      value: execCommand,
      configurable: true,
    });

    const health: HealthResponse = {
      status: "setup_required",
      version: "test",
      components: {},
      setup: {
        required: true,
        title: "Setup required",
        message: "Recover intentionally.",
        dataDir: ".data",
        actions: [
          {
            label: "Copy reset command",
            kind: "copy",
            value: "spartan reset-data",
          },
        ],
      },
    };

    render(
      <SystemStatusPanel
        health={health}
        onNavigate={vi.fn()}
        onRefresh={vi.fn()}
      />,
    );

    await user.click(
      screen.getByRole("button", { name: "Copy Copy reset command" }),
    );

    expect(writeText).toHaveBeenCalledWith("spartan reset-data");
    expect(execCommand).toHaveBeenCalledWith("copy");
    expect(
      screen.getByRole("button", { name: "Copy Copy reset command" }),
    ).toHaveTextContent("Copied!");
  });

  it("copies setup commands and renders one-click results", async () => {
    const user = userEvent.setup();
    const onRefresh = vi.fn().mockResolvedValue(undefined);

    const health: HealthResponse = {
      status: "setup_required",
      version: "test",
      components: {
        browser: {
          status: "degraded",
          message: "Chrome is missing.",
          actions: [
            {
              label: "Re-check browser tooling",
              kind: "one-click",
              value: "/v1/diagnostics/browser-check",
            },
          ],
        },
      },
      setup: {
        required: true,
        title: "Setup required",
        message: "Recover intentionally.",
        dataDir: ".data",
        actions: [
          {
            label: "Copy reset command",
            kind: "copy",
            value: "spartan reset-data",
          },
        ],
      },
    };

    render(
      <SystemStatusPanel
        health={health}
        onNavigate={vi.fn()}
        onRefresh={onRefresh}
      />,
    );

    await user.click(
      screen.getByRole("button", { name: "Copy Copy reset command" }),
    );
    expect(
      screen.getByRole("button", { name: "Copy Copy reset command" }),
    ).toHaveTextContent("Copied!");

    await user.click(
      screen.getByRole("button", { name: "Re-check browser tooling" }),
    );

    expect(
      await screen.findByText("Detected browser tooling at /usr/bin/chromium."),
    ).toBeInTheDocument();
    expect(postV1DiagnosticsBrowserCheck).toHaveBeenCalledWith({
      baseUrl: "http://127.0.0.1:8741",
    });
    expect(onRefresh).toHaveBeenCalled();
  });

  it("renders follow-up diagnostic actions without recursing infinitely", async () => {
    const user = userEvent.setup();
    const onRefresh = vi.fn().mockResolvedValue(undefined);

    vi.mocked(postV1DiagnosticsBrowserCheck).mockResolvedValue({
      data: {
        status: "degraded",
        title: "Playwright is still unavailable",
        message: "Chrome is present, but Playwright is not ready.",
        actions: [
          {
            label: "Re-check browser tooling",
            kind: "one-click",
            value: "/v1/diagnostics/browser-check",
          },
          {
            label: "Install Playwright drivers",
            kind: "copy",
            value:
              "go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install --with-deps",
          },
        ],
      },
      request: new Request(
        "http://127.0.0.1:8741/v1/diagnostics/browser-check",
      ),
      response: new Response(null, { status: 200 }),
    } as never);

    const health: HealthResponse = {
      status: "degraded",
      version: "test",
      components: {
        browser: {
          status: "degraded",
          message: "Browser tooling missing.",
          actions: [
            {
              label: "Re-check browser tooling",
              kind: "one-click",
              value: "/v1/diagnostics/browser-check",
            },
          ],
        },
      },
    };

    render(
      <SystemStatusPanel
        health={health}
        onNavigate={vi.fn()}
        onRefresh={onRefresh}
      />,
    );

    await user.click(
      screen.getByRole("button", { name: "Re-check browser tooling" }),
    );

    expect(
      await screen.findByText(
        "Chrome is present, but Playwright is not ready.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getAllByRole("button", { name: "Re-check browser tooling" }),
    ).toHaveLength(2);
    expect(screen.getByText("Install Playwright drivers")).toBeInTheDocument();
    expect(onRefresh).toHaveBeenCalled();
  });

  it("renders setup recovery alongside degraded subsystem actions without duplicating setup notices", async () => {
    const user = userEvent.setup();
    const onRefresh = vi.fn().mockResolvedValue(undefined);

    const health: HealthResponse = {
      status: "setup_required",
      version: "test",
      components: {
        database: {
          status: "setup_required",
          message: "Legacy jobs.db detected.",
        },
        queue: {
          status: "setup_required",
          message: "Job processing stays unavailable until setup is completed.",
        },
        browser: {
          status: "degraded",
          message: "Chrome is missing.",
          actions: [
            {
              label: "Install Chromium on Fedora",
              kind: "copy",
              value: "sudo dnf install chromium",
            },
          ],
        },
        ai: {
          status: "degraded",
          message: "AI failed to initialize.",
          actions: [
            {
              label: "Re-check AI prerequisites",
              kind: "one-click",
              value: "/v1/diagnostics/ai-check",
            },
          ],
        },
        proxy_pool: {
          status: "degraded",
          message: "Proxy pool configuration is present but not loaded.",
          actions: [
            {
              label: "Disable proxy pool intentionally",
              kind: "copy",
              value: "PROXY_POOL_FILE=",
            },
          ],
        },
      },
      notices: [
        {
          id: "setup-required",
          scope: "setup",
          severity: "error",
          title: "Setup required",
          message: "Recover intentionally.",
        },
      ],
      setup: {
        required: true,
        title: "Setup required",
        message: "Recover intentionally.",
        dataDir: ".data",
        actions: [
          {
            label: "Copy reset command",
            kind: "copy",
            value: "spartan reset-data",
          },
        ],
      },
    };

    render(
      <SystemStatusPanel
        health={health}
        onNavigate={vi.fn()}
        onRefresh={onRefresh}
      />,
    );

    expect(screen.getByText("Chrome is missing.")).toBeInTheDocument();
    expect(screen.getByText("AI failed to initialize.")).toBeInTheDocument();
    expect(
      screen.getByText("Proxy pool configuration is present but not loaded."),
    ).toBeInTheDocument();
    expect(screen.queryByText("Recover intentionally.")).toBeInTheDocument();
    expect(screen.queryAllByText("Recover intentionally.")).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: "Refresh status" }));
    expect(onRefresh).toHaveBeenCalled();
  });

  it("renders component and notice recovery actions", async () => {
    const user = userEvent.setup();
    const onNavigate = vi.fn();

    const health: HealthResponse = {
      status: "degraded",
      version: "test",
      components: {
        browser: {
          status: "degraded",
          message: "Chrome is missing.",
          actions: [
            {
              label: "Install Chrome on macOS",
              kind: "copy",
              value: "brew install --cask google-chrome",
            },
          ],
        },
      },
      notices: [
        {
          id: "config-warning",
          scope: "config",
          severity: "warning",
          title: "Review docs",
          message: "Something needs attention.",
          actions: [
            {
              label: "Open settings docs",
              kind: "doc",
              value: "/settings/operations",
            },
          ],
        },
      ],
    };

    render(
      <SystemStatusPanel
        health={health}
        onNavigate={onNavigate}
        onRefresh={vi.fn()}
      />,
    );

    expect(screen.getByText("Chrome is missing.")).toBeInTheDocument();
    expect(screen.getByText("Review docs")).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: "Open settings docs" }),
    );
    expect(onNavigate).toHaveBeenCalledWith("/settings/operations");
  });

  it("renders external recovery links with safe new-tab attributes", () => {
    const health: HealthResponse = {
      status: "degraded",
      version: "test",
      components: {
        browser: {
          status: "degraded",
          message: "Browser tooling missing.",
          actions: [
            {
              label: "Browser install guide",
              kind: "external-link",
              value: "https://playwright.dev/docs/browsers",
            },
          ],
        },
      },
    };

    render(
      <SystemStatusPanel
        health={health}
        onNavigate={vi.fn()}
        onRefresh={vi.fn()}
      />,
    );

    const link = screen.getByRole("link", { name: "Browser install guide" });
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", expect.stringContaining("noreferrer"));
  });
});
