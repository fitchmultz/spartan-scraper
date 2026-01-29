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
			Name:        "scrape_page",
			Description: "Scrape a single page (static or headless)",
			InputSchema: schema(map[string]string{"url": "string"}, map[string]string{"authProfile": "string", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "extractTemplate": "string", "extractValidate": "boolean", "preProcessors": "array", "postProcessors": "array", "transformers": "array", "incremental": "boolean"}),
		},
		{
			Name:        "crawl_site",
			Description: "Crawl a site with depth and page limits",
			InputSchema: schema(map[string]string{"url": "string"}, map[string]string{"authProfile": "string", "maxDepth": "number", "maxPages": "number", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "extractTemplate": "string", "extractValidate": "boolean", "preProcessors": "array", "postProcessors": "array", "transformers": "array", "incremental": "boolean"}),
		},
		{
			Name:        "research",
			Description: "Deep research across multiple sources",
			InputSchema: schema(map[string]string{"query": "string", "urls": "array"}, map[string]string{"authProfile": "string", "maxDepth": "number", "maxPages": "number", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "extractTemplate": "string", "extractValidate": "boolean", "preProcessors": "array", "postProcessors": "array", "transformers": "array"}),
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
			Description: "Export job results in specified format (jsonl, json, md, csv)",
			InputSchema: schema(map[string]string{"id": "string"}, map[string]string{"format": "string"}),
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
