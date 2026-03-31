// Package mcp defines the typed registry for MCP tool execution.
//
// Purpose:
// - Keep MCP tool-name dispatch explicit, validated, and split by domain.
//
// Responsibilities:
// - Declare per-domain tool registries.
// - Merge registries with duplicate-name protection.
// - Select the correct registry for setup or runtime servers.
//
// Scope:
// - Tool registration and dispatch metadata only; tool implementations live elsewhere.
//
// Usage:
// - Used by handleToolCall to resolve a tool name into a domain handler.
//
// Invariants/Assumptions:
// - Each tool name is registered at most once.
// - Setup-mode registry is a strict subset of the runtime registry.
// - Registry contents stay aligned with toolsList().
package mcp

import "context"

type toolHandler func(*Server, context.Context, callParams) (interface{}, error)
type toolRegistry map[string]toolHandler

var diagnosticToolRegistry = toolRegistry{
	"health_status":    (*Server).handleHealthStatusTool,
	"diagnostic_check": (*Server).handleDiagnosticCheckTool,
}

var aiToolRegistry = toolRegistry{
	"ai_extract_preview":         (*Server).handleAIExtractPreviewTool,
	"ai_template_generate":       (*Server).handleAITemplateGenerateTool,
	"ai_template_debug":          (*Server).handleAITemplateDebugTool,
	"ai_render_profile_generate": (*Server).handleAIRenderProfileGenerateTool,
	"ai_render_profile_debug":    (*Server).handleAIRenderProfileDebugTool,
	"ai_pipeline_js_generate":    (*Server).handleAIPipelineJSGenerateTool,
	"ai_pipeline_js_debug":       (*Server).handleAIPipelineJSDebugTool,
	"ai_research_refine":         (*Server).handleAIResearchRefineTool,
	"ai_export_shape":            (*Server).handleAIExportShapeTool,
	"ai_transform_generate":      (*Server).handleAITransformGenerateTool,
}

var jobToolRegistry = toolRegistry{
	"scrape_page":      (*Server).handleScrapePageTool,
	"crawl_site":       (*Server).handleCrawlSiteTool,
	"research":         (*Server).handleResearchTool,
	"job_status":       (*Server).handleJobStatusTool,
	"job_results":      (*Server).handleJobResultsTool,
	"job_list":         (*Server).handleJobListTool,
	"job_failure_list": (*Server).handleJobFailureListTool,
	"job_cancel":       (*Server).handleJobCancelTool,
}

var batchToolRegistry = toolRegistry{
	"batch_scrape":   (*Server).handleBatchScrapeTool,
	"batch_crawl":    (*Server).handleBatchCrawlTool,
	"batch_research": (*Server).handleBatchResearchTool,
	"batch_list":     (*Server).handleBatchListTool,
	"batch_status":   (*Server).handleBatchStatusTool,
	"batch_cancel":   (*Server).handleBatchCancelTool,
}

var exportToolRegistry = toolRegistry{
	"job_export":              (*Server).handleJobExportTool,
	"job_export_history":      (*Server).handleJobExportHistoryTool,
	"export_outcome_get":      (*Server).handleExportOutcomeGetTool,
	"export_schedule_list":    (*Server).handleExportScheduleListTool,
	"export_schedule_get":     (*Server).handleExportScheduleGetTool,
	"export_schedule_create":  (*Server).handleExportScheduleCreateTool,
	"export_schedule_update":  (*Server).handleExportScheduleUpdateTool,
	"export_schedule_delete":  (*Server).handleExportScheduleDeleteTool,
	"export_schedule_history": (*Server).handleExportScheduleHistoryTool,
}

var watchToolRegistry = toolRegistry{
	"watch_list":          (*Server).handleWatchListTool,
	"watch_get":           (*Server).handleWatchGetTool,
	"watch_create":        (*Server).handleWatchCreateTool,
	"watch_update":        (*Server).handleWatchUpdateTool,
	"watch_delete":        (*Server).handleWatchDeleteTool,
	"watch_check":         (*Server).handleWatchCheckTool,
	"watch_check_history": (*Server).handleWatchCheckHistoryTool,
	"watch_check_get":     (*Server).handleWatchCheckGetTool,
}

var observabilityToolRegistry = toolRegistry{
	"webhook_delivery_list": (*Server).handleWebhookDeliveryListTool,
	"webhook_delivery_get":  (*Server).handleWebhookDeliveryGetTool,
	"proxy_pool_status":     (*Server).handleProxyPoolStatusTool,
}

var runtimeToolRegistry = mustMergeToolRegistries(
	diagnosticToolRegistry,
	aiToolRegistry,
	jobToolRegistry,
	batchToolRegistry,
	exportToolRegistry,
	watchToolRegistry,
	observabilityToolRegistry,
)

var setupToolRegistry = mustMergeToolRegistries(diagnosticToolRegistry)

func mustMergeToolRegistries(registries ...toolRegistry) toolRegistry {
	merged := make(toolRegistry)
	for _, registry := range registries {
		for name, handler := range registry {
			if _, exists := merged[name]; exists {
				panic("duplicate MCP tool registration: " + name)
			}
			merged[name] = handler
		}
	}
	return merged
}

func (s *Server) activeToolRegistry() toolRegistry {
	if s.setup != nil {
		return setupToolRegistry
	}
	return runtimeToolRegistry
}
