// Package ai manages the bridge process used for pi-backed LLM operations.
//
// Purpose:
// - Provide the public bridge client operations and shared request execution path.
//
// Responsibilities:
// - Construct the client, expose operation-specific methods, decode bridge responses,
// - and apply per-request timeout handling before delegating to transport helpers.
//
// Scope:
// - Bridge operation orchestration only; process lifecycle and health helpers live in adjacent files.
//
// Usage:
// - Called by higher-level AI authoring and research services.
//
// Invariants/Assumptions:
// - Every bridge request goes through `call` so timeout, startup, and response decoding stay consistent.
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func NewClient(cfg config.AIConfig) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Health(ctx context.Context) (HealthResponse, error) {
	var resp HealthResponse
	err := c.call(ctx, OperationHealth, nil, &resp)
	if err == nil {
		return resp, nil
	}
	var healthErr *BridgeHealthError
	if errors.As(err, &healthErr) {
		return healthErr.Health, err
	}
	return HealthResponse{}, err
}

func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.Health(ctx)
	return err
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

func (c *Client) GenerateExportShape(ctx context.Context, req ExportShapeRequest) (ExportShapeResult, error) {
	var resp ExportShapeResult
	err := c.call(ctx, OperationExportShape, req, &resp)
	return resp, err
}

func (c *Client) GenerateTransform(ctx context.Context, req GenerateTransformRequest) (GenerateTransformResult, error) {
	var resp GenerateTransformResult
	err := c.call(ctx, OperationGenerateTransform, req, &resp)
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
