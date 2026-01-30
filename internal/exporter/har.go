// Package exporter provides functionality for exporting job results to various formats.
//
// This file implements HAR (HTTP Archive) v1.2 format export for network interception data.
package exporter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// HARLog represents the root of a HAR file.
type HARLog struct {
	Version string     `json:"version"`
	Creator HARCreator `json:"creator"`
	Entries []HAREntry `json:"entries"`
	Pages   []HARPage  `json:"pages,omitempty"`
}

// HARCreator represents the tool that created the HAR.
type HARCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// HARPage represents a page in the HAR (for browser navigation).
type HARPage struct {
	StartedDateTime time.Time      `json:"startedDateTime"`
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	PageTimings     HARPageTimings `json:"pageTimings"`
}

// HARPageTimings represents timing information for a page.
type HARPageTimings struct {
	OnContentLoad float64 `json:"onContentLoad,omitempty"`
	OnLoad        float64 `json:"onLoad,omitempty"`
}

// HAREntry represents a single request/response pair.
type HAREntry struct {
	StartedDateTime time.Time   `json:"startedDateTime"`
	Time            float64     `json:"time"`
	Request         HARRequest  `json:"request"`
	Response        HARResponse `json:"response"`
	Cache           HARCache    `json:"cache"`
	Timings         HARTimings  `json:"timings"`
	Connection      string      `json:"connection,omitempty"`
	ServerIPAddress string      `json:"serverIPAddress,omitempty"`
}

// HARRequest represents a request in HAR format.
type HARRequest struct {
	Method      string          `json:"method"`
	URL         string          `json:"url"`
	HTTPVersion string          `json:"httpVersion"`
	Cookies     []HARCookie     `json:"cookies"`
	Headers     []HARHeader     `json:"headers"`
	QueryString []HARQueryParam `json:"queryString"`
	PostData    *HARPostData    `json:"postData,omitempty"`
	HeadersSize int             `json:"headersSize"`
	BodySize    int             `json:"bodySize"`
}

// HARResponse represents a response in HAR format.
type HARResponse struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Cookies     []HARCookie `json:"cookies"`
	Headers     []HARHeader `json:"headers"`
	Content     HARContent  `json:"content"`
	RedirectURL string      `json:"redirectURL"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int         `json:"bodySize"`
}

// HARCookie represents a cookie in HAR format.
type HARCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Path     string    `json:"path,omitempty"`
	Domain   string    `json:"domain,omitempty"`
	Expires  time.Time `json:"expires,omitempty"`
	HTTPOnly bool      `json:"httpOnly,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
}

// HARHeader represents a header in HAR format.
type HARHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARQueryParam represents a query parameter in HAR format.
type HARQueryParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARPostData represents POST data in HAR format.
type HARPostData struct {
	MimeType string         `json:"mimeType"`
	Text     string         `json:"text"`
	Params   []HARPostParam `json:"params,omitempty"`
}

// HARPostParam represents a POST parameter in HAR format.
type HARPostParam struct {
	Name        string `json:"name"`
	Value       string `json:"value,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}

// HARContent represents response content in HAR format.
type HARContent struct {
	Size        int    `json:"size"`
	Compression int    `json:"compression,omitempty"`
	MimeType    string `json:"mimeType"`
	Text        string `json:"text,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
}

// HARCache represents cache information in HAR format.
type HARCache struct {
	// Empty for now - we don't track cache
}

// HARTimings represents timing information in HAR format.
type HARTimings struct {
	Blocked float64 `json:"blocked,omitempty"`
	DNS     float64 `json:"dns,omitempty"`
	Connect float64 `json:"connect,omitempty"`
	Send    float64 `json:"send,omitempty"`
	Wait    float64 `json:"wait,omitempty"`
	Receive float64 `json:"receive,omitempty"`
	SSL     float64 `json:"ssl,omitempty"`
}

// exportHARStream exports job results with network interception data to HAR format.
func exportHARStream(job model.Job, r io.Reader, w io.Writer) error {
	// Read all data from reader
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read job results: %w", err)
	}

	// Parse the job results to extract intercepted data
	var results []struct {
		URL             string                   `json:"url"`
		Status          int                      `json:"status"`
		HTML            string                   `json:"html"`
		FetchedAt       time.Time                `json:"fetchedAt"`
		InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
	}

	if err := json.Unmarshal(data, &results); err != nil {
		// If we can't decode as array, try single object
		var singleResult struct {
			URL             string                   `json:"url"`
			Status          int                      `json:"status"`
			HTML            string                   `json:"html"`
			FetchedAt       time.Time                `json:"fetchedAt"`
			InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
		}
		if err := json.Unmarshal(data, &singleResult); err != nil {
			return fmt.Errorf("failed to decode job results: %w", err)
		}
		results = []struct {
			URL             string                   `json:"url"`
			Status          int                      `json:"status"`
			HTML            string                   `json:"html"`
			FetchedAt       time.Time                `json:"fetchedAt"`
			InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
		}{singleResult}
	}

	// Build HAR log
	harLog := HARLog{
		Version: "1.2",
		Creator: HARCreator{
			Name:    "Spartan Scraper",
			Version: "0.1.0",
		},
		Entries: []HAREntry{},
		Pages:   []HARPage{},
	}

	// Convert intercepted entries to HAR format
	for _, result := range results {
		// Create a page entry for the main navigation
		pageID := fmt.Sprintf("page_%d", len(harLog.Pages))
		page := HARPage{
			StartedDateTime: result.FetchedAt,
			ID:              pageID,
			Title:           result.URL,
			PageTimings:     HARPageTimings{},
		}
		harLog.Pages = append(harLog.Pages, page)

		// Convert intercepted entries
		for _, entry := range result.InterceptedData {
			harEntry := convertToHAREntry(entry, pageID)
			harLog.Entries = append(harLog.Entries, harEntry)
		}
	}

	// Write HAR to output
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(harLog)
}

// convertToHAREntry converts an InterceptedEntry to HAR format.
func convertToHAREntry(entry fetch.InterceptedEntry, pageID string) HAREntry {
	req := entry.Request
	resp := entry.Response

	// Parse URL for query string extraction
	parsedURL, _ := url.Parse(req.URL)
	if parsedURL == nil {
		parsedURL = &url.URL{}
	}

	// Build HAR request
	harReq := HARRequest{
		Method:      req.Method,
		URL:         req.URL,
		HTTPVersion: "HTTP/1.1", // We don't track the actual version
		Cookies:     extractCookies(req.Headers),
		Headers:     mapToHARHeaders(req.Headers),
		QueryString: extractQueryString(parsedURL),
		HeadersSize: -1, // Not tracked
		BodySize:    int(req.BodySize),
	}

	// Add POST data if present
	if req.Body != "" {
		contentType := req.Headers["Content-Type"]
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		harReq.PostData = &HARPostData{
			MimeType: contentType,
			Text:     req.Body,
		}
	}

	// Build HAR response
	harResp := HARResponse{
		Status:      0,
		StatusText:  "",
		HTTPVersion: "HTTP/1.1",
		Cookies:     []HARCookie{},
		Headers:     []HARHeader{},
		Content: HARContent{
			Size:     0,
			MimeType: "application/octet-stream",
		},
		HeadersSize: -1,
		BodySize:    0,
	}

	if resp != nil {
		harResp.Status = resp.Status
		harResp.StatusText = resp.StatusText
		harResp.Cookies = extractCookies(resp.Headers)
		harResp.Headers = mapToHARHeaders(resp.Headers)
		harResp.BodySize = int(resp.BodySize)

		// Determine content type and encoding
		contentType := resp.Headers["Content-Type"]
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		harResp.Content.MimeType = contentType
		harResp.Content.Size = int(resp.BodySize)

		// Add response body if present
		if resp.Body != "" {
			// Check if content is binary
			if isBinaryContent(contentType) {
				harResp.Content.Encoding = "base64"
				harResp.Content.Text = base64.StdEncoding.EncodeToString([]byte(resp.Body))
			} else {
				harResp.Content.Text = resp.Body
			}
		}
	}

	// Calculate timing
	duration := entry.Duration.Seconds() * 1000 // Convert to milliseconds

	return HAREntry{
		StartedDateTime: req.Timestamp,
		Time:            duration,
		Request:         harReq,
		Response:        harResp,
		Cache:           HARCache{},
		Timings: HARTimings{
			// We don't have detailed timing, so put everything in "wait"
			Wait:    duration,
			Receive: 0,
		},
	}
}

// mapToHARHeaders converts a map of headers to HAR header format.
func mapToHARHeaders(headers map[string]string) []HARHeader {
	result := make([]HARHeader, 0, len(headers))
	for name, value := range headers {
		result = append(result, HARHeader{
			Name:  name,
			Value: value,
		})
	}
	return result
}

// extractCookies extracts cookies from headers.
func extractCookies(headers map[string]string) []HARCookie {
	cookies := []HARCookie{}
	cookieHeader, exists := headers["Cookie"]
	if !exists {
		cookieHeader, exists = headers["cookie"]
	}
	if !exists {
		return cookies
	}

	// Simple cookie parsing
	parts := strings.Split(cookieHeader, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			cookies = append(cookies, HARCookie{
				Name:  kv[0],
				Value: kv[1],
			})
		}
	}
	return cookies
}

// extractQueryString extracts query parameters from URL.
func extractQueryString(u *url.URL) []HARQueryParam {
	params := []HARQueryParam{}
	for key, values := range u.Query() {
		for _, value := range values {
			params = append(params, HARQueryParam{
				Name:  key,
				Value: value,
			})
		}
	}
	return params
}

// isBinaryContent checks if the content type indicates binary data.
func isBinaryContent(contentType string) bool {
	binaryTypes := []string{
		"image/",
		"audio/",
		"video/",
		"application/octet-stream",
		"application/pdf",
		"application/zip",
		"application/gzip",
		"application/x-protobuf",
		"application/x-msgpack",
	}

	lowerType := strings.ToLower(contentType)
	for _, bt := range binaryTypes {
		if strings.Contains(lowerType, bt) {
			return true
		}
	}
	return false
}
