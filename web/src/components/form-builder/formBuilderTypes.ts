/**
 * Purpose: Define the local request, response, and view-model contracts used by the form builder workflow.
 * Responsibilities: Export typed shapes for detected forms, fill requests, fill results, and the root form builder component props.
 * Scope: Form builder type contracts only; rendering and network behavior stay in adjacent modules.
 * Usage: Import from `FormBuilder.tsx` and related extracted form-builder section modules.
 * Invariants/Assumptions: These types mirror the current form detect/fill API contract and the local UI state that depends on it.
 */

export interface FieldMatch {
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

export interface DetectedForm {
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

export interface FormDetectRequest {
  url: string;
  formType?: string;
  headless?: boolean;
}

export interface FormDetectResponse {
  url: string;
  forms: DetectedForm[];
  formCount: number;
  detectedTypes: string[];
}

export interface FormFillRequest {
  url: string;
  formSelector?: string;
  fields: Record<string, string>;
  submit?: boolean;
  waitFor?: string;
  headless?: boolean;
  timeoutSeconds?: number;
  formTypeFilter?: string;
}

export interface FormFillResponse {
  success: boolean;
  formSelector: string;
  formType?: string;
  filledFields: string[];
  errors?: string[];
  pageUrl?: string;
  pageHtml?: string;
  detectedForms?: DetectedForm[];
}

export interface FormBuilderProps {
  onClose?: () => void;
}
