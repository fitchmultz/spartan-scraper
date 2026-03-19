/**
 * Purpose: Verify proxy-pool settings render guided capability explanations and actionable recovery flows.
 * Responsibilities: Assert degraded/disabled guidance, inline diagnostic execution, and runtime status details remain visible when available.
 * Scope: ProxyPoolStatusPanel behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Proxy pooling is optional, but degraded configuration should expose recovery actions without hiding operational detail.
 */

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  getProxyPoolStatus,
  postV1DiagnosticsProxyPoolCheck,
  type HealthResponse,
} from "../api";
import { ProxyPoolStatusPanel } from "./ProxyPoolStatusPanel";

vi.mock("../api", async () => {
  const actual = await vi.importActual<Record<string, unknown>>("../api");
  return {
    ...actual,
    getProxyPoolStatus: vi.fn(),
    postV1DiagnosticsProxyPoolCheck: vi.fn(),
    postV1DiagnosticsAiCheck: vi.fn(),
    postV1DiagnosticsBrowserCheck: vi.fn(),
  };
});

vi.mock("../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

describe("ProxyPoolStatusPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state initially", () => {
    vi.mocked(getProxyPoolStatus).mockReturnValue(
      new Promise(() => {}) as ReturnType<typeof getProxyPoolStatus>,
    );

    render(
      <ProxyPoolStatusPanel
        health={null}
        onNavigate={vi.fn()}
        onRefreshHealth={vi.fn()}
      />,
    );

    expect(screen.getByText(/loading proxy pool status/i)).toBeInTheDocument();
  });

  it("renders disabled guidance when proxy pooling is unconfigured", async () => {
    vi.mocked(getProxyPoolStatus).mockResolvedValue({
      data: {
        strategy: "none",
        total_proxies: 0,
        healthy_proxies: 0,
        regions: [],
        tags: [],
        proxies: [],
      },
      request: new Request("http://localhost:8741/v1/proxy-pool/status"),
      response: new Response(),
    } as never);

    const health: HealthResponse = {
      status: "ok",
      version: "test",
      components: {
        proxy_pool: {
          status: "disabled",
          message:
            "Proxy pooling is currently off. Spartan does not need a proxy pool for normal operation.",
          actions: [
            {
              label: "Set PROXY_POOL_FILE when you need pooled routing",
              kind: "env",
              value: "PROXY_POOL_FILE=/absolute/path/to/proxy-pool.json",
            },
          ],
        },
      },
      notices: [],
    };

    render(
      <ProxyPoolStatusPanel
        health={health}
        onNavigate={vi.fn()}
        onRefreshHealth={vi.fn()}
      />,
    );

    expect(
      await screen.findByText("Proxy pooling is currently off"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Proxy pooling is currently off. Spartan does not need a proxy pool for normal operation.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText("PROXY_POOL_FILE=/absolute/path/to/proxy-pool.json"),
    ).toBeInTheDocument();
  });

  it("renders degraded health guidance and runs inline diagnostics", async () => {
    const user = userEvent.setup();
    const onRefreshHealth = vi.fn().mockResolvedValue(undefined);

    vi.mocked(getProxyPoolStatus).mockResolvedValue({
      data: {
        strategy: "round_robin",
        total_proxies: 0,
        healthy_proxies: 0,
        regions: [],
        tags: [],
        proxies: [],
      },
      request: new Request("http://localhost:8741/v1/proxy-pool/status"),
      response: new Response(),
    } as never);

    vi.mocked(postV1DiagnosticsProxyPoolCheck).mockResolvedValue({
      data: {
        status: "degraded",
        title: "Proxy pool file is missing",
        message: "Configured proxy pool file is still missing.",
        actions: [],
      },
      request: new Request(
        "http://localhost:8741/v1/diagnostics/proxy-pool-check",
      ),
      response: new Response(),
    } as never);

    const health: HealthResponse = {
      status: "degraded",
      version: "test",
      components: {
        proxy_pool: {
          status: "degraded",
          message: "Configured proxy pool file is missing or unreadable.",
          details: { path: "/tmp/proxies.txt" },
          actions: [
            {
              label: "Re-check proxy pool configuration",
              kind: "one-click",
              value: "/v1/diagnostics/proxy-pool-check",
            },
            {
              label: "Disable proxy pool intentionally",
              kind: "env",
              value: "PROXY_POOL_FILE=",
            },
          ],
        },
      },
      notices: [],
    };

    render(
      <ProxyPoolStatusPanel
        health={health}
        onNavigate={vi.fn()}
        onRefreshHealth={onRefreshHealth}
      />,
    );

    expect(
      await screen.findByText("Proxy pool needs attention"),
    ).toBeInTheDocument();
    expect(screen.getByText("Configured file")).toBeInTheDocument();
    expect(screen.getByText("/tmp/proxies.txt")).toBeInTheDocument();
    expect(screen.getByText("PROXY_POOL_FILE=")).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", {
        name: "Re-check proxy pool configuration",
      }),
    );

    await waitFor(() => {
      expect(postV1DiagnosticsProxyPoolCheck).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
      });
    });

    expect(
      await screen.findByText("Configured proxy pool file is still missing."),
    ).toBeInTheDocument();
    expect(onRefreshHealth).toHaveBeenCalled();
  });

  it("renders loaded proxy pool stats when proxies exist", async () => {
    vi.mocked(getProxyPoolStatus).mockResolvedValue({
      data: {
        strategy: "round_robin",
        total_proxies: 2,
        healthy_proxies: 1,
        regions: ["us-east", "us-west"],
        tags: ["datacenter", "residential"],
        proxies: [
          {
            id: "proxy-east",
            region: "us-east",
            tags: ["residential"],
            is_healthy: true,
            request_count: 3,
            success_count: 2,
            failure_count: 1,
            success_rate: 66.67,
            avg_latency_ms: 120,
            consecutive_fails: 0,
          },
        ],
      },
      request: new Request("http://localhost:8741/v1/proxy-pool/status"),
      response: new Response(),
    } as never);

    render(
      <ProxyPoolStatusPanel
        health={null}
        onNavigate={vi.fn()}
        onRefreshHealth={vi.fn()}
      />,
    );

    expect(await screen.findByText("round_robin")).toBeInTheDocument();
    expect(screen.getByText("proxy-east")).toBeInTheDocument();
    expect(screen.getByText("Available regions:")).toBeInTheDocument();
    expect(screen.getByText(/datacenter, residential/i)).toBeInTheDocument();
    expect(screen.getByText("66.67%")).toBeInTheDocument();
  });
});
