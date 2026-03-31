/**
 * Purpose: Verify the shared device-emulation picker stays operable and accessible.
 * Responsibilities: Assert category-chip state, icon rendering, and combobox labeling for assistive technologies.
 * Scope: Component coverage for `DeviceSelector` only.
 * Usage: Run with Vitest as part of the Web test suite.
 * Invariants/Assumptions: The preset combobox should expose the visible Device Emulation label, and category chips keep a single active state.
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DeviceSelector } from "./DeviceSelector";

describe("DeviceSelector", () => {
  it("exposes the preset combobox with the visible device-emulation label", () => {
    render(<DeviceSelector device={null} onChange={vi.fn()} />);

    expect(
      screen.getByRole("combobox", { name: /device emulation/i }),
    ).toBeInTheDocument();
  });

  it("keeps a single active category chip", () => {
    render(<DeviceSelector device={null} onChange={vi.fn()} />);

    const allButton = screen.getByRole("button", { name: "All" });
    const mobileButton = screen.getByRole("button", { name: "Mobile" });
    const tabletButton = screen.getByRole("button", { name: "Tablet" });

    expect(allButton).toHaveClass("active");
    expect(mobileButton).not.toHaveClass("active");
    expect(tabletButton).not.toHaveClass("active");

    fireEvent.click(mobileButton);

    expect(allButton).not.toHaveClass("active");
    expect(mobileButton).toHaveClass("active");
    expect(tabletButton).not.toHaveClass("active");
  });

  it("renders intentional svg iconography for category chips", () => {
    const { container } = render(
      <DeviceSelector device={null} onChange={vi.fn()} />,
    );

    expect(
      container.querySelectorAll(".device-category-filters svg"),
    ).toHaveLength(4);
  });
});
