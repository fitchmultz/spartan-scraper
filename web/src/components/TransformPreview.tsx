/**
 * Transform Preview Component
 *
 * Provides a UI for testing JMESPath and JSONata transformations on job results.
 * Allows users to input expressions, preview the transformed output, apply
 * transformations to exports, and generate bounded transform suggestions from
 * saved job results.
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

interface AITransformGenerateResponse {
  issues?: string[];
  inputStats?: {
    sampleRecordCount: number;
    fieldPathCount: number;
    currentTransformProvided: boolean;
  };
  transform: {
    expression: string;
    language: "jmespath" | "jsonata";
  };
  preview?: unknown[];
  explanation?: string;
  route_id?: string;
  provider?: string;
  model?: string;
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

async function generateTransformWithAI(
  jobId: string,
  language: "jmespath" | "jsonata",
  instructions: string,
  currentExpression: string,
): Promise<AITransformGenerateResponse> {
  const response = await fetch("/v1/ai/transform-generate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      job_id: jobId,
      preferredLanguage: language,
      currentTransform: currentExpression.trim()
        ? {
            expression: currentExpression.trim(),
            language,
          }
        : undefined,
      instructions: instructions.trim() || undefined,
    }),
  });

  const data = (await response.json()) as
    | AITransformGenerateResponse
    | { error?: string; message?: string };
  if (!response.ok) {
    const errorData = data as { error?: string; message?: string };
    throw new Error(
      errorData.error || errorData.message || "AI transform generation failed",
    );
  }

  return data as AITransformGenerateResponse;
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
  const [showAIAssistant, setShowAIAssistant] = useState(false);
  const [aiInstructions, setAIInstructions] = useState("");
  const [isGeneratingAI, setIsGeneratingAI] = useState(false);
  const [aiResult, setAIResult] = useState<AITransformGenerateResponse | null>(
    null,
  );
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

  const handleGenerateAI = useCallback(async () => {
    setIsGeneratingAI(true);
    setError(null);
    setAIResult(null);

    try {
      const result = await generateTransformWithAI(
        jobId,
        language,
        aiInstructions,
        expression,
      );
      setAIResult(result);
      setExpression(result.transform.expression);
      setLanguage(result.transform.language);
      setValidationResult({
        valid: true,
        message:
          "AI-generated transform validated against representative results.",
      });
      setPreviewResult({
        results: result.preview || [],
        resultCount: result.preview?.length || 0,
      });
    } catch (err) {
      setError(String(err));
    } finally {
      setIsGeneratingAI(false);
    }
  }, [aiInstructions, expression, jobId, language]);

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
              onClick={() => setShowAIAssistant((value) => !value)}
              disabled={isGeneratingAI}
            >
              {showAIAssistant
                ? "Hide AI"
                : expression.trim()
                  ? "Revise with AI"
                  : "Generate with AI"}
            </button>
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

        {showAIAssistant && (
          <div className="panel" style={{ marginTop: 12 }}>
            <div style={{ fontWeight: 600, marginBottom: 8 }}>
              Generate bounded transform with AI
            </div>
            <textarea
              value={aiInstructions}
              onChange={(event) => setAIInstructions(event.target.value)}
              rows={3}
              placeholder="Project the URL, title, and pricing fields for a lightweight export."
              className="form-textarea"
              aria-label="AI transform instructions"
              disabled={isGeneratingAI}
            />
            <p className="form-help" style={{ marginTop: 8 }}>
              Spartan sends only representative saved result records. The AI
              does not fetch new data or browse.
            </p>
            <div className="row" style={{ gap: 8, marginTop: 8 }}>
              <button
                type="button"
                onClick={() => void handleGenerateAI()}
                disabled={isGeneratingAI}
              >
                {isGeneratingAI ? "Generating..." : "Generate Transform"}
              </button>
            </div>

            {aiResult ? (
              <div style={{ marginTop: 12 }}>
                <div className="flex gap-2" style={{ flexWrap: "wrap" }}>
                  <div className="badge running">
                    {aiResult.transform.language}
                  </div>
                  {aiResult.route_id ? (
                    <div className="badge running">{aiResult.route_id}</div>
                  ) : null}
                  {aiResult.provider && aiResult.model ? (
                    <div className="badge running">
                      {aiResult.provider}/{aiResult.model}
                    </div>
                  ) : null}
                </div>
                {aiResult.issues?.length ? (
                  <div style={{ marginTop: 12 }}>
                    <div style={{ fontWeight: 600 }}>Input diagnostics</div>
                    <ul>
                      {aiResult.issues.map((issue) => (
                        <li key={issue}>{issue}</li>
                      ))}
                    </ul>
                  </div>
                ) : null}
                {aiResult.inputStats ? (
                  <div style={{ marginTop: 12 }}>
                    <div style={{ fontWeight: 600 }}>Input stats</div>
                    <div className="job-list">
                      <div className="job-item">
                        Sample records {aiResult.inputStats.sampleRecordCount}
                      </div>
                      <div className="job-item">
                        Field paths {aiResult.inputStats.fieldPathCount}
                      </div>
                      <div className="job-item">
                        Current transform{" "}
                        {aiResult.inputStats.currentTransformProvided
                          ? "provided"
                          : "not provided"}
                      </div>
                    </div>
                  </div>
                ) : null}
                <div style={{ marginTop: 12 }}>
                  <div style={{ fontWeight: 600 }}>Suggested expression</div>
                  <pre className="preview-output">
                    {aiResult.transform.expression}
                  </pre>
                </div>
                {aiResult.explanation ? (
                  <div style={{ marginTop: 12 }}>
                    <div style={{ fontWeight: 600 }}>Model explanation</div>
                    <p>{aiResult.explanation}</p>
                  </div>
                ) : null}
              </div>
            ) : null}
          </div>
        )}

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
