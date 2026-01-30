/**
 * Transform Preview Component
 *
 * Provides a UI for testing JMESPath and JSONata transformations on job results.
 * Allows users to input expressions, preview the transformed output, and
 * apply transformations to exports.
 *
 * @module TransformPreview
 */
import { useState, useCallback, useEffect } from "react";

interface TransformPreviewProps {
  jobId: string;
  onApply?: (expression: string, language: "jmespath" | "jsonata") => void;
}

interface TransformResult {
  results: unknown[];
  error?: string;
  resultCount: number;
}

/**
 * Validates a transformation expression via API.
 */
async function validateTransform(
  expression: string,
  language: "jmespath" | "jsonata",
): Promise<{ valid: boolean; error?: string; message?: string }> {
  const response = await fetch("/v1/transform/validate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ expression, language }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || "Validation failed");
  }

  return response.json();
}

/**
 * Previews a transformation on job results via API.
 */
async function previewTransform(
  jobId: string,
  expression: string,
  language: "jmespath" | "jsonata",
  limit = 10,
): Promise<TransformResult> {
  const response = await fetch(`/v1/jobs/${jobId}/preview-transform`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ jobId, expression, language, limit }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || "Preview failed");
  }

  return response.json();
}

/**
 * Main TransformPreview component.
 */
export function TransformPreview({ jobId, onApply }: TransformPreviewProps) {
  const [expression, setExpression] = useState("");
  const [language, setLanguage] = useState<"jmespath" | "jsonata">("jmespath");
  const [limit, setLimit] = useState(10);
  const [isValidating, setIsValidating] = useState(false);
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [validationResult, setValidationResult] = useState<{
    valid: boolean;
    message?: string;
  } | null>(null);
  const [previewResult, setPreviewResult] = useState<TransformResult | null>(
    null,
  );
  const [error, setError] = useState<string | null>(null);

  // Debounced validation
  useEffect(() => {
    if (!expression.trim()) {
      setValidationResult(null);
      return;
    }

    const timer = setTimeout(async () => {
      setIsValidating(true);
      setError(null);
      try {
        const result = await validateTransform(expression, language);
        setValidationResult({
          valid: result.valid,
          message: result.message,
        });
      } catch (err) {
        setValidationResult({ valid: false, message: String(err) });
      } finally {
        setIsValidating(false);
      }
    }, 500);

    return () => clearTimeout(timer);
  }, [expression, language]);

  const handlePreview = useCallback(async () => {
    if (!expression.trim()) {
      setError("Please enter an expression");
      return;
    }

    setIsPreviewing(true);
    setError(null);
    setPreviewResult(null);

    try {
      const result = await previewTransform(jobId, expression, language, limit);
      setPreviewResult(result);
    } catch (err) {
      setError(String(err));
    } finally {
      setIsPreviewing(false);
    }
  }, [jobId, expression, language, limit]);

  const handleApply = useCallback(() => {
    if (onApply && validationResult?.valid) {
      onApply(expression, language);
    }
  }, [onApply, expression, language, validationResult]);

  const insertExample = useCallback((exampleExpr: string) => {
    setExpression(exampleExpr);
  }, []);

  return (
    <div className="transform-preview">
      {/* Expression Input Section */}
      <div className="transform-input-section">
        <div className="transform-header">
          <h4>Data Transformation</h4>
          <div className="language-selector">
            <button
              type="button"
              className={language === "jmespath" ? "active" : ""}
              onClick={() => setLanguage("jmespath")}
            >
              JMESPath
            </button>
            <button
              type="button"
              className={language === "jsonata" ? "active" : ""}
              onClick={() => setLanguage("jsonata")}
            >
              JSONata
            </button>
          </div>
        </div>

        <div className="expression-input">
          <textarea
            value={expression}
            onChange={(e) => setExpression(e.target.value)}
            placeholder={
              language === "jmespath"
                ? 'e.g., [?status=="success"].{url: url, title: title}'
                : 'e.g., $[status="success"].{"url": url, "title": title}'
            }
            rows={3}
          />
          {isValidating && (
            <span className="validation-status validating">Validating...</span>
          )}
          {!isValidating && validationResult && (
            <span
              className={`validation-status ${validationResult.valid ? "valid" : "invalid"}`}
            >
              {validationResult.valid ? "✓ Valid" : "✗ Invalid"}
              {validationResult.message && ` - ${validationResult.message}`}
            </span>
          )}
        </div>

        <div className="transform-controls">
          <div className="limit-selector">
            <label>
              Preview limit:
              <select
                value={limit}
                onChange={(e) => setLimit(Number(e.target.value))}
              >
                <option value={5}>5</option>
                <option value={10}>10</option>
                <option value={25}>25</option>
                <option value={50}>50</option>
              </select>
            </label>
          </div>
          <div className="transform-actions">
            <button
              type="button"
              className="secondary"
              onClick={handlePreview}
              disabled={isPreviewing || !validationResult?.valid}
            >
              {isPreviewing ? "Previewing..." : "Preview"}
            </button>
            {onApply && (
              <button
                type="button"
                className="primary"
                onClick={handleApply}
                disabled={!validationResult?.valid}
              >
                Apply to Export
              </button>
            )}
          </div>
        </div>

        {error && <div className="transform-error">Error: {error}</div>}
      </div>

      {/* Examples Section */}
      <div className="transform-examples">
        <h5>Examples</h5>
        <div className="example-buttons">
          {language === "jmespath" ? (
            <>
              <button
                type="button"
                onClick={() => insertExample('[?status=="success"]')}
              >
                Filter success
              </button>
              <button
                type="button"
                onClick={() => insertExample("[].{url: url, title: title}")}
              >
                Project fields
              </button>
              <button type="button" onClick={() => insertExample("[][].links")}>
                Flatten links
              </button>
              <button
                type="button"
                onClick={() => insertExample("sort_by(@, \u0026timestamp)")}
              >
                Sort by timestamp
              </button>
              <button type="button" onClick={() => insertExample("[0:10]")}>
                First 10 items
              </button>
            </>
          ) : (
            <>
              <button
                type="button"
                onClick={() => insertExample('$[status="success"]')}
              >
                Filter success
              </button>
              <button
                type="button"
                onClick={() => insertExample('$.{"url": url, "title": title}')}
              >
                Project fields
              </button>
              <button
                type="button"
                onClick={() =>
                  insertExample('{"pages": $count($), "urls": $.url}')
                }
              >
                Aggregate
              </button>
              <button
                type="button"
                onClick={() => insertExample("$sum($.(price * quantity))")}
              >
                Sum calculation
              </button>
            </>
          )}
        </div>
      </div>

      {/* Preview Results Section */}
      {previewResult && (
        <div className="transform-results">
          <h5>
            Preview Results ({previewResult.resultCount} items
            {previewResult.resultCount >= limit ? " (limited)" : ""})
          </h5>
          {previewResult.error ? (
            <div className="preview-error">{previewResult.error}</div>
          ) : (
            <pre className="preview-output">
              {JSON.stringify(previewResult.results, null, 2)}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}

export default TransformPreview;
