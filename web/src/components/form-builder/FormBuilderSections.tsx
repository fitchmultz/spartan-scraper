/**
 * Purpose: Render the extracted presentational sections used by the form builder workflow.
 * Responsibilities: Present detected forms, editable field inputs, and fill results while keeping the root form builder focused on state and network calls.
 * Scope: Form builder presentation only; request execution and local state ownership stay in `FormBuilder.tsx`.
 * Usage: Import from `FormBuilder.tsx` to compose the form builder surface.
 * Invariants/Assumptions: Section props remain fully controlled by the parent and current labels/class names stay stable for existing styling.
 */

import type { DetectedForm, FormFillResponse } from "./formBuilderTypes";

export function DetectedFormsList({
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
            {form.allFields ? (
              <div className="form-fields-count">
                {form.allFields.length} field(s)
              </div>
            ) : null}
          </button>
        ))}
      </div>
    </div>
  );
}

export function FormFieldsSection({
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
                {field.required ? <span className="required">*</span> : null}
              </span>
              <span className="field-type">{field.fieldType}</span>
              {field.placeholder ? (
                <span className="field-placeholder">({field.placeholder})</span>
              ) : null}
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

        {submit ? (
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
        ) : null}
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

export function FormResultSection({
  result,
}: {
  result: FormFillResponse | null;
}) {
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
        {result.filledFields.length > 0 ? (
          <div className="filled-fields">
            <strong>Filled Fields:</strong>
            <ul>
              {result.filledFields.map((field) => (
                <li key={`filled-${field}`}>{field}</li>
              ))}
            </ul>
          </div>
        ) : null}
        {result.errors && result.errors.length > 0 ? (
          <div className="errors">
            <strong>Errors:</strong>
            <ul>
              {result.errors.map((error) => (
                <li key={`error-${error.substring(0, 30)}`}>{error}</li>
              ))}
            </ul>
          </div>
        ) : null}
        {result.pageUrl ? (
          <div className="page-url">
            <strong>Final URL:</strong> {result.pageUrl}
          </div>
        ) : null}
      </div>
    </div>
  );
}
