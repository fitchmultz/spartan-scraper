// Package api provides HTTP handlers for plugin management endpoints.
// Plugin handlers support listing, installing, enabling, disabling, and
// configuring third-party WASM plugins.
package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/plugins"
)

// PluginListResponse represents the response for listing plugins.
type PluginListResponse struct {
	Plugins []*plugins.PluginInfo `json:"plugins"`
}

// PluginInstallRequest represents a plugin installation request.
type PluginInstallRequest struct {
	Source string `json:"source"` // Path to plugin directory
}

// PluginInstallFromUploadRequest represents a plugin installation from uploaded file.
type PluginInstallFromUploadRequest struct {
	Name    string `json:"name"`
	Content []byte `json:"content"` // Base64-encoded zip or wasm
}

// PluginConfigureRequest represents a plugin configuration update.
type PluginConfigureRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// handlePlugins handles requests to /v1/plugins
func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListPlugins(w, r)
	case http.MethodPost:
		s.handleInstallPlugin(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handlePlugin handles requests to /v1/plugins/{name}
func (s *Server) handlePlugin(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/plugins/")
	name := strings.TrimSpace(path)

	if name == "" {
		writeError(w, r, apperrors.Validation("plugin name is required"))
		return
	}

	// Handle sub-paths for enable/disable
	if strings.HasSuffix(name, "/enable") {
		name = strings.TrimSuffix(name, "/enable")
		if r.Method == http.MethodPost {
			s.handleEnablePlugin(w, r, name)
		} else {
			writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		}
		return
	}

	if strings.HasSuffix(name, "/disable") {
		name = strings.TrimSuffix(name, "/disable")
		if r.Method == http.MethodPost {
			s.handleDisablePlugin(w, r, name)
		} else {
			writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetPlugin(w, r, name)
	case http.MethodPut:
		s.handleUpdatePlugin(w, r, name)
	case http.MethodDelete:
		s.handleDeletePlugin(w, r, name)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleListPlugins handles GET /v1/plugins
func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	loader := plugins.NewLoader(s.cfg.DataDir)
	pluginList, err := loader.Discover()
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, PluginListResponse{Plugins: pluginList})
}

// handleGetPlugin handles GET /v1/plugins/{name}
func (s *Server) handleGetPlugin(w http.ResponseWriter, r *http.Request, name string) {
	loader := plugins.NewLoader(s.cfg.DataDir)
	manifest, pluginDir, err := loader.LoadPlugin(name)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			writeError(w, r, apperrors.NotFound(fmt.Sprintf("plugin not found: %s", name)))
			return
		}
		writeError(w, r, err)
		return
	}

	info, err := manifest.ToInfo(pluginDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, info)
}

// handleInstallPlugin handles POST /v1/plugins
func (s *Server) handleInstallPlugin(w http.ResponseWriter, r *http.Request) {
	var req PluginInstallRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	if req.Source == "" {
		writeError(w, r, apperrors.Validation("source is required"))
		return
	}

	loader := plugins.NewLoader(s.cfg.DataDir)
	info, err := loader.Install(req.Source)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindValidation) {
			writeError(w, r, apperrors.Wrap(apperrors.KindValidation, "invalid plugin", err))
		} else {
			writeError(w, r, err)
		}
		return
	}

	writeCreatedJSON(w, info)
}

// handleUpdatePlugin handles PUT /v1/plugins/{name}
func (s *Server) handleUpdatePlugin(w http.ResponseWriter, r *http.Request, name string) {
	var req PluginConfigureRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	if req.Key == "" {
		writeError(w, r, apperrors.Validation("key is required"))
		return
	}

	loader := plugins.NewLoader(s.cfg.DataDir)
	if err := loader.Configure(name, req.Key, req.Value); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			writeError(w, r, apperrors.NotFound(fmt.Sprintf("plugin not found: %s", name)))
			return
		}
		writeError(w, r, err)
		return
	}

	// Return updated plugin info
	manifest, pluginDir, err := loader.LoadPlugin(name)
	if err != nil {
		writeError(w, r, err)
		return
	}

	info, err := manifest.ToInfo(pluginDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, info)
}

// handleDeletePlugin handles DELETE /v1/plugins/{name}
func (s *Server) handleDeletePlugin(w http.ResponseWriter, r *http.Request, name string) {
	loader := plugins.NewLoader(s.cfg.DataDir)
	if err := loader.Uninstall(name); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			writeError(w, r, apperrors.NotFound(fmt.Sprintf("plugin not found: %s", name)))
			return
		}
		writeError(w, r, err)
		return
	}

	writeNoContent(w)
}

// handleEnablePlugin handles POST /v1/plugins/{name}/enable
func (s *Server) handleEnablePlugin(w http.ResponseWriter, r *http.Request, name string) {
	loader := plugins.NewLoader(s.cfg.DataDir)
	if err := loader.Enable(name); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			writeError(w, r, apperrors.NotFound(fmt.Sprintf("plugin not found: %s", name)))
			return
		}
		writeError(w, r, err)
		return
	}

	// Return updated plugin info
	manifest, pluginDir, err := loader.LoadPlugin(name)
	if err != nil {
		writeError(w, r, err)
		return
	}

	info, err := manifest.ToInfo(pluginDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, info)
}

// handleDisablePlugin handles POST /v1/plugins/{name}/disable
func (s *Server) handleDisablePlugin(w http.ResponseWriter, r *http.Request, name string) {
	loader := plugins.NewLoader(s.cfg.DataDir)
	if err := loader.Disable(name); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			writeError(w, r, apperrors.NotFound(fmt.Sprintf("plugin not found: %s", name)))
			return
		}
		writeError(w, r, err)
		return
	}

	// Return updated plugin info
	manifest, pluginDir, err := loader.LoadPlugin(name)
	if err != nil {
		writeError(w, r, err)
		return
	}

	info, err := manifest.ToInfo(pluginDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, info)
}
