/**
 * Template A/B Test Manager Component.
 *
 * Purpose:
 * - Coordinate the top-level template A/B test management view.
 *
 * Responsibilities:
 * - Load available templates and existing A/B tests.
 * - Orchestrate create, edit, and list workflows through focused child
 *   components.
 *
 * Scope:
 * - Page-level management for template A/B tests.
 *
 * Usage:
 * - Render from the main application when template experimentation is needed.
 *
 * Invariants/Assumptions:
 * - Child components own local interaction state; this component owns shared
 *   list/template refresh behavior.
 */

import { useCallback, useEffect, useState } from "react";
import type { TemplateAbTest } from "../api";
import { getV1TemplateAbTests, listTemplates } from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import { VisualSelectorBuilder } from "./VisualSelectorBuilder";
import { ABTestCard } from "./templates/ab-tests/ABTestCard";
import { CreateTestForm } from "./templates/ab-tests/CreateTestForm";

export function TemplateABTestManager() {
  const [tests, setTests] = useState<TemplateAbTest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<string | null>(null);
  const [templates, setTemplates] = useState<string[]>([]);

  // Fetch available templates
  const fetchTemplates = useCallback(async () => {
    try {
      const { data, error } = await listTemplates({
        baseUrl: getApiBaseUrl(),
      });
      if (!error && data) {
        setTemplates(data.templates || []);
      }
    } catch {
      // Silently fail - templates are not critical
    }
  }, []);

  useEffect(() => {
    fetchTemplates();
  }, [fetchTemplates]);

  const fetchTests = useCallback(async () => {
    try {
      setLoading(true);
      const { data, error } = await getV1TemplateAbTests({
        baseUrl: getApiBaseUrl(),
      });

      if (error) {
        throw new Error(String(error));
      }

      setTests(data?.tests || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTests();
  }, [fetchTests]);

  if (loading) {
    return <div className="ab-test-manager loading">Loading A/B tests...</div>;
  }

  if (error) {
    return <div className="ab-test-manager error">Error: {error}</div>;
  }

  return (
    <div className="ab-test-manager">
      <div className="ab-test-manager__header">
        <h2>A/B Tests</h2>
        <button
          type="button"
          className="btn btn--primary"
          onClick={() => setShowCreateForm(!showCreateForm)}
        >
          {showCreateForm ? "Cancel" : "Create Test"}
        </button>
      </div>

      {showCreateForm && (
        <CreateTestForm
          templates={templates}
          onSuccess={() => {
            setShowCreateForm(false);
            fetchTests();
          }}
          onCancel={() => setShowCreateForm(false)}
        />
      )}

      {editingTemplate && (
        // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
        // biome-ignore lint/a11y/useKeyWithClickEvents: handled by escape key in component
        <div className="modal-overlay" onClick={() => setEditingTemplate(null)}>
          {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled by child component */}
          {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <VisualSelectorBuilder
              onSave={() => {
                setEditingTemplate(null);
                fetchTemplates();
              }}
              onCancel={() => setEditingTemplate(null)}
            />
          </div>
        </div>
      )}

      <div className="ab-test-manager__list">
        {tests.length === 0 ? (
          <div className="ab-test-manager__empty">
            <p>No A/B tests yet.</p>
            <p className="empty-hint">
              Create one to compare template performance and find the best
              extraction strategy.
            </p>
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => setShowCreateForm(true)}
            >
              Create Your First Test
            </button>
          </div>
        ) : (
          tests.map((test) => (
            <ABTestCard
              key={test.id}
              test={test}
              onUpdate={fetchTests}
              onEditTemplate={setEditingTemplate}
            />
          ))
        )}
      </div>
    </div>
  );
}
