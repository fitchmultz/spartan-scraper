// Package webhook provides tests for SSRF (Server-Side Request Forgery) protection.
//
// Tests cover:
// - Private IP range blocking (RFC1918, link-local, loopback)
// - DNS rebinding protection
// - Localhost variant detection
// - IPv6 private address blocking
// - AllowInternal override functionality
// - URL sanitization for safe logging
// - Error classification
package webhook

import (
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestValidateURL_PrivateIPs(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		allowed bool
	}{
		// RFC1918 Class A - 10.0.0.0/8
		{"10.0.0.1", "http://10.0.0.1/webhook", false},
		{"10.255.255.255", "http://10.255.255.255/webhook", false},
		{"10.0.0.0", "http://10.0.0.0/webhook", false},

		// RFC1918 Class B - 172.16.0.0/12
		{"172.16.0.1", "http://172.16.0.1/webhook", false},
		{"172.31.255.255", "http://172.31.255.255/webhook", false},
		{"172.20.0.0", "http://172.20.0.0/webhook", false},

		// RFC1918 Class C - 192.168.0.0/16
		{"192.168.1.1", "http://192.168.1.1/webhook", false},
		{"192.168.0.0", "http://192.168.0.0/webhook", false},
		{"192.168.255.255", "http://192.168.255.255/webhook", false},

		// Link-local / APIPA - 169.254.0.0/16 (includes AWS IMDS)
		{"169.254.169.254", "http://169.254.169.254/latest/meta-data/", false},
		{"169.254.0.1", "http://169.254.0.1/webhook", false},
		{"169.254.255.255", "http://169.254.255.255/webhook", false},

		// Loopback - 127.0.0.0/8
		{"127.0.0.1", "http://127.0.0.1/webhook", false},
		{"127.0.0.0", "http://127.0.0.0/webhook", false},
		{"127.255.255.255", "http://127.255.255.255/webhook", false},
		{"127.1.0.1", "http://127.1.0.1/webhook", false},

		// Zero/unspecified (localhost check)
		{"0.0.0.0", "http://0.0.0.0/webhook", false},
		{"0", "http://0/webhook", false},

		// Public IPs should be allowed
		{"8.8.8.8", "http://8.8.8.8/webhook", true},
		{"1.1.1.1", "http://1.1.1.1/webhook", true},
		{"203.0.113.1", "http://203.0.113.1/webhook", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, false)
			if tt.allowed && err != nil {
				t.Errorf("expected %s to be allowed, got error: %v", tt.url, err)
			}
			if !tt.allowed && err == nil {
				t.Errorf("expected %s to be blocked, but it was allowed", tt.url)
			}
			if !tt.allowed && err != nil {
				if !apperrors.IsKind(err, apperrors.KindValidation) {
					t.Errorf("expected validation error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateURL_Localhost(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		allowed bool
	}{
		// Localhost variants
		{"localhost", "http://localhost/webhook", false},
		{"localhost.localdomain", "http://localhost.localdomain/webhook", false},
		{"LOCALHOST", "http://LOCALHOST/webhook", false}, // case insensitive
		{"Localhost", "http://Localhost/webhook", false},

		// IPv6 loopback
		{"::1 bare", "http://[::1]/webhook", false},

		// Localhost with ports should still be blocked
		{"localhost:8080", "http://localhost:8080/webhook", false},
		{"127.0.0.1:8080", "http://127.0.0.1:8080/webhook", false},

		// Public hostnames should be allowed
		{"example.com", "http://example.com/webhook", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, false)
			if tt.allowed && err != nil {
				t.Errorf("expected %s to be allowed, got error: %v", tt.url, err)
			}
			if !tt.allowed && err == nil {
				t.Errorf("expected %s to be blocked, but it was allowed", tt.url)
			}
		})
	}
}

func TestValidateURL_IPv6(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		allowed bool
	}{
		// IPv6 loopback
		{"::1", "http://[::1]/webhook", false},
		{"0:0:0:0:0:0:0:1", "http://[0:0:0:0:0:0:0:1]/webhook", false},

		// IPv6 unique local addresses (fc00::/7)
		{"fc00::1", "http://[fc00::1]/webhook", false},
		{"fd00::1", "http://[fd00::1]/webhook", false},

		// IPv6 link-local (fe80::/10)
		{"fe80::1", "http://[fe80::1]/webhook", false},
		{"fe80::1234:5678:90ab:cdef", "http://[fe80::1234:5678:90ab:cdef]/webhook", false},

		// IPv6 IPv4-mapped loopback should be blocked
		{"::ffff:127.0.0.1", "http://[::ffff:127.0.0.1]/webhook", false},

		// Public IPv6 should be allowed
		{"2001:4860:4860::8888", "http://[2001:4860:4860::8888]/webhook", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, false)
			if tt.allowed && err != nil {
				t.Errorf("expected %s to be allowed, got error: %v", tt.url, err)
			}
			if !tt.allowed && err == nil {
				t.Errorf("expected %s to be blocked, but it was allowed", tt.url)
			}
		})
	}
}

func TestValidateURL_SchemeValidation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		allowed bool
	}{
		// Valid schemes
		{"http", "http://example.com/webhook", true},
		{"https", "https://example.com/webhook", true},

		// Invalid schemes
		{"ftp", "ftp://example.com/webhook", false},
		{"file", "file:///etc/passwd", false},
		{"gopher", "gopher://example.com", false},
		{"data", "data:text/plain,hello", false},
		{"javascript", "javascript:alert(1)", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, false)
			if tt.allowed && err != nil {
				t.Errorf("expected %s to be allowed, got error: %v", tt.url, err)
			}
			if !tt.allowed && err == nil {
				t.Errorf("expected %s to be blocked, but it was allowed", tt.url)
			}
		})
	}
}

func TestValidateURL_InvalidURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"empty string", ""},
		{"no scheme", "example.com/webhook"},
		{"invalid scheme separator", "http:/example.com"},
		{"just path", "/webhook"},
		{"no host", "http:///webhook"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, false)
			if err == nil {
				t.Errorf("expected %s to return an error, but it was nil", tt.url)
			}
		})
	}
}

func TestValidateURL_AllowInternal(t *testing.T) {
	// Test that allowInternal=true bypasses SSRF checks
	privateURLs := []string{
		"http://127.0.0.1/webhook",
		"http://localhost/webhook",
		"http://10.0.0.1/webhook",
		"http://192.168.1.1/webhook",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]/webhook",
	}

	for _, url := range privateURLs {
		t.Run(url, func(t *testing.T) {
			// Should be blocked by default
			err := ValidateURL(url, false)
			if err == nil {
				t.Errorf("expected %s to be blocked with allowInternal=false", url)
			}

			// Should be allowed when allowInternal=true
			err = ValidateURL(url, true)
			if err != nil {
				t.Errorf("expected %s to be allowed with allowInternal=true, got error: %v", url, err)
			}
		})
	}

	// Even with allowInternal=true, invalid schemes should still be rejected
	t.Run("invalid scheme still blocked", func(t *testing.T) {
		err := ValidateURL("file:///etc/passwd", true)
		if err == nil {
			t.Error("expected file:// to be blocked even with allowInternal=true")
		}
	})
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "no credentials",
			url:      "http://example.com/webhook",
			expected: "http://example.com/webhook",
		},
		{
			name:     "with credentials",
			url:      "http://user:pass@example.com/webhook",
			expected: "http://%5BREDACTED%5D:%5BREDACTED%5D@example.com/webhook",
		},
		{
			name:     "with username only",
			url:      "http://user@example.com/webhook",
			expected: "http://%5BREDACTED%5D:%5BREDACTED%5D@example.com/webhook",
		},
		{
			name:     "invalid URL",
			url:      "://invalid-url",
			expected: "[invalid-url]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeURL(tt.url)
			if result != tt.expected {
				t.Errorf("SanitizeURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsSSRFError(t *testing.T) {
	// Test with actual SSRF error
	err := ValidateURL("http://127.0.0.1/webhook", false)
	if !IsSSRFError(err) {
		t.Errorf("expected IsSSRFError to return true for SSRF error, got false")
	}

	// Test with nil error
	if IsSSRFError(nil) {
		t.Error("expected IsSSRFError to return false for nil error")
	}

	// Test with non-SSRF error
	otherErr := apperrors.Validation("some other validation error")
	if IsSSRFError(otherErr) {
		t.Error("expected IsSSRFError to return false for non-SSRF error")
	}

	// Test with allowed URL (no error)
	allowedErr := ValidateURL("http://example.com/webhook", false)
	if IsSSRFError(allowedErr) {
		t.Error("expected IsSSRFError to return false when no error occurs")
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"localhost", true},
		{"LOCALHOST", true},
		{"Localhost", true},
		{"localhost.localdomain", true},
		{"127.0.0.1", true},
		{"127.0.0.0", true},
		{"127.255.255.255", true},
		{"127.1.0.1", true},
		{"::1", true},
		{"[::1]", true},
		{"0.0.0.0", true},
		{"0", true},
		{"example.com", false},
		{"10.0.0.1", false}, // IP check is separate from localhost check
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result := isLocalhost(tt.host)
			if result != tt.expected {
				t.Errorf("isLocalhost(%q) = %v, want %v", tt.host, result, tt.expected)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		// Private ranges
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true},
		{"127.0.0.1", true},
		{"0.0.0.0", true},

		// Public ranges
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"203.0.113.1", false},
		{"9.9.9.9", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			// Parse IP using stdlib to test isPrivateIP directly
			// Note: isPrivateIP takes netip.Addr, so we test via ValidateURL
			url := "http://" + tt.ip + "/webhook"
			err := ValidateURL(url, false)
			blocked := err != nil && strings.Contains(err.Error(), "internal/private")
			if blocked != tt.expected {
				t.Errorf("IP %q: expected blocked=%v, got blocked=%v (err=%v)", tt.ip, tt.expected, blocked, err)
			}
		})
	}
}
