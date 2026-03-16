// Package webhook classifies and validates webhook URLs for SSRF-safe delivery.
//
// Purpose:
// - Provide shared SSRF validation helpers for webhook-target configuration and delivery.
//
// Responsibilities:
// - Block localhost, private, link-local, and loopback IP ranges by default.
// - Resolve hostname targets before delivery planning so unsafe answers are rejected.
// - Sanitize webhook URLs for safe logging.
// - Classify SSRF validation failures with a stable sentinel error.
//
// Scope:
// - Webhook URL validation and IP-range classification only.
//
// Usage:
// - Dispatcher calls ValidateURL/resolveDeliveryTarget before making outbound webhook requests.
// - Other packages may reuse ValidateURL when they need the same webhook URL safety policy.
//
// Invariants/Assumptions:
// - Hostname validation is fail-closed when DNS resolution fails.
// - DNS-rebinding protection requires delivery-time IP pinning in addition to preflight validation.
package webhook

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// SSRFError is returned when a URL fails SSRF validation.
var SSRFError = apperrors.Validation("webhook URL targets internal/private address")

// Private CIDR ranges that should be blocked by default.
// These include RFC1918 private addresses, link-local, and loopback ranges.
var privateCIDRs = []string{
	// RFC1918 private addresses
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	// Link-local / APIPA (includes AWS IMDS at 169.254.169.254)
	"169.254.0.0/16",
	// Loopback
	"127.0.0.0/8",
	// IPv6 loopback
	"::1/128",
	// IPv6 unique local addresses
	"fc00::/7",
	// IPv6 link-local
	"fe80::/10",
}

// Parsed CIDR blocks for efficient matching.
var parsedPrivateCIDRs []*net.IPNet

func init() {
	for _, cidr := range privateCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err == nil {
			parsedPrivateCIDRs = append(parsedPrivateCIDRs, ipnet)
		}
	}
}

// localhostNames contains common localhost hostnames that should be blocked.
var localhostNames = []string{
	"localhost",
	"localhost.localdomain",
	"ip6-localhost",
	"ip6-loopback",
}

// ValidateURL checks if a webhook URL is safe to request.
//
// If allowInternal is true, private IP ranges and localhost are allowed.
// This should only be enabled in trusted environments.
//
// Returns an apperrors.KindValidation error if the URL is blocked.
func ValidateURL(rawURL string, allowInternal bool) error {
	_, err := resolveDeliveryTarget(context.Background(), rawURL, allowInternal, systemIPResolver{resolver: net.DefaultResolver})
	return err
}

// isLocalhost checks if the hostname is a localhost variant.
func isLocalhost(host string) bool {
	lowerHost := strings.ToLower(host)

	for _, name := range localhostNames {
		if lowerHost == name {
			return true
		}
	}

	// Check for 127.x.x.x format hostnames (e.g., 127.0.0.1, 127.1.0.1)
	if strings.HasPrefix(lowerHost, "127.") {
		return true
	}

	// Check for ::1 (IPv6 loopback)
	if lowerHost == "::1" || lowerHost == "[::1]" {
		return true
	}

	// Check for 0.0.0.0 or just 0
	if lowerHost == "0.0.0.0" || lowerHost == "0" {
		return true
	}

	return false
}

// isPrivateIP checks if the IP address is in a private/restricted range.
func isPrivateIP(ip netip.Addr) bool {
	// Unmap IPv4-mapped IPv6 addresses for consistent checking
	ip = ip.Unmap()

	// Check against all private CIDR ranges
	ipBytes := ip.AsSlice()
	for _, cidr := range parsedPrivateCIDRs {
		if cidr.Contains(ipBytes) {
			return true
		}
	}

	return false
}

// SanitizeURL removes credentials from a URL for safe logging.
// Returns the URL with any userinfo redacted.
func SanitizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		// If we can't parse, return a redacted placeholder
		return "[invalid-url]"
	}

	if u.User != nil {
		// Redact credentials
		u.User = url.UserPassword("[REDACTED]", "[REDACTED]")
	}

	return u.String()
}

// IsSSRFError checks if an error is an SSRF validation error.
func IsSSRFError(err error) bool {
	if err == nil {
		return false
	}
	return err == SSRFError || errors.Is(err, SSRFError)
}
