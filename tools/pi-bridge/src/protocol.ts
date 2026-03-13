export const CAPABILITY_EXTRACT_NATURAL = "extract.natural_language";
export const CAPABILITY_EXTRACT_SCHEMA = "extract.schema_guided";
export const CAPABILITY_TEMPLATE_GENERATE = "template.generate";

export const OP_HEALTH = "health";
export const OP_EXTRACT_PREVIEW = "extract_preview";
export const OP_GENERATE_TEMPLATE = "generate_template";

export type BridgeOperation =
  | typeof OP_HEALTH
  | typeof OP_EXTRACT_PREVIEW
  | typeof OP_GENERATE_TEMPLATE;

export interface BridgeRequest<TPayload = unknown> {
  id: string;
  op: BridgeOperation;
  payload?: TPayload;
}

export interface BridgeError {
  code?: string;
  message: string;
}

export interface BridgeResponse<TResult = unknown> {
  id: string;
  ok: boolean;
  result?: TResult;
  error?: BridgeError;
}

export interface ExtractPayload {
  html: string;
  url: string;
  mode: "natural_language" | "schema_guided" | string;
  prompt?: string;
  schema_example?: Record<string, unknown>;
  fields?: string[];
  max_content_chars?: number;
}

export interface BridgeFieldValue {
  values: string[];
  source: "llm";
  rawObject?: string;
}

export interface ExtractResult {
  fields: Record<string, BridgeFieldValue>;
  confidence: number;
  explanation?: string;
  tokens_used?: number;
  provider?: string;
  model?: string;
}

export interface GenerateTemplatePayload {
  html: string;
  url: string;
  description: string;
  sample_fields?: string[];
  feedback?: string;
}

export interface SelectorRule {
  name: string;
  selector: string;
  attr?: string;
  all?: boolean;
  join?: string;
  trim?: boolean;
  required?: boolean;
}

export interface JsonLdRule {
  name: string;
  type?: string;
  path?: string;
  all?: boolean;
  required?: boolean;
}

export interface RegexRule {
  name: string;
  pattern: string;
  group?: number;
  all?: boolean;
  source?: string;
  required?: boolean;
}

export interface NormalizeSpec {
  titleField?: string;
  descriptionField?: string;
  textField?: string;
  metaFields?: Record<string, string>;
}

export interface TemplateResult {
  template: {
    name: string;
    version?: string;
    selectors?: SelectorRule[];
    jsonld?: JsonLdRule[];
    regex?: RegexRule[];
    normalize?: NormalizeSpec;
  };
  explanation?: string;
  provider?: string;
  model?: string;
}

export interface HealthResult {
  mode: string;
  agent_dir?: string;
  resolved?: Record<string, string[]>;
  available?: Record<string, string[]>;
  load_error?: string;
}
