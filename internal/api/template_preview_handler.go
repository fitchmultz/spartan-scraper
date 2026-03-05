// Package api implements the REST API server for Spartan Scraper.
// This file handles template preview endpoints for visual selector building.
package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

// DOMNode represents a simplified DOM element for the visual builder
type DOMNode struct {
	Tag        string            `json:"tag"`
	ID         string            `json:"id,omitempty"`
	Classes    []string          `json:"classes,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Text       string            `json:"text,omitempty"`
	Children   []DOMNode         `json:"children,omitempty"`
	Path       string            `json:"path"` // CSS path to this element
	Depth      int               `json:"depth"`
}

// TemplatePreviewResponse returned by the preview endpoint
type TemplatePreviewResponse struct {
	URL       string  `json:"url"`
	Title     string  `json:"title"`
	DOMTree   DOMNode `json:"dom_tree"`
	FetchTime int64   `json:"fetch_time_ms"`
	Fetcher   string  `json:"fetcher"` // "http", "chromedp", or "playwright"
}

// TestSelectorRequest for testing selectors
type TestSelectorRequest struct {
	URL        string `json:"url"`
	Selector   string `json:"selector"`
	Headless   bool   `json:"headless,omitempty"`
	Playwright bool   `json:"playwright,omitempty"`
}

// TestSelectorResponse with matching elements
type TestSelectorResponse struct {
	Selector string       `json:"selector"`
	Matches  int          `json:"matches"`
	Elements []DOMElement `json:"elements"`
	Error    string       `json:"error,omitempty"`
}

// DOMElement represents a matched element in selector testing
type DOMElement struct {
	Tag  string `json:"tag"`
	Text string `json:"text"`
	HTML string `json:"html"` // truncated preview
	Path string `json:"path"`
}

// handleTemplatePreview handles requests to /v1/template-preview
func (s *Server) handleTemplatePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Parse query parameters
	pageURL := r.URL.Query().Get("url")
	if pageURL == "" {
		writeError(w, r, apperrors.Validation("url parameter is required"))
		return
	}

	// Validate URL
	parsedURL, err := url.Parse(pageURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		writeError(w, r, apperrors.Validation("invalid URL format"))
		return
	}

	headless := r.URL.Query().Get("headless") == "true"
	usePlaywright := r.URL.Query().Get("playwright") == "true"

	// Fetch the page
	startTime := time.Now()

	fetchReq := fetch.Request{
		URL:           pageURL,
		Method:        "GET",
		Timeout:       30 * time.Second,
		Headless:      headless,
		UsePlaywright: usePlaywright,
		DataDir:       s.cfg.DataDir,
	}

	fetcher := fetch.NewFetcher(s.cfg.DataDir)
	result, err := fetcher.Fetch(r.Context(), fetchReq)
	if err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to fetch page", err))
		return
	}

	fetchTime := time.Since(startTime).Milliseconds()

	// Parse HTML and build DOM tree
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(result.HTML))
	if err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to parse HTML", err))
		return
	}

	// Get page title
	title := doc.Find("title").Text()

	// Build simplified DOM tree
	rootElement := doc.Find("html")
	if rootElement.Length() == 0 {
		rootElement = doc.Find("body")
	}
	if rootElement.Length() == 0 {
		// Fallback to document root
		domTree := s.buildDOMTree(doc.Selection, "", 0)
		response := TemplatePreviewResponse{
			URL:       pageURL,
			Title:     title,
			DOMTree:   domTree,
			FetchTime: fetchTime,
			Fetcher:   string(result.Engine),
		}
		writeJSON(w, response)
		return
	}

	domTree := s.buildDOMTree(rootElement.First(), "", 0)

	response := TemplatePreviewResponse{
		URL:       pageURL,
		Title:     title,
		DOMTree:   domTree,
		FetchTime: fetchTime,
		Fetcher:   string(result.Engine),
	}

	writeJSON(w, response)
}

// handleTestSelector handles requests to /v1/template-preview/test-selector
func (s *Server) handleTestSelector(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req TestSelectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid request body"))
		return
	}

	if req.URL == "" {
		writeError(w, r, apperrors.Validation("url is required"))
		return
	}
	if req.Selector == "" {
		writeError(w, r, apperrors.Validation("selector is required"))
		return
	}

	// Validate URL
	parsedURL, err := url.Parse(req.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		writeError(w, r, apperrors.Validation("invalid URL format"))
		return
	}

	// Fetch the page
	fetchReq := fetch.Request{
		URL:           req.URL,
		Method:        "GET",
		Timeout:       30 * time.Second,
		Headless:      req.Headless,
		UsePlaywright: req.Playwright,
		DataDir:       s.cfg.DataDir,
	}

	fetcher := fetch.NewFetcher(s.cfg.DataDir)
	result, err := fetcher.Fetch(r.Context(), fetchReq)
	if err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to fetch page", err))
		return
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(result.HTML))
	if err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to parse HTML", err))
		return
	}

	// Test the selector
	selection := doc.Find(req.Selector)
	matches := selection.Length()

	elements := make([]DOMElement, 0, matches)
	selection.Each(func(i int, sel *goquery.Selection) {
		elem := DOMElement{
			Tag:  goquery.NodeName(sel),
			Text: truncateText(sel.Text(), 100),
			HTML: truncateText(getOuterHTML(sel), 200),
			Path: s.getElementPath(sel),
		}
		elements = append(elements, elem)
	})

	response := TestSelectorResponse{
		Selector: req.Selector,
		Matches:  matches,
		Elements: elements,
	}

	writeJSON(w, response)
}

// buildDOMTree builds a simplified DOM tree from a goquery Selection
func (s *Server) buildDOMTree(sel *goquery.Selection, parentPath string, depth int) DOMNode {
	node := DOMNode{
		Depth: depth,
	}

	// Get the underlying node
	goqueryNode := sel.Get(0)
	if goqueryNode == nil {
		return node
	}

	node.Tag = goqueryNode.Data

	// Skip certain tags
	if node.Tag == "script" || node.Tag == "style" || node.Tag == "meta" || node.Tag == "link" {
		return node
	}

	// Build path
	if parentPath == "" {
		node.Path = node.Tag
	} else {
		// Count siblings of same type for nth-child
		siblingIndex := 1
		for prev := goqueryNode.PrevSibling; prev != nil; prev = prev.PrevSibling {
			if prev.Type == goqueryNode.Type && prev.Data == goqueryNode.Data {
				siblingIndex++
			}
		}
		node.Path = parentPath + " > " + node.Tag + ":nth-child(" + strconv.Itoa(siblingIndex) + ")"
	}

	// Extract attributes
	if goqueryNode.Type == 1 { // Element node
		node.Attributes = make(map[string]string)
		for _, attr := range goqueryNode.Attr {
			switch attr.Key {
			case "id":
				node.ID = attr.Val
			case "class":
				node.Classes = strings.Fields(attr.Val)
			default:
				node.Attributes[attr.Key] = attr.Val
			}
		}
	}

	// Extract text (truncated)
	node.Text = truncateText(sel.Text(), 100)

	// Limit depth and children to prevent excessive data
	if depth >= 10 {
		return node
	}

	// Process children
	children := sel.Children()
	childCount := children.Length()
	maxChildren := 100
	if childCount > maxChildren {
		childCount = maxChildren
	}

	node.Children = make([]DOMNode, 0, childCount)
	children.Each(func(i int, childSel *goquery.Selection) {
		if i >= maxChildren {
			return
		}
		childNode := s.buildDOMTree(childSel, node.Path, depth+1)
		if childNode.Tag != "" && childNode.Tag != "script" && childNode.Tag != "style" {
			node.Children = append(node.Children, childNode)
		}
	})

	return node
}

// getElementPath generates a CSS path for an element
func (s *Server) getElementPath(sel *goquery.Selection) string {
	var parts []string

	for {
		if sel.Length() == 0 {
			break
		}

		node := sel.Get(0)
		if node == nil {
			break
		}

		tag := node.Data
		if tag == "" || tag == "html" {
			parts = append([]string{"html"}, parts...)
			break
		}

		// Try to use ID for shorter path
		id := sel.AttrOr("id", "")
		if id != "" {
			parts = append([]string{"#" + id}, parts...)
			break
		}

		// Use tag with nth-child
		siblingIndex := 1
		for prev := node.PrevSibling; prev != nil; prev = prev.PrevSibling {
			if prev.Type == node.Type && prev.Data == node.Data {
				siblingIndex++
			}
		}

		part := tag + ":nth-child(" + strconv.Itoa(siblingIndex) + ")"
		parts = append([]string{part}, parts...)

		// Move to parent
		sel = sel.Parent()
	}

	return strings.Join(parts, " > ")
}

// truncateText truncates text to max length
func truncateText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// getOuterHTML returns the outer HTML of a selection
func getOuterHTML(sel *goquery.Selection) string {
	if sel.Length() == 0 {
		return ""
	}

	node := sel.Get(0)
	if node == nil {
		return ""
	}

	// For element nodes, try to reconstruct outer HTML
	if node.Type == 1 { // Element node
		var buf strings.Builder
		buf.WriteString("<")
		buf.WriteString(node.Data)

		for _, attr := range node.Attr {
			buf.WriteString(" ")
			buf.WriteString(attr.Key)
			buf.WriteString(`="`)
			buf.WriteString(attr.Val)
			buf.WriteString(`"`)
		}

		if node.FirstChild == nil {
			buf.WriteString("/>")
		} else {
			buf.WriteString(">...")
			buf.WriteString("</")
			buf.WriteString(node.Data)
			buf.WriteString(">")
		}

		return buf.String()
	}

	return sel.Text()
}
