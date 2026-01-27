package apperrors

import "testing"

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal HTTPS URL",
			input:    "https://api.example.com/v1/users",
			expected: "https://api.example.com/v1/users",
		},
		{
			name:     "URL with query parameters",
			input:    "https://api.example.com/v1/users?token=secret123&api_key=abc",
			expected: "https://api.example.com/v1/users",
		},
		{
			name:     "URL with fragment",
			input:    "https://example.com/page#section",
			expected: "https://example.com/page",
		},
		{
			name:     "URL with userinfo",
			input:    "https://user:pass@example.com/path",
			expected: "https://example.com/path",
		},
		{
			name:     "URL with userinfo, query, and fragment",
			input:    "https://user:pass@api.example.com/v1/users?key=val#frag",
			expected: "https://api.example.com/v1/users",
		},
		{
			name:     "relative URL",
			input:    "/api/v1/users",
			expected: "/api/v1/users",
		},
		{
			name:     "relative URL with query",
			input:    "/api/v1/users?token=secret",
			expected: "/api/v1/users",
		},
		{
			name:     "malformed URL",
			input:    "not-a-valid-url",
			expected: "not-a-valid-url",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "URL with port",
			input:    "https://api.example.com:8080/path",
			expected: "https://api.example.com:8080/path",
		},
		{
			name:     "URL with port and query",
			input:    "https://api.example.com:8080/path?token=secret",
			expected: "https://api.example.com:8080/path",
		},
		{
			name:     "HTTP URL",
			input:    "http://example.com/path",
			expected: "http://example.com/path",
		},
		{
			name:     "complex query string",
			input:    "https://api.example.com/search?q=query&page=1&limit=10&sort=desc",
			expected: "https://api.example.com/search",
		},
		{
			name:     "URL with encoded characters in path",
			input:    "https://example.com/path%20with%20spaces",
			expected: "https://example.com/path%20with%20spaces",
		},
		{
			name:     "URL with only userinfo",
			input:    "https://admin:password@example.com",
			expected: "https://example.com",
		},
		{
			name:     "URL with only fragment",
			input:    "https://example.com#anchor",
			expected: "https://example.com",
		},
		{
			name:     "URL with only query",
			input:    "https://example.com?param=value",
			expected: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeURL(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
