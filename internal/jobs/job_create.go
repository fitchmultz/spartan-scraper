// Package jobs provides job creation and persistence logic for scrape, crawl, and research jobs.
//
// Purpose:
// - Convert creation-time job requests into persisted typed spec envelopes.
//
// Responsibilities:
// - Validate incoming job specs.
// - Build the typed persisted spec payload for the job kind.
// - Create queued job records with local artifact paths.
//
// Scope:
// - Job creation only. Execution happens elsewhere.
//
// Usage:
// - Called by API, scheduler, MCP, batch, and chain submission flows.
//
// Invariants/Assumptions:
// - Every created job persists a typed spec and spec version.
// - ResultPath is always DATA_DIR/jobs/<id>/results.jsonl.
// - Version 1 is the only supported persisted job spec version.
package jobs

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func buildExecutionSpec(spec JobSpec) model.ExecutionSpec {
	exec := model.ExecutionSpec{
		RequestID:        spec.RequestID,
		Headless:         spec.Headless,
		UsePlaywright:    spec.UsePlaywright,
		TimeoutSeconds:   spec.TimeoutSeconds,
		AuthProfile:      spec.AuthProfile,
		Auth:             spec.Auth,
		Extract:          spec.Extract,
		Pipeline:         spec.Pipeline,
		Screenshot:       spec.Screenshot,
		NetworkIntercept: spec.NetworkIntercept,
		Device:           spec.Device,
	}

	if spec.WebhookURL != "" {
		exec.Webhook = &model.WebhookSpec{
			URL:    spec.WebhookURL,
			Events: spec.WebhookEvents,
			Secret: spec.WebhookSecret,
		}
	}

	return exec
}

func persistedSpecFromCreateSpec(spec JobSpec) (int, any, error) {
	exec := buildExecutionSpec(spec)

	switch spec.Kind {
	case model.KindScrape:
		return model.JobSpecVersion1, model.ScrapeSpecV1{
			Version:     model.JobSpecVersion1,
			URL:         spec.URL,
			Method:      spec.Method,
			Body:        spec.Body,
			ContentType: spec.ContentType,
			Incremental: spec.Incremental,
			Execution:   exec,
		}, nil
	case model.KindCrawl:
		return model.JobSpecVersion1, model.CrawlSpecV1{
			Version:           model.JobSpecVersion1,
			URL:               spec.URL,
			MaxDepth:          spec.MaxDepth,
			MaxPages:          spec.MaxPages,
			Incremental:       spec.Incremental,
			SitemapURL:        spec.SitemapURL,
			SitemapOnly:       spec.SitemapOnly,
			IncludePatterns:   spec.IncludePatterns,
			ExcludePatterns:   spec.ExcludePatterns,
			RespectRobotsTxt:  spec.RespectRobotsTxt,
			SkipDuplicates:    spec.SkipDuplicates,
			SimHashThreshold:  spec.SimHashThreshold,
			CrossJobDedup:     false,
			CrossJobThreshold: 0,
			Execution:         exec,
		}, nil
	case model.KindResearch:
		return model.JobSpecVersion1, model.ResearchSpecV1{
			Version:   model.JobSpecVersion1,
			Query:     spec.Query,
			URLs:      spec.URLs,
			MaxDepth:  spec.MaxDepth,
			MaxPages:  spec.MaxPages,
			Agentic:   model.NormalizeResearchAgenticConfig(spec.Agentic),
			Execution: exec,
		}, nil
	default:
		return 0, nil, fmt.Errorf("unknown job kind: %s", spec.Kind)
	}
}

// TypedSpecFromJobSpec converts a create-time JobSpec into the persisted typed spec envelope.
func TypedSpecFromJobSpec(spec JobSpec) (int, any, error) {
	if err := spec.Validate(); err != nil {
		return 0, nil, fmt.Errorf("invalid job spec: %w", err)
	}
	return persistedSpecFromCreateSpec(spec)
}

// CreateJob creates and persists a new job from a unified JobSpec.
func (m *Manager) CreateJob(ctx context.Context, spec JobSpec) (model.Job, error) {
	if err := spec.Validate(); err != nil {
		return model.Job{}, fmt.Errorf("invalid job spec: %w", err)
	}

	specVersion, persistedSpec, err := persistedSpecFromCreateSpec(spec)
	if err != nil {
		return model.Job{}, err
	}

	now := time.Now()
	job := model.Job{
		ID:               uuid.NewString(),
		Kind:             spec.Kind,
		Status:           model.StatusQueued,
		CreatedAt:        now,
		UpdatedAt:        now,
		SpecVersion:      specVersion,
		Spec:             persistedSpec,
		DependencyStatus: model.DependencyStatusReady,
		ResultPath:       filepath.Join(m.DataDir, "jobs", uuid.Nil.String(), "results.jsonl"),
	}
	job.ResultPath = filepath.Join(m.DataDir, "jobs", job.ID, "results.jsonl")

	if err := m.store.Create(ctx, job); err != nil {
		return model.Job{}, err
	}

	return job, nil
}

func legacyScrapeSpec(args []any) (JobSpec, error) {
	if len(args) == 1 {
		spec, ok := args[0].(JobSpec)
		if !ok {
			return JobSpec{}, fmt.Errorf("expected JobSpec, got %T", args[0])
		}
		spec.Kind = model.KindScrape
		return spec, nil
	}
	if len(args) != 15 {
		return JobSpec{}, fmt.Errorf("expected 1 or 15 scrape arguments, got %d", len(args))
	}
	body, _ := args[2].([]byte)
	auth, _ := args[6].(fetch.AuthOptions)
	extractOpts, _ := args[8].(extract.ExtractOptions)
	pipelineOpts, _ := args[9].(pipeline.Options)
	screenshot, _ := args[13].(*fetch.ScreenshotConfig)
	spec := JobSpec{
		Kind:           model.KindScrape,
		URL:            args[0].(string),
		Method:         args[1].(string),
		Body:           body,
		ContentType:    args[3].(string),
		Headless:       args[4].(bool),
		UsePlaywright:  args[5].(bool),
		AuthProfile:    "",
		Auth:           auth,
		TimeoutSeconds: args[7].(int),
		Extract:        extractOpts,
		Pipeline:       pipelineOpts,
		Incremental:    args[10].(bool),
		RequestID:      args[11].(string),
		Screenshot:     screenshot,
	}
	if webhookURL, _ := args[12].(string); webhookURL != "" {
		spec.WebhookURL = webhookURL
	}
	if webhookSecret, _ := args[14].(string); webhookSecret != "" {
		spec.WebhookSecret = webhookSecret
	}
	return spec, nil
}

func legacyCrawlSpec(args []any) (JobSpec, error) {
	if len(args) == 1 {
		spec, ok := args[0].(JobSpec)
		if !ok {
			return JobSpec{}, fmt.Errorf("expected JobSpec, got %T", args[0])
		}
		spec.Kind = model.KindCrawl
		return spec, nil
	}
	if len(args) != 16 {
		return JobSpec{}, fmt.Errorf("expected 1 or 16 crawl arguments, got %d", len(args))
	}
	auth, _ := args[5].(fetch.AuthOptions)
	extractOpts, _ := args[7].(extract.ExtractOptions)
	pipelineOpts, _ := args[8].(pipeline.Options)
	screenshot, _ := args[14].(*fetch.ScreenshotConfig)
	spec := JobSpec{
		Kind:             model.KindCrawl,
		URL:              args[0].(string),
		MaxDepth:         args[1].(int),
		MaxPages:         args[2].(int),
		Headless:         args[3].(bool),
		UsePlaywright:    args[4].(bool),
		AuthProfile:      "",
		Auth:             auth,
		TimeoutSeconds:   args[6].(int),
		Extract:          extractOpts,
		Pipeline:         pipelineOpts,
		Incremental:      args[9].(bool),
		RequestID:        args[10].(string),
		SitemapURL:       args[11].(string),
		SitemapOnly:      args[12].(bool),
		Screenshot:       screenshot,
		WebhookSecret:    args[15].(string),
		RespectRobotsTxt: false,
	}
	if webhookURL, ok := args[13].(string); ok {
		spec.WebhookURL = webhookURL
	} else {
		spec.WebhookURL = ""
	}
	return spec, nil
}

func legacyResearchSpec(args []any) (JobSpec, error) {
	if len(args) == 1 {
		spec, ok := args[0].(JobSpec)
		if !ok {
			return JobSpec{}, fmt.Errorf("expected JobSpec, got %T", args[0])
		}
		spec.Kind = model.KindResearch
		return spec, nil
	}
	if len(args) != 14 {
		return JobSpec{}, fmt.Errorf("expected 1 or 14 research arguments, got %d", len(args))
	}
	auth, _ := args[6].(fetch.AuthOptions)
	extractOpts, _ := args[8].(extract.ExtractOptions)
	pipelineOpts, _ := args[9].(pipeline.Options)
	screenshot, _ := args[12].(*fetch.ScreenshotConfig)
	spec := JobSpec{
		Kind:           model.KindResearch,
		Query:          args[0].(string),
		URLs:           args[1].([]string),
		MaxDepth:       args[2].(int),
		MaxPages:       args[3].(int),
		Headless:       args[4].(bool),
		UsePlaywright:  args[5].(bool),
		AuthProfile:    "",
		Auth:           auth,
		TimeoutSeconds: args[7].(int),
		Extract:        extractOpts,
		Pipeline:       pipelineOpts,
		RequestID:      args[10].(string),
		WebhookURL:     args[11].(string),
		Screenshot:     screenshot,
	}
	if webhookSecret, _ := args[13].(string); webhookSecret != "" {
		spec.WebhookSecret = webhookSecret
	}
	return spec, nil
}

// CreateScrapeJob creates and persists a scrape job.
func (m *Manager) CreateScrapeJob(ctx context.Context, args ...any) (model.Job, error) {
	spec, err := legacyScrapeSpec(args)
	if err != nil {
		return model.Job{}, err
	}
	return m.CreateJob(ctx, spec)
}

// CreateCrawlJob creates and persists a crawl job.
func (m *Manager) CreateCrawlJob(ctx context.Context, args ...any) (model.Job, error) {
	spec, err := legacyCrawlSpec(args)
	if err != nil {
		return model.Job{}, err
	}
	return m.CreateJob(ctx, spec)
}

// CreateResearchJob creates and persists a research job.
func (m *Manager) CreateResearchJob(ctx context.Context, args ...any) (model.Job, error) {
	spec, err := legacyResearchSpec(args)
	if err != nil {
		return model.Job{}, err
	}
	return m.CreateJob(ctx, spec)
}
