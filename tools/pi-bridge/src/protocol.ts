/**
 * Purpose: Define the bridge protocol shared between Spartan and the pi-bridge process.
 * Responsibilities: Declare operation names, request/response envelopes, and bounded payload/result shapes.
 * Scope: Runtime bridge protocol types only.
 * Usage: Imported by bridge backends, the process entrypoint, and tests.
 * Invariants/Assumptions: Capability IDs stay stable, payloads mirror Spartan's source-of-truth contracts, and optional guidance fields remain optional across the bridge.
 */
export const CAPABILITY_EXTRACT_NATURAL = "extract.natural_language";
export const CAPABILITY_EXTRACT_SCHEMA = "extract.schema_guided";
export const CAPABILITY_TEMPLATE_GENERATE = "template.generate";
export const CAPABILITY_RENDER_PROFILE_GENERATE = "render_profile.generate";
export const CAPABILITY_PIPELINE_JS_GENERATE = "pipeline_js.generate";
export const CAPABILITY_RESEARCH_REFINE = "research.refine";
export const CAPABILITY_EXPORT_SHAPE = "export.shape";
export const CAPABILITY_TRANSFORM_GENERATE = "transform.generate";

export const OP_HEALTH = "health";
export const OP_EXTRACT_PREVIEW = "extract_preview";
export const OP_GENERATE_TEMPLATE = "generate_template";
export const OP_GENERATE_RENDER_PROFILE = "generate_render_profile";
export const OP_GENERATE_PIPELINE_JS = "generate_pipeline_js";
export const OP_RESEARCH_REFINE = "research_refine";
export const OP_EXPORT_SHAPE = "export_shape";
export const OP_GENERATE_TRANSFORM = "generate_transform";

export type BridgeOperation =
  | typeof OP_HEALTH
  | typeof OP_EXTRACT_PREVIEW
  | typeof OP_GENERATE_TEMPLATE
  | typeof OP_GENERATE_RENDER_PROFILE
  | typeof OP_GENERATE_PIPELINE_JS
  | typeof OP_RESEARCH_REFINE
  | typeof OP_EXPORT_SHAPE
  | typeof OP_GENERATE_TRANSFORM;

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

export interface ImageInput {
  data: string;
  mime_type: string;
}

export interface ExtractPayload {
  html: string;
  url: string;
  mode: "natural_language" | "schema_guided" | string;
  prompt?: string;
  schema_example?: Record<string, unknown>;
  fields?: string[];
  images?: ImageInput[];
  max_content_chars?: number;
}

export interface BridgeFieldValue {
  values: string[];
  source: string;
  rawObject?: string;
}

export interface ExtractResult {
  fields: Record<string, BridgeFieldValue>;
  confidence: number;
  explanation?: string;
  tokens_used?: number;
  route_id?: string;
  provider?: string;
  model?: string;
}

export interface GenerateTemplatePayload {
  html: string;
  url: string;
  description: string;
  sample_fields?: string[];
  feedback?: string;
  images?: ImageInput[];
}

export interface GenerateRenderProfilePayload {
  html: string;
  url: string;
  instructions?: string;
  context_summary?: string;
  feedback?: string;
  images?: ImageInput[];
}

export interface GeneratePipelineJsPayload {
  html: string;
  url: string;
  instructions?: string;
  context_summary?: string;
  feedback?: string;
  images?: ImageInput[];
}

export interface ResearchEvidence {
  url: string;
  title?: string;
  snippet?: string;
  score?: number;
  simhash?: number;
  clusterId?: string;
  confidence?: number;
  citationUrl?: string;
  fields?: Record<string, BridgeFieldValue>;
}

export interface ResearchEvidenceCluster {
  id: string;
  label?: string;
  confidence?: number;
  evidence?: ResearchEvidence[];
}

export interface ResearchCitation {
  url?: string;
  anchor?: string;
  canonical?: string;
}

export interface ResearchAgenticRound {
  round: number;
  goal?: string;
  focusAreas?: string[];
  selectedUrls?: string[];
  addedEvidenceCount?: number;
  reasoning?: string;
}

export interface ResearchAgenticResult {
  status: string;
  instructions?: string;
  summary?: string;
  objective?: string;
  focusAreas?: string[];
  keyFindings?: string[];
  openQuestions?: string[];
  recommendedNextSteps?: string[];
  followUpUrls?: string[];
  rounds?: ResearchAgenticRound[];
  confidence?: number;
  route_id?: string;
  provider?: string;
  model?: string;
  cached?: boolean;
  error?: string;
}

export interface ResearchResultInput {
  query?: string;
  summary?: string;
  confidence?: number;
  evidence?: ResearchEvidence[];
  clusters?: ResearchEvidenceCluster[];
  citations?: ResearchCitation[];
  agentic?: ResearchAgenticResult;
}

export interface ResearchRefinePayload {
  result: ResearchResultInput;
  instructions?: string;
  feedback?: string;
}

export interface ResearchEvidenceHighlight {
  url: string;
  title?: string;
  finding: string;
  relevance?: string;
  citationUrl?: string;
}

export interface ResearchRefinedContent {
  summary: string;
  conciseSummary: string;
  keyFindings: string[];
  openQuestions?: string[];
  recommendedNextSteps?: string[];
  evidenceHighlights?: ResearchEvidenceHighlight[];
  confidence?: number;
}

export interface ResearchRefineResult {
  refined: ResearchRefinedContent;
  explanation?: string;
  route_id?: string;
  provider?: string;
  model?: string;
}

export interface ExportShapeFieldOption {
  key: string;
  category?: string;
  label?: string;
  sampleValues?: string[];
}

export interface ExportFormattingHints {
  emptyValue?: string;
  multiValueJoin?: string;
  markdownTitle?: string;
}

export interface ExportShapeConfig {
  topLevelFields?: string[];
  normalizedFields?: string[];
  evidenceFields?: string[];
  summaryFields?: string[];
  fieldLabels?: Record<string, string>;
  formatting?: ExportFormattingHints;
}

export interface ExportShapePayload {
  jobKind: string;
  format: string;
  fieldOptions?: ExportShapeFieldOption[];
  currentShape?: ExportShapeConfig;
  instructions?: string;
  feedback?: string;
}

export interface ExportShapeResult {
  shape: ExportShapeConfig;
  explanation?: string;
  route_id?: string;
  provider?: string;
  model?: string;
}

export interface TransformConfig {
  expression?: string;
  language?: "jmespath" | "jsonata" | string;
}

export interface TransformSampleField {
  path: string;
  sampleValues?: string[];
}

export interface GenerateTransformPayload {
  jobKind?: string;
  sampleRecords?: Record<string, unknown>[];
  sampleFields?: TransformSampleField[];
  currentTransform?: TransformConfig;
  preferredLanguage?: "jmespath" | "jsonata" | string;
  instructions?: string;
  feedback?: string;
}

export interface TransformResult {
  transform: TransformConfig;
  explanation?: string;
  route_id?: string;
  provider?: string;
  model?: string;
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
  route_id?: string;
  provider?: string;
  model?: string;
}

export interface RenderProfileResult {
  profile: Record<string, unknown>;
  explanation?: string;
  route_id?: string;
  provider?: string;
  model?: string;
}

export interface PipelineJsResult {
  script: Record<string, unknown>;
  explanation?: string;
  route_id?: string;
  provider?: string;
  model?: string;
}

export interface HealthRouteStatus {
  route_id: string;
  provider?: string;
  model?: string;
  status: "ready" | "missing_auth" | "missing_model" | "invalid_route";
  message?: string;
  model_found: boolean;
  auth_configured: boolean;
}

export interface HealthResult {
  mode: string;
  agent_dir?: string;
  resolved?: Record<string, string[]>;
  available?: Record<string, string[]>;
  route_status?: Record<string, HealthRouteStatus[]>;
  load_error?: string;
  auth_errors?: string[];
}
