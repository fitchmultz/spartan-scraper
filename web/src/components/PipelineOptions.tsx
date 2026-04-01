/**
 * Purpose: Render the pipeline options UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

interface PipelineOptionsProps {
  extractTemplate: string;
  setExtractTemplate: (value: string) => void;
  extractValidate: boolean;
  setExtractValidate: (value: boolean) => void;
  preProcessors: string;
  setPreProcessors: (value: string) => void;
  postProcessors: string;
  setPostProcessors: (value: string) => void;
  transformers: string;
  setTransformers: (value: string) => void;
  incremental?: boolean;
  setIncremental?: (value: boolean) => void;
  inputPrefix: string;
}

export function PipelineOptions({
  extractTemplate,
  setExtractTemplate,
  extractValidate,
  setExtractValidate,
  preProcessors,
  setPreProcessors,
  postProcessors,
  setPostProcessors,
  transformers,
  setTransformers,
  incremental,
  setIncremental,
  inputPrefix,
}: PipelineOptionsProps) {
  return (
    <div data-tour="templates">
      <div className="row" style={{ marginTop: 12 }}>
        <label>
          Extract Template
          <input
            value={extractTemplate}
            onChange={(e) => setExtractTemplate(e.target.value)}
            placeholder="default, article, product..."
          />
        </label>
        <label>
          <input
            type="checkbox"
            checked={extractValidate}
            onChange={(e) => setExtractValidate(e.target.checked)}
          />{" "}
          Validate Schema
        </label>
      </div>
      <details>
        <summary
          style={{
            cursor: "pointer",
            marginBottom: "8px",
            color: "var(--accent)",
          }}
        >
          Pipeline Options
        </summary>
        <div
          style={{
            marginTop: "12px",
            padding: "12px",
            borderRadius: "12px",
            background: "rgba(0, 0, 0, 0.25)",
          }}
        >
          <label htmlFor={`${inputPrefix}-pre-processors`}>
            Pre-Processors (comma-separated)
          </label>
          <input
            id={`${inputPrefix}-pre-processors`}
            value={preProcessors}
            onChange={(event) => setPreProcessors(event.target.value)}
            placeholder="redact, sanitize"
          />
          <label
            htmlFor={`${inputPrefix}-post-processors`}
            style={{ marginTop: 12 }}
          >
            Post-Processors (comma-separated)
          </label>
          <input
            id={`${inputPrefix}-post-processors`}
            value={postProcessors}
            onChange={(event) => setPostProcessors(event.target.value)}
            placeholder="cleanup, normalize"
          />
          <label
            htmlFor={`${inputPrefix}-transformers`}
            style={{ marginTop: 12 }}
          >
            Transformers (comma-separated)
          </label>
          <input
            id={`${inputPrefix}-transformers`}
            value={transformers}
            onChange={(event) => setTransformers(event.target.value)}
            placeholder="json-clean, csv-export"
          />
          {incremental !== undefined && setIncremental && (
            <label style={{ marginTop: 12 }}>
              <input
                type="checkbox"
                checked={incremental}
                onChange={(event) => setIncremental(event.target.checked)}
              />{" "}
              Incremental (ETag/Hash tracking)
            </label>
          )}
        </div>
      </details>
    </div>
  );
}
