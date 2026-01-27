package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestServer_DefaultBindingIsLocalhost(t *testing.T) {
	t.Setenv("BIND_ADDR", "")
	t.Setenv("PORT", "8741")

	// Ensure defaults apply (no accidental inheritance from developer env).
	t.Setenv("SERVER_READ_HEADER_TIMEOUT_SECONDS", "")
	t.Setenv("SERVER_READ_TIMEOUT_SECONDS", "")
	t.Setenv("SERVER_WRITE_TIMEOUT_SECONDS", "")
	t.Setenv("SERVER_IDLE_TIMEOUT_SECONDS", "")

	cfg := config.Load()
	if cfg.BindAddr != "127.0.0.1" {
		t.Fatalf("expected default BindAddr 127.0.0.1, got %q", cfg.BindAddr)
	}

	srv := newHTTPServer(cfg, http.NewServeMux())
	if srv.Addr != "127.0.0.1:8741" {
		t.Fatalf("expected server Addr %q, got %q", "127.0.0.1:8741", srv.Addr)
	}
}

func TestServer_TimeoutsAreNonZeroAndSane(t *testing.T) {
	t.Setenv("BIND_ADDR", "")
	t.Setenv("PORT", "8741")

	// Force fallback defaults.
	t.Setenv("SERVER_READ_HEADER_TIMEOUT_SECONDS", "")
	t.Setenv("SERVER_READ_TIMEOUT_SECONDS", "")
	t.Setenv("SERVER_WRITE_TIMEOUT_SECONDS", "")
	t.Setenv("SERVER_IDLE_TIMEOUT_SECONDS", "")

	cfg := config.Load()
	srv := newHTTPServer(cfg, http.NewServeMux())

	if srv.ReadHeaderTimeout <= 0 {
		t.Fatalf("expected ReadHeaderTimeout > 0, got %s", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout <= 0 {
		t.Fatalf("expected ReadTimeout > 0, got %s", srv.ReadTimeout)
	}
	if srv.WriteTimeout <= 0 {
		t.Fatalf("expected WriteTimeout > 0, got %s", srv.WriteTimeout)
	}
	if srv.IdleTimeout <= 0 {
		t.Fatalf("expected IdleTimeout > 0, got %s", srv.IdleTimeout)
	}

	// Validate conservative default expectations (tight enough to be meaningful; stable for CI).
	if srv.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("expected ReadHeaderTimeout %s, got %s", 10*time.Second, srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout != 30*time.Second {
		t.Fatalf("expected ReadTimeout %s, got %s", 30*time.Second, srv.ReadTimeout)
	}
	if srv.WriteTimeout != 60*time.Second {
		t.Fatalf("expected WriteTimeout %s, got %s", 60*time.Second, srv.WriteTimeout)
	}
	if srv.IdleTimeout != 120*time.Second {
		t.Fatalf("expected IdleTimeout %s, got %s", 120*time.Second, srv.IdleTimeout)
	}

	// Basic sanity ordering.
	if srv.ReadTimeout < srv.ReadHeaderTimeout {
		t.Fatalf("expected ReadTimeout >= ReadHeaderTimeout (%s < %s)", srv.ReadTimeout, srv.ReadHeaderTimeout)
	}
}

func TestServer_CanOptInToBindAllInterfaces(t *testing.T) {
	t.Setenv("BIND_ADDR", "0.0.0.0")
	t.Setenv("PORT", "9999")

	// Set explicit values to ensure they are used.
	t.Setenv("SERVER_READ_HEADER_TIMEOUT_SECONDS", "11")
	t.Setenv("SERVER_READ_TIMEOUT_SECONDS", "31")
	t.Setenv("SERVER_WRITE_TIMEOUT_SECONDS", "61")
	t.Setenv("SERVER_IDLE_TIMEOUT_SECONDS", "121")

	cfg := config.Load()
	if cfg.BindAddr != "0.0.0.0" {
		t.Fatalf("expected BindAddr %q, got %q", "0.0.0.0", cfg.BindAddr)
	}

	srv := newHTTPServer(cfg, http.NewServeMux())
	if srv.Addr != "0.0.0.0:9999" {
		t.Fatalf("expected server Addr %q, got %q", "0.0.0.0:9999", srv.Addr)
	}

	if srv.ReadHeaderTimeout != 11*time.Second ||
		srv.ReadTimeout != 31*time.Second ||
		srv.WriteTimeout != 61*time.Second ||
		srv.IdleTimeout != 121*time.Second {
		t.Fatalf("expected timeouts to reflect env overrides, got RH=%s R=%s W=%s I=%s",
			srv.ReadHeaderTimeout, srv.ReadTimeout, srv.WriteTimeout, srv.IdleTimeout)
	}
}
