/**
 * formBuilderUtils
 *
 * Purpose:
 * - Hold pure helpers and shared option data for the form builder UI.
 *
 * Responsibilities:
 * - Define stable form-type filter options.
 * - Build editable field-value maps from detected forms.
 * - Limit submitted field values to the selected form.
 *
 * Scope:
 * - Pure utility logic only; no React state or API calls.
 *
 * Usage:
 * - Used by FormBuilder and its focused unit tests.
 *
 * Invariants/Assumptions:
 * - Field names are stable identifiers within a detected form.
 * - Empty field values are omitted from submission payloads.
 * - Form-type options stay in a predictable order for the UI.
 */

export interface FieldMatchLike {
  fieldName: string;
}

export interface DetectedFormLike {
  allFields?: FieldMatchLike[];
}

export const formTypeOptions = [
  { value: "", label: "All Types" },
  { value: "login", label: "Login" },
  { value: "register", label: "Register" },
  { value: "search", label: "Search" },
  { value: "contact", label: "Contact" },
  { value: "newsletter", label: "Newsletter" },
  { value: "checkout", label: "Checkout" },
  { value: "survey", label: "Survey" },
] as const;

export function buildInitialFieldValues(
  forms: DetectedFormLike[],
): Record<string, string> {
  const initialValues: Record<string, string> = {};
  forms.forEach((form) => {
    form.allFields?.forEach((field) => {
      initialValues[field.fieldName] = "";
    });
  });
  return initialValues;
}

export function buildSelectedFormFieldValues(
  form: DetectedFormLike | undefined,
  fieldValues: Record<string, string>,
): Record<string, string> {
  const selectedValues: Record<string, string> = {};
  form?.allFields?.forEach((field) => {
    const value = fieldValues[field.fieldName];
    if (value) {
      selectedValues[field.fieldName] = value;
    }
  });
  return selectedValues;
}
