/**
 * Tests for WelcomeModal immediate dismissal behavior.
 *
 * @module WelcomeModal.test
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { WelcomeModal } from "./WelcomeModal";

describe("WelcomeModal", () => {
  it("skips immediately when Escape is pressed", () => {
    const onSkip = vi.fn();

    render(<WelcomeModal isOpen onStartTour={vi.fn()} onSkip={onSkip} />);

    fireEvent.keyDown(window, { key: "Escape" });

    expect(onSkip).toHaveBeenCalledTimes(1);
  });

  it("skips immediately when the backdrop is clicked", () => {
    const onSkip = vi.fn();

    render(<WelcomeModal isOpen onStartTour={vi.fn()} onSkip={onSkip} />);

    fireEvent.click(screen.getByRole("dialog"));

    expect(onSkip).toHaveBeenCalledTimes(1);
  });
});
