// Package api implements the REST API server for Spartan Scraper.
//
// This file contains handlers for form detection and filling endpoints.
// It provides:
//   - POST /v1/forms/detect - Detect forms on a page
//   - POST /v1/forms/fill - Fill and optionally submit a form
//
// These endpoints use headless browser automation via chromedp.
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

// FormDetectRequest represents a request to detect forms on a page.
type FormDetectRequest struct {
	URL      string `json:"url"`
	FormType string `json:"formType,omitempty"`
	Headless bool   `json:"headless"`
}

// FormDetectResponse represents the response from form detection.
type FormDetectResponse struct {
	URL           string               `json:"url"`
	Forms         []fetch.DetectedForm `json:"forms"`
	FormCount     int                  `json:"formCount"`
	DetectedTypes []string             `json:"detectedTypes"`
}

// FormFillRequest represents a request to fill a form.
type FormFillRequest struct {
	URL            string            `json:"url"`
	FormSelector   string            `json:"formSelector,omitempty"`
	Fields         map[string]string `json:"fields"`
	Submit         bool              `json:"submit"`
	WaitFor        string            `json:"waitFor,omitempty"`
	Headless       bool              `json:"headless"`
	TimeoutSeconds int               `json:"timeoutSeconds,omitempty"`
	FormTypeFilter string            `json:"formTypeFilter,omitempty"`
}

// FormFillResponse represents the response from form filling.
type FormFillResponse struct {
	Success       bool                 `json:"success"`
	FormSelector  string               `json:"formSelector"`
	FormType      fetch.FormType       `json:"formType,omitempty"`
	FilledFields  []string             `json:"filledFields"`
	Errors        []string             `json:"errors,omitempty"`
	PageURL       string               `json:"pageUrl,omitempty"`
	PageHTML      string               `json:"pageHtml,omitempty"`
	DetectedForms []fetch.DetectedForm `json:"detectedForms,omitempty"`
}

// handleForms routes form-related requests to the appropriate handler.
func (s *Server) handleForms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		switch r.URL.Path {
		case "/v1/forms/detect":
			s.handleFormDetect(w, r)
		case "/v1/forms/fill":
			s.handleFormFill(w, r)
		default:
			writeError(w, r, apperrors.NotFound("endpoint not found"))
		}
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleFormDetect handles POST /v1/forms/detect.
func (s *Server) handleFormDetect(w http.ResponseWriter, r *http.Request) {
	var req FormDetectRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	if req.URL == "" {
		writeError(w, r, apperrors.Validation("url is required"))
		return
	}

	// Create form filler
	filler := fetch.NewFormFiller(nil)

	// Set default timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	detectReq := fetch.FormDetectRequest{
		URL:      req.URL,
		FormType: req.FormType,
		Headless: req.Headless,
	}

	result, err := filler.Detect(ctx, detectReq)
	if err != nil {
		writeError(w, r, apperrors.Internal(fmt.Sprintf("failed to detect forms: %v", err)))
		return
	}

	// Convert to API response
	response := FormDetectResponse{
		URL:           result.URL,
		Forms:         result.Forms,
		FormCount:     result.FormCount,
		DetectedTypes: result.DetectedTypes,
	}

	writeJSONStatus(w, http.StatusOK, response)
}

// handleFormFill handles POST /v1/forms/fill.
func (s *Server) handleFormFill(w http.ResponseWriter, r *http.Request) {
	var req FormFillRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	if req.URL == "" {
		writeError(w, r, apperrors.Validation("url is required"))
		return
	}

	// Set default timeout
	timeout := req.TimeoutSeconds
	if timeout == 0 {
		timeout = 30
	}

	// Create form filler
	filler := fetch.NewFormFiller(nil)

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeout)*time.Second)
	defer cancel()

	fillReq := fetch.FormFillRequest{
		URL:            req.URL,
		FormSelector:   req.FormSelector,
		Fields:         req.Fields,
		Submit:         req.Submit,
		WaitFor:        req.WaitFor,
		Timeout:        time.Duration(timeout) * time.Second,
		Headless:       req.Headless,
		FormTypeFilter: req.FormTypeFilter,
	}

	result, err := filler.FillForm(ctx, fillReq)
	if err != nil && result == nil {
		writeError(w, r, apperrors.Internal(fmt.Sprintf("failed to fill form: %v", err)))
		return
	}

	// Convert to API response
	response := FormFillResponse{
		Success:       result.Success,
		FormSelector:  result.FormSelector,
		FormType:      result.FormType,
		FilledFields:  result.FilledFields,
		Errors:        result.Errors,
		PageURL:       result.PageURL,
		PageHTML:      result.PageHTML,
		DetectedForms: result.DetectedForms,
	}

	if result.Success {
		writeJSONStatus(w, http.StatusOK, response)
	} else {
		writeJSONStatus(w, http.StatusUnprocessableEntity, response)
	}
}
