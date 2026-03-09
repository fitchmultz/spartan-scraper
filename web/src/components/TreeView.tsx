/**
 * Tree View Component
 *
 * Displays a hierarchical tree structure of crawl results with collapsible nodes,
 * status indicators, and keyboard navigation. Supports virtualization for large
 * trees and provides search/filter capabilities.
 *
 * @module TreeView
 */
import { useCallback, useMemo, useRef, useState } from "react";
import { getSimpleHttpStatusClass } from "../lib/http-status";
import type { TreeNode } from "../lib/tree-utils";
import { flattenTree, filterTreeNodes } from "../lib/tree-utils";

interface TreeViewProps {
  /** Root nodes of the tree */
  nodes: TreeNode[];
  /** Currently selected node ID */
  selectedId: string | null;
  /** Callback when a node is selected */
  onSelect: (node: TreeNode) => void;
  /** Callback when a node's expanded state is toggled */
  onToggleExpand: (nodeId: string) => void;
  /** Set of expanded node IDs */
  expandedIds: Set<string>;
  /** Optional search query for filtering */
  searchQuery?: string;
  /** Optional status filter: 'all' | 'success' | 'error' */
  statusFilter?: "all" | "success" | "error";
}

const ITEM_HEIGHT = 36;
const VIRTUALIZATION_THRESHOLD = 50;
const CONTAINER_HEIGHT = 400;

/**
 * Chevron icon for expand/collapse indicator.
 */
function ChevronIcon({ expanded }: { expanded: boolean }) {
  return (
    <svg
      width="12"
      height="12"
      viewBox="0 0 12 12"
      fill="none"
      aria-hidden="true"
      style={{
        transform: expanded ? "rotate(90deg)" : "rotate(0deg)",
        transition: "transform 0.15s ease",
      }}
    >
      <path
        d="M4.5 2.5L8 6L4.5 9.5"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

/**
 * File icon for leaf nodes.
 */
function FileIcon() {
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 14 14"
      fill="none"
      aria-hidden="true"
    >
      <path
        d="M7.5 2H4C3.44772 2 3 2.44772 3 3V11C3 11.5523 3.44772 12 4 12H10C10.5523 12 11 11.5523 11 11V5.5L7.5 2Z"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M7.5 2V5.5H11"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

/**
 * Folder icon for directory nodes.
 */
function FolderIcon({ open }: { open: boolean }) {
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 14 14"
      fill="none"
      aria-hidden="true"
    >
      <path
        d={
          open
            ? "M1.5 3.5C1.5 2.94772 1.94772 2.5 2.5 2.5H5.5L7 4H11.5C12.0523 4 12.5 4.44772 12.5 5V10.5C12.5 11.0523 12.0523 11.5 11.5 11.5H2.5C1.94772 11.5 1.5 11.0523 1.5 10.5V3.5Z"
            : "M1.5 3C1.5 2.44772 1.94772 2 2.5 2H5.79289C6.05811 2 6.31247 2.10536 6.5 2.29289L7.20711 3H11.5C12.0523 3 12.5 3.44772 12.5 4V10.5C12.5 11.0523 12.0523 11.5 11.5 11.5H2.5C1.94772 11.5 1.5 11.0523 1.5 10.5V3Z"
        }
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

/**
 * Single tree item row component.
 */
interface TreeItemProps {
  node: TreeNode;
  isSelected: boolean;
  isExpanded: boolean;
  onSelect: () => void;
  onToggle: () => void;
  style: React.CSSProperties;
}

function TreeItem({
  node,
  isSelected,
  isExpanded,
  onSelect,
  onToggle,
  style,
}: TreeItemProps) {
  const hasChildren = node.children.length > 0;
  const isLeaf = !hasChildren;

  const handleClick = useCallback(() => {
    onSelect();
  }, [onSelect]);

  const handleToggleClick = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onToggle();
    },
    [onToggle],
  );

  return (
    <div
      className={`tree-item ${isSelected ? "selected" : ""}`}
      style={{
        ...style,
        paddingLeft: `${12 + node.depth * 20}px`,
      }}
      onClick={handleClick}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          handleClick();
        }
      }}
      role="treeitem"
      aria-selected={isSelected}
      aria-expanded={hasChildren ? isExpanded : undefined}
      tabIndex={0}
    >
      <button
        type="button"
        className="tree-item-toggle"
        onClick={handleToggleClick}
        disabled={!hasChildren}
        aria-label={isExpanded ? "Collapse" : "Expand"}
      >
        {hasChildren ? <ChevronIcon expanded={isExpanded} /> : null}
      </button>
      <span className="tree-item-icon">
        {isLeaf ? <FileIcon /> : <FolderIcon open={isExpanded} />}
      </span>
      <span className="tree-item-title" title={node.url}>
        {node.title}
      </span>
      {node.status > 0 && (
        <span
          className={`tree-item-status badge ${getSimpleHttpStatusClass(node.status, { emptyWhenZero: true })}`}
        >
          {node.status}
        </span>
      )}
      {hasChildren && (
        <span className="tree-item-count">({node.resultCount})</span>
      )}
    </div>
  );
}

/**
 * Virtualized tree list component for large trees.
 */
interface VirtualTreeListProps {
  nodes: TreeNode[];
  selectedId: string | null;
  expandedIds: Set<string>;
  onSelect: (node: TreeNode) => void;
  onToggleExpand: (nodeId: string) => void;
}

function VirtualTreeList({
  nodes,
  selectedId,
  expandedIds,
  onSelect,
  onToggleExpand,
}: VirtualTreeListProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [scrollTop, setScrollTop] = useState(0);

  const flatNodes = useMemo(
    () => flattenTree(nodes, expandedIds),
    [nodes, expandedIds],
  );

  const totalHeight = flatNodes.length * ITEM_HEIGHT;
  const startIndex = Math.floor(scrollTop / ITEM_HEIGHT);
  const endIndex = Math.min(
    flatNodes.length,
    Math.ceil((scrollTop + CONTAINER_HEIGHT) / ITEM_HEIGHT),
  );
  const visibleNodes = flatNodes.slice(startIndex, endIndex);
  const offsetY = startIndex * ITEM_HEIGHT;

  const handleScroll = useCallback((e: React.UIEvent<HTMLDivElement>) => {
    setScrollTop(e.currentTarget.scrollTop);
  }, []);

  return (
    <div
      ref={containerRef}
      className="tree-view-virtual-container"
      onScroll={handleScroll}
      style={{ height: CONTAINER_HEIGHT }}
    >
      <div style={{ height: totalHeight, position: "relative" }}>
        <div style={{ transform: `translateY(${offsetY}px)` }}>
          {visibleNodes.map((node, index) => {
            const actualIndex = startIndex + index;
            return (
              <TreeItem
                key={node.id}
                node={node}
                isSelected={selectedId === node.id}
                isExpanded={expandedIds.has(node.id)}
                onSelect={() => onSelect(node)}
                onToggle={() => onToggleExpand(node.id)}
                style={{
                  height: ITEM_HEIGHT,
                  position: "absolute",
                  top: actualIndex * ITEM_HEIGHT,
                  left: 0,
                  right: 0,
                }}
              />
            );
          })}
        </div>
      </div>
    </div>
  );
}

/**
 * Simple tree list component for smaller trees.
 */
interface SimpleTreeListProps {
  nodes: TreeNode[];
  selectedId: string | null;
  expandedIds: Set<string>;
  onSelect: (node: TreeNode) => void;
  onToggleExpand: (nodeId: string) => void;
}

function SimpleTreeList({
  nodes,
  selectedId,
  expandedIds,
  onSelect,
  onToggleExpand,
}: SimpleTreeListProps) {
  const flatNodes = useMemo(
    () => flattenTree(nodes, expandedIds),
    [nodes, expandedIds],
  );

  return (
    <div className="tree-view-simple-container">
      {flatNodes.map((node) => (
        <TreeItem
          key={node.id}
          node={node}
          isSelected={selectedId === node.id}
          isExpanded={expandedIds.has(node.id)}
          onSelect={() => onSelect(node)}
          onToggle={() => onToggleExpand(node.id)}
          style={{
            height: ITEM_HEIGHT,
            position: "relative",
          }}
        />
      ))}
    </div>
  );
}

/**
 * Main TreeView component.
 *
 * Displays a hierarchical tree with virtualization support for large datasets.
 * Supports keyboard navigation, search filtering, and status filtering.
 */
export function TreeView({
  nodes,
  selectedId,
  onSelect,
  onToggleExpand,
  expandedIds,
  searchQuery = "",
  statusFilter = "all",
}: TreeViewProps) {
  // Filter nodes based on search query
  const filteredNodes = useMemo(() => {
    let filtered = nodes;

    if (searchQuery.trim()) {
      filtered = filterTreeNodes(nodes, searchQuery);
    }

    // Apply status filter
    if (statusFilter !== "all") {
      const filterByStatus = (node: TreeNode): TreeNode | null => {
        const filteredChildren = node.children
          .map(filterByStatus)
          .filter((child): child is TreeNode => child !== null);

        const matchesStatus =
          node.status === 0 || // Directory nodes pass through
          (statusFilter === "success" &&
            node.status >= 200 &&
            node.status < 300) ||
          (statusFilter === "error" && node.status >= 400);

        if (matchesStatus || filteredChildren.length > 0) {
          return {
            ...node,
            children: filteredChildren,
          };
        }

        return null;
      };

      filtered = filtered
        .map(filterByStatus)
        .filter((node): node is TreeNode => node !== null);
    }

    return filtered;
  }, [nodes, searchQuery, statusFilter]);

  // Count total visible nodes
  const totalVisibleNodes = useMemo(
    () => flattenTree(filteredNodes, expandedIds).length,
    [filteredNodes, expandedIds],
  );

  // Keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const flatNodes = flattenTree(filteredNodes, expandedIds);
      const currentIndex = flatNodes.findIndex((n) => n.id === selectedId);

      switch (e.key) {
        case "ArrowDown": {
          e.preventDefault();
          const nextIndex = Math.min(currentIndex + 1, flatNodes.length - 1);
          if (nextIndex >= 0) {
            onSelect(flatNodes[nextIndex]);
          }
          break;
        }
        case "ArrowUp": {
          e.preventDefault();
          const prevIndex = Math.max(currentIndex - 1, 0);
          if (prevIndex >= 0) {
            onSelect(flatNodes[prevIndex]);
          }
          break;
        }
        case "ArrowLeft": {
          e.preventDefault();
          const current = flatNodes[currentIndex];
          if (current && expandedIds.has(current.id)) {
            onToggleExpand(current.id);
          }
          break;
        }
        case "ArrowRight": {
          e.preventDefault();
          const current = flatNodes[currentIndex];
          if (
            current &&
            current.children.length > 0 &&
            !expandedIds.has(current.id)
          ) {
            onToggleExpand(current.id);
          }
          break;
        }
        case "Enter": {
          e.preventDefault();
          const current = flatNodes[currentIndex];
          if (current && current.children.length > 0) {
            onToggleExpand(current.id);
          }
          break;
        }
      }
    },
    [filteredNodes, expandedIds, selectedId, onSelect, onToggleExpand],
  );

  const useVirtualization = totalVisibleNodes > VIRTUALIZATION_THRESHOLD;

  return (
    <div
      className="tree-view"
      onKeyDown={handleKeyDown}
      role="tree"
      aria-label="Results tree"
      tabIndex={0}
    >
      {filteredNodes.length === 0 ? (
        <div className="tree-view-empty">No results match your filter</div>
      ) : useVirtualization ? (
        <VirtualTreeList
          nodes={filteredNodes}
          selectedId={selectedId}
          expandedIds={expandedIds}
          onSelect={onSelect}
          onToggleExpand={onToggleExpand}
        />
      ) : (
        <SimpleTreeList
          nodes={filteredNodes}
          selectedId={selectedId}
          expandedIds={expandedIds}
          onSelect={onSelect}
          onToggleExpand={onToggleExpand}
        />
      )}
    </div>
  );
}

export default TreeView;
