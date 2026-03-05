// Package api provides HTTP handlers for extraction template listing endpoints.
// Template handlers support listing available extraction templates for use
// in scrape, crawl, and research jobs.
package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

// TemplateResponse with full template details
type TemplateResponse struct {
	Name      string           `json:"name"`
	IsBuiltIn bool             `json:"is_built_in"`
	Template  extract.Template `json:"template"`
}

// CreateTemplateRequest for POST /v1/templates
type CreateTemplateRequest struct {
	Name      string                 `json:"name"`
	Selectors []extract.SelectorRule `json:"selectors"`
	JSONLD    []extract.JSONLDRule   `json:"jsonld,omitempty"`
	Regex     []extract.RegexRule    `json:"regex,omitempty"`
	Normalize extract.NormalizeSpec  `json:"normalize,omitempty"`
}

// builtInTemplateNames contains the list of built-in template names that cannot be modified
var builtInTemplateNames = map[string]bool{
	"default": true,
	"article": true,
	"product": true,
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListTemplates(w, r)
	case http.MethodPost:
		s.handleCreateTemplate(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleTemplate handles requests to /v1/templates/{name}
func (s *Server) handleTemplate(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/templates/")
	name := strings.TrimSpace(path)

	if name == "" {
		writeError(w, r, apperrors.Validation("template name is required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetTemplate(w, r, name)
	case http.MethodPut:
		s.handleUpdateTemplate(w, r, name)
	case http.MethodDelete:
		s.handleDeleteTemplate(w, r, name)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleListTemplates handles GET /v1/templates
func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	names, err := extract.ListTemplateNames(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, map[string]interface{}{"templates": names})
}

// handleGetTemplate handles GET /v1/templates/{name}
func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request, name string) {
	registry, err := extract.LoadTemplateRegistry(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	template, err := registry.GetTemplate(name)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			writeError(w, r, apperrors.NotFound("template not found: "+name))
			return
		}
		writeError(w, r, err)
		return
	}

	_, isBuiltIn := builtInTemplateNames[name]

	response := TemplateResponse{
		Name:      name,
		IsBuiltIn: isBuiltIn,
		Template:  template,
	}

	writeJSON(w, response)
}

// handleCreateTemplate handles POST /v1/templates
func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid request body: "+err.Error()))
		return
	}

	// Validate request
	if req.Name == "" {
		writeError(w, r, apperrors.Validation("name is required"))
		return
	}

	if len(req.Selectors) == 0 {
		writeError(w, r, apperrors.Validation("at least one selector is required"))
		return
	}

	// Check if trying to create a built-in template
	if _, exists := builtInTemplateNames[req.Name]; exists {
		writeError(w, r, apperrors.Validation("cannot create template with reserved name: "+req.Name))
		return
	}

	// Check if template already exists
	registry, err := extract.LoadTemplateRegistry(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if _, exists := registry.Templates[req.Name]; exists {
		writeError(w, r, apperrors.Validation("template already exists: "+req.Name))
		return
	}

	// Create the template
	template := extract.Template{
		Name:      req.Name,
		Selectors: req.Selectors,
		JSONLD:    req.JSONLD,
		Regex:     req.Regex,
		Normalize: req.Normalize,
	}

	// Save to file
	if err := s.saveTemplateToFile(template); err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to save template", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, TemplateResponse{
		Name:      req.Name,
		IsBuiltIn: false,
		Template:  template,
	})
}

// handleUpdateTemplate handles PUT /v1/templates/{name}
func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request, name string) {
	// Protect built-in templates
	if _, isBuiltIn := builtInTemplateNames[name]; isBuiltIn {
		writeError(w, r, apperrors.Permission("cannot modify built-in template: "+name))
		return
	}

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid request body: "+err.Error()))
		return
	}

	// Validate request
	if len(req.Selectors) == 0 {
		writeError(w, r, apperrors.Validation("at least one selector is required"))
		return
	}

	// Check if template exists
	registry, err := extract.LoadTemplateRegistry(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if _, exists := registry.Templates[name]; !exists {
		writeError(w, r, apperrors.NotFound("template not found: "+name))
		return
	}

	// If name is changing, validate new name doesn't conflict
	if req.Name != "" && req.Name != name {
		if _, exists := builtInTemplateNames[req.Name]; exists {
			writeError(w, r, apperrors.Validation("cannot rename to reserved name: "+req.Name))
			return
		}
		if _, exists := registry.Templates[req.Name]; exists {
			writeError(w, r, apperrors.Validation("template already exists with name: "+req.Name))
			return
		}
	}

	// Use original name if not provided
	templateName := req.Name
	if templateName == "" {
		templateName = name
	}

	// Create the updated template
	template := extract.Template{
		Name:      templateName,
		Selectors: req.Selectors,
		JSONLD:    req.JSONLD,
		Regex:     req.Regex,
		Normalize: req.Normalize,
	}

	// Save to file
	if err := s.saveTemplateToFile(template); err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to save template", err))
		return
	}

	// If name changed, delete the old template
	if req.Name != "" && req.Name != name {
		if err := s.deleteTemplateFromFile(name); err != nil {
			// Log but don't fail - the new template is already saved
			// In production, you'd want proper logging
			_ = err
		}
	}

	writeJSON(w, TemplateResponse{
		Name:      templateName,
		IsBuiltIn: false,
		Template:  template,
	})
}

// handleDeleteTemplate handles DELETE /v1/templates/{name}
func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request, name string) {
	// Protect built-in templates
	if _, isBuiltIn := builtInTemplateNames[name]; isBuiltIn {
		writeError(w, r, apperrors.Permission("cannot delete built-in template: "+name))
		return
	}

	// Check if template exists
	registry, err := extract.LoadTemplateRegistry(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if _, exists := registry.Templates[name]; !exists {
		writeError(w, r, apperrors.NotFound("template not found: "+name))
		return
	}

	// Delete from file
	if err := s.deleteTemplateFromFile(name); err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to delete template", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// saveTemplateToFile saves a template to the extract_templates.json file
func (s *Server) saveTemplateToFile(template extract.Template) error {
	path := filepath.Join(s.cfg.DataDir, "extract_templates.json")

	// Load existing templates
	var templateFile extract.TemplateFile
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// File doesn't exist, create new
		templateFile = extract.TemplateFile{Templates: []extract.Template{}}
	} else {
		if err := json.Unmarshal(data, &templateFile); err != nil {
			return err
		}
	}

	// Find and update or append
	found := false
	for i, t := range templateFile.Templates {
		if t.Name == template.Name {
			templateFile.Templates[i] = template
			found = true
			break
		}
	}

	if !found {
		templateFile.Templates = append(templateFile.Templates, template)
	}

	// Save back to file
	data, err = json.MarshalIndent(templateFile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// deleteTemplateFromFile removes a template from the extract_templates.json file
func (s *Server) deleteTemplateFromFile(name string) error {
	path := filepath.Join(s.cfg.DataDir, "extract_templates.json")

	// Load existing templates
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var templateFile extract.TemplateFile
	if err := json.Unmarshal(data, &templateFile); err != nil {
		return err
	}

	// Remove the template
	found := false
	newTemplates := make([]extract.Template, 0, len(templateFile.Templates))
	for _, t := range templateFile.Templates {
		if t.Name == name {
			found = true
			continue
		}
		newTemplates = append(newTemplates, t)
	}

	if !found {
		return apperrors.NotFound("template not found: " + name)
	}

	templateFile.Templates = newTemplates

	// Save back to file
	data, err = json.MarshalIndent(templateFile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
