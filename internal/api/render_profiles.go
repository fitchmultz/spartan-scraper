// Package api provides REST API handlers for render profile management.
//
// Purpose:
// - Expose file-backed CRUD endpoints for render profiles.
//
// Responsibilities:
// - Route collection and item requests for /v1/render-profiles.
// - Reuse the shared named-resource CRUD helpers with render-profile storage.
//
// Scope:
// - HTTP handler wiring only; render-profile persistence and validation live in internal/fetch.
//
// Usage:
// - Registered by the API server for GET/POST collection requests and GET/PUT/DELETE item requests.
//
// Invariants/Assumptions:
// - Profile names are stable identifiers carried in the URL path.
// - Validation is delegated to the fetch package before files are written.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

var renderProfileStore = namedResourceStore[fetch.RenderProfile]{
	pathSegment:   "render-profiles",
	singularLabel: "profile",
	collectionKey: "profiles",
	list: func(dataDir string) ([]fetch.RenderProfile, error) {
		file, err := fetch.LoadRenderProfilesFile(dataDir)
		if err != nil {
			return nil, err
		}
		return file.Profiles, nil
	},
	get:    fetch.GetRenderProfile,
	upsert: fetch.UpsertRenderProfile,
	delete: fetch.DeleteRenderProfile,
	nameOf: func(profile fetch.RenderProfile) string {
		return profile.Name
	},
	setName: func(profile *fetch.RenderProfile, name string) {
		profile.Name = name
	},
}

// handleRenderProfiles handles requests to /v1/render-profiles.
func (s *Server) handleRenderProfiles(w http.ResponseWriter, r *http.Request) {
	handleNamedResourceCollection(s, w, r, renderProfileStore)
}

// handleRenderProfile handles requests to /v1/render-profiles/{name}.
func (s *Server) handleRenderProfile(w http.ResponseWriter, r *http.Request) {
	handleNamedResourceItem(s, w, r, renderProfileStore)
}
