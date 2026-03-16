// Package webhook resolves and pins webhook delivery targets.
//
// Purpose:
// - Resolve webhook URLs into a validated, dialable target plan before delivery.
//
// Responsibilities:
// - Parse and validate webhook URL syntax.
// - Resolve hostname targets into a stable set of IPs for the lifetime of one delivery.
// - Reject private/internal targets unless AllowInternal is enabled.
// - Build request-scoped HTTP clients that pin outbound dialing to the validated IP set.
//
// Scope:
// - Webhook URL resolution and request-scoped HTTP transport construction only.
//
// Usage:
// - Called by Dispatcher before retries begin so every attempt uses the same resolved IP set.
//
// Invariants/Assumptions:
// - Redirects are disabled so a validated target cannot pivot to a new host mid-request.
// - Webhook delivery does not honor outbound proxy settings; direct dialing is required for IP pinning.
package webhook

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

type ipResolver interface {
	LookupNetIP(ctx context.Context, network string, host string) ([]netip.Addr, error)
}

type systemIPResolver struct {
	resolver *net.Resolver
}

func (r systemIPResolver) LookupNetIP(ctx context.Context, network string, host string) ([]netip.Addr, error) {
	return r.resolver.LookupNetIP(ctx, network, host)
}

type dialContextFunc func(ctx context.Context, network string, address string) (net.Conn, error)

type deliveryTarget struct {
	hostname      string
	port          string
	dialAddresses []string
	tlsServerName string
}

func resolveDeliveryTarget(ctx context.Context, rawURL string, allowInternal bool, resolver ipResolver) (deliveryTarget, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if resolver == nil {
		resolver = systemIPResolver{resolver: net.DefaultResolver}
	}

	u, err := parseWebhookURL(rawURL)
	if err != nil {
		return deliveryTarget{}, err
	}

	host := u.Hostname()
	if isLocalhost(host) && !allowInternal {
		return deliveryTarget{}, SSRFError
	}

	port := u.Port()
	if port == "" {
		port = defaultPortForScheme(u.Scheme)
	}

	if literalIP, err := netip.ParseAddr(host); err == nil {
		literalIP = literalIP.Unmap()
		if !allowInternal && isPrivateIP(literalIP) {
			return deliveryTarget{}, SSRFError
		}
		return deliveryTarget{
			hostname:      host,
			port:          port,
			dialAddresses: []string{net.JoinHostPort(literalIP.String(), port)},
			tlsServerName: tlsServerName(u, literalIP),
		}, nil
	}

	resolvedIPs, err := resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return deliveryTarget{}, apperrors.Wrap(apperrors.KindValidation, "failed to resolve webhook URL hostname", err)
	}
	if len(resolvedIPs) == 0 {
		return deliveryTarget{}, apperrors.Validation("failed to resolve webhook URL hostname")
	}

	dialAddresses := make([]string, 0, len(resolvedIPs))
	seen := make(map[string]struct{}, len(resolvedIPs))
	for _, resolvedIP := range resolvedIPs {
		resolvedIP = resolvedIP.Unmap()
		if !resolvedIP.IsValid() {
			continue
		}
		if !allowInternal && isPrivateIP(resolvedIP) {
			return deliveryTarget{}, SSRFError
		}
		dialAddress := net.JoinHostPort(resolvedIP.String(), port)
		if _, ok := seen[dialAddress]; ok {
			continue
		}
		seen[dialAddress] = struct{}{}
		dialAddresses = append(dialAddresses, dialAddress)
	}
	if len(dialAddresses) == 0 {
		return deliveryTarget{}, apperrors.Validation("failed to resolve webhook URL hostname")
	}

	return deliveryTarget{
		hostname:      host,
		port:          port,
		dialAddresses: dialAddresses,
		tlsServerName: tlsServerName(u, netip.Addr{}),
	}, nil
}

func parseWebhookURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindValidation, "invalid webhook URL", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, apperrors.Validation("webhook URL must use http or https scheme")
	}
	if u.Hostname() == "" {
		return nil, apperrors.Validation("webhook URL must have a host")
	}
	return u, nil
}

func defaultPortForScheme(scheme string) string {
	switch scheme {
	case "https":
		return "443"
	default:
		return "80"
	}
}

func tlsServerName(u *url.URL, literalIP netip.Addr) string {
	if u.Scheme != "https" {
		return ""
	}
	if literalIP.IsValid() {
		return ""
	}
	return u.Hostname()
}

func newPinnedHTTPClient(timeout time.Duration, target deliveryTarget, dialContext dialContextFunc, baseTLSConfig *tls.Config) (*http.Client, func()) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = newPinnedDialContext(target, dialContext)
	transport.DialTLSContext = nil
	if target.tlsServerName != "" {
		tlsConfig := &tls.Config{}
		if baseTLSConfig != nil {
			tlsConfig = baseTLSConfig.Clone()
		}
		tlsConfig.ServerName = target.tlsServerName
		transport.TLSClientConfig = tlsConfig
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return client, transport.CloseIdleConnections
}

func newPinnedDialContext(target deliveryTarget, dialContext dialContextFunc) dialContextFunc {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		if !matchesTargetAddress(address, target) {
			return nil, fmt.Errorf("webhook transport blocked unexpected dial target %q", address)
		}
		var lastErr error
		for _, dialAddress := range target.dialAddresses {
			conn, err := dialContext(ctx, network, dialAddress)
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("no resolved webhook addresses available for %s", target.hostname)
		}
		return nil, lastErr
	}
}

func matchesTargetAddress(address string, target deliveryTarget) bool {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	return strings.EqualFold(host, target.hostname) && port == target.port
}
