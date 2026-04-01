/**
 * Purpose: Render the form builder utils UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
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
