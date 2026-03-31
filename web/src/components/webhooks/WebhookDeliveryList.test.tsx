/**
 * Purpose: Verify the webhook-delivery list keeps clipboard and detail actions routed through the shared UI contracts.
 * Responsibilities: Assert copy actions use the Clipboard API and report clipboard failures through the shared runtime reporter.
 * Scope: `WebhookDeliveryList` behavior only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Clipboard access is mocked and row data is already sanitized for browser display.
 */

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";

import type { WebhookDeliveryRecord } from "../../api";
import { reportRuntimeError } from "../../lib/runtime-errors";
import { WebhookDeliveryList } from "./WebhookDeliveryList";

vi.mock("../../lib/runtime-errors", () => ({
  reportRuntimeError: vi.fn(),
}));

describe("WebhookDeliveryList", () => {
  const delivery: WebhookDeliveryRecord = {
    id: "delivery-12345678",
    eventType: "job.completed",
    jobId: "job-12345678",
    url: "https://example.com/webhook",
    status: "delivered",
    attempts: 1,
    createdAt: "2026-03-31T12:00:00Z",
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("copies the full delivery id to the clipboard", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", {
      clipboard: { writeText },
    });

    render(
      <WebhookDeliveryList deliveries={[delivery]} onViewDetail={vi.fn()} />,
    );

    await user.click(screen.getByRole("button", { name: "Copy" }));

    expect(writeText).toHaveBeenCalledWith("delivery-12345678");
    expect(reportRuntimeError).not.toHaveBeenCalled();
  });

  it("reports clipboard failures through runtime error reporting", async () => {
    const user = userEvent.setup();
    const clipboardError = new Error("denied");
    const writeText = vi.fn().mockRejectedValue(clipboardError);
    vi.stubGlobal("navigator", {
      clipboard: { writeText },
    });

    render(
      <WebhookDeliveryList deliveries={[delivery]} onViewDetail={vi.fn()} />,
    );

    await user.click(screen.getByRole("button", { name: "Copy" }));

    expect(reportRuntimeError).toHaveBeenCalledWith(
      "Failed to copy to clipboard",
      clipboardError,
    );
  });
});
