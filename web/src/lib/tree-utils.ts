/**
 * Tree View Utilities
 *
 * Provides functions for building hierarchical tree structures from flat crawl results.
 * Groups URLs by domain and path segments to create a visual site hierarchy.
 *
 * @module tree-utils
 */
import type { CrawlResultItem } from "../types";

/**
 * Represents a node in the URL hierarchy tree.
 */
export interface TreeNode {
  /** Unique identifier for the node */
  id: string;
  /** Full URL for leaf nodes, path segment for intermediate nodes */
  url: string;
  /** Display title (page title for leaves, path segment for intermediate) */
  title: string;
  /** HTTP status code (only for leaf nodes) */
  status: number;
  /** Child nodes */
  children: TreeNode[];
  /** Depth in the tree (0 = root/domain) */
  depth: number;
  /** Whether the node is expanded in the UI */
  isExpanded?: boolean;
  /** Number of crawl results in this subtree */
  resultCount: number;
  /** Original result item (only for leaf nodes) */
  result?: CrawlResultItem;
}

/**
 * Extract the domain from a URL.
 *
 * @param url - The URL to parse
 * @returns The domain (hostname) or empty string if invalid
 */
export function getDomain(url: string): string {
  try {
    const urlObj = new URL(url);
    return urlObj.hostname;
  } catch {
    return "";
  }
}

/**
 * Extract path segments from a URL.
 *
 * @param url - The URL to parse
 * @returns Array of path segments (excluding empty segments)
 */
export function getUrlPathSegments(url: string): string[] {
  try {
    const urlObj = new URL(url);
    return urlObj.pathname.split("/").filter(Boolean);
  } catch {
    return [];
  }
}

/**
 * Group crawl results by their domain.
 *
 * @param results - Array of crawl result items
 * @returns Map of domain to results for that domain
 */
export function groupByDomain(
  results: CrawlResultItem[],
): Map<string, CrawlResultItem[]> {
  const groups = new Map<string, CrawlResultItem[]>();

  for (const result of results) {
    const domain = getDomain(result.url);
    if (!domain) continue;

    const existing = groups.get(domain);
    if (existing) {
      existing.push(result);
    } else {
      groups.set(domain, [result]);
    }
  }

  return groups;
}

/**
 * Sort tree nodes alphabetically by title, with directories before files.
 *
 * @param nodes - Array of tree nodes to sort
 * @returns Sorted array of tree nodes
 */
export function sortTreeNodes(nodes: TreeNode[]): TreeNode[] {
  return [...nodes].sort((a, b) => {
    // Directories (nodes with children) come before files
    const aIsDir = a.children.length > 0;
    const bIsDir = b.children.length > 0;

    if (aIsDir && !bIsDir) return -1;
    if (!aIsDir && bIsDir) return 1;

    // Then sort alphabetically by title
    return a.title.localeCompare(b.title);
  });
}

/**
 * Build a hierarchical tree from flat crawl results.
 *
 * Groups results by domain, then builds a tree structure based on URL path segments.
 * Each domain becomes a root node, with path segments as intermediate nodes.
 *
 * @param results - Array of crawl result items
 * @returns Array of root tree nodes (one per domain)
 */
export function buildUrlTree(results: CrawlResultItem[]): TreeNode[] {
  const domainGroups = groupByDomain(results);
  const roots: TreeNode[] = [];

  for (const [domain, domainResults] of domainGroups) {
    // Create domain root node
    const domainNode: TreeNode = {
      id: `domain:${domain}`,
      url: `https://${domain}`,
      title: domain,
      status: 0,
      children: [],
      depth: 0,
      isExpanded: true,
      resultCount: domainResults.length,
    };

    // Build path-based tree for this domain
    const pathMap = new Map<string, TreeNode>();

    for (const result of domainResults) {
      const segments = getUrlPathSegments(result.url);

      if (segments.length === 0) {
        // Root path of domain - add as direct child
        const rootNode: TreeNode = {
          id: `page:${result.url}`,
          url: result.url,
          title: result.title || "/",
          status: result.status,
          children: [],
          depth: 1,
          resultCount: 1,
          result,
        };
        domainNode.children.push(rootNode);
      } else {
        // Build intermediate nodes for path segments
        let currentPath = "";
        let parentNode = domainNode;

        for (let i = 0; i < segments.length; i++) {
          const segment = segments[i];
          const isLastSegment = i === segments.length - 1;
          currentPath = currentPath ? `${currentPath}/${segment}` : segment;
          const fullPath = `https://${domain}/${currentPath}`;

          if (isLastSegment) {
            // Leaf node - actual page
            const leafNode: TreeNode = {
              id: `page:${result.url}`,
              url: result.url,
              title: result.title || segment,
              status: result.status,
              children: [],
              depth: i + 1,
              resultCount: 1,
              result,
            };
            parentNode.children.push(leafNode);
          } else {
            // Intermediate directory node
            const pathKey = `dir:${domain}:${currentPath}`;
            let dirNode = pathMap.get(pathKey);

            if (!dirNode) {
              dirNode = {
                id: pathKey,
                url: fullPath,
                title: segment,
                status: 0,
                children: [],
                depth: i + 1,
                isExpanded: false,
                resultCount: 0,
              };
              pathMap.set(pathKey, dirNode);
              parentNode.children.push(dirNode);
            }

            parentNode = dirNode;
          }
        }
      }
    }

    // Update result counts for directory nodes
    const updateResultCount = (node: TreeNode): number => {
      if (node.children.length === 0) {
        return node.resultCount;
      }
      let count = 0;
      for (const child of node.children) {
        count += updateResultCount(child);
      }
      node.resultCount = count;
      return count;
    };

    updateResultCount(domainNode);

    // Sort all children recursively
    const sortRecursively = (node: TreeNode) => {
      node.children = sortTreeNodes(node.children);
      for (const child of node.children) {
        sortRecursively(child);
      }
    };
    sortRecursively(domainNode);

    roots.push(domainNode);
  }

  // Sort root domains alphabetically
  return roots.sort((a, b) => a.title.localeCompare(b.title));
}

/**
 * Flatten a tree structure for virtualization.
 *
 * Returns only visible nodes based on expanded state.
 *
 * @param nodes - Root nodes to flatten
 * @param expandedIds - Set of expanded node IDs
 * @returns Flat array of visible tree nodes
 */
export function flattenTree(
  nodes: TreeNode[],
  expandedIds: Set<string>,
): TreeNode[] {
  const result: TreeNode[] = [];

  const traverse = (node: TreeNode) => {
    result.push(node);
    if (expandedIds.has(node.id) && node.children.length > 0) {
      for (const child of node.children) {
        traverse(child);
      }
    }
  };

  for (const node of nodes) {
    traverse(node);
  }

  return result;
}

/**
 * Get all node IDs in a tree.
 *
 * @param nodes - Root nodes to traverse
 * @returns Set of all node IDs
 */
export function getAllNodeIds(nodes: TreeNode[]): Set<string> {
  const ids = new Set<string>();

  const traverse = (node: TreeNode) => {
    ids.add(node.id);
    for (const child of node.children) {
      traverse(child);
    }
  };

  for (const node of nodes) {
    traverse(node);
  }

  return ids;
}

/**
 * Find a node by ID in the tree.
 *
 * @param nodes - Root nodes to search
 * @param id - Node ID to find
 * @returns The found node or undefined
 */
export function findNodeById(
  nodes: TreeNode[],
  id: string,
): TreeNode | undefined {
  for (const node of nodes) {
    if (node.id === id) {
      return node;
    }
    if (node.children.length > 0) {
      const found = findNodeById(node.children, id);
      if (found) return found;
    }
  }
  return undefined;
}

/**
 * Get the path from root to a specific node.
 *
 * @param nodes - Root nodes to search
 * @param targetId - Target node ID
 * @returns Array of nodes from root to target, or empty array if not found
 */
export function getNodePath(nodes: TreeNode[], targetId: string): TreeNode[] {
  const path: TreeNode[] = [];

  const findPath = (currentNodes: TreeNode[]): boolean => {
    for (const node of currentNodes) {
      if (node.id === targetId) {
        path.push(node);
        return true;
      }
      if (node.children.length > 0) {
        if (findPath(node.children)) {
          path.unshift(node);
          return true;
        }
      }
    }
    return false;
  };

  findPath(nodes);
  return path;
}

/**
 * Filter tree nodes by a search query.
 *
 * Returns nodes that match the query in their URL or title,
 * including all ancestors to maintain tree structure.
 *
 * @param nodes - Root nodes to filter
 * @param query - Search query string
 * @returns Filtered tree nodes
 */
export function filterTreeNodes(nodes: TreeNode[], query: string): TreeNode[] {
  if (!query.trim()) {
    return nodes;
  }

  const lowerQuery = query.toLowerCase();

  const matches = (node: TreeNode): boolean => {
    return (
      node.url.toLowerCase().includes(lowerQuery) ||
      node.title.toLowerCase().includes(lowerQuery)
    );
  };

  const filterRecursively = (node: TreeNode): TreeNode | null => {
    const filteredChildren = node.children
      .map(filterRecursively)
      .filter((child): child is TreeNode => child !== null);

    if (matches(node) || filteredChildren.length > 0) {
      return {
        ...node,
        children: filteredChildren,
        isExpanded: true, // Auto-expand filtered results
      };
    }

    return null;
  };

  return nodes
    .map(filterRecursively)
    .filter((node): node is TreeNode => node !== null);
}
