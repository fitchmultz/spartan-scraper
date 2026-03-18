// Package mcp defines tool metadata and schemas for the Spartan MCP surface.
//
// Purpose:
// - Describe every MCP tool exposed by the server in one deterministic list.
//
// Responsibilities:
// - Define tool names, descriptions, and JSON Schemas.
// - Keep MCP batch, job, watch, export, and AI authoring tool contracts aligned with other operator surfaces.
//
// Scope:
// - Tool metadata only; handler execution lives in handlers.go.
//
// Usage:
// - Returned from `tools/list` and used by tests to verify parity.
//
// Invariants/Assumptions:
// - Tool names match MCP `tools/call` method names.
// - Schema generation is deterministic for stable snapshots and docs.
package mcp

import (
	"sort"
)

func (s *Server) toolsList() []tool {
	diagnostics := []tool{
		{
			Name:        "health_status",
			Description: "Return the same structured setup/runtime health payload used by /healthz",
			InputSchema: schema(nil, nil),
		},
		{
			Name:        "diagnostic_check",
			Description: "Run a read-only browser, ai, or proxy_pool diagnostic re-check and return follow-up actions",
			InputSchema: schema(map[string]string{"component": "string"}, nil),
		},
	}
	if s.setup != nil {
		return diagnostics
	}
	return append(diagnostics, []tool{
		{
			Name:        "ai_extract_preview",
			Description: "Preview AI-powered extraction against fetched or pasted HTML without creating a job",
			InputSchema: schema(nil, map[string]string{"url": "string", "html": "string", "mode": "string", "prompt": "string", "schema": "object", "fields": "array", "images": "array", "headless": "boolean", "playwright": "boolean", "visual": "boolean"}),
		},
		{
			Name:        "ai_template_generate",
			Description: "Generate an extraction template from fetched or pasted HTML without creating a job",
			InputSchema: schema(map[string]string{"description": "string"}, map[string]string{"url": "string", "html": "string", "sampleFields": "array", "images": "array", "headless": "boolean", "playwright": "boolean", "visual": "boolean"}),
		},
		{
			Name:        "ai_template_debug",
			Description: "Debug and repair an extraction template against fetched or pasted HTML without creating a job",
			InputSchema: schema(map[string]string{"template": "object"}, map[string]string{"url": "string", "html": "string", "instructions": "string", "images": "array", "headless": "boolean", "playwright": "boolean", "visual": "boolean"}),
		},
		{
			Name:        "ai_render_profile_generate",
			Description: "Generate a render profile for a live page without creating a job",
			InputSchema: schema(map[string]string{"url": "string", "instructions": "string"}, map[string]string{"name": "string", "hostPatterns": "array", "images": "array", "headless": "boolean", "playwright": "boolean", "visual": "boolean"}),
		},
		{
			Name:        "ai_render_profile_debug",
			Description: "Debug and tune an existing render profile against a live page without creating a job",
			InputSchema: schema(map[string]string{"url": "string", "profile": "object"}, map[string]string{"instructions": "string", "images": "array", "headless": "boolean", "playwright": "boolean", "visual": "boolean"}),
		},
		{
			Name:        "ai_pipeline_js_generate",
			Description: "Generate a pipeline JS script for a live page without creating a job",
			InputSchema: schema(map[string]string{"url": "string", "instructions": "string"}, map[string]string{"name": "string", "hostPatterns": "array", "images": "array", "headless": "boolean", "playwright": "boolean", "visual": "boolean"}),
		},
		{
			Name:        "ai_pipeline_js_debug",
			Description: "Debug and tune an existing pipeline JS script against a live page without creating a job",
			InputSchema: schema(map[string]string{"url": "string", "script": "object"}, map[string]string{"instructions": "string", "images": "array", "headless": "boolean", "playwright": "boolean", "visual": "boolean"}),
		},
		{
			Name:        "ai_research_refine",
			Description: "Refine an existing research result into a bounded operator-ready brief without creating a job",
			InputSchema: schema(map[string]string{"result": "object"}, map[string]string{"instructions": "string"}),
		},
		{
			Name:        "ai_export_shape",
			Description: "Generate or tune a bounded export shape for a representative job result before configuring recurring exports",
			InputSchema: schema(map[string]string{"jobId": "string", "format": "string"}, map[string]string{"currentShape": "object", "instructions": "string"}),
		},
		{
			Name:        "ai_transform_generate",
			Description: "Generate or tune a bounded result transform for a representative job result without creating a job",
			InputSchema: schema(map[string]string{"jobId": "string"}, map[string]string{"currentTransform": "object", "preferredLanguage": "string", "instructions": "string"}),
		},
		{
			Name:        "scrape_page",
			Description: "Scrape a single page using the same request contract as POST /v1/scrape",
			InputSchema: schema(map[string]string{"url": "string"}, map[string]string{"method": "string", "body": "string", "contentType": "string", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "authProfile": "string", "auth": "object", "extract": "object", "pipeline": "object", "incremental": "boolean", "webhook": "object", "screenshot": "object", "device": "object", "networkIntercept": "object"}),
		},
		{
			Name:        "crawl_site",
			Description: "Crawl a site using the same request contract as POST /v1/crawl",
			InputSchema: schema(map[string]string{"url": "string"}, map[string]string{"maxDepth": "number", "maxPages": "number", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "authProfile": "string", "auth": "object", "extract": "object", "pipeline": "object", "incremental": "boolean", "sitemapURL": "string", "sitemapOnly": "boolean", "includePatterns": "array", "excludePatterns": "array", "respectRobotsTxt": "boolean", "skipDuplicates": "boolean", "simHashThreshold": "number", "webhook": "object", "screenshot": "object", "device": "object", "networkIntercept": "object"}),
		},
		{
			Name:        "research",
			Description: "Run research using the same request contract as POST /v1/research",
			InputSchema: schema(map[string]string{"query": "string", "urls": "array"}, map[string]string{"maxDepth": "number", "maxPages": "number", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "authProfile": "string", "auth": "object", "extract": "object", "pipeline": "object", "webhook": "object", "screenshot": "object", "device": "object", "networkIntercept": "object", "agentic": "object"}),
		},
		{
			Name:        "job_status",
			Description: "Get a single job envelope by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "job_results",
			Description: "Get job results by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "job_list",
			Description: "List recent job run envelopes with pagination metadata and optional status filtering",
			InputSchema: schema(nil, map[string]string{"limit": "number", "offset": "number", "status": "string"}),
		},
		{
			Name:        "job_failure_list",
			Description: "List recent failed job runs with derived failure context",
			InputSchema: schema(nil, map[string]string{"limit": "number", "offset": "number"}),
		},
		{
			Name:        "job_cancel",
			Description: "Cancel a running or queued job and return the updated job envelope",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "batch_scrape",
			Description: "Create a batch of scrape jobs using the same request contract as POST /v1/jobs/batch/scrape",
			InputSchema: schema(map[string]string{"jobs": "array"}, map[string]string{"outputFormat": "string", "extractionName": "string", "extractionMode": "string", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "authProfile": "string", "auth": "object", "extract": "object", "pipeline": "object", "incremental": "boolean", "webhook": "object", "screenshot": "object", "device": "object", "networkIntercept": "object"}),
		},
		{
			Name:        "batch_crawl",
			Description: "Create a batch of crawl jobs using the same request contract as POST /v1/jobs/batch/crawl",
			InputSchema: schema(map[string]string{"jobs": "array"}, map[string]string{"maxDepth": "number", "maxPages": "number", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "authProfile": "string", "auth": "object", "extract": "object", "pipeline": "object", "incremental": "boolean", "sitemapURL": "string", "sitemapOnly": "boolean", "includePatterns": "array", "excludePatterns": "array", "respectRobotsTxt": "boolean", "skipDuplicates": "boolean", "simHashThreshold": "number", "webhook": "object", "screenshot": "object", "device": "object", "networkIntercept": "object"}),
		},
		{
			Name:        "batch_research",
			Description: "Create a batch of research jobs using the same request contract as POST /v1/jobs/batch/research",
			InputSchema: schema(map[string]string{"jobs": "array", "query": "string"}, map[string]string{"maxDepth": "number", "maxPages": "number", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "authProfile": "string", "auth": "object", "extract": "object", "pipeline": "object", "webhook": "object", "screenshot": "object", "device": "object", "networkIntercept": "object", "agentic": "object"}),
		},
		{
			Name:        "batch_list",
			Description: "List batch summaries with pagination metadata",
			InputSchema: schema(nil, map[string]string{"limit": "number", "offset": "number"}),
		},
		{
			Name:        "batch_status",
			Description: "Get a batch envelope by id with optional included jobs",
			InputSchema: schema(map[string]string{"id": "string"}, map[string]string{"includeJobs": "boolean", "limit": "number", "offset": "number"}),
		},
		{
			Name:        "batch_cancel",
			Description: "Cancel a batch and return the updated batch envelope",
			InputSchema: schema(map[string]string{"id": "string"}, map[string]string{"includeJobs": "boolean", "limit": "number", "offset": "number"}),
		},
		{
			Name:        "job_export",
			Description: "Export saved job results in jsonl, json, md, csv, or xlsx with optional shape or transform controls",
			InputSchema: schema(map[string]string{"id": "string"}, map[string]string{"format": "string", "shape": "object", "transform": "object"}),
		},
		{
			Name:        "watch_list",
			Description: "List configured watches",
			InputSchema: schema(nil, nil),
		},
		{
			Name:        "watch_get",
			Description: "Get a single watch by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "watch_create",
			Description: "Create a new watch for content change monitoring",
			InputSchema: schema(map[string]string{"url": "string"}, map[string]string{"selector": "string", "intervalSeconds": "number", "enabled": "boolean", "diffFormat": "string", "webhookConfig": "object", "notifyOnChange": "boolean", "minChangeSize": "number", "ignorePatterns": "array", "headless": "boolean", "usePlaywright": "boolean", "extractMode": "string", "screenshotEnabled": "boolean", "screenshotConfig": "object", "visualDiffThreshold": "number", "jobTrigger": "object"}),
		},
		{
			Name:        "watch_update",
			Description: "Update an existing watch",
			InputSchema: schema(map[string]string{"id": "string"}, map[string]string{"url": "string", "selector": "string", "intervalSeconds": "number", "enabled": "boolean", "diffFormat": "string", "webhookConfig": "object", "notifyOnChange": "boolean", "minChangeSize": "number", "ignorePatterns": "array", "headless": "boolean", "usePlaywright": "boolean", "extractMode": "string", "screenshotEnabled": "boolean", "screenshotConfig": "object", "visualDiffThreshold": "number", "jobTrigger": "object"}),
		},
		{
			Name:        "watch_delete",
			Description: "Delete a watch by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "watch_check",
			Description: "Run a manual check for a watch and return the check result",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "export_schedule_list",
			Description: "List automated export schedules",
			InputSchema: schema(nil, nil),
		},
		{
			Name:        "export_schedule_get",
			Description: "Get a single automated export schedule by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "export_schedule_create",
			Description: "Create an automated export schedule",
			InputSchema: schema(map[string]string{"name": "string", "filters": "object", "export": "object"}, map[string]string{"enabled": "boolean", "retry": "object"}),
		},
		{
			Name:        "export_schedule_update",
			Description: "Update an existing automated export schedule",
			InputSchema: schema(map[string]string{"id": "string", "name": "string", "filters": "object", "export": "object"}, map[string]string{"enabled": "boolean", "retry": "object"}),
		},
		{
			Name:        "export_schedule_delete",
			Description: "Delete an automated export schedule",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "export_schedule_history",
			Description: "Get export history for an automated export schedule",
			InputSchema: schema(map[string]string{"id": "string"}, map[string]string{"limit": "number", "offset": "number"}),
		},
		{
			Name:        "webhook_delivery_list",
			Description: "List persisted webhook delivery attempts, including failures, retries, and response metadata",
			InputSchema: schema(nil, map[string]string{"jobId": "string", "limit": "number", "offset": "number"}),
		},
		{
			Name:        "webhook_delivery_get",
			Description: "Get a single persisted webhook delivery attempt by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "proxy_pool_status",
			Description: "Inspect the currently loaded proxy pool strategy and per-proxy health/runtime stats",
			InputSchema: schema(nil, nil),
		},
	}...)
}

func schema(required map[string]string, optional map[string]string) map[string]interface{} {
	properties := map[string]interface{}{}

	keys := make([]string, 0, len(required)+len(optional))
	for key := range required {
		keys = append(keys, key)
	}
	for key := range optional {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	requiredKeys := make([]string, 0, len(required))
	for key := range required {
		requiredKeys = append(requiredKeys, key)
	}
	sort.Strings(requiredKeys)

	for _, key := range keys {
		kind := ""
		if k, ok := required[key]; ok {
			kind = k
		} else if k, ok := optional[key]; ok {
			kind = k
		}
		properties[key] = map[string]string{"type": kind}
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   requiredKeys,
	}
}
