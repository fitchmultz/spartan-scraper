/**
 * Purpose: Verify capability action lists render disabled diagnostic results as informational guidance.
 * Responsibilities: Assert one-click optional-off responses stay quiet and avoid warning styling.
 * Scope: CapabilityActionList behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Optional disabled states should never look like runtime failures.
 */

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { postV1DiagnosticsProxyPoolCheck } from "../api";
import { CapabilityActionList } from "./CapabilityActionList";

vi.mock("../api", () => ({
  postV1DiagnosticsAiCheck: vi.fn(),
  postV1DiagnosticsBrowserCheck: vi.fn(),
  postV1DiagnosticsProxyPoolCheck: vi.fn(),
}));

vi.mock("../lib/api-config", () => ({
  getApiBaseUrl: () => "http://127.0.0.1:8741",
}));

describe("CapabilityActionList", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(postV1DiagnosticsProxyPoolCheck).mockResolvedValue({
      data: {
        status: "disabled",
        title: "Proxy pool is off",
        message:
          "No proxy-pool file is configured. Spartan does not need pooled proxy routing unless you explicitly opt into it.",
        actions: [],
      },
      request: new Request(
        "http://127.0.0.1:8741/v1/diagnostics/proxy-pool-check",
      ),
      response: new Response(null, { status: 200 }),
    } as never);
  });

  it("renders disabled diagnostic responses with informational styling", async () => {
    const user = userEvent.setup();
    const { container } = render(
      <CapabilityActionList
        actions={[
          {
            label: "Re-check proxy pool configuration",
            kind: "one-click",
            value: "/v1/diagnostics/proxy-pool-check",
          },
        ]}
        onNavigate={vi.fn()}
        onRefresh={vi.fn()}
      />,
    );

    await user.click(
      screen.getByRole("button", { name: "Re-check proxy pool configuration" }),
    );

    expect(await screen.findByText("Proxy pool is off")).toBeInTheDocument();
    expect(
      screen.getByText(/spartan does not need pooled proxy routing/i),
    ).toBeInTheDocument();

    const result = container.querySelector(".system-status__action-result");
    expect(result).toHaveClass("system-status__action-result--info");
    expect(result).not.toHaveClass("system-status__action-result--warning");
  });
});
