/**
 * Purpose: Render the selector builder utils UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
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
