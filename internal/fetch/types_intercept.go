// Package fetch provides HTTP and headless browser content fetching capabilities.
// Network interception types for capturing XHR/Fetch API traffic.
package fetch

import "time"

// InterceptedResourceType represents the type of network resource.
type InterceptedResourceType string

const (
	ResourceTypeXHR        InterceptedResourceType = "xhr"
	ResourceTypeFetch      InterceptedResourceType = "fetch"
	ResourceTypeDocument   InterceptedResourceType = "document"
	ResourceTypeScript     InterceptedResourceType = "script"
	ResourceTypeStylesheet InterceptedResourceType = "stylesheet"
	ResourceTypeImage      InterceptedResourceType = "image"
	ResourceTypeMedia      InterceptedResourceType = "media"
	ResourceTypeFont       InterceptedResourceType = "font"
	ResourceTypeWebSocket  InterceptedResourceType = "websocket"
	ResourceTypeOther      InterceptedResourceType = "other"
)

// NetworkInterceptConfig defines configuration for network request/response interception.
// Used to capture XHR/Fetch API traffic from SPAs for API scraping.
type NetworkInterceptConfig struct {
	Enabled             bool                      `json:"enabled"`             // Toggle interception
	URLPatterns         []string                  `json:"urlPatterns"`         // Glob patterns for URLs to intercept (e.g., "**/api/**", "*.json")
	ResourceTypes       []InterceptedResourceType `json:"resourceTypes"`       // Resource types to capture
	CaptureRequestBody  bool                      `json:"captureRequestBody"`  // Whether to capture request bodies
	CaptureResponseBody bool                      `json:"captureResponseBody"` // Whether to capture response bodies
	MaxBodySize         int64                     `json:"maxBodySize"`         // Max bytes to capture per body (default 1MB)
	MaxEntries          int                       `json:"maxEntries"`          // Max number of entries to capture (default 1000)
}

// DefaultNetworkInterceptConfig returns a default configuration with sensible limits.
func DefaultNetworkInterceptConfig() NetworkInterceptConfig {
	return NetworkInterceptConfig{
		Enabled:             false,
		URLPatterns:         []string{},
		ResourceTypes:       []InterceptedResourceType{ResourceTypeXHR, ResourceTypeFetch},
		CaptureRequestBody:  true,
		CaptureResponseBody: true,
		MaxBodySize:         1024 * 1024, // 1MB
		MaxEntries:          1000,
	}
}

// InterceptedRequest represents a captured network request.
type InterceptedRequest struct {
	RequestID    string                  `json:"requestId"`    // Unique identifier
	URL          string                  `json:"url"`          // Request URL
	Method       string                  `json:"method"`       // HTTP method
	Headers      map[string]string       `json:"headers"`      // Request headers
	Body         string                  `json:"body"`         // Request body (base64 if binary)
	BodySize     int64                   `json:"bodySize"`     // Original body size
	Timestamp    time.Time               `json:"timestamp"`    // When request was sent
	ResourceType InterceptedResourceType `json:"resourceType"` // Type of resource
}

// InterceptedResponse represents a captured network response.
type InterceptedResponse struct {
	RequestID  string            `json:"requestId"`  // Matches request
	Status     int               `json:"status"`     // HTTP status code
	StatusText string            `json:"statusText"` // HTTP status text
	Headers    map[string]string `json:"headers"`    // Response headers
	Body       string            `json:"body"`       // Response body (base64 if binary)
	BodySize   int64             `json:"bodySize"`   // Size of response body
	Timestamp  time.Time         `json:"timestamp"`  // When response received
}

// InterceptedEntry combines a request/response pair with timing data.
type InterceptedEntry struct {
	Request  InterceptedRequest   `json:"request"`
	Response *InterceptedResponse `json:"response,omitempty"` // nil if response not received
	Duration time.Duration        `json:"duration"`           // Time between request and response
}
