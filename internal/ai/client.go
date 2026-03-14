// Package ai manages the bridge process used for pi-backed LLM operations.
package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

const (
	OperationHealth                = "health"
	OperationExtractPreview        = "extract_preview"
	OperationGenerateTemplate      = "generate_template"
	OperationGenerateRenderProfile = "generate_render_profile"
	OperationGeneratePipelineJS    = "generate_pipeline_js"
	OperationResearchRefine        = "research_refine"
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

func NewClient(cfg config.AIConfig) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) HealthCheck(ctx context.Context) error {
	var resp HealthResponse
	return c.call(ctx, OperationHealth, nil, &resp)
}

func (c *Client) Extract(ctx context.Context, req ExtractRequest) (ExtractResult, error) {
	var resp ExtractResult
	err := c.call(ctx, OperationExtractPreview, req, &resp)
	if err != nil {
		return ExtractResult{}, err
	}
	if err := resp.Canonicalize(); err != nil {
		return ExtractResult{}, fmt.Errorf("validate bridge extract result: %w", err)
	}
	return resp, nil
}

func (c *Client) GenerateTemplate(ctx context.Context, req GenerateTemplateRequest) (GenerateTemplateResult, error) {
	var resp GenerateTemplateResult
	err := c.call(ctx, OperationGenerateTemplate, req, &resp)
	return resp, err
}

func (c *Client) GenerateRenderProfile(ctx context.Context, req GenerateRenderProfileRequest) (GenerateRenderProfileResult, error) {
	var resp GenerateRenderProfileResult
	err := c.call(ctx, OperationGenerateRenderProfile, req, &resp)
	return resp, err
}

func (c *Client) GeneratePipelineJS(ctx context.Context, req GeneratePipelineJSRequest) (GeneratePipelineJSResult, error) {
	var resp GeneratePipelineJSResult
	err := c.call(ctx, OperationGeneratePipelineJS, req, &resp)
	return resp, err
}

func (c *Client) GenerateResearchRefinement(ctx context.Context, req ResearchRefineRequest) (ResearchRefineResult, error) {
	var resp ResearchRefineResult
	err := c.call(ctx, OperationResearchRefine, req, &resp)
	return resp, err
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopLocked()
}

func (c *Client) call(ctx context.Context, op string, payload interface{}, target interface{}) error {
	ctx, cancel := withConfiguredTimeout(ctx, time.Duration(c.cfg.RequestTimeoutSecs)*time.Second)
	defer cancel()

	if err := c.ensureStarted(ctx); err != nil {
		return err
	}

	c.reqMu.Lock()
	resp, err := c.sendRequest(ctx, op, payload)
	c.reqMu.Unlock()
	if err != nil {
		c.resetProcess()
		return err
	}
	if !resp.OK {
		c.resetProcessOnFatal(resp.Error)
		if resp.Error == nil {
			return fmt.Errorf("bridge request %s failed", op)
		}
		if resp.Error.Code != "" {
			return fmt.Errorf("bridge %s: %s", resp.Error.Code, resp.Error.Message)
		}
		return fmt.Errorf("bridge error: %s", resp.Error.Message)
	}
	if target == nil || len(resp.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Result, target); err != nil {
		return fmt.Errorf("decode bridge result: %w", err)
	}
	return nil
}

func (c *Client) ensureStarted(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd != nil && c.cmd.Process != nil {
		return nil
	}

	scriptPath, err := c.resolveBridgeScriptPath()
	if err != nil {
		return err
	}

	cmd := exec.Command(c.cfg.NodeBin, scriptPath)
	cmd.Env = append(os.Environ(), c.bridgeEnv()...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open bridge stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open bridge stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("open bridge stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start bridge process: %w", err)
	}

	go streamBridgeStderr(stderr)

	c.cmd = cmd
	c.stdin = stdin
	c.stdout = bufio.NewReader(stdout)

	startupCtx, cancel := withConfiguredTimeout(ctx, time.Duration(c.cfg.StartupTimeoutSecs)*time.Second)
	defer cancel()

	c.reqMu.Lock()
	defer c.reqMu.Unlock()

	resp, err := c.sendRequest(startupCtx, OperationHealth, nil)
	if err != nil {
		_ = c.stopLocked()
		return fmt.Errorf("wait for bridge health: %w", err)
	}
	if !resp.OK {
		_ = c.stopLocked()
		if resp.Error == nil {
			return fmt.Errorf("bridge health check failed")
		}
		if resp.Error.Code != "" {
			return fmt.Errorf("bridge %s: %s", resp.Error.Code, resp.Error.Message)
		}
		return fmt.Errorf("bridge error: %s", resp.Error.Message)
	}

	var health HealthResponse
	if len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, &health); err != nil {
			_ = c.stopLocked()
			return fmt.Errorf("decode bridge health: %w", err)
		}
	}
	logBridgeHealth(health)
	if err := validateBridgeHealth(health); err != nil {
		_ = c.stopLocked()
		return err
	}
	return nil
}

func (c *Client) bridgeEnv() []string {
	env := []string{
		"PI_MODE=" + c.cfg.Mode,
	}
	if c.cfg.ConfigPath != "" {
		env = append(env, "PI_CONFIG_PATH="+c.cfg.ConfigPath)
	}
	return env
}

func (c *Client) resetProcess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.stopLocked()
}

func (c *Client) resetProcessOnFatal(err *bridgeError) {
	if err == nil {
		return
	}
	if strings.EqualFold(err.Code, "bad_request") {
		return
	}
	c.resetProcess()
}

func (c *Client) stopLocked() error {
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	var waitErr error
	if c.cmd != nil {
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		waitErr = c.cmd.Wait()
	}
	c.cmd = nil
	c.stdin = nil
	c.stdout = nil
	if waitErr != nil && strings.Contains(waitErr.Error(), "signal: killed") {
		return nil
	}
	return waitErr
}

func streamBridgeStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		slog.Debug("pi bridge", "stderr", scanner.Text())
	}
}

func (c *Client) sendRequest(ctx context.Context, op string, payload interface{}) (responseEnvelope, error) {
	req := requestEnvelope{
		ID:      fmt.Sprintf("req-%d", atomic.AddUint64(&c.nextID, 1)),
		Op:      op,
		Payload: payload,
	}

	line, err := json.Marshal(req)
	if err != nil {
		return responseEnvelope{}, fmt.Errorf("marshal bridge request: %w", err)
	}
	line = append(line, '\n')

	if _, err := c.stdin.Write(line); err != nil {
		return responseEnvelope{}, fmt.Errorf("write bridge request: %w", err)
	}

	respCh := make(chan responseEnvelope, 1)
	errCh := make(chan error, 1)
	go func() {
		raw, err := c.stdout.ReadBytes('\n')
		if err != nil {
			errCh <- fmt.Errorf("read bridge response: %w", err)
			return
		}
		var resp responseEnvelope
		if err := json.Unmarshal(raw, &resp); err != nil {
			errCh <- fmt.Errorf("decode bridge response: %w", err)
			return
		}
		respCh <- resp
	}()

	select {
	case <-ctx.Done():
		return responseEnvelope{}, ctx.Err()
	case err := <-errCh:
		return responseEnvelope{}, err
	case resp := <-respCh:
		return resp, nil
	}
}

func (c *Client) resolveBridgeScriptPath() (string, error) {
	wd, _ := os.Getwd()
	executablePath, _ := os.Executable()
	return resolveBridgeScriptPath(
		c.cfg.BridgeScript,
		bridgeScriptSearchRoots(wd, executablePath, c.cfg.ConfigPath),
	)
}

func bridgeScriptSearchRoots(workingDir string, executablePath string, configPath string) []string {
	roots := make([]string, 0, 4)
	if configPath != "" {
		roots = append(roots, filepath.Dir(configPath))
	}
	if workingDir != "" {
		roots = append(roots, workingDir)
	}
	if executablePath != "" {
		executableDir := filepath.Dir(executablePath)
		roots = append(roots, executableDir, filepath.Join(executableDir, ".."))
	}
	return roots
}

func resolveBridgeScriptPath(scriptPath string, searchRoots []string) (string, error) {
	if filepath.IsAbs(scriptPath) {
		return scriptPath, nil
	}

	seen := make(map[string]struct{}, len(searchRoots))
	for _, root := range searchRoots {
		if root == "" {
			continue
		}
		candidate := filepath.Clean(filepath.Join(root, scriptPath))
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf(
		"resolve bridge script path %q: not found relative to PI_CONFIG_PATH, cwd, or executable; set PI_BRIDGE_SCRIPT to an absolute path",
		scriptPath,
	)
}

func logBridgeHealth(health HealthResponse) {
	if health.Mode == "" && len(health.Resolved) == 0 && len(health.Available) == 0 && len(health.RouteStatus) == 0 && health.LoadError == "" && len(health.AuthErrors) == 0 {
		return
	}

	slog.Info(
		"pi bridge startup health",
		"mode", health.Mode,
		"ready_routes", summarizeRouteCounts(health.Available),
		"configured_routes", summarizeRouteCounts(health.Resolved),
	)

	if health.LoadError != "" || len(health.AuthErrors) > 0 {
		slog.Warn(
			"pi bridge startup diagnostics",
			"models_error", health.LoadError,
			"auth_errors", health.AuthErrors,
		)
	}
}

func validateBridgeHealth(health HealthResponse) error {
	var issues []string
	for capability, routes := range health.Resolved {
		if len(routes) == 0 {
			continue
		}
		if len(health.Available[capability]) > 0 {
			continue
		}
		issues = append(issues, fmt.Sprintf("%s: %s", capability, formatRouteStatuses(health.RouteStatus[capability], routes)))
	}
	if len(issues) == 0 {
		return nil
	}

	parts := make([]string, 0, 2+len(health.AuthErrors))
	parts = append(parts, "no auth-ready pi routes available for "+strings.Join(issues, "; "))
	if health.LoadError != "" {
		parts = append(parts, "models.json: "+health.LoadError)
	}
	for _, authErr := range health.AuthErrors {
		if strings.TrimSpace(authErr) == "" {
			continue
		}
		parts = append(parts, "auth: "+authErr)
	}
	return fmt.Errorf("bridge startup diagnostics: %s", strings.Join(parts, " | "))
}

func summarizeRouteCounts(routes map[string][]string) map[string]string {
	if len(routes) == 0 {
		return nil
	}
	counts := make(map[string]string, len(routes))
	for capability, entries := range routes {
		counts[capability] = fmt.Sprintf("%d", len(entries))
	}
	return counts
}

func formatRouteStatuses(statuses []HealthRouteStatus, fallbackRoutes []string) string {
	if len(statuses) == 0 {
		if len(fallbackRoutes) == 0 {
			return "no routes configured"
		}
		return strings.Join(fallbackRoutes, ", ")
	}

	parts := make([]string, 0, len(statuses))
	for _, status := range statuses {
		label := status.RouteID
		if strings.TrimSpace(label) == "" {
			label = "<unknown-route>"
		}
		if strings.TrimSpace(status.Message) != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", label, status.Message))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", label, status.Status))
	}
	return strings.Join(parts, ", ")
}

func withConfiguredTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	if deadline, ok := ctx.Deadline(); ok {
		if time.Until(deadline) <= timeout {
			return ctx, func() {}
		}
	}
	return context.WithTimeout(ctx, timeout)
}
