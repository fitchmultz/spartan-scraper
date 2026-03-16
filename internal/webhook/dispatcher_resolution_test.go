// Package webhook verifies resolver-driven delivery pinning and redirect hardening.
//
// Purpose:
// - Prove webhook delivery uses the IP set validated before dispatch rather than re-resolving later.
//
// Responsibilities:
// - Exercise dispatcher delivery against resolver-controlled hostname answers.
// - Verify mixed public/private DNS answers are rejected.
// - Verify redirect responses cannot pivot a validated delivery onto a new host.
//
// Scope:
// - Dispatcher security behavior around hostname resolution and redirect handling.
//
// Usage:
// - Run with `go test ./internal/webhook`.
//
// Invariants/Assumptions:
// - Tests reroute pinned public test-net IPs into local httptest servers through an injected dialer.
// - The dispatcher package may use unexported resolver and dialer hooks for deterministic coverage.
package webhook

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
)

type stubResolver struct {
	lookup func(ctx context.Context, network string, host string) ([]netip.Addr, error)
}

func (r stubResolver) LookupNetIP(ctx context.Context, network string, host string) ([]netip.Addr, error) {
	return r.lookup(ctx, network, host)
}

func mustAddr(t *testing.T, raw string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		t.Fatalf("ParseAddr(%q) failed: %v", raw, err)
	}
	return addr
}

func serverAddress(t *testing.T, serverURL string) string {
	t.Helper()
	parsed, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("url.Parse(%q) failed: %v", serverURL, err)
	}
	return parsed.Host
}

func rerouteDialContext(t *testing.T, routes map[string]string, dialed *[]string, mu *sync.Mutex) dialContextFunc {
	t.Helper()
	baseDialer := &net.Dialer{}
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		mu.Lock()
		*dialed = append(*dialed, address)
		mu.Unlock()
		target, ok := routes[address]
		if !ok {
			return nil, fmt.Errorf("unexpected dial target %q", address)
		}
		return baseDialer.DialContext(ctx, network, target)
	}
}

func TestResolveDeliveryTarget_BlocksPrivateDNSAnswers(t *testing.T) {
	resolver := stubResolver{lookup: func(_ context.Context, _ string, host string) ([]netip.Addr, error) {
		if host != "safe.example" {
			return nil, fmt.Errorf("unexpected host %q", host)
		}
		return []netip.Addr{mustAddr(t, "203.0.113.10"), mustAddr(t, "127.0.0.1")}, nil
	}}

	_, err := resolveDeliveryTarget(context.Background(), "http://safe.example/webhook", false, resolver)
	if !IsSSRFError(err) {
		t.Fatalf("expected SSRF validation error, got %v", err)
	}
}

func TestDeliver_PinsResolvedIPsDuringDial(t *testing.T) {
	var (
		received    atomic.Bool
		hostHeader  string
		hostHeaderM sync.Mutex
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Store(true)
		hostHeaderM.Lock()
		hostHeader = r.Host
		hostHeaderM.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var (
		resolverCalls atomic.Int32
		dialed        []string
		dialMu        sync.Mutex
	)

	d := NewDispatcher(Config{MaxRetries: 1})
	d.resolver = stubResolver{lookup: func(_ context.Context, _ string, host string) ([]netip.Addr, error) {
		resolverCalls.Add(1)
		if host != "safe.example" {
			return nil, fmt.Errorf("unexpected host %q", host)
		}
		if resolverCalls.Load() == 1 {
			return []netip.Addr{mustAddr(t, "203.0.113.10")}, nil
		}
		return []netip.Addr{mustAddr(t, "127.0.0.1")}, nil
	}}
	d.dialContext = rerouteDialContext(t, map[string]string{
		"203.0.113.10:8080": serverAddress(t, server.URL),
	}, &dialed, &dialMu)

	err := d.Deliver(context.Background(), "http://safe.example:8080/webhook", testPayload(), "")
	if err != nil {
		t.Fatalf("Deliver() failed: %v", err)
	}
	if !received.Load() {
		t.Fatal("expected pinned webhook request to reach receiver")
	}
	if resolverCalls.Load() != 1 {
		t.Fatalf("expected one resolver call, got %d", resolverCalls.Load())
	}

	dialMu.Lock()
	defer dialMu.Unlock()
	if len(dialed) != 1 {
		t.Fatalf("expected one dial attempt, got %#v", dialed)
	}
	if dialed[0] != "203.0.113.10:8080" {
		t.Fatalf("expected pinned dial to 203.0.113.10:8080, got %#v", dialed)
	}
	hostHeaderM.Lock()
	defer hostHeaderM.Unlock()
	if hostHeader != "safe.example:8080" {
		t.Fatalf("expected host header safe.example:8080, got %q", hostHeader)
	}
}

func TestDeliver_DoesNotFollowRedirects(t *testing.T) {
	var redirected atomic.Bool
	internalReceiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer internalReceiver.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, internalReceiver.URL, http.StatusFound)
	}))
	defer redirector.Close()

	var (
		dialed []string
		dialMu sync.Mutex
	)

	d := NewDispatcher(Config{MaxRetries: 1})
	d.resolver = stubResolver{lookup: func(_ context.Context, _ string, host string) ([]netip.Addr, error) {
		if host != "safe.example" {
			return nil, fmt.Errorf("unexpected host %q", host)
		}
		return []netip.Addr{mustAddr(t, "203.0.113.20")}, nil
	}}
	d.dialContext = rerouteDialContext(t, map[string]string{
		"203.0.113.20:8080": serverAddress(t, redirector.URL),
	}, &dialed, &dialMu)

	err := d.Deliver(context.Background(), "http://safe.example:8080/webhook", testPayload(), "")
	if err == nil {
		t.Fatal("expected redirecting webhook delivery to fail")
	}
	if got := err.Error(); got != "exhausted retries: unexpected status code: 302" {
		t.Fatalf("expected redirect failure, got %q", got)
	}
	if redirected.Load() {
		t.Fatal("expected redirect target to remain untouched")
	}

	dialMu.Lock()
	defer dialMu.Unlock()
	if len(dialed) != 1 || dialed[0] != "203.0.113.20:8080" {
		t.Fatalf("expected only the validated redirector IP to be dialed, got %#v", dialed)
	}
}
