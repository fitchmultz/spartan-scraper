/**
 * Chain Builder Component
 *
 * Provides a form for creating new job chains via JSON input.
 * Includes validation for chain definition structure.
 *
 * @module ChainBuilder
 */
import { useState, useCallback } from "react";
import type { ChainCreateRequest } from "../api";

interface ChainBuilderProps {
  onCreate: (chain: ChainCreateRequest) => Promise<void>;
  onCancel?: () => void;
}

// Sample chain template to help users get started
const SAMPLE_CHAIN = JSON.stringify(
  {
    name: "Example Chain",
    description: "A simple two-step workflow",
    definition: {
      nodes: [
        {
          id: "step1",
          kind: "scrape",
          params: {
            url: "https://example.com",
            headless: true,
          },
          metadata: {
            name: "Get Homepage",
            description: "Scrape the homepage",
          },
        },
        {
          id: "step2",
          kind: "crawl",
          params: {
            url: "https://example.com",
            maxDepth: 2,
            maxPages: 50,
          },
          metadata: {
            name: "Crawl Links",
            description: "Follow links from homepage",
          },
        },
      ],
      edges: [
        {
          from: "step1",
          to: "step2",
        },
      ],
    },
  },
  null,
  2,
);

export function ChainBuilder({ onCreate, onCancel }: ChainBuilderProps) {
  const [jsonInput, setJsonInput] = useState(SAMPLE_CHAIN);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const validateChain = useCallback(
    (data: unknown): data is ChainCreateRequest => {
      if (typeof data !== "object" || data === null) {
        throw new Error("Chain definition must be an object");
      }

      const chain = data as Record<string, unknown>;

      // Validate name
      if (typeof chain.name !== "string" || chain.name.trim() === "") {
        throw new Error("name is required and must be a string");
      }

      // Validate definition
      if (typeof chain.definition !== "object" || chain.definition === null) {
        throw new Error("definition is required and must be an object");
      }

      const def = chain.definition as Record<string, unknown>;

      // Validate nodes
      if (!Array.isArray(def.nodes)) {
        throw new Error("definition.nodes must be an array");
      }

      if (def.nodes.length === 0) {
        throw new Error("definition.nodes must contain at least one node");
      }

      const nodeIds = new Set<string>();
      for (let i = 0; i < def.nodes.length; i++) {
        const node = def.nodes[i] as Record<string, unknown>;

        if (typeof node.id !== "string" || node.id.trim() === "") {
          throw new Error(`definition.nodes[${i}].id is required`);
        }

        if (nodeIds.has(node.id)) {
          throw new Error(`definition.nodes[${i}].id is duplicate: ${node.id}`);
        }
        nodeIds.add(node.id);

        if (typeof node.kind !== "string" || node.kind.trim() === "") {
          throw new Error(`definition.nodes[${i}].kind is required`);
        }

        const validKinds = ["scrape", "crawl", "research"];
        if (!validKinds.includes(node.kind)) {
          throw new Error(
            `definition.nodes[${i}].kind must be one of: ${validKinds.join(", ")}`,
          );
        }

        if (typeof node.params !== "object" || node.params === null) {
          throw new Error(
            `definition.nodes[${i}].params is required and must be an object`,
          );
        }
      }

      // Validate edges if present
      if (def.edges !== undefined) {
        if (!Array.isArray(def.edges)) {
          throw new Error("definition.edges must be an array");
        }

        for (let i = 0; i < def.edges.length; i++) {
          const edge = def.edges[i] as Record<string, unknown>;

          if (typeof edge.from !== "string" || edge.from.trim() === "") {
            throw new Error(`definition.edges[${i}].from is required`);
          }

          if (typeof edge.to !== "string" || edge.to.trim() === "") {
            throw new Error(`definition.edges[${i}].to is required`);
          }

          if (!nodeIds.has(edge.from)) {
            throw new Error(
              `definition.edges[${i}].from references unknown node: ${edge.from}`,
            );
          }

          if (!nodeIds.has(edge.to)) {
            throw new Error(
              `definition.edges[${i}].to references unknown node: ${edge.to}`,
            );
          }

          if (edge.from === edge.to) {
            throw new Error(
              `definition.edges[${i}] is a self-reference: ${edge.from}`,
            );
          }
        }

        // Check for cycles using DFS
        const adjacency = new Map<string, string[]>();
        for (const edge of def.edges as Array<{ from: string; to: string }>) {
          if (!adjacency.has(edge.from)) {
            adjacency.set(edge.from, []);
          }
          adjacency.get(edge.from)?.push(edge.to);
        }

        const visited = new Set<string>();
        const recursionStack = new Set<string>();

        const hasCycle = (nodeId: string): boolean => {
          visited.add(nodeId);
          recursionStack.add(nodeId);

          const neighbors = adjacency.get(nodeId) || [];
          for (const neighbor of neighbors) {
            if (!visited.has(neighbor)) {
              if (hasCycle(neighbor)) {
                return true;
              }
            } else if (recursionStack.has(neighbor)) {
              return true;
            }
          }

          recursionStack.delete(nodeId);
          return false;
        };

        for (const nodeId of nodeIds) {
          if (!visited.has(nodeId)) {
            if (hasCycle(nodeId)) {
              throw new Error("Chain definition contains a cycle");
            }
          }
        }
      }

      return true;
    },
    [],
  );

  const handleSubmit = useCallback(async () => {
    setError(null);

    // Parse JSON
    let data: unknown;
    try {
      data = JSON.parse(jsonInput);
    } catch (err) {
      setError(`Invalid JSON: ${String(err)}`);
      return;
    }

    // Validate structure
    try {
      validateChain(data);
    } catch (err) {
      setError(`Validation error: ${String(err)}`);
      return;
    }

    // Submit
    setIsSubmitting(true);
    try {
      await onCreate(data as ChainCreateRequest);
      // Reset form on success
      setJsonInput(SAMPLE_CHAIN);
    } catch (err) {
      setError(`Failed to create chain: ${String(err)}`);
    } finally {
      setIsSubmitting(false);
    }
  }, [jsonInput, validateChain, onCreate]);

  const handleLoadSample = useCallback(() => {
    setJsonInput(SAMPLE_CHAIN);
    setError(null);
  }, []);

  return (
    <div className="panel">
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 16,
        }}
      >
        <h2>Create Chain</h2>
        <div style={{ display: "flex", gap: 8 }}>
          <button
            type="button"
            className="secondary"
            onClick={handleLoadSample}
            disabled={isSubmitting}
          >
            Reset to Sample
          </button>
          {onCancel && (
            <button
              type="button"
              className="secondary"
              onClick={onCancel}
              disabled={isSubmitting}
            >
              Cancel
            </button>
          )}
        </div>
      </div>

      <div style={{ marginBottom: 16 }}>
        <p
          style={{ fontSize: 14, color: "var(--text-muted)", marginBottom: 12 }}
        >
          Define a reusable job chain using JSON. Each chain consists of nodes
          (job templates) and edges (dependencies).
        </p>

        <div
          style={{
            background: "var(--bg)",
            padding: 12,
            borderRadius: 4,
            fontSize: 13,
            marginBottom: 12,
          }}
        >
          <strong>Quick Reference:</strong>
          <ul style={{ margin: "8px 0", paddingLeft: 20 }}>
            <li>
              <code>name</code> (required): Unique name for the chain
            </li>
            <li>
              <code>description</code> (optional): Human-readable description
            </li>
            <li>
              <code>definition.nodes</code> (required): Array of job templates
            </li>
            <li>
              <code>definition.edges</code> (optional): Array of dependencies{" "}
              {"{"}from, to{"}"}
            </li>
            <li>
              Each node needs: <code>id</code>, <code>kind</code>{" "}
              (scrape/crawl/research), <code>params</code>
            </li>
          </ul>
        </div>
      </div>

      <div style={{ marginBottom: 16 }}>
        <label
          htmlFor="chain-json"
          style={{
            display: "block",
            fontSize: 14,
            fontWeight: 600,
            marginBottom: 8,
          }}
        >
          Chain Definition JSON
        </label>
        <textarea
          id="chain-json"
          value={jsonInput}
          onChange={(e) => {
            setJsonInput(e.target.value);
            setError(null);
          }}
          disabled={isSubmitting}
          style={{
            width: "100%",
            minHeight: 400,
            fontFamily: "monospace",
            fontSize: 13,
            padding: 12,
            border: error
              ? "1px solid var(--error)"
              : "1px solid var(--border)",
            borderRadius: 4,
            background: "var(--bg)",
            color: "var(--text)",
            resize: "vertical",
            lineHeight: 1.5,
          }}
        />
      </div>

      {error && (
        <div
          style={{
            background: "rgba(239, 68, 68, 0.1)",
            border: "1px solid var(--error)",
            borderRadius: 4,
            padding: 12,
            marginBottom: 16,
          }}
        >
          <div
            style={{
              color: "var(--error)",
              fontSize: 14,
              fontWeight: 600,
              marginBottom: 4,
            }}
          >
            Error
          </div>
          <div style={{ color: "var(--error)", fontSize: 13 }}>{error}</div>
        </div>
      )}

      <div style={{ display: "flex", gap: 8 }}>
        <button
          type="button"
          onClick={handleSubmit}
          disabled={isSubmitting}
          style={{ minWidth: 120 }}
        >
          {isSubmitting ? "Creating..." : "Create Chain"}
        </button>
        {onCancel && (
          <button
            type="button"
            className="secondary"
            onClick={onCancel}
            disabled={isSubmitting}
          >
            Cancel
          </button>
        )}
      </div>
    </div>
  );
}
