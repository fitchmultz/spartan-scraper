import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DeviceSelector } from "./DeviceSelector";

describe("DeviceSelector", () => {
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
