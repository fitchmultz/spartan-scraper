/**
 * Visual CSS Selector Builder Component
 *
 * Provides a visual, point-and-click interface for building CSS selectors
 * by inspecting remote page DOM structure. Supports selector testing and
 * template creation/editing.
 *
 * @module VisualSelectorBuilder
 */

import { useEffect, useState } from "react";

import {
  getTemplatePreview,
  testSelector,
  type DomNode,
  type Template,
  type TestSelectorRequest,
  type TestSelectorResponse,
} from "../api";
import { useTemplateBuilder } from "../hooks/useTemplateBuilder";

import {
  buildExpandedPaths,
  createSelectorRule,
  generateSelectorOptions,
  nodeMatchesSearch,
} from "./visual-selector-builder/selectorBuilderUtils";

interface VisualSelectorBuilderProps {
  initialTemplate?: Template;
  onSave: (template: Template) => void;
  onCancel: () => void;
}

// DOM Tree Node Component
interface DOMTreeNodeProps {
  node: DomNode;
  selectedPath: string | null;
  expandedPaths: Set<string>;
  onSelect: (node: DomNode) => void;
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
  const nodePath = node.path ?? "";
  const isExpanded = expandedPaths.has(nodePath);
  const isSelected = selectedPath === node.path;
  const hasChildren = node.children && node.children.length > 0;

  // Filter visibility based on search
  const matchesSearch = searchQuery
    ? nodeMatchesSearch(node, searchQuery)
    : true;
  const childMatchesSearch = searchQuery
    ? (node.children?.some((child) => nodeMatchesSearch(child, searchQuery)) ??
      false)
    : true;

  const shouldShow = matchesSearch || childMatchesSearch;

  if (!shouldShow) return null;

  return (
    <div className="dom-tree-node">
      <button
        type="button"
        className={`dom-tree-node__row ${isSelected ? "selected" : ""}`}
        style={{ paddingLeft: `${(node.depth ?? 0) * 16}px` }}
        onClick={() => onSelect(node)}
        aria-label={`Select ${node.tag ?? "element"} element`}
      >
        {hasChildren && (
          <button
            type="button"
            className="dom-tree-node__toggle"
            onClick={(e) => {
              e.stopPropagation();
              onToggleExpand(nodePath);
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
            key={child.path ?? `${child.tag ?? "node"}-${child.depth ?? 0}`}
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
  const [domTree, setDomTree] = useState<DomNode | null>(null);
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());
  const [selectedNode, setSelectedNode] = useState<DomNode | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  // Selector testing state
  const [generatedSelector, setGeneratedSelector] = useState("");
  const [testing, setTesting] = useState(false);
  const [testResults, setTestResults] = useState<TestSelectorResponse | null>(
    null,
  );

  const {
    template,
    updateTemplate,
    addSelector,
    updateSelector,
    removeSelector,
    saveTemplate,
    isSaving,
    error: saveError,
    clearError,
  } = useTemplateBuilder({ initialTemplate, onSave });

  // Initialize expanded paths when DOM tree loads
  useEffect(() => {
    setExpandedPaths(buildExpandedPaths(domTree));
  }, [domTree]);

  // Fetch page
  const handleFetch = async () => {
    if (!url) {
      setFetchError("Please enter a URL");
      return;
    }

    setFetching(true);
    setFetchError(null);
    setDomTree(null);

    try {
      const response = await getTemplatePreview({
        query: {
          url,
          ...(headless ? { headless: true } : {}),
          ...(playwright ? { playwright: true } : {}),
        },
      });
      if (response.error) {
        throw new Error(String(response.error) || "Failed to fetch page");
      }
      setDomTree(response.data?.dom_tree ?? null);
    } catch (err) {
      setFetchError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setFetching(false);
    }
  };

  // Handle node selection
  const handleNodeSelect = (node: DomNode) => {
    setSelectedNode(node);
    const options = generateSelectorOptions(node);
    if (options.length > 0) {
      setGeneratedSelector(options[0]);
    }
  };

  // Toggle expand
  const handleToggleExpand = (path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  };

  // Test selector
  const handleTestSelector = async () => {
    if (!url || !generatedSelector) return;

    setTesting(true);
    setTestResults(null);

    try {
      const request: TestSelectorRequest = {
        url,
        selector: generatedSelector,
        ...(headless ? { headless: true } : {}),
        ...(playwright ? { playwright: true } : {}),
      };
      const response = await testSelector({
        body: request,
      });
      if (response.error) {
        throw new Error(String(response.error) || "Failed to test selector");
      }
      const data = response.data;
      setTestResults({
        selector: data?.selector ?? generatedSelector,
        matches: data?.matches ?? 0,
        elements: data?.elements ?? [],
        error: data?.error,
      });
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
  };

  // Add selector to template
  const handleAddSelector = () => {
    if (!generatedSelector) return;
    clearError();
    addSelector(
      createSelectorRule(generatedSelector, template.selectors?.length ?? 0),
    );
  };

  // Save template
  const handleSave = async () => {
    await saveTemplate();
  };

  return (
    <div className="visual-selector-builder">
      <div className="visual-selector-builder__header">
        <h3>Visual Selector Builder</h3>
        <div className="visual-selector-builder__actions">
          <button
            type="button"
            className="btn btn--secondary"
            onClick={onCancel}
            disabled={isSaving}
          >
            Cancel
          </button>
          <button
            type="button"
            className="btn btn--primary"
            onClick={handleSave}
            disabled={isSaving || (template.selectors?.length ?? 0) === 0}
          >
            {isSaving ? "Saving..." : "Save Template"}
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
                className={`test-results ${(testResults.matches ?? 0) > 0 ? "success" : testResults.error ? "error" : ""}`}
              >
                <div className="test-results__count">
                  {testResults.matches ?? 0} match
                  {(testResults.matches ?? 0) !== 1 ? "es" : ""}
                </div>
                {testResults.error && (
                  <div className="test-results__error">{testResults.error}</div>
                )}
                {(testResults.elements?.length ?? 0) > 0 && (
                  <div className="test-results__elements">
                    {testResults.elements?.slice(0, 5).map((elem) => (
                      <div
                        key={`${elem.tag ?? "element"}-${elem.path ?? "no-path"}`}
                        className="test-result-element"
                      >
                        <code>&lt;{elem.tag}&gt;</code>
                        <span>{elem.text}</span>
                      </div>
                    ))}
                    {(testResults.elements?.length ?? 0) > 5 && (
                      <div className="test-results__more">
                        +{(testResults.elements?.length ?? 0) - 5} more
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
              value={template.name ?? ""}
              onChange={(e) => updateTemplate({ name: e.target.value })}
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
              {(template.selectors?.length ?? 0) === 0 ? (
                <div className="selector-rules__empty">
                  No selectors added yet. Select an element from the DOM tree.
                </div>
              ) : (
                template.selectors?.map((rule, index) => (
                  <div key={rule.name} className="selector-rule">
                    <input
                      type="text"
                      value={rule.name ?? ""}
                      onChange={(e) =>
                        updateSelector(index, { name: e.target.value })
                      }
                      placeholder="Field name"
                    />
                    <input
                      type="text"
                      value={rule.selector ?? ""}
                      onChange={(e) =>
                        updateSelector(index, { selector: e.target.value })
                      }
                      placeholder="CSS selector"
                    />
                    <select
                      value={rule.attr ?? "text"}
                      onChange={(e) =>
                        updateSelector(index, { attr: e.target.value })
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
                          checked={rule.trim ?? true}
                          onChange={(e) =>
                            updateSelector(index, { trim: e.target.checked })
                          }
                        />
                        Trim
                      </label>
                      <button
                        type="button"
                        className="btn btn--danger btn--small"
                        onClick={() => removeSelector(index)}
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
