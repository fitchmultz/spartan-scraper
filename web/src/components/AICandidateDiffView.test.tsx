/**
 * Purpose: Verify shared AI candidate diff rendering behavior.
 * Responsibilities: Assert changed-field highlighting, unchanged-field omission, latest-only summaries, and raw-JSON fallback behavior.
 * Scope: `AICandidateDiffView` only.
 * Usage: Run with `pnpm --dir web test`.
 * Invariants/Assumptions: Comparison mode hides unchanged fields, latest-only mode shows supported highlights, and unsupported runtime changes default to raw JSON.
 */

import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AICandidateDiffView } from "./AICandidateDiffView";

describe("AICandidateDiffView", () => {
  it("highlights changed render-profile fields and omits unchanged ones", () => {
    render(
      <AICandidateDiffView
        artifactKind="render-profile"
        baselineArtifact={{
          name: "example-app",
          hostPatterns: ["example.com"],
          preferHeadless: false,
          wait: { mode: "selector", selector: "main" },
        }}
        selectedArtifact={{
          name: "example-app",
          hostPatterns: ["example.com"],
          preferHeadless: true,
          wait: { mode: "selector", selector: "#app-root" },
        }}
      />,
    );

    expect(screen.getByText("Prefer headless")).toBeInTheDocument();
    expect(screen.getByText("Wait selector")).toBeInTheDocument();
    expect(screen.queryByText("Profile name")).not.toBeInTheDocument();
    expect(screen.queryByText("Host patterns")).not.toBeInTheDocument();

    const selectorChange = screen.getByRole("region", {
      name: /wait selector/i,
    });
    expect(within(selectorChange).getByText("main")).toBeInTheDocument();
    expect(within(selectorChange).getByText("#app-root")).toBeInTheDocument();
  });

  it("shows latest-only pipeline highlights before a retry exists", () => {
    render(
      <AICandidateDiffView
        artifactKind="pipeline-js"
        selectedArtifact={{
          name: "example-app",
          hostPatterns: ["example.com"],
          selectors: ["#app-root"],
          postNav: "window.scrollTo(0, 0);",
        }}
      />,
    );

    expect(screen.getByText("Script name")).toBeInTheDocument();
    expect(screen.getByText("Wait selectors")).toBeInTheDocument();
    expect(screen.getByText("Post-navigation logic")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /show raw json/i }),
    ).toBeInTheDocument();
  });

  it("falls back to raw JSON when unsupported fields changed", () => {
    render(
      <AICandidateDiffView
        artifactKind="pipeline-js"
        baselineArtifact={
          {
            name: "example-app",
            hostPatterns: ["example.com"],
            selectors: ["main"],
            experimental: { enabled: false },
          } as unknown as never
        }
        selectedArtifact={
          {
            name: "example-app",
            hostPatterns: ["example.com"],
            selectors: ["main"],
            experimental: { enabled: true },
          } as unknown as never
        }
      />,
    );

    expect(
      screen.getByText(
        /showing raw json because the pipeline js script changed in unsupported fields: experimental.enabled/i,
      ),
    ).toBeInTheDocument();
    expect(screen.getAllByText(/"experimental": \{/i)).toHaveLength(2);

    fireEvent.click(screen.getByRole("button", { name: /hide raw json/i }));
    expect(
      screen.getByRole("button", { name: /show raw json/i }),
    ).toBeInTheDocument();
  });
});
