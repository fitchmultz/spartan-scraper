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
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
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
	name, err := requireResourceID(r, "templates", "template name")
	if err != nil {
		writeError(w, r, err)
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

	writeCollectionJSON(w, "templates", names)
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
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	template, err := templateFromRequest("", req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if isBuiltInTemplate(template.Name) {
		writeError(w, r, apperrors.Validation("cannot create template with reserved name: "+template.Name))
		return
	}

	registry, err := extract.LoadTemplateRegistry(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if _, exists := registry.Templates[template.Name]; exists {
		writeError(w, r, apperrors.Validation("template already exists: "+template.Name))
		return
	}

	if err := s.upsertCustomTemplate(template); err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to save template", err))
		return
	}

	writeCreatedJSON(w, TemplateResponse{
		Name:      template.Name,
		IsBuiltIn: false,
		Template:  template,
	})
}

// handleUpdateTemplate handles PUT /v1/templates/{name}
func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request, name string) {
	if isBuiltInTemplate(name) {
		writeError(w, r, apperrors.Permission("cannot modify built-in template: "+name))
		return
	}

	var req CreateTemplateRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	template, err := templateFromRequest(name, req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	registry, err := extract.LoadTemplateRegistry(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if _, exists := registry.Templates[name]; !exists {
		writeError(w, r, apperrors.NotFound("template not found: "+name))
		return
	}

	if template.Name != name {
		if isBuiltInTemplate(template.Name) {
			writeError(w, r, apperrors.Validation("cannot rename to reserved name: "+template.Name))
			return
		}
		if _, exists := registry.Templates[template.Name]; exists {
			writeError(w, r, apperrors.Validation("template already exists with name: "+template.Name))
			return
		}
	}

	if err := s.replaceCustomTemplate(name, template); err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to save template", err))
		return
	}

	writeJSON(w, TemplateResponse{
		Name:      template.Name,
		IsBuiltIn: false,
		Template:  template,
	})
}

// handleDeleteTemplate handles DELETE /v1/templates/{name}
func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request, name string) {
	if isBuiltInTemplate(name) {
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

	if err := s.deleteCustomTemplate(name); err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to delete template", err))
		return
	}

	writeNoContent(w)
}

func templateFromRequest(fallbackName string, req CreateTemplateRequest) (extract.Template, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(fallbackName)
	}
	if name == "" {
		return extract.Template{}, apperrors.Validation("name is required")
	}
	if len(req.Selectors) == 0 {
		return extract.Template{}, apperrors.Validation("at least one selector is required")
	}

	return extract.Template{
		Name:      name,
		Selectors: req.Selectors,
		JSONLD:    req.JSONLD,
		Regex:     req.Regex,
		Normalize: req.Normalize,
	}, nil
}

func isBuiltInTemplate(name string) bool {
	_, ok := builtInTemplateNames[name]
	return ok
}

func (s *Server) customTemplateFilePath() string {
	return filepath.Join(s.cfg.DataDir, "extract_templates.json")
}

func (s *Server) loadCustomTemplateFile() (extract.TemplateFile, error) {
	path := s.customTemplateFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return extract.TemplateFile{Templates: []extract.Template{}}, nil
		}
		return extract.TemplateFile{}, err
	}

	var templateFile extract.TemplateFile
	if err := json.Unmarshal(data, &templateFile); err != nil {
		return extract.TemplateFile{}, err
	}
	if templateFile.Templates == nil {
		templateFile.Templates = []extract.Template{}
	}
	return templateFile, nil
}

func (s *Server) saveCustomTemplateFile(templateFile extract.TemplateFile) error {
	data, err := json.MarshalIndent(templateFile, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(s.customTemplateFilePath(), data, 0o644)
}

func (s *Server) upsertCustomTemplate(template extract.Template) error {
	templateFile, err := s.loadCustomTemplateFile()
	if err != nil {
		return err
	}

	for i := range templateFile.Templates {
		if templateFile.Templates[i].Name == template.Name {
			templateFile.Templates[i] = template
			return s.saveCustomTemplateFile(templateFile)
		}
	}

	templateFile.Templates = append(templateFile.Templates, template)
	return s.saveCustomTemplateFile(templateFile)
}

func (s *Server) replaceCustomTemplate(existingName string, replacement extract.Template) error {
	templateFile, err := s.loadCustomTemplateFile()
	if err != nil {
		return err
	}

	for i := range templateFile.Templates {
		if templateFile.Templates[i].Name != existingName {
			continue
		}
		templateFile.Templates[i] = replacement
		return s.saveCustomTemplateFile(templateFile)
	}

	return apperrors.NotFound("template not found: " + existingName)
}

func (s *Server) deleteCustomTemplate(name string) error {
	templateFile, err := s.loadCustomTemplateFile()
	if err != nil {
		return err
	}

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
	return s.saveCustomTemplateFile(templateFile)
}
