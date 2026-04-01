/**
 * Purpose: Verify the shared shell primitives preserve navigation affordances on mobile layouts.
 * Responsibilities: Exercise the top-bar mobile menu state and ensure route changes and navigation clicks collapse the drawer predictably.
 * Scope: App shell primitive behavior only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Mobile navigation keeps toolbar affordances visible, uses an explicit menu toggle, and closes after route changes or successful navigation.
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import {
  AppTopBar,
  RouteHeader,
  type AppShellNavItem,
} from "./ShellPrimitives";

const navItems: ReadonlyArray<AppShellNavItem<"jobs" | "templates">> = [
  {
    kind: "jobs",
    label: "Jobs",
    path: "/jobs",
    description: "Go to jobs",
  },
  {
    kind: "templates",
    label: "Templates",
    path: "/templates",
    description: "Go to templates",
  },
];

describe("RouteHeader", () => {
  it("renders workspace signals inside the sticky route header when requested", () => {
    render(
      <RouteHeader
        title="Automation"
        stickyOnShortViewport
        signals={<div>Current section: Watches</div>}
      />,
    );

    expect(
      screen.getByRole("region", { name: "Automation overview" }),
    ).toHaveClass("route-header--workspace");
    expect(screen.getByText("Current section: Watches")).toBeInTheDocument();
  });
});

describe("AppTopBar", () => {
  it("opens and closes the mobile nav menu around navigation", () => {
    const onNavigate = vi.fn();
    window.innerWidth = 390;

    render(
      <AppTopBar
        activeRoute="jobs"
        navItems={navItems}
        onNavigate={onNavigate}
        globalAction={<button type="button">Create Job</button>}
        utilities={<button type="button">Shortcuts</button>}
      />,
    );

    const menuButton = screen.getByRole("button", {
      name: /open navigation menu/i,
    });
    const nav = screen.getByRole("navigation", { name: /primary/i });

    expect(menuButton).toHaveAttribute("aria-expanded", "false");
    expect(nav).not.toHaveClass("is-open");

    fireEvent.click(menuButton);

    expect(menuButton).toHaveAttribute("aria-expanded", "true");
    expect(nav).toHaveClass("is-open");

    fireEvent.click(screen.getByRole("button", { name: "Templates" }));

    expect(onNavigate).toHaveBeenCalledWith("/templates");
    expect(nav).not.toHaveClass("is-open");
  });

  it("closes the mobile nav when the active route changes", () => {
    const onNavigate = vi.fn();
    window.innerWidth = 390;

    const { rerender } = render(
      <AppTopBar
        activeRoute="jobs"
        navItems={navItems}
        onNavigate={onNavigate}
        utilities={<button type="button">Shortcuts</button>}
      />,
    );

    fireEvent.click(
      screen.getByRole("button", { name: /open navigation menu/i }),
    );
    expect(screen.getByRole("navigation", { name: /primary/i })).toHaveClass(
      "is-open",
    );

    rerender(
      <AppTopBar
        activeRoute="templates"
        navItems={navItems}
        onNavigate={onNavigate}
        utilities={<button type="button">Shortcuts</button>}
      />,
    );

    expect(
      screen.getByRole("navigation", { name: /primary/i }),
    ).not.toHaveClass("is-open");
  });
});
