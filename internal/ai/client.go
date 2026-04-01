// Package ai manages the bridge process used for pi-backed LLM operations.
//
// Purpose:
// - Define the bridge client contracts, request/response payloads, and shared client state.
//
// Responsibilities:
// - Hold the operation IDs, transport envelopes, bridge-facing payload types,
// - and the long-lived bridge client struct shared across split client modules.
//
// Scope:
// - Bridge client contracts and state only; request execution, process transport,
// - and health validation live in adjacent files in this package.
//
// Usage:
// - Used by AI extraction, authoring, and research flows that call the pi bridge.
//
// Invariants/Assumptions:
// - JSON payload shapes must stay stable across the Go client and pi bridge.
// - Client state remains guarded by the mutexes defined here.
package ai

import (
	"bufio"
	"encoding/json"
	"io"
	"os/exec"
	"sync"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

const (
	OperationHealth                = "health"
	OperationExtractPreview        = "extract_preview"
	OperationGenerateTemplate      = "generate_template"
	OperationGenerateRenderProfile = "generate_render_profile"
	OperationGeneratePipelineJS    = "generate_pipeline_js"
	OperationResearchRefine        = "research_refine"
	OperationExportShape           = "export_shape"
	OperationGenerateTransform     = "generate_transform"
)

type requestEnvelope struct {
	ID      string      `json:"id"`
	Op      string      `json:"op"`
	Payload interface{} `json:"payload,omitempty"`
}

type responseEnvelope struct {
	ID     string          `json:"id"`
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *bridgeError    `json:"error,omitempty"`
}

type bridgeError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type HealthRouteStatus struct {
	RouteID        string `json:"route_id"`
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model,omitempty"`
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
	ModelFound     bool   `json:"model_found"`
	AuthConfigured bool   `json:"auth_configured"`
}

type HealthResponse struct {
	Mode        string                         `json:"mode"`
	AgentDir    string                         `json:"agent_dir,omitempty"`
	Resolved    map[string][]string            `json:"resolved,omitempty"`
	Available   map[string][]string            `json:"available,omitempty"`
	RouteStatus map[string][]HealthRouteStatus `json:"route_status,omitempty"`
	LoadError   string                         `json:"load_error,omitempty"`
	AuthErrors  []string                       `json:"auth_errors,omitempty"`
}

type ImageInput struct {
	Data     string `json:"data"`
	MimeType string `json:"mime_type"`
}

type ExtractRequest struct {
	HTML            string                 `json:"html"`
	URL             string                 `json:"url"`
	Mode            string                 `json:"mode"`
	Prompt          string                 `json:"prompt,omitempty"`
	SchemaExample   map[string]interface{} `json:"schema_example,omitempty"`
	Fields          []string               `json:"fields,omitempty"`
	Images          []ImageInput           `json:"images,omitempty"`
	MaxContentChars int                    `json:"max_content_chars,omitempty"`
}

type ExtractResult struct {
	Fields      map[string]BridgeFieldValue `json:"fields"`
	Confidence  float64                     `json:"confidence"`
	Explanation string                      `json:"explanation,omitempty"`
	TokensUsed  int                         `json:"tokens_used,omitempty"`
	RouteID     string                      `json:"route_id,omitempty"`
	Provider    string                      `json:"provider,omitempty"`
	Model       string                      `json:"model,omitempty"`
}

type GenerateTemplateRequest struct {
	HTML         string       `json:"html"`
	URL          string       `json:"url"`
	Description  string       `json:"description"`
	SampleFields []string     `json:"sample_fields,omitempty"`
	Feedback     string       `json:"feedback,omitempty"`
	Images       []ImageInput `json:"images,omitempty"`
}

type GenerateTemplateResult struct {
	Template    BridgeTemplate `json:"template"`
	Explanation string         `json:"explanation,omitempty"`
	RouteID     string         `json:"route_id,omitempty"`
	Provider    string         `json:"provider,omitempty"`
	Model       string         `json:"model,omitempty"`
}

type GenerateRenderProfileRequest struct {
	HTML           string       `json:"html"`
	URL            string       `json:"url"`
	Instructions   string       `json:"instructions"`
	ContextSummary string       `json:"context_summary,omitempty"`
	Feedback       string       `json:"feedback,omitempty"`
	Images         []ImageInput `json:"images,omitempty"`
}

type BridgeRenderBlockPolicy struct {
	ResourceTypes []string `json:"resourceTypes,omitempty"`
	URLPatterns   []string `json:"urlPatterns,omitempty"`
}

type BridgeRenderWaitPolicy struct {
	Mode                string `json:"mode,omitempty"`
	Selector            string `json:"selector,omitempty"`
	NetworkIdleQuietMs  int    `json:"networkIdleQuietMs,omitempty"`
	MinTextLength       int    `json:"minTextLength,omitempty"`
	StabilityPollMs     int    `json:"stabilityPollMs,omitempty"`
	StabilityIterations int    `json:"stabilityIterations,omitempty"`
	ExtraSleepMs        int    `json:"extraSleepMs,omitempty"`
}

type BridgeRenderTimeoutPolicy struct {
	MaxRenderMs  int `json:"maxRenderMs,omitempty"`
	ScriptEvalMs int `json:"scriptEvalMs,omitempty"`
	NavigationMs int `json:"navigationMs,omitempty"`
}

type BridgeScreenshotConfig struct {
	Enabled  bool   `json:"enabled,omitempty"`
	FullPage bool   `json:"fullPage,omitempty"`
	Format   string `json:"format,omitempty"`
	Quality  int    `json:"quality,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

type BridgeRenderProfile struct {
	ForceEngine      string                    `json:"forceEngine,omitempty"`
	PreferHeadless   bool                      `json:"preferHeadless,omitempty"`
	AssumeJSHeavy    bool                      `json:"assumeJsHeavy,omitempty"`
	NeverHeadless    bool                      `json:"neverHeadless,omitempty"`
	JSHeavyThreshold float64                   `json:"jsHeavyThreshold,omitempty"`
	RateLimitQPS     int                       `json:"rateLimitQPS,omitempty"`
	RateLimitBurst   int                       `json:"rateLimitBurst,omitempty"`
	Block            BridgeRenderBlockPolicy   `json:"block,omitempty"`
	Wait             BridgeRenderWaitPolicy    `json:"wait,omitempty"`
	Timeouts         BridgeRenderTimeoutPolicy `json:"timeouts,omitempty"`
	Screenshot       BridgeScreenshotConfig    `json:"screenshot,omitempty"`
}

type GenerateRenderProfileResult struct {
	Profile     BridgeRenderProfile `json:"profile"`
	Explanation string              `json:"explanation,omitempty"`
	RouteID     string              `json:"route_id,omitempty"`
	Provider    string              `json:"provider,omitempty"`
	Model       string              `json:"model,omitempty"`
}

type GeneratePipelineJSRequest struct {
	HTML           string       `json:"html"`
	URL            string       `json:"url"`
	Instructions   string       `json:"instructions"`
	ContextSummary string       `json:"context_summary,omitempty"`
	Feedback       string       `json:"feedback,omitempty"`
	Images         []ImageInput `json:"images,omitempty"`
}

type BridgePipelineJSScript struct {
	Engine    string   `json:"engine,omitempty"`
	PreNav    string   `json:"preNav,omitempty"`
	PostNav   string   `json:"postNav,omitempty"`
	Selectors []string `json:"selectors,omitempty"`
}

type GeneratePipelineJSResult struct {
	Script      BridgePipelineJSScript `json:"script"`
	Explanation string                 `json:"explanation,omitempty"`
	RouteID     string                 `json:"route_id,omitempty"`
	Provider    string                 `json:"provider,omitempty"`
	Model       string                 `json:"model,omitempty"`
}

type BridgeResearchEvidence struct {
	URL         string                      `json:"url"`
	Title       string                      `json:"title,omitempty"`
	Snippet     string                      `json:"snippet,omitempty"`
	Score       float64                     `json:"score,omitempty"`
	SimHash     uint64                      `json:"simhash,omitempty"`
	ClusterID   string                      `json:"clusterId,omitempty"`
	Confidence  float64                     `json:"confidence,omitempty"`
	CitationURL string                      `json:"citationUrl,omitempty"`
	Fields      map[string]BridgeFieldValue `json:"fields,omitempty"`
}

type BridgeResearchEvidenceCluster struct {
	ID         string                   `json:"id"`
	Label      string                   `json:"label,omitempty"`
	Evidence   []BridgeResearchEvidence `json:"evidence,omitempty"`
	Confidence float64                  `json:"confidence,omitempty"`
}

type BridgeResearchCitation struct {
	URL       string `json:"url,omitempty"`
	Anchor    string `json:"anchor,omitempty"`
	Canonical string `json:"canonical,omitempty"`
}

type BridgeResearchAgenticRound struct {
	Round              int      `json:"round"`
	Goal               string   `json:"goal,omitempty"`
	FocusAreas         []string `json:"focusAreas,omitempty"`
	SelectedURLs       []string `json:"selectedUrls,omitempty"`
	AddedEvidenceCount int      `json:"addedEvidenceCount,omitempty"`
	Reasoning          string   `json:"reasoning,omitempty"`
}

type BridgeResearchAgenticResult struct {
	Status               string                       `json:"status"`
	Instructions         string                       `json:"instructions,omitempty"`
	Summary              string                       `json:"summary,omitempty"`
	Objective            string                       `json:"objective,omitempty"`
	FocusAreas           []string                     `json:"focusAreas,omitempty"`
	KeyFindings          []string                     `json:"keyFindings,omitempty"`
	OpenQuestions        []string                     `json:"openQuestions,omitempty"`
	RecommendedNextSteps []string                     `json:"recommendedNextSteps,omitempty"`
	FollowUpURLs         []string                     `json:"followUpUrls,omitempty"`
	Rounds               []BridgeResearchAgenticRound `json:"rounds,omitempty"`
	Confidence           float64                      `json:"confidence,omitempty"`
	RouteID              string                       `json:"route_id,omitempty"`
	Provider             string                       `json:"provider,omitempty"`
	Model                string                       `json:"model,omitempty"`
	Cached               bool                         `json:"cached,omitempty"`
	Error                string                       `json:"error,omitempty"`
}

type BridgeResearchResult struct {
	Query      string                          `json:"query,omitempty"`
	Summary    string                          `json:"summary,omitempty"`
	Confidence float64                         `json:"confidence,omitempty"`
	Evidence   []BridgeResearchEvidence        `json:"evidence,omitempty"`
	Clusters   []BridgeResearchEvidenceCluster `json:"clusters,omitempty"`
	Citations  []BridgeResearchCitation        `json:"citations,omitempty"`
	Agentic    *BridgeResearchAgenticResult    `json:"agentic,omitempty"`
}

type ResearchRefineRequest struct {
	Result       BridgeResearchResult `json:"result"`
	Instructions string               `json:"instructions,omitempty"`
	Feedback     string               `json:"feedback,omitempty"`
}

type ResearchEvidenceHighlight struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Finding     string `json:"finding"`
	Relevance   string `json:"relevance,omitempty"`
	CitationURL string `json:"citationUrl,omitempty"`
}

type ResearchRefinedContent struct {
	Summary              string                      `json:"summary"`
	ConciseSummary       string                      `json:"conciseSummary"`
	KeyFindings          []string                    `json:"keyFindings"`
	OpenQuestions        []string                    `json:"openQuestions,omitempty"`
	RecommendedNextSteps []string                    `json:"recommendedNextSteps,omitempty"`
	EvidenceHighlights   []ResearchEvidenceHighlight `json:"evidenceHighlights,omitempty"`
	Confidence           float64                     `json:"confidence,omitempty"`
}

type ResearchRefineResult struct {
	Refined     ResearchRefinedContent `json:"refined"`
	Explanation string                 `json:"explanation,omitempty"`
	RouteID     string                 `json:"route_id,omitempty"`
	Provider    string                 `json:"provider,omitempty"`
	Model       string                 `json:"model,omitempty"`
}

type ExportShapeFieldOption struct {
	Key          string   `json:"key"`
	Category     string   `json:"category,omitempty"`
	Label        string   `json:"label,omitempty"`
	SampleValues []string `json:"sampleValues,omitempty"`
}

type ExportFormattingHints struct {
	EmptyValue     string `json:"emptyValue,omitempty"`
	MultiValueJoin string `json:"multiValueJoin,omitempty"`
	MarkdownTitle  string `json:"markdownTitle,omitempty"`
}

type BridgeExportShapeConfig struct {
	TopLevelFields   []string              `json:"topLevelFields,omitempty"`
	NormalizedFields []string              `json:"normalizedFields,omitempty"`
	EvidenceFields   []string              `json:"evidenceFields,omitempty"`
	SummaryFields    []string              `json:"summaryFields,omitempty"`
	FieldLabels      map[string]string     `json:"fieldLabels,omitempty"`
	Formatting       ExportFormattingHints `json:"formatting,omitempty"`
}

type ExportShapeRequest struct {
	JobKind      string                   `json:"jobKind"`
	Format       string                   `json:"format"`
	FieldOptions []ExportShapeFieldOption `json:"fieldOptions,omitempty"`
	CurrentShape BridgeExportShapeConfig  `json:"currentShape,omitempty"`
	Instructions string                   `json:"instructions,omitempty"`
	Feedback     string                   `json:"feedback,omitempty"`
}

type ExportShapeResult struct {
	Shape       BridgeExportShapeConfig `json:"shape"`
	Explanation string                  `json:"explanation,omitempty"`
	RouteID     string                  `json:"route_id,omitempty"`
	Provider    string                  `json:"provider,omitempty"`
	Model       string                  `json:"model,omitempty"`
}

type BridgeTransformConfig struct {
	Expression string `json:"expression,omitempty"`
	Language   string `json:"language,omitempty"`
}

type TransformSampleField struct {
	Path         string   `json:"path"`
	SampleValues []string `json:"sampleValues,omitempty"`
}

type GenerateTransformRequest struct {
	JobKind           string                 `json:"jobKind,omitempty"`
	SampleRecords     []map[string]any       `json:"sampleRecords,omitempty"`
	SampleFields      []TransformSampleField `json:"sampleFields,omitempty"`
	CurrentTransform  BridgeTransformConfig  `json:"currentTransform,omitempty"`
	PreferredLanguage string                 `json:"preferredLanguage,omitempty"`
	Instructions      string                 `json:"instructions,omitempty"`
	Feedback          string                 `json:"feedback,omitempty"`
}

type GenerateTransformResult struct {
	Transform   BridgeTransformConfig `json:"transform"`
	Explanation string                `json:"explanation,omitempty"`
	RouteID     string                `json:"route_id,omitempty"`
	Provider    string                `json:"provider,omitempty"`
	Model       string                `json:"model,omitempty"`
}

type BridgeFieldValue struct {
	Values    []string `json:"values,omitempty"`
	Source    string   `json:"source"`
	RawObject string   `json:"rawObject,omitempty"`
}

type BridgeTemplate struct {
	Name      string               `json:"name"`
	Version   string               `json:"version,omitempty"`
	Selectors []BridgeSelectorRule `json:"selectors,omitempty"`
	JSONLD    []BridgeJSONLDRule   `json:"jsonld,omitempty"`
	Regex     []BridgeRegexRule    `json:"regex,omitempty"`
	Normalize BridgeNormalizeSpec  `json:"normalize,omitempty"`
}

type BridgeSelectorRule struct {
	Name     string `json:"name"`
	Selector string `json:"selector"`
	Attr     string `json:"attr,omitempty"`
	All      bool   `json:"all,omitempty"`
	Join     string `json:"join,omitempty"`
	Trim     bool   `json:"trim,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type BridgeJSONLDRule struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Path     string `json:"path,omitempty"`
	All      bool   `json:"all,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type BridgeRegexRule struct {
	Name     string `json:"name"`
	Pattern  string `json:"pattern"`
	Group    int    `json:"group,omitempty"`
	All      bool   `json:"all,omitempty"`
	Source   string `json:"source,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type BridgeNormalizeSpec struct {
	TitleField       string            `json:"titleField,omitempty"`
	DescriptionField string            `json:"descriptionField,omitempty"`
	TextField        string            `json:"textField,omitempty"`
	MetaFields       map[string]string `json:"metaFields,omitempty"`
}

// Client manages a long-lived pi bridge subprocess.
type Client struct {
	cfg    config.AIConfig
	mu     sync.Mutex
	reqMu  sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	nextID uint64
}

// BridgeHealthError preserves the bridge health snapshot that caused startup to fail.
type BridgeHealthError struct {
	Health HealthResponse
	Err    error
}
