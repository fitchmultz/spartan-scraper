import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ProxyPoolStatusPanel } from "./ProxyPoolStatusPanel";
import * as api from "../api";

vi.mock("../api", () => ({
  getProxyPoolStatus: vi.fn(),
}));

vi.mock("../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

describe("ProxyPoolStatusPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state initially", () => {
    vi.mocked(api.getProxyPoolStatus).mockReturnValue(
      new Promise(() => {}) as ReturnType<typeof api.getProxyPoolStatus>,
    );

    render(<ProxyPoolStatusPanel />);

    expect(screen.getByText(/loading proxy pool status/i)).toBeInTheDocument();
  });

  it("renders loaded proxy pool stats", async () => {
    vi.mocked(api.getProxyPoolStatus).mockResolvedValue({
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
    });

    render(<ProxyPoolStatusPanel />);

    await waitFor(() => {
      expect(screen.getByText("round_robin")).toBeInTheDocument();
    });

    expect(screen.getByText("Loaded")).toBeInTheDocument();
    expect(screen.getByText("proxy-east")).toBeInTheDocument();
    expect(screen.getByText("residential")).toBeInTheDocument();
    expect(screen.getByText("Available regions:")).toBeInTheDocument();
    expect(screen.getByText(/datacenter, residential/i)).toBeInTheDocument();
    expect(screen.getByText("66.67%")).toBeInTheDocument();
  });

  it("refreshes proxy pool status when requested", async () => {
    vi.mocked(api.getProxyPoolStatus).mockResolvedValue({
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
    });

    render(<ProxyPoolStatusPanel />);

    await waitFor(() => {
      expect(
        screen.getByText(/no proxy pool is currently loaded/i),
      ).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: /refresh/i }));

    await waitFor(() => {
      expect(api.getProxyPoolStatus).toHaveBeenCalledTimes(2);
    });
  });
});
