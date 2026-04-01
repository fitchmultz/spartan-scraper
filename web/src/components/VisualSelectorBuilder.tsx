/**
 * Purpose: Render the visual selector builder UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import { useCallback, useEffect, useRef, useState } from "react";

import {
  getTemplatePreview,
  testSelector,
  type DomNode,
  type Template,
  type TestSelectorRequest,
  type TestSelectorResponse,
} from "../api";
import { useTemplateBuilder } from "../hooks/useTemplateBuilder";
import { buildBrowserRuntimeFields, isValidHttpUrl } from "../lib/form-utils";
import { getApiErrorMessage } from "../lib/api-errors";

import { BrowserExecutionControls } from "./BrowserExecutionControls";

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

interface VisualSelectorPreviewPanelProps {
  url: string;
  setUrl: (value: string) => void;
  headless: boolean;
  setHeadless: (value: boolean) => void;
  playwright: boolean;
  setPlaywright: (value: boolean) => void;
  onAddSelector: (selector: string) => void;
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
      <div
        className={`dom-tree-node__row ${isSelected ? "selected" : ""}`}
        style={{ paddingLeft: `${(node.depth ?? 0) * 16}px` }}
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
        {!hasChildren && <span className="dom-tree-node__toggle-spacer" />}
        <button
          type="button"
          className="dom-tree-node__select"
          onClick={() => onSelect(node)}
          aria-label={`Select ${node.tag ?? "element"} element`}
        >
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
      </div>
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

function VisualSelectorPreviewPanel({
  url,
  setUrl,
  headless,
  setHeadless,
  playwright,
  setPlaywright,
  onAddSelector,
}: VisualSelectorPreviewPanelProps) {
  // URL/Fetch state
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

  const fetchRunRef = useRef(0);
  const selectorTestRunRef = useRef(0);

  const invalidateSelectorTest = useCallback(() => {
    selectorTestRunRef.current += 1;
    setTesting(false);
    setTestResults(null);
  }, []);

  // Initialize expanded paths when DOM tree loads
  useEffect(() => {
    setExpandedPaths(buildExpandedPaths(domTree));
  }, [domTree]);

  // Fetch page
  const handleFetch = async () => {
    const trimmedURL = url.trim();
    if (!trimmedURL) {
      setFetchError("Please enter a URL");
      return;
    }
    if (!isValidHttpUrl(trimmedURL)) {
      setFetchError("Please enter a valid URL");
      return;
    }

    const runID = ++fetchRunRef.current;
    setFetching(true);
    setFetchError(null);
    setDomTree(null);
    setSelectedNode(null);
    setGeneratedSelector("");
    invalidateSelectorTest();
    setSearchQuery("");

    try {
      const response = await getTemplatePreview({
        query: {
          url: trimmedURL,
          ...buildBrowserRuntimeFields({
            headless,
            playwright,
          }),
        },
      });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to fetch page"),
        );
      }
      if (runID !== fetchRunRef.current) {
        return;
      }
      setDomTree(response.data?.dom_tree ?? null);
    } catch (err) {
      if (runID !== fetchRunRef.current) {
        return;
      }
      setFetchError(getApiErrorMessage(err, "Failed to fetch page"));
    } finally {
      if (runID === fetchRunRef.current) {
        setFetching(false);
      }
    }
  };

  // Handle node selection
  const handleNodeSelect = (node: DomNode) => {
    setSelectedNode(node);
    invalidateSelectorTest();
    const options = generateSelectorOptions(node);
    setGeneratedSelector(options[0] ?? "");
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
    const trimmedURL = url.trim();
    const trimmedSelector = generatedSelector.trim();
    if (!trimmedURL || !trimmedSelector) return;

    const runID = ++selectorTestRunRef.current;
    setTesting(true);
    setTestResults(null);

    try {
      const request: TestSelectorRequest = {
        url: trimmedURL,
        selector: trimmedSelector,
        ...buildBrowserRuntimeFields({
          headless,
          playwright,
        }),
      };
      const response = await testSelector({
        body: request,
      });
      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to test selector"),
        );
      }
      const data = response.data;
      if (runID !== selectorTestRunRef.current) {
        return;
      }
      setTestResults({
        selector: data?.selector ?? trimmedSelector,
        matches: data?.matches ?? 0,
        elements: data?.elements ?? [],
        error: data?.error,
      });
    } catch (err) {
      if (runID !== selectorTestRunRef.current) {
        return;
      }
      setTestResults({
        selector: trimmedSelector,
        matches: 0,
        elements: [],
        error: getApiErrorMessage(err, "Failed to test selector"),
      });
    } finally {
      if (runID === selectorTestRunRef.current) {
        setTesting(false);
      }
    }
  };

  // Add selector to template
  const handleAddSelector = () => {
    const trimmedSelector = generatedSelector.trim();
    if (!trimmedSelector) return;
    onAddSelector(trimmedSelector);
  };

  return (
    <>
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
            <div aria-hidden="true">&nbsp;</div>
            <button
              id="fetch-button"
              type="button"
              className="btn btn--primary"
              aria-label={fetching ? "Fetching page" : "Fetch Page"}
              onClick={handleFetch}
              disabled={fetching || !url}
            >
              {fetching ? "Fetching..." : "Fetch Page"}
            </button>
          </div>
        </div>

        <div className="form-row form-row--inline">
          <BrowserExecutionControls
            headless={headless}
            setHeadless={(value) => {
              setHeadless(value);
              if (!value) {
                setPlaywright(false);
              }
            }}
            usePlaywright={playwright}
            setUsePlaywright={(value) => {
              setPlaywright(value);
              if (value) {
                setHeadless(true);
              }
            }}
            headlessLabel="Use Headless Browser"
            playwrightLabel="Use Playwright"
            helperText="Enable headless to unlock Playwright for DOM preview."
            showTimeout={false}
            disabled={fetching}
          />
        </div>

        {fetchError && (
          <div className="form-error" role="alert">
            {fetchError}
          </div>
        )}
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
                  onChange={(e) => {
                    setGeneratedSelector(e.target.value);
                    invalidateSelectorTest();
                  }}
                  placeholder="CSS selector"
                />
                <button
                  type="button"
                  className="btn btn--secondary btn--small"
                  onClick={handleTestSelector}
                  disabled={testing || !generatedSelector.trim()}
                >
                  {testing ? "Testing..." : "Test"}
                </button>
                <button
                  type="button"
                  className="btn btn--primary btn--small"
                  onClick={handleAddSelector}
                  disabled={!generatedSelector.trim()}
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
                    onClick={() => {
                      setGeneratedSelector(opt);
                      invalidateSelectorTest();
                    }}
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
        </div>
      </div>
    </>
  );
}

export function VisualSelectorBuilder({
  initialTemplate,
  onSave,
  onCancel,
}: VisualSelectorBuilderProps) {
  const [url, setUrl] = useState("");
  const [headless, setHeadless] = useState(false);
  const [playwright, setPlaywright] = useState(false);

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

  const selectorRowKeysRef = useRef(
    (template.selectors ?? []).map((_, index) => `selector-rule-${index}`),
  );
  const nextSelectorRowKeyRef = useRef(selectorRowKeysRef.current.length);

  const loadedPageStateKey = `${url.trim()}::${headless}::${playwright}`;

  useEffect(() => {
    const selectorCount = template.selectors?.length ?? 0;
    const keys = selectorRowKeysRef.current;
    if (keys.length === selectorCount) {
      return;
    }
    if (keys.length < selectorCount) {
      while (keys.length < selectorCount) {
        keys.push(`selector-rule-${nextSelectorRowKeyRef.current}`);
        nextSelectorRowKeyRef.current += 1;
      }
      return;
    }
    keys.length = selectorCount;
  }, [template.selectors]);

  const handleAddSelector = useCallback(
    (selector: string) => {
      const trimmedSelector = selector.trim();
      if (!trimmedSelector) {
        return;
      }

      clearError();
      selectorRowKeysRef.current.push(
        `selector-rule-${nextSelectorRowKeyRef.current}`,
      );
      nextSelectorRowKeyRef.current += 1;
      addSelector(
        createSelectorRule(trimmedSelector, template.selectors?.length ?? 0),
      );
    },
    [addSelector, clearError, template.selectors?.length],
  );

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

      <VisualSelectorPreviewPanel
        key={loadedPageStateKey}
        url={url}
        setUrl={setUrl}
        headless={headless}
        setHeadless={(value) => {
          setHeadless(value);
          if (!value) {
            setPlaywright(false);
          }
        }}
        playwright={playwright}
        setPlaywright={(value) => {
          setPlaywright(value);
          if (value) {
            setHeadless(true);
          }
        }}
        onAddSelector={handleAddSelector}
      />

      <div className="visual-selector-builder__panel">
        <h4>Template Configuration</h4>
        <div className="selector-builder-section">
          <label htmlFor="template-name">Template Name</label>
          <input
            id="template-name"
            type="text"
            value={template.name ?? ""}
            onChange={(e) => updateTemplate({ name: e.target.value })}
            placeholder="Template name"
            className="template-name-input"
          />

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
                <div
                  key={
                    selectorRowKeysRef.current[index] ??
                    `selector-rule-${index}`
                  }
                  className="selector-rule"
                >
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
                      onClick={() => {
                        selectorRowKeysRef.current.splice(index, 1);
                        removeSelector(index);
                      }}
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
  );
}

export default VisualSelectorBuilder;
