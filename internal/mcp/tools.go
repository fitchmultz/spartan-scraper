// MCP tool definitions and schema builder.
//
// Responsibilities:
// - Define all available MCP tools (scrape_page, crawl_site, research, job_*)
// - Generate JSON Schema for tool input validation
//
// Does NOT handle:
// - Tool execution logic (handled by handlers.go)
// - Request parsing or validation
//
// Invariants:
// - Tool names match MCP tool/call method names
// - Schema is deterministic (sorted keys for consistent output)
package mcp

import (
	"sort"
)

func (s *Server) toolsList() []tool {
	return []tool{
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
			Description: "Scrape a single page (static or headless) with optional AI extraction controls",
			InputSchema: schema(map[string]string{"url": "string"}, map[string]string{"authProfile": "string", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "extractTemplate": "string", "extractValidate": "boolean", "aiExtract": "boolean", "aiMode": "string", "aiPrompt": "string", "aiSchema": "object", "aiFields": "array", "preProcessors": "array", "postProcessors": "array", "transformers": "array", "incremental": "boolean"}),
		},
		{
			Name:        "crawl_site",
			Description: "Crawl a site with depth and page limits plus optional AI extraction controls",
			InputSchema: schema(map[string]string{"url": "string"}, map[string]string{"authProfile": "string", "maxDepth": "number", "maxPages": "number", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "extractTemplate": "string", "extractValidate": "boolean", "aiExtract": "boolean", "aiMode": "string", "aiPrompt": "string", "aiSchema": "object", "aiFields": "array", "preProcessors": "array", "postProcessors": "array", "transformers": "array", "incremental": "boolean"}),
		},
		{
			Name:        "research",
			Description: "Deep research across multiple sources with optional AI extraction controls and bounded pi-powered follow-up synthesis",
			InputSchema: schema(map[string]string{"query": "string", "urls": "array"}, map[string]string{"authProfile": "string", "maxDepth": "number", "maxPages": "number", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "extractTemplate": "string", "extractValidate": "boolean", "aiExtract": "boolean", "aiMode": "string", "aiPrompt": "string", "aiSchema": "object", "aiFields": "array", "agentic": "boolean", "agenticInstructions": "string", "agenticMaxRounds": "number", "agenticMaxFollowUpUrls": "number", "preProcessors": "array", "postProcessors": "array", "transformers": "array"}),
		},
		{
			Name:        "job_status",
			Description: "Get job status by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "job_results",
			Description: "Get job results by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "job_list",
			Description: "List all jobs with pagination",
			InputSchema: schema(nil, map[string]string{"limit": "number", "offset": "number"}),
		},
		{
			Name:        "job_cancel",
			Description: "Cancel a running or queued job by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "job_export",
			Description: "Export saved job results in jsonl, json, md, csv, or xlsx with optional shape or transform controls",
			InputSchema: schema(map[string]string{"id": "string"}, map[string]string{"format": "string", "shape": "object", "transform": "object"}),
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
			Name:        "proxy_pool_status",
			Description: "Inspect the currently loaded proxy pool strategy and per-proxy health/runtime stats",
			InputSchema: schema(nil, nil),
		},
	}
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
