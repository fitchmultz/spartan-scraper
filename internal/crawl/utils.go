package crawl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

// normalizeURL normalizes a URL for deduplication purposes.
// It lowercases the host and removes the fragment.
func normalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	return u.String()
}

// resolveURL resolves a relative URL against a base URL.
// Returns empty string if the href is invalid.
func resolveURL(base *url.URL, href string) string {
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return ""
	}
	return base.ResolveReference(u).String()
}

// sameHost checks if a URL has the same host as the base URL.
func sameHost(base *url.URL, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Host == base.Host
}

// dispatchPageEvent sends a webhook notification for a crawled page.
func (req *Request) dispatchPageEvent(ctx context.Context, result PageResult, depth int, seqNum int) {
	if req.WebhookDispatcher == nil || req.WebhookConfig == nil {
		return
	}
	if !webhook.ShouldSendEvent(webhook.EventPageCrawled, "", req.WebhookConfig.Events) {
		return
	}

	payload := webhook.Payload{
		EventID:     fmt.Sprintf("%s-page-%d", req.RequestID, seqNum),
		EventType:   webhook.EventPageCrawled,
		Timestamp:   time.Now(),
		JobID:       req.RequestID,
		JobKind:     string(model.KindCrawl),
		PageURL:     result.URL,
		PageStatus:  result.Status,
		PageTitle:   result.Title,
		PageDepth:   depth,
		IsDuplicate: result.DuplicateOf != "",
		DuplicateOf: result.DuplicateOf,
		CrawlSeqNum: seqNum,
	}

	req.WebhookDispatcher.Dispatch(ctx, req.WebhookConfig.URL, payload, req.WebhookConfig.Secret)
}

// applyCrawlOutputPipeline applies the output pipeline stages to a crawl result.
// It runs pre-output hooks, transformers, and post-output hooks.
func applyCrawlOutputPipeline(ctx context.Context, registry *pipeline.Registry, baseCtx pipeline.HookContext, result PageResult) (PageResult, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return PageResult{}, apperrors.Wrap(apperrors.KindInternal, "failed to marshal result", err)
	}
	input := pipeline.OutputInput{
		Target:     baseCtx.Target,
		Kind:       string(model.KindCrawl),
		Raw:        raw,
		Structured: result,
	}

	preCtx := baseCtx
	preCtx.Stage = pipeline.StagePreOutput
	outInput, err := registry.RunPreOutput(preCtx, input)
	if err != nil {
		return PageResult{}, err
	}
	if typed, ok := outInput.Structured.(PageResult); ok {
		result = typed
		outInput.Structured = result
	}

	transformCtx := baseCtx
	transformCtx.Stage = pipeline.StagePreOutput
	out, err := registry.RunTransformers(transformCtx, outInput)
	if err != nil {
		return PageResult{}, err
	}

	postCtx := baseCtx
	postCtx.Stage = pipeline.StagePostOutput
	out, err = registry.RunPostOutput(postCtx, outInput, out)
	if err != nil {
		return PageResult{}, err
	}

	if out.Structured == nil {
		return result, nil
	}
	typed, ok := out.Structured.(PageResult)
	if !ok {
		return PageResult{}, apperrors.Internal("pipeline output type mismatch for crawl")
	}
	return typed, nil
}
