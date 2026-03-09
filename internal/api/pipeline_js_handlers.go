// Package api provides REST API handlers for pipeline JavaScript management.
//
// Purpose:
// - Expose file-backed CRUD endpoints for pipeline JS scripts.
//
// Responsibilities:
// - Route collection and item requests for /v1/pipeline-js.
// - Reuse the shared named-resource CRUD helpers with pipeline JS storage.
//
// Scope:
// - HTTP handler wiring only; script persistence and validation live in internal/pipeline.
//
// Usage:
// - Registered by the API server for GET/POST collection requests and GET/PUT/DELETE item requests.
//
// Invariants/Assumptions:
// - Script names are stable identifiers carried in the URL path.
// - Validation is delegated to the pipeline package before files are written.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

var pipelineJSStore = namedResourceStore[pipeline.JSTargetScript]{
	pathSegment:   "pipeline-js",
	singularLabel: "script",
	collectionKey: "scripts",
	list: func(dataDir string) ([]pipeline.JSTargetScript, error) {
		registry, err := pipeline.LoadJSRegistryStrict(dataDir)
		if err != nil {
			return nil, err
		}
		return registry.Scripts, nil
	},
	get:    pipeline.GetJSScript,
	upsert: pipeline.UpsertJSScript,
	delete: pipeline.DeleteJSScript,
	nameOf: func(script pipeline.JSTargetScript) string {
		return script.Name
	},
	setName: func(script *pipeline.JSTargetScript, name string) {
		script.Name = name
	},
}

// handlePipelineJS handles requests to /v1/pipeline-js.
func (s *Server) handlePipelineJS(w http.ResponseWriter, r *http.Request) {
	handleNamedResourceCollection(s, w, r, pipelineJSStore)
}

// handlePipelineJSScript handles requests to /v1/pipeline-js/{name}.
func (s *Server) handlePipelineJSScript(w http.ResponseWriter, r *http.Request) {
	handleNamedResourceItem(s, w, r, pipelineJSStore)
}
