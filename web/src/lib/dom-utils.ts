/**
 * DOM Utilities
 *
 * Utility functions for DOM manipulation and selector generation.
 *
 * @module dom-utils
 */

// DOMNode type definition (duplicated to avoid circular imports)
interface DOMNode {
  tag: string;
  id?: string;
  classes?: string[];
  attributes?: Record<string, string>;
  text?: string;
  children?: DOMNode[];
  path: string;
  depth: number;
}

/**
 * Generate multiple CSS selector options for a DOM node.
 * Returns selectors ordered by specificity (most specific first).
 */
export function generateSelectorOptions(node: DOMNode): string[] {
  const options: string[] = [];

  // ID-based (most specific)
  if (node.id) {
    options.push(`#${node.id}`);
  }

  // Class-based
  if (node.classes && node.classes.length > 0) {
    const classSelector = `${node.tag}.${node.classes.join(".")}`;
    options.push(classSelector);

    // Just the classes (if not too many)
    if (node.classes.length <= 3) {
      options.push(`.${node.classes.join(".")}`);
    }
  }

  // Tag with data attributes
  if (node.attributes) {
    for (const [key, value] of Object.entries(node.attributes)) {
      if (key.startsWith("data-") && value) {
        options.push(`${node.tag}[${key}="${value}"]`);
      }
    }
  }

  // Tag only
  options.push(node.tag);

  // Full path (least preferred)
  if (node.path && node.path !== node.tag) {
    options.push(node.path);
  }

  return options;
}

/**
 * Simplify a selector to make it more robust.
 * Removes nth-child if possible, prefers classes.
 */
export function simplifySelector(selector: string): string {
  // Remove nth-child if present and replace with just the tag
  let simplified = selector.replace(/:nth-child\(\d+\)/g, "");

  // Remove extra spaces
  simplified = simplified.replace(/\s+/g, " ").trim();

  return simplified;
}

/**
 * Check if a selector is valid CSS.
 */
export function isValidSelector(selector: string): boolean {
  if (!selector) return false;

  // Basic validation - try to catch obvious errors
  const invalidPatterns = [
    /^[0-9]/, // Starts with number
    /\s\s+/, // Multiple spaces
    /\[\]/, // Empty attribute
    /\(\)/, // Empty pseudo
  ];

  return !invalidPatterns.some((pattern) => pattern.test(selector));
}

/**
 * Extract text content from a DOM node, truncated.
 */
export function extractNodeText(node: DOMNode, maxLength = 100): string {
  const text = node.text ?? "";
  if (text.length <= maxLength) return text;
  return `${text.slice(0, maxLength)}...`;
}

/**
 * Get a display name for a DOM node.
 */
export function getNodeDisplayName(node: DOMNode): string {
  const parts: string[] = [node.tag];

  if (node.id) {
    parts.push(`#${node.id.slice(0, 20)}`);
  }

  if (node.classes && node.classes.length > 0) {
    const classStr = node.classes.slice(0, 2).join(".");
    parts.push(`.${classStr}`);
    if (node.classes.length > 2) {
      parts.push("...");
    }
  }

  return parts.join("");
}

/**
 * Flatten a DOM tree into an array for easier traversal.
 */
export function flattenDOMTree(
  root: DOMNode,
  expandedPaths: Set<string>,
): DOMNode[] {
  const result: DOMNode[] = [];

  function addNode(node: DOMNode) {
    result.push(node);

    if (expandedPaths.has(node.path) && node.children) {
      for (const child of node.children) {
        addNode(child);
      }
    }
  }

  addNode(root);
  return result;
}

/**
 * Find a node by its path in the DOM tree.
 */
export function findNodeByPath(root: DOMNode, path: string): DOMNode | null {
  if (root.path === path) return root;

  if (root.children) {
    for (const child of root.children) {
      const found = findNodeByPath(child, path);
      if (found) return found;
    }
  }

  return null;
}

/**
 * Search DOM tree for nodes matching a query.
 */
export function searchDOMTree(root: DOMNode, query: string): DOMNode[] {
  const results: DOMNode[] = [];
  const lowerQuery = query.toLowerCase();

  function searchNode(node: DOMNode) {
    const matches =
      (node.tag?.toLowerCase().includes(lowerQuery) ?? false) ||
      (node.id?.toLowerCase().includes(lowerQuery) ?? false) ||
      (node.classes?.some((c: string) =>
        c.toLowerCase().includes(lowerQuery),
      ) ??
        false) ||
      (node.text?.toLowerCase().includes(lowerQuery) ?? false);

    if (matches) {
      results.push(node);
    }

    if (node.children) {
      for (const child of node.children) {
        searchNode(child);
      }
    }
  }

  searchNode(root);
  return results;
}

/**
 * Get all paths that should be expanded to show search results.
 */
export function getExpansionPathsForSearch(
  root: DOMNode,
  query: string,
): Set<string> {
  const paths = new Set<string>();
  const lowerQuery = query.toLowerCase();

  function checkNode(node: DOMNode, parentPath: string | null): boolean {
    const matches =
      (node.tag?.toLowerCase().includes(lowerQuery) ?? false) ||
      (node.id?.toLowerCase().includes(lowerQuery) ?? false) ||
      (node.classes?.some((c: string) =>
        c.toLowerCase().includes(lowerQuery),
      ) ??
        false) ||
      (node.text?.toLowerCase().includes(lowerQuery) ?? false);

    let childMatches = false;
    if (node.children) {
      for (const child of node.children) {
        if (checkNode(child, node.path)) {
          childMatches = true;
        }
      }
    }

    if ((matches || childMatches) && parentPath) {
      paths.add(parentPath);
    }

    return matches || childMatches;
  }

  checkNode(root, null);
  return paths;
}

/**
 * Truncate text with ellipsis.
 */
export function truncateText(text: string, maxLength: number): string {
  if (!text) return "";
  if (text.length <= maxLength) return text;
  return `${text.slice(0, maxLength)}...`;
}
