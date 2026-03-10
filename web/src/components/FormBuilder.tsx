/**
 * Form Builder Component
 *
 * Visual form interaction tool for detecting and filling forms on web pages.
 * Provides URL input, form type filtering, visual form preview, field mapping,
 * and form submission capabilities.
 *
 * @module FormBuilder
 */
import { useState } from "react";
import { client } from "../api/client.gen";
import {
  buildInitialFieldValues,
  buildSelectedFormFieldValues,
  formTypeOptions,
} from "./form-builder/formBuilderUtils";

// Types for form detection and filling
interface FieldMatch {
  selector: string;
  attribute: string;
  matchValue: string;
  confidence: number;
  matchReasons?: string[];
  fieldType: string;
  fieldName: string;
  required?: boolean;
  placeholder?: string;
}

interface DetectedForm {
  formIndex: number;
  formSelector: string;
  score: number;
  formType: string;
  userField?: FieldMatch;
  passField?: FieldMatch;
  submitField?: FieldMatch;
  allFields?: FieldMatch[];
  html?: string;
  action?: string;
  method?: string;
  name?: string;
  id?: string;
}

interface FormDetectRequest {
  url: string;
  formType?: string;
  headless?: boolean;
}

interface FormDetectResponse {
  url: string;
  forms: DetectedForm[];
  formCount: number;
  detectedTypes: string[];
}

interface FormFillRequest {
  url: string;
  formSelector?: string;
  fields: Record<string, string>;
  submit?: boolean;
  waitFor?: string;
  headless?: boolean;
  timeoutSeconds?: number;
  formTypeFilter?: string;
}

interface FormFillResponse {
  success: boolean;
  formSelector: string;
  formType?: string;
  filledFields: string[];
  errors?: string[];
  pageUrl?: string;
  pageHtml?: string;
  detectedForms?: DetectedForm[];
}

interface FormBuilderProps {
  onClose?: () => void;
}

function DetectedFormsList({
  forms,
  selectedFormIndex,
  onSelectForm,
}: {
  forms: DetectedForm[];
  selectedFormIndex: number | null;
  onSelectForm: (index: number) => void;
}) {
  if (forms.length === 0) {
    return null;
  }

  return (
    <div className="section">
      <h3>Detected Forms ({forms.length})</h3>
      <div className="forms-list">
        {forms.map((form, index) => (
          <button
            type="button"
            key={`form-${form.formIndex}-${form.formSelector}`}
            className={`form-card ${selectedFormIndex === index ? "selected" : ""}`}
            onClick={() => onSelectForm(index)}
            aria-pressed={selectedFormIndex === index}
          >
            <div className="form-card-header">
              <span className="form-type">{form.formType}</span>
              <span className="form-score">
                {(form.score * 100).toFixed(0)}% match
              </span>
            </div>
            <div className="form-selector">{form.formSelector}</div>
            {form.allFields && (
              <div className="form-fields-count">
                {form.allFields.length} field(s)
              </div>
            )}
          </button>
        ))}
      </div>
    </div>
  );
}

function FormFieldsSection({
  selectedForm,
  fieldValues,
  submit,
  waitFor,
  loading,
  onChangeField,
  onChangeSubmit,
  onChangeWaitFor,
  onFillForm,
}: {
  selectedForm: DetectedForm;
  fieldValues: Record<string, string>;
  submit: boolean;
  waitFor: string;
  loading: boolean;
  onChangeField: (fieldName: string, value: string) => void;
  onChangeSubmit: (value: boolean) => void;
  onChangeWaitFor: (value: string) => void;
  onFillForm: () => void;
}) {
  return (
    <div className="section">
      <h3>Form Fields</h3>
      <div className="fields-table">
        {selectedForm.allFields?.map((field) => (
          <div
            key={`${field.fieldName}-${field.selector}-${field.attribute}`}
            className="field-row"
          >
            <label
              className="field-info"
              htmlFor={`field-input-${field.fieldName}`}
            >
              <span className="field-name">
                {field.fieldName}
                {field.required && <span className="required">*</span>}
              </span>
              <span className="field-type">{field.fieldType}</span>
              {field.placeholder && (
                <span className="field-placeholder">({field.placeholder})</span>
              )}
            </label>
            <input
              id={`field-input-${field.fieldName}`}
              type="text"
              value={fieldValues[field.fieldName] || ""}
              onChange={(event) =>
                onChangeField(field.fieldName, event.target.value)
              }
              placeholder={`Enter ${field.fieldName}...`}
              className="field-input"
            />
          </div>
        ))}
      </div>

      <div className="submit-options">
        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={submit}
            onChange={(event) => onChangeSubmit(event.target.checked)}
          />
          Submit form after filling
        </label>

        {submit && (
          <div className="wait-for-input">
            <label htmlFor="wait-for-selector">
              Wait for selector (optional):
            </label>
            <input
              id="wait-for-selector"
              type="text"
              value={waitFor}
              onChange={(event) => onChangeWaitFor(event.target.value)}
              placeholder=".success-message"
            />
          </div>
        )}
      </div>

      <button
        type="button"
        onClick={onFillForm}
        disabled={loading}
        className="execute-button"
      >
        {loading ? "Processing..." : submit ? "Fill & Submit" : "Fill Form"}
      </button>
    </div>
  );
}

function FormResultSection({ result }: { result: FormFillResponse | null }) {
  if (!result) {
    return null;
  }

  return (
    <div
      className={`section result-section ${result.success ? "success" : "error"}`}
    >
      <h3>Result</h3>
      <div className="result-content">
        <div className="result-status">
          Status: {result.success ? "Success" : "Failed"}
        </div>
        {result.filledFields.length > 0 && (
          <div className="filled-fields">
            <strong>Filled Fields:</strong>
            <ul>
              {result.filledFields.map((field) => (
                <li key={`filled-${field}`}>{field}</li>
              ))}
            </ul>
          </div>
        )}
        {result.errors && result.errors.length > 0 && (
          <div className="errors">
            <strong>Errors:</strong>
            <ul>
              {result.errors.map((error) => (
                <li key={`error-${error.substring(0, 30)}`}>{error}</li>
              ))}
            </ul>
          </div>
        )}
        {result.pageUrl && (
          <div className="page-url">
            <strong>Final URL:</strong> {result.pageUrl}
          </div>
        )}
      </div>
    </div>
  );
}

export function FormBuilder({ onClose }: FormBuilderProps) {
  // State
  const [url, setUrl] = useState("");
  const [formTypeFilter, setFormTypeFilter] = useState("");
  const [detectedForms, setDetectedForms] = useState<DetectedForm[]>([]);
  const [selectedFormIndex, setSelectedFormIndex] = useState<number | null>(
    null,
  );
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({});
  const [submit, setSubmit] = useState(false);
  const [waitFor, setWaitFor] = useState("");
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<FormFillResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Detect forms on the page
  const detectForms = async () => {
    if (!url) {
      setError("Please enter a URL");
      return;
    }

    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const request: FormDetectRequest = {
        url,
        formType: formTypeFilter || undefined,
        headless: true,
      };

      const response = await client.post<FormDetectResponse>({
        url: "/v1/forms/detect",
        body: request,
      });

      if (response.data) {
        setDetectedForms(response.data.forms);
        setSelectedFormIndex(response.data.forms.length > 0 ? 0 : null);
        setFieldValues(buildInitialFieldValues(response.data.forms));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to detect forms");
    } finally {
      setLoading(false);
    }
  };

  // Fill and optionally submit the form
  const fillForm = async () => {
    if (selectedFormIndex === null || detectedForms.length === 0) {
      setError("Please select a form");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const selectedForm = detectedForms[selectedFormIndex];

      const request: FormFillRequest = {
        url,
        formSelector: selectedForm.formSelector,
        fields: buildSelectedFormFieldValues(selectedForm, fieldValues),
        submit,
        waitFor: waitFor || undefined,
        headless: true,
        timeoutSeconds: 30,
      };

      const response = await client.post<FormFillResponse>({
        url: "/v1/forms/fill",
        body: request,
      });

      if (response.data) {
        setResult(response.data);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fill form");
    } finally {
      setLoading(false);
    }
  };

  // Handle field value change
  const handleFieldChange = (fieldName: string, value: string) => {
    setFieldValues((prev) => ({
      ...prev,
      [fieldName]: value,
    }));
  };

  // Get selected form
  const selectedForm =
    selectedFormIndex !== null ? detectedForms[selectedFormIndex] : null;

  return (
    <div className="form-builder">
      <div className="form-builder-header">
        <h2>Form Builder</h2>
        {onClose && (
          <button
            type="button"
            className="close-button"
            onClick={onClose}
            aria-label="Close"
          >
            ×
          </button>
        )}
      </div>

      <div className="form-builder-content">
        {/* URL Input Section */}
        <div className="section">
          <h3>Page URL</h3>
          <div className="url-input-group">
            <input
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://example.com/contact"
              className="url-input"
            />
            <select
              value={formTypeFilter}
              onChange={(e) => setFormTypeFilter(e.target.value)}
              className="form-type-select"
            >
              {formTypeOptions.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
            <button
              type="button"
              onClick={detectForms}
              disabled={loading || !url}
              className="detect-button"
            >
              {loading ? "Detecting..." : "Detect Forms"}
            </button>
          </div>
        </div>

        {/* Error Display */}
        {error && (
          <div className="error-message">
            <strong>Error:</strong> {error}
          </div>
        )}

        {/* Detected Forms List */}
        <DetectedFormsList
          forms={detectedForms}
          selectedFormIndex={selectedFormIndex}
          onSelectForm={setSelectedFormIndex}
        />

        {/* Selected Form Fields */}
        {selectedForm && (
          <FormFieldsSection
            selectedForm={selectedForm}
            fieldValues={fieldValues}
            submit={submit}
            waitFor={waitFor}
            loading={loading}
            onChangeField={handleFieldChange}
            onChangeSubmit={setSubmit}
            onChangeWaitFor={setWaitFor}
            onFillForm={fillForm}
          />
        )}

        {/* Result Display */}
        <FormResultSection result={result} />
      </div>

      <style>{`
        .form-builder {
          background: var(--bg-primary, #1a1a2e);
          border-radius: 12px;
          padding: 24px;
          max-width: 800px;
          margin: 0 auto;
        }

        .form-builder-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 24px;
          padding-bottom: 16px;
          border-bottom: 1px solid var(--border-color, #2d2d44);
        }

        .form-builder-header h2 {
          margin: 0;
          color: var(--text-primary, #eaeaea);
        }

        .close-button {
          background: none;
          border: none;
          font-size: 24px;
          color: var(--text-secondary, #a0a0a0);
          cursor: pointer;
          padding: 0;
          width: 32px;
          height: 32px;
          display: flex;
          align-items: center;
          justify-content: center;
          border-radius: 4px;
        }

        .close-button:hover {
          background: var(--bg-hover, #2d2d44);
          color: var(--text-primary, #eaeaea);
        }

        .section {
          margin-bottom: 24px;
        }

        .section h3 {
          margin: 0 0 12px 0;
          color: var(--text-primary, #eaeaea);
          font-size: 16px;
        }

        .url-input-group {
          display: flex;
          gap: 8px;
          flex-wrap: wrap;
        }

        .url-input {
          flex: 1;
          min-width: 250px;
          padding: 10px 14px;
          border: 1px solid var(--border-color, #2d2d44);
          border-radius: 6px;
          background: var(--bg-secondary, #16213e);
          color: var(--text-primary, #eaeaea);
          font-size: 14px;
        }

        .form-type-select {
          padding: 10px 14px;
          border: 1px solid var(--border-color, #2d2d44);
          border-radius: 6px;
          background: var(--bg-secondary, #16213e);
          color: var(--text-primary, #eaeaea);
          font-size: 14px;
          min-width: 120px;
        }

        .detect-button,
        .execute-button {
          padding: 10px 20px;
          border: none;
          border-radius: 6px;
          background: var(--accent-primary, #0f3460);
          color: white;
          font-size: 14px;
          font-weight: 500;
          cursor: pointer;
          transition: background 0.2s;
        }

        .detect-button:hover:not(:disabled),
        .execute-button:hover:not(:disabled) {
          background: var(--accent-hover, #1a4a7a);
        }

        .detect-button:disabled,
        .execute-button:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        .error-message {
          padding: 12px 16px;
          background: rgba(231, 76, 60, 0.1);
          border: 1px solid rgba(231, 76, 60, 0.3);
          border-radius: 6px;
          color: #e74c3c;
          margin-bottom: 16px;
        }

        .forms-list {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }

        .form-card {
          padding: 12px 16px;
          background: var(--bg-secondary, #16213e);
          border: 2px solid transparent;
          border-radius: 8px;
          cursor: pointer;
          transition: all 0.2s;
        }

        .form-card:hover {
          border-color: var(--accent-primary, #0f3460);
        }

        .form-card.selected {
          border-color: var(--accent-primary, #0f3460);
          background: rgba(15, 52, 96, 0.2);
        }

        .form-card-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 4px;
        }

        .form-type {
          font-weight: 600;
          color: var(--accent-primary, #4a9eff);
          text-transform: capitalize;
        }

        .form-score {
          font-size: 12px;
          color: var(--text-secondary, #a0a0a0);
        }

        .form-selector {
          font-family: monospace;
          font-size: 12px;
          color: var(--text-secondary, #a0a0a0);
          margin-bottom: 4px;
        }

        .form-fields-count {
          font-size: 12px;
          color: var(--text-muted, #666);
        }

        .fields-table {
          display: flex;
          flex-direction: column;
          gap: 12px;
        }

        .field-row {
          display: flex;
          align-items: center;
          gap: 16px;
          padding: 12px;
          background: var(--bg-secondary, #16213e);
          border-radius: 6px;
        }

        .field-info {
          flex: 0 0 200px;
          display: flex;
          flex-direction: column;
          gap: 4px;
        }

        .field-name {
          font-weight: 500;
          color: var(--text-primary, #eaeaea);
        }

        .required {
          color: #e74c3c;
          margin-left: 4px;
        }

        .field-type {
          font-size: 12px;
          color: var(--accent-primary, #4a9eff);
          text-transform: capitalize;
        }

        .field-placeholder {
          font-size: 11px;
          color: var(--text-muted, #666);
        }

        .field-input {
          flex: 1;
          padding: 8px 12px;
          border: 1px solid var(--border-color, #2d2d44);
          border-radius: 4px;
          background: var(--bg-primary, #1a1a2e);
          color: var(--text-primary, #eaeaea);
          font-size: 14px;
        }

        .submit-options {
          margin-top: 16px;
          padding: 16px;
          background: var(--bg-secondary, #16213e);
          border-radius: 6px;
        }

        .checkbox-label {
          display: flex;
          align-items: center;
          gap: 8px;
          color: var(--text-primary, #eaeaea);
          cursor: pointer;
        }

        .wait-for-input {
          margin-top: 12px;
          display: flex;
          align-items: center;
          gap: 8px;
        }

        .wait-for-input label {
          color: var(--text-secondary, #a0a0a0);
          font-size: 14px;
        }

        .wait-for-input input {
          flex: 1;
          padding: 8px 12px;
          border: 1px solid var(--border-color, #2d2d44);
          border-radius: 4px;
          background: var(--bg-primary, #1a1a2e);
          color: var(--text-primary, #eaeaea);
          font-size: 14px;
        }

        .execute-button {
          margin-top: 16px;
          width: 100%;
          padding: 12px;
        }

        .result-section {
          padding: 16px;
          border-radius: 6px;
        }

        .result-section.success {
          background: rgba(46, 204, 113, 0.1);
          border: 1px solid rgba(46, 204, 113, 0.3);
        }

        .result-section.error {
          background: rgba(231, 76, 60, 0.1);
          border: 1px solid rgba(231, 76, 60, 0.3);
        }

        .result-content {
          color: var(--text-primary, #eaeaea);
        }

        .result-status {
          font-weight: 600;
          margin-bottom: 12px;
        }

        .result-section.success .result-status {
          color: #2ecc71;
        }

        .result-section.error .result-status {
          color: #e74c3c;
        }

        .filled-fields,
        .errors,
        .page-url {
          margin-top: 8px;
        }

        .filled-fields ul,
        .errors ul {
          margin: 4px 0;
          padding-left: 20px;
        }

        .filled-fields li,
        .errors li {
          margin: 2px 0;
        }

        .errors li {
          color: #e74c3c;
        }
      `}</style>
    </div>
  );
}
