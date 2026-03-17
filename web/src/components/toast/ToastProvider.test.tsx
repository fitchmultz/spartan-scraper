/**
 * Purpose: Verify the global toast provider's notification and confirmation behavior.
 * Responsibilities: Cover timed dismissal, in-place loading updates, and confirm dialog resolution.
 * Scope: Toast provider integration only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Tests execute in jsdom with fake timers available for deterministic dismissal timing.
 */

import { act, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ToastProvider } from "./ToastProvider";
import { useToast } from "./useToast";

function ToastHarness() {
  const toast = useToast();
  const [confirmResult, setConfirmResult] = useState("pending");

  return (
    <div>
      <button
        type="button"
        onClick={() => {
          toast.show({
            id: "success-toast",
            tone: "success",
            title: "Saved",
            description: "Template changes were stored.",
            durationMs: 1_000,
          });
        }}
      >
        Show success toast
      </button>
      <button
        type="button"
        onClick={() => {
          toast.show({
            id: "loading-toast",
            tone: "loading",
            title: "Submitting",
            description: "Queueing the scrape job.",
          });
        }}
      >
        Show loading toast
      </button>
      <button
        type="button"
        onClick={() => {
          toast.update("loading-toast", {
            tone: "success",
            title: "Submitted",
            description: "Your scrape job is now running.",
          });
        }}
      >
        Upgrade loading toast
      </button>
      <button
        type="button"
        onClick={async () => {
          const confirmed = await toast.confirm({
            title: "Delete this template?",
            description: "This removes the saved custom template.",
            confirmLabel: "Delete template",
            cancelLabel: "Keep template",
            tone: "error",
          });
          setConfirmResult(confirmed ? "confirmed" : "canceled");
        }}
      >
        Open confirm dialog
      </button>
      <output aria-label="confirm result">{confirmResult}</output>
    </div>
  );
}

describe("ToastProvider", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.runOnlyPendingTimers();
    vi.useRealTimers();
  });

  it("auto-dismisses timed success toasts", () => {
    render(
      <ToastProvider>
        <ToastHarness />
      </ToastProvider>,
    );

    fireEvent.click(
      screen.getByRole("button", { name: /show success toast/i }),
    );

    expect(screen.getByText("Saved")).toBeInTheDocument();
    expect(
      screen.getByText("Template changes were stored."),
    ).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(1_000);
    });

    expect(screen.queryByText("Saved")).not.toBeInTheDocument();
  });

  it("updates a loading toast in place and dismisses it after the new tone duration", () => {
    render(
      <ToastProvider>
        <ToastHarness />
      </ToastProvider>,
    );

    fireEvent.click(
      screen.getByRole("button", { name: /show loading toast/i }),
    );
    expect(screen.getByText("Submitting")).toBeInTheDocument();
    expect(screen.getByText("Queueing the scrape job.")).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: /upgrade loading toast/i }),
    );

    expect(screen.getByText("Submitted")).toBeInTheDocument();
    expect(
      screen.getByText("Your scrape job is now running."),
    ).toBeInTheDocument();
    expect(screen.queryByText("Submitting")).not.toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(4_500);
    });

    expect(screen.queryByText("Submitted")).not.toBeInTheDocument();
  });

  it("resolves confirmation requests when the operator confirms or cancels", async () => {
    render(
      <ToastProvider>
        <ToastHarness />
      </ToastProvider>,
    );

    fireEvent.click(
      screen.getByRole("button", { name: /open confirm dialog/i }),
    );
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(screen.getByText("Delete this template?")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /keep template/i }));
    await act(async () => {
      await Promise.resolve();
    });
    expect(screen.getByLabelText("confirm result")).toHaveTextContent(
      "canceled",
    );

    fireEvent.click(
      screen.getByRole("button", { name: /open confirm dialog/i }),
    );
    fireEvent.click(screen.getByRole("button", { name: /delete template/i }));
    await act(async () => {
      await Promise.resolve();
    });
    expect(screen.getByLabelText("confirm result")).toHaveTextContent(
      "confirmed",
    );
  });
});
