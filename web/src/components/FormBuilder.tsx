/**
 * Purpose: Render the form builder UI surface for the web operator experience.
 * Responsibilities: Detect forms on a target page, manage field input state, call the form detect/fill APIs, and compose the extracted form-builder sections.
 * Scope: File-local form builder orchestration only; reusable types, sections, and styles stay in `./form-builder/*`.
 * Usage: Import from the surrounding feature components that expose the form builder workflow.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import { useState } from "react";

import { client } from "../api/client.gen";
import {
  buildInitialFieldValues,
  buildSelectedFormFieldValues,
  formTypeOptions,
} from "./form-builder/formBuilderUtils";
import {
  DetectedFormsList,
  FormFieldsSection,
  FormResultSection,
} from "./form-builder/FormBuilderSections";
import { FORM_BUILDER_STYLES } from "./form-builder/formBuilderStyles";
import type {
  DetectedForm,
  FormBuilderProps,
  FormDetectRequest,
  FormDetectResponse,
  FormFillRequest,
  FormFillResponse,
} from "./form-builder/formBuilderTypes";

export function FormBuilder({ onClose }: FormBuilderProps) {
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

  const handleFieldChange = (fieldName: string, value: string) => {
    setFieldValues((previous) => ({
      ...previous,
      [fieldName]: value,
    }));
  };

  const selectedForm =
    selectedFormIndex !== null ? detectedForms[selectedFormIndex] : null;

  return (
    <div className="form-builder">
      <div className="form-builder-header">
        <h2>Form Builder</h2>
        {onClose ? (
          <button
            type="button"
            className="close-button"
            onClick={onClose}
            aria-label="Close"
          >
            ×
          </button>
        ) : null}
      </div>

      <div className="form-builder-content">
        <div className="section">
          <h3>Page URL</h3>
          <div className="url-input-group">
            <input
              type="url"
              value={url}
              onChange={(event) => setUrl(event.target.value)}
              placeholder="https://example.com/contact"
              className="url-input"
            />
            <select
              value={formTypeFilter}
              onChange={(event) => setFormTypeFilter(event.target.value)}
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

        {error ? (
          <div className="error-message">
            <strong>Error:</strong> {error}
          </div>
        ) : null}

        <DetectedFormsList
          forms={detectedForms}
          selectedFormIndex={selectedFormIndex}
          onSelectForm={setSelectedFormIndex}
        />

        {selectedForm ? (
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
        ) : null}

        <FormResultSection result={result} />
      </div>

      <style>{FORM_BUILDER_STYLES}</style>
    </div>
  );
}
