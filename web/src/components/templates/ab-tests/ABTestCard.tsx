/**
 * Template A/B test summary card.
 *
 * Purpose:
 * - Render one template A/B test with actions and expandable results.
 *
 * Responsibilities:
 * - Handle start, stop, delete, and edit-template actions for one test.
 * - Render summary details, status, and child results content.
 *
 * Scope:
 * - Single-card UI for template A/B tests only.
 *
 * Usage:
 * - Render inside TemplateABTestManager for each fetched A/B test.
 *
 * Invariants/Assumptions:
 * - Parent owns list refresh behavior and passes it through as `onUpdate`.
 */

import { useState } from "react";

import {
  deleteV1TemplateAbTestsById,
  postV1TemplateAbTestsByIdStart,
  postV1TemplateAbTestsByIdStop,
} from "../../../api";
import type { TemplateAbTest } from "../../../api";
import { getApiBaseUrl } from "../../../lib/api-config";
import {
  formatTemplateABTestAllocation,
  formatTemplateABTestConfidence,
  getTemplateABTestStatusBadgeClass,
  getTemplateABTestWinnerName,
} from "../../../lib/template-ab-tests";
import { TestResults } from "./TestResults";

interface ABTestCardProps {
  test: TemplateAbTest;
  onUpdate: () => void;
  onEditTemplate: (templateName: string) => void;
}

export function ABTestCard({
  test,
  onUpdate,
  onEditTemplate,
}: ABTestCardProps) {
  const [showResults, setShowResults] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleStart = async () => {
    try {
      setLoading(true);
      const { error } = await postV1TemplateAbTestsByIdStart({
        baseUrl: getApiBaseUrl(),
        path: { id: test.id ?? "" },
      });

      if (error) {
        throw new Error(String(error));
      }

      onUpdate();
    } catch (err) {
      console.error("Failed to start test:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    try {
      setLoading(true);
      const { error } = await postV1TemplateAbTestsByIdStop({
        baseUrl: getApiBaseUrl(),
        path: { id: test.id || "" },
      });

      if (error) {
        throw new Error(String(error));
      }

      onUpdate();
    } catch (err) {
      console.error("Failed to stop test:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!confirm("Are you sure you want to delete this A/B test?")) {
      return;
    }

    try {
      setLoading(true);
      const { error } = await deleteV1TemplateAbTestsById({
        baseUrl: getApiBaseUrl(),
        path: { id: test.id || "" },
      });

      if (error) {
        throw new Error(String(error));
      }

      onUpdate();
    } catch (err) {
      console.error("Failed to delete test:", err);
    } finally {
      setLoading(false);
    }
  };

  const winnerName = getTemplateABTestWinnerName(test);
  const status = test.status ?? "pending";

  return (
    <div className={`ab-test-card ab-test-card--${test.status}`}>
      <div className="ab-test-card__header">
        <div className="ab-test-card__title">
          <h4>{test.name}</h4>
          <span
            className={`badge ${getTemplateABTestStatusBadgeClass(status)}`}
          >
            {status}
          </span>
        </div>
        <div className="ab-test-card__actions">
          {test.status === "pending" && (
            <button
              type="button"
              className="btn btn--small btn--primary"
              onClick={handleStart}
              disabled={loading}
            >
              Start
            </button>
          )}
          {test.status === "running" && (
            <button
              type="button"
              className="btn btn--small btn--warning"
              onClick={handleStop}
              disabled={loading}
            >
              Stop
            </button>
          )}
          <button
            type="button"
            className="btn btn--small btn--secondary"
            onClick={() => setShowResults(!showResults)}
          >
            {showResults ? "Hide Results" : "View Results"}
          </button>
          <button
            type="button"
            className="btn btn--small btn--danger"
            onClick={handleDelete}
            disabled={loading}
          >
            Delete
          </button>
        </div>
      </div>

      {test.description && (
        <p className="ab-test-card__description">{test.description}</p>
      )}

      <div className="ab-test-card__details">
        <div className="detail-row">
          <span className="detail-label">Templates:</span>
          <span className="detail-value">
            {test.baseline_template}
            <button
              type="button"
              className="btn btn--link btn--small"
              onClick={() => onEditTemplate(test.baseline_template ?? "")}
              title="Edit baseline template"
            >
              Edit
            </button>
            {" vs "}
            {test.variant_template}
            <button
              type="button"
              className="btn btn--link btn--small"
              onClick={() => onEditTemplate(test.variant_template ?? "")}
              title="Edit variant template"
            >
              Edit
            </button>
          </span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Allocation:</span>
          <span className="detail-value">
            {formatTemplateABTestAllocation(test.allocation)}
          </span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Min Samples:</span>
          <span className="detail-value">{test.min_sample_size}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Confidence:</span>
          <span className="detail-value">
            {formatTemplateABTestConfidence(test.confidence_level)}
          </span>
        </div>
      </div>

      {test.status === "running" ? (
        <div className="test-progress">
          <div className="progress-bar progress-bar--indeterminate" />
          <span className="progress-label">Running...</span>
        </div>
      ) : winnerName ? (
        <div className="test-progress">
          <span className="winner-badge">Winner: {winnerName}</span>
        </div>
      ) : null}

      {showResults && test.id ? (
        <TestResults testId={test.id} onUpdate={onUpdate} />
      ) : null}
    </div>
  );
}
