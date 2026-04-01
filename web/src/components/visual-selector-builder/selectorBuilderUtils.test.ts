/**
 * Purpose: Verify selector builder utils behavior with automated regression coverage.
 * Responsibilities: Define focused test cases, fixtures, and assertions for the module under test.
 * Scope: Automated test coverage only; production logic stays in the adjacent source modules.
 * Usage: Run through the repo test entrypoints or the feature-local test command.
 * Invariants/Assumptions: Tests should describe the current contract clearly and remain deterministic under local CI settings.
 */

import { describe, expect, it } from "vitest";

import type { DomNode } from "../../api";

import {
  buildExpandedPaths,
  createSelectorRule,
  generateSelectorOptions,
  nodeMatchesSearch,
} from "./selectorBuilderUtils";

const tree: DomNode = {
  tag: "body",
  path: "body",
  depth: 0,
  children: [
    {
      tag: "main",
      path: "body > main",
      depth: 1,
      classes: ["content"],
      children: [
        {
          tag: "button",
          id: "buy-now",
          path: "body > main > button",
          depth: 2,
          classes: ["cta", "primary"],
          attributes: { "data-testid": "buy-button" },
          text: "Buy now",
        },
      ],
    },
  ],
};

describe("selectorBuilderUtils", () => {
  it("builds expanded paths through the configured depth", () => {
    expect([...buildExpandedPaths(tree, 1)]).toEqual(["body", "body > main"]);
  });

  it("matches nested DOM tree search terms", () => {
    expect(nodeMatchesSearch(tree, "buy now")).toBe(true);
    expect(nodeMatchesSearch(tree, "cta")).toBe(true);
    expect(nodeMatchesSearch(tree, "missing")).toBe(false);
  });

  it("generates ordered selector options", () => {
    const node = tree.children?.[0]?.children?.[0];
    if (!node) {
      throw new Error("expected selector fixture node");
    }
    expect(generateSelectorOptions(node)).toEqual([
      "#buy-now",
      "button.cta.primary",
      ".cta.primary",
      'button[data-testid="buy-button"]',
      "button",
      "body > main > button",
    ]);
  });

  it("creates selector rules with stable defaults", () => {
    expect(createSelectorRule(".price", 1)).toEqual({
      name: "field_2",
      selector: ".price",
      attr: "text",
      all: false,
      join: "",
      trim: true,
      required: false,
    });
  });
});
