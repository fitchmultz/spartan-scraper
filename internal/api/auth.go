// Package api provides HTTP handlers for authentication profile management endpoints.
// Auth handlers support listing, creating, updating, deleting auth profiles,
// importing/exporting profiles, and managing profile presets.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

func (s *Server) handleAuthProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, apperrors.MethodNotAllowed("method not allowed"))
		return
	}
	vault, err := auth.LoadVault(s.cfg.DataDir)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]interface{}{"profiles": vault.Profiles})
}

func (s *Server) handleAuthProfile(w http.ResponseWriter, r *http.Request) {
	name := extractID(r.URL.Path, "profiles")
	if name == "" {
		writeError(w, apperrors.Validation("name required"))
		return
	}
	switch r.Method {
	case http.MethodPut:
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			writeError(w, apperrors.UnsupportedMediaType("content-type must be application/json"))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		var profile auth.Profile
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&profile); err != nil {
			writeError(w, apperrors.Validation("invalid json: "+err.Error()))
			return
		}
		if profile.Name == "" {
			profile.Name = name
		}
		if profile.Name != name {
			writeError(w, apperrors.Validation("profile name mismatch"))
			return
		}
		if err := validate.ValidateAuthProfileName(profile.Name); err != nil {
			writeError(w, apperrors.Validation(err.Error()))
			return
		}
		if err := auth.UpsertProfile(s.cfg.DataDir, profile); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, profile)
	case http.MethodDelete:
		if err := auth.DeleteProfile(s.cfg.DataDir, name); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		writeError(w, apperrors.MethodNotAllowed("method not allowed"))
	}
}

func (s *Server) handleAuthImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, apperrors.MethodNotAllowed("method not allowed"))
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var payload struct {
		Path string `json:"path"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, apperrors.Validation("invalid json: "+err.Error()))
		return
	}
	if err := auth.ImportVault(s.cfg.DataDir, payload.Path); err != nil {
		if errors.Is(err, auth.ErrInvalidPath) || err.Error() == "path is required" {
			writeError(w, err)
			return
		}
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleAuthExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, apperrors.MethodNotAllowed("method not allowed"))
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var payload struct {
		Path string `json:"path"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, apperrors.Validation("invalid json: "+err.Error()))
		return
	}
	if err := auth.ExportVault(s.cfg.DataDir, payload.Path); err != nil {
		if errors.Is(err, auth.ErrInvalidPath) || err.Error() == "path is required" {
			writeError(w, err)
			return
		}
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}
