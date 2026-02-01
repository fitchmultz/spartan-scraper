/**
 * Visual CSS Selector Builder Component
 *
 * Provides a visual, point-and-click interface for building CSS selectors
 * by inspecting remote page DOM structure. Supports selector testing and
 * template creation/editing.
 *
 * @module VisualSelectorBuilder
 */

import { useState, useCallback, useEffect } from "react";

// Types matching the API responses
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

interface TemplatePreviewResponse {
  url: string;
  title: string;
  dom_tree: DOMNode;
  fetch_time_ms: number;
  fetcher: string;
}

interface TestSelectorResponse {
  selector: string;
  matches: number;
  elements: DOMElement[];
  error?: string;
}

interface DOMElement {
  tag: string;
  text: string;
  html: string;
  path: string;
}

interface SelectorRule {
  name: string;
  selector: string;
  attr: string;
  all: boolean;
  join: string;
  trim: boolean;
  required: boolean;
}

interface Template {
  name: string;
  selectors: SelectorRule[];
  jsonld?: unknown[];
  regex?: unknown[];
  normalize?: {
    titleField?: string;
    descriptionField?: string;
    textField?: string;
    metaFields?: Record<string, string>;
  };
}

interface VisualSelectorBuilderProps {
  initialTemplate?: Template;
  onSave: (template: Template) => void;
  onCancel: () => void;
}

// DOM Tree Node Component
interface DOMTreeNodeProps {
  node: DOMNode;
  selectedPath: string | null;
  expandedPaths: Set<string>;
  onSelect: (node: DOMNode) => void;
  onToggleExpand: (path: string) => void;
  searchQuery: string;
}

function DOMTreeNode({
  node,
  selectedPath,
  expandedPaths,
  onSelect,
  onToggleExpand,
  searchQuery,
}: DOMTreeNodeProps) {
  const isExpanded = expandedPaths.has(node.path);
  const isSelected = selectedPath === node.path;
  const hasChildren = node.children && node.children.length > 0;

  // Filter visibility based on search
  const matchesSearch = searchQuery
    ? (node.tag?.toLowerCase().includes(searchQuery.toLowerCase()) ?? false) ||
      (node.id?.toLowerCase().includes(searchQuery.toLowerCase()) ?? false) ||
      (node.classes?.some((c) =>
        c.toLowerCase().includes(searchQuery.toLowerCase()),
      ) ??
        false) ||
      (node.text?.toLowerCase().includes(searchQuery.toLowerCase()) ?? false)
    : true;

  const childMatchesSearch = searchQuery
    ? (node.children?.some((child) =>
        DOMTreeNodeMatchesSearch(child, searchQuery),
      ) ?? false)
    : true;

  const shouldShow = matchesSearch || childMatchesSearch;

  if (!shouldShow) return null;

  return (
    <div className="dom-tree-node">
      <button
        type="button"
        className={`dom-tree-node__row ${isSelected ? "selected" : ""}`}
        style={{ paddingLeft: `${node.depth * 16}px` }}
        onClick={() => onSelect(node)}
        aria-label={`Select ${node.tag} element`}
      >
        {hasChildren && (
          <button
            type="button"
            className="dom-tree-node__toggle"
            onClick={(e) => {
              e.stopPropagation();
              onToggleExpand(node.path);
            }}
            aria-label={isExpanded ? "Collapse" : "Expand"}
          >
            {isExpanded ? "▼" : "▶"}
          </button>
        )}
        <span className="dom-tree-node__tag">{node.tag}</span>
        {node.id && (
          <span className="dom-tree-node__id">#{node.id.slice(0, 20)}</span>
        )}
        {node.classes && node.classes.length > 0 && (
          <span className="dom-tree-node__classes">
            .{node.classes.slice(0, 2).join(".")}
            {node.classes.length > 2 && "..."}
          </span>
        )}
        {node.text && (
          <span className="dom-tree-node__text">
            {node.text.slice(0, 50)}
            {node.text.length > 50 && "..."}
          </span>
        )}
      </button>
      {isExpanded &&
        node.children?.map((child) => (
          <DOMTreeNode
            key={child.path}
            node={child}
            selectedPath={selectedPath}
            expandedPaths={expandedPaths}
            onSelect={onSelect}
            onToggleExpand={onToggleExpand}
            searchQuery={searchQuery}
          />
        ))}
    </div>
  );
}

// Helper to check if a node matches search (for parent visibility)
function DOMTreeNodeMatchesSearch(node: DOMNode, query: string): boolean {
  const matches =
    (node.tag?.toLowerCase().includes(query.toLowerCase()) ?? false) ||
    (node.id?.toLowerCase().includes(query.toLowerCase()) ?? false) ||
    (node.classes?.some((c) => c.toLowerCase().includes(query.toLowerCase())) ??
      false) ||
    (node.text?.toLowerCase().includes(query.toLowerCase()) ?? false);

  if (matches) return true;

  return (
    node.children?.some((child) => DOMTreeNodeMatchesSearch(child, query)) ??
    false
  );
}

// Generate CSS selector options for a node
function generateSelectorOptions(node: DOMNode): string[] {
  const options: string[] = [];

  // ID-based (most specific)
  if (node.id) {
    options.push(`#${node.id}`);
  }

  // Class-based
  if (node.classes && node.classes.length > 0) {
    const classSelector = `${node.tag}.${node.classes.join(".")}`;
    options.push(classSelector);

    // Just the classes
    if (node.classes.length <= 3) {
      options.push(`.${node.classes.join(".")}`);
    }
  }

  // Tag with attribute
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
  options.push(node.path);

  return options;
}

export function VisualSelectorBuilder({
  initialTemplate,
  onSave,
  onCancel,
}: VisualSelectorBuilderProps) {
  // URL/Fetch state
  const [url, setUrl] = useState("");
  const [headless, setHeadless] = useState(false);
  const [playwright, setPlaywright] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);

  // DOM tree state
  const [domTree, setDomTree] = useState<DOMNode | null>(null);
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());
  const [selectedNode, setSelectedNode] = useState<DOMNode | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  // Selector testing state
  const [generatedSelector, setGeneratedSelector] = useState("");
  const [testing, setTesting] = useState(false);
  const [testResults, setTestResults] = useState<TestSelectorResponse | null>(
    null,
  );

  // Template state
  const [templateName, setTemplateName] = useState(initialTemplate?.name ?? "");
  const [selectors, setSelectors] = useState<SelectorRule[]>(
    initialTemplate?.selectors ?? [],
  );
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Initialize expanded paths when DOM tree loads
  useEffect(() => {
    if (domTree) {
      const paths = new Set<string>();
      // Expand first 2 levels
      const addPaths = (node: DOMNode, depth: number) => {
        if (depth <= 2) {
          paths.add(node.path);
        }
        node.children?.forEach((child) => {
          addPaths(child, depth + 1);
        });
      };
      addPaths(domTree, 0);
      setExpandedPaths(paths);
    }
  }, [domTree]);

  // Fetch page
  const handleFetch = useCallback(async () => {
    if (!url) {
      setFetchError("Please enter a URL");
      return;
    }

    setFetching(true);
    setFetchError(null);
    setDomTree(null);

    try {
      const params = new URLSearchParams();
      params.set("url", url);
      if (headless) params.set("headless", "true");
      if (playwright) params.set("playwright", "true");

      const response = await fetch(`/v1/template-preview?${params}`);

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to fetch page");
      }

      const data: TemplatePreviewResponse = await response.json();
      setDomTree(data.dom_tree);
    } catch (err) {
      setFetchError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setFetching(false);
    }
  }, [url, headless, playwright]);

  // Handle node selection
  const handleNodeSelect = useCallback((node: DOMNode) => {
    setSelectedNode(node);
    const options = generateSelectorOptions(node);
    if (options.length > 0) {
      setGeneratedSelector(options[0]);
    }
  }, []);

  // Toggle expand
  const handleToggleExpand = useCallback((path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }, []);

  // Test selector
  const handleTestSelector = useCallback(async () => {
    if (!url || !generatedSelector) return;

    setTesting(true);
    setTestResults(null);

    try {
      const response = await fetch("/v1/template-preview/test-selector", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          url,
          selector: generatedSelector,
          headless,
          playwright,
        }),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to test selector");
      }

      const data: TestSelectorResponse = await response.json();
      setTestResults(data);
    } catch (err) {
      setTestResults({
        selector: generatedSelector,
        matches: 0,
        elements: [],
        error: err instanceof Error ? err.message : "Unknown error",
      });
    } finally {
      setTesting(false);
    }
  }, [url, generatedSelector, headless, playwright]);

  // Add selector to template
  const handleAddSelector = useCallback(() => {
    if (!generatedSelector) return;

    const newRule: SelectorRule = {
      name: `field_${selectors.length + 1}`,
      selector: generatedSelector,
      attr: "text",
      all: false,
      join: "",
      trim: true,
      required: false,
    };

    setSelectors((prev) => [...prev, newRule]);
  }, [generatedSelector, selectors.length]);

  // Update selector rule
  const handleUpdateRule = useCallback(
    (index: number, updates: Partial<SelectorRule>) => {
      setSelectors((prev) =>
        prev.map((rule, i) => (i === index ? { ...rule, ...updates } : rule)),
      );
    },
    [],
  );

  // Remove selector rule
  const handleRemoveRule = useCallback((index: number) => {
    setSelectors((prev) => prev.filter((_, i) => i !== index));
  }, []);

  // Save template
  const handleSave = useCallback(async () => {
    if (!templateName) {
      setSaveError("Template name is required");
      return;
    }

    if (selectors.length === 0) {
      setSaveError("At least one selector is required");
      return;
    }

    setSaving(true);
    setSaveError(null);

    try {
      const template: Template = {
        name: templateName,
        selectors,
      };

      const isUpdate = initialTemplate?.name === templateName;
      const method = isUpdate ? "PUT" : "POST";
      const endpoint = isUpdate
        ? `/v1/templates/${encodeURIComponent(templateName)}`
        : "/v1/templates";

      const response = await fetch(endpoint, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(template),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to save template");
      }

      onSave(template);
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setSaving(false);
    }
  }, [templateName, selectors, initialTemplate, onSave]);

  return (
    <div className="visual-selector-builder">
      <div className="visual-selector-builder__header">
        <h3>Visual Selector Builder</h3>
        <div className="visual-selector-builder__actions">
          <button
            type="button"
            className="btn btn--secondary"
            onClick={onCancel}
            disabled={saving}
          >
            Cancel
          </button>
          <button
            type="button"
            className="btn btn--primary"
            onClick={handleSave}
            disabled={saving || selectors.length === 0}
          >
            {saving ? "Saving..." : "Save Template"}
          </button>
        </div>
      </div>

      {/* URL Input Section */}
      <div className="visual-selector-builder__url-section">
        <div className="form-row">
          <div className="form-group form-group--grow">
            <label htmlFor="preview-url">URL to Analyze</label>
            <input
              id="preview-url"
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://example.com/article"
              disabled={fetching}
            />
          </div>
          <div className="form-group form-group--shrink">
            <label htmlFor="fetch-button">&nbsp;</label>
            <button
              id="fetch-button"
              type="button"
              className="btn btn--primary"
              onClick={handleFetch}
              disabled={fetching || !url}
            >
              {fetching ? "Fetching..." : "Fetch Page"}
            </button>
          </div>
        </div>

        <div className="form-row form-row--inline">
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={headless}
              onChange={(e) => setHeadless(e.target.checked)}
              disabled={fetching}
            />
            Use Headless Browser
          </label>
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={playwright}
              onChange={(e) => setPlaywright(e.target.checked)}
              disabled={fetching}
            />
            Use Playwright
          </label>
        </div>

        {fetchError && <div className="form-error">{fetchError}</div>}
      </div>

      {/* Main Content */}
      <div className="visual-selector-builder__content">
        {/* DOM Tree Panel */}
        <div className="visual-selector-builder__panel">
          <h4>DOM Tree</h4>
          {domTree ? (
            <>
              <input
                type="text"
                className="dom-tree-search"
                placeholder="Search elements..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
              <div className="dom-tree">
                <DOMTreeNode
                  node={domTree}
                  selectedPath={selectedNode?.path ?? null}
                  expandedPaths={expandedPaths}
                  onSelect={handleNodeSelect}
                  onToggleExpand={handleToggleExpand}
                  searchQuery={searchQuery}
                />
              </div>
            </>
          ) : (
            <div className="dom-tree-placeholder">
              Enter a URL and click Fetch Page to see the DOM structure
            </div>
          )}
        </div>

        {/* Selector Builder Panel */}
        <div className="visual-selector-builder__panel">
          <h4>Selector Builder</h4>

          {/* Generated Selector */}
          {selectedNode && (
            <div className="selector-builder-section">
              <label htmlFor="generated-selector">Generated Selector</label>
              <div className="generated-selector">
                <input
                  id="generated-selector"
                  type="text"
                  value={generatedSelector}
                  onChange={(e) => setGeneratedSelector(e.target.value)}
                  placeholder="CSS selector"
                />
                <button
                  type="button"
                  className="btn btn--secondary btn--small"
                  onClick={handleTestSelector}
                  disabled={testing || !generatedSelector}
                >
                  {testing ? "Testing..." : "Test"}
                </button>
                <button
                  type="button"
                  className="btn btn--primary btn--small"
                  onClick={handleAddSelector}
                  disabled={!generatedSelector}
                >
                  Add to Template
                </button>
              </div>

              {/* Selector Options */}
              <div className="selector-options">
                {generateSelectorOptions(selectedNode).map((opt) => (
                  <button
                    key={opt}
                    type="button"
                    className={`selector-option ${generatedSelector === opt ? "active" : ""}`}
                    onClick={() => setGeneratedSelector(opt)}
                  >
                    {opt}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Test Results */}
          {testResults && (
            <div className="selector-builder-section">
              <label htmlFor="test-results">Test Results</label>
              <div
                id="test-results"
                className={`test-results ${testResults.matches > 0 ? "success" : testResults.error ? "error" : ""}`}
              >
                <div className="test-results__count">
                  {testResults.matches} match
                  {testResults.matches !== 1 ? "es" : ""}
                </div>
                {testResults.error && (
                  <div className="test-results__error">{testResults.error}</div>
                )}
                {testResults.elements.length > 0 && (
                  <div className="test-results__elements">
                    {testResults.elements.slice(0, 5).map((elem) => (
                      <div
                        key={`${elem.tag}-${elem.path}`}
                        className="test-result-element"
                      >
                        <code>&lt;{elem.tag}&gt;</code>
                        <span>{elem.text}</span>
                      </div>
                    ))}
                    {testResults.elements.length > 5 && (
                      <div className="test-results__more">
                        +{testResults.elements.length - 5} more
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Template Configuration */}
          <div className="selector-builder-section">
            <label htmlFor="template-name">Template Configuration</label>
            <input
              id="template-name"
              type="text"
              value={templateName}
              onChange={(e) => setTemplateName(e.target.value)}
              placeholder="Template name"
              className="template-name-input"
            />

            {/* Selector Rules Table */}
            <div className="selector-rules">
              <div className="selector-rules__header">
                <span>Field</span>
                <span>Selector</span>
                <span>Attr</span>
                <span>Actions</span>
              </div>
              {selectors.length === 0 ? (
                <div className="selector-rules__empty">
                  No selectors added yet. Select an element from the DOM tree.
                </div>
              ) : (
                selectors.map((rule, index) => (
                  <div key={rule.name} className="selector-rule">
                    <input
                      type="text"
                      value={rule.name}
                      onChange={(e) =>
                        handleUpdateRule(index, { name: e.target.value })
                      }
                      placeholder="Field name"
                    />
                    <input
                      type="text"
                      value={rule.selector}
                      onChange={(e) =>
                        handleUpdateRule(index, { selector: e.target.value })
                      }
                      placeholder="CSS selector"
                    />
                    <select
                      value={rule.attr}
                      onChange={(e) =>
                        handleUpdateRule(index, { attr: e.target.value })
                      }
                    >
                      <option value="text">text</option>
                      <option value="content">content</option>
                      <option value="href">href</option>
                      <option value="src">src</option>
                      <option value="alt">alt</option>
                      <option value="title">title</option>
                      <option value="value">value</option>
                    </select>
                    <div className="selector-rule__actions">
                      <label className="checkbox-label checkbox-label--small">
                        <input
                          type="checkbox"
                          checked={rule.trim}
                          onChange={(e) =>
                            handleUpdateRule(index, { trim: e.target.checked })
                          }
                        />
                        Trim
                      </label>
                      <button
                        type="button"
                        className="btn btn--danger btn--small"
                        onClick={() => handleRemoveRule(index)}
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

          {saveError && <div className="form-error">{saveError}</div>}
        </div>
      </div>
    </div>
  );
}

export default VisualSelectorBuilder;
