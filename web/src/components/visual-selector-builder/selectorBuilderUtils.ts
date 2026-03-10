/**
 * selectorBuilderUtils
 *
 * Purpose:
 * - Hold pure DOM-tree and selector-builder helpers used by the visual selector UI.
 *
 * Responsibilities:
 * - Compute expanded DOM paths for initial tree display.
 * - Generate selector suggestions for a selected DOM node.
 * - Evaluate DOM-tree search matches recursively.
 * - Create new selector rules with consistent defaults.
 *
 * Scope:
 * - Pure helper logic only; no React state or network calls.
 *
 * Usage:
 * - Used by VisualSelectorBuilder and its focused tests.
 *
 * Invariants/Assumptions:
 * - Missing DOM fields fall back to safe empty values.
 * - Selector suggestions are ordered from more-specific to less-specific.
 * - New selector rules always start with the same default extraction settings.
 */

import type { DomNode, SelectorRule } from "../../api";

export function buildExpandedPaths(
  root: DomNode | null,
  maxDepth = 2,
): Set<string> {
  const paths = new Set<string>();
  if (!root) {
    return paths;
  }

  const visit = (node: DomNode, depth: number) => {
    if (node.path && depth <= maxDepth) {
      paths.add(node.path);
    }
    node.children?.forEach((child) => {
      visit(child, depth + 1);
    });
  };

  visit(root, 0);
  return paths;
}

export function nodeMatchesSearch(node: DomNode, query: string): boolean {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) {
    return true;
  }

  const matches =
    (node.tag?.toLowerCase().includes(normalizedQuery) ?? false) ||
    (node.id?.toLowerCase().includes(normalizedQuery) ?? false) ||
    (node.classes?.some((className) =>
      className.toLowerCase().includes(normalizedQuery),
    ) ??
      false) ||
    (node.text?.toLowerCase().includes(normalizedQuery) ?? false);

  if (matches) {
    return true;
  }

  return (
    node.children?.some((child) => nodeMatchesSearch(child, normalizedQuery)) ??
    false
  );
}

export function generateSelectorOptions(node: DomNode): string[] {
  const options: string[] = [];

  if (node.id) {
    options.push(`#${node.id}`);
  }

  if (node.tag && node.classes && node.classes.length > 0) {
    options.push(`${node.tag}.${node.classes.join(".")}`);
    if (node.classes.length <= 3) {
      options.push(`.${node.classes.join(".")}`);
    }
  }

  if (node.tag && node.attributes) {
    for (const [key, value] of Object.entries(node.attributes)) {
      if (key.startsWith("data-") && value) {
        options.push(`${node.tag}[${key}="${value}"]`);
      }
    }
  }

  if (node.tag) {
    options.push(node.tag);
  }
  if (node.path) {
    options.push(node.path);
  }

  return options;
}

export function createSelectorRule(
  selector: string,
  index: number,
): SelectorRule {
  return {
    name: `field_${index + 1}`,
    selector,
    attr: "text",
    all: false,
    join: "",
    trim: true,
    required: false,
  };
}
