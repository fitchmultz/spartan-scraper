package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

func TestHandleProxyPoolStatus_NoPoolConfigured(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/v1/proxy-pool/status", nil)
	rr := httptest.NewRecorder()

	srv.handleProxyPoolStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got ProxyPoolStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() failed: %v", err)
	}

	if got.Strategy != "none" || got.TotalProxies != 0 || got.HealthyProxies != 0 || len(got.Proxies) != 0 {
		t.Fatalf("unexpected empty proxy pool response: %#v", got)
	}
}

func TestHandleProxyPoolStatus_ReturnsProxyStats(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	pool, err := fetch.NewProxyPool(fetch.ProxyPoolConfig{
		DefaultStrategy: "least_used",
		HealthCheck:     fetch.HealthCheckConfig{Enabled: false},
		Proxies: []fetch.ProxyEntry{
			{ID: "proxy-east", URL: "http://127.0.0.1:18080", Region: "us-east", Tags: []string{"residential"}},
			{ID: "proxy-west", URL: "http://127.0.0.1:18081", Region: "us-west", Tags: []string{"datacenter"}},
		},
	})
	if err != nil {
		t.Fatalf("NewProxyPool() failed: %v", err)
	}
	defer pool.Stop()
	srv.manager.SetProxyPool(pool)

	pool.RecordSuccess("proxy-east", 120)
	pool.RecordFailure("proxy-west", fmt.Errorf("dial tcp timeout"))

	req := httptest.NewRequest(http.MethodGet, "/v1/proxy-pool/status", nil)
	rr := httptest.NewRecorder()

	srv.handleProxyPoolStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got ProxyPoolStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() failed: %v", err)
	}

	if got.Strategy != "least_used" {
		t.Fatalf("strategy = %q, want least_used", got.Strategy)
	}
	if got.TotalProxies != 2 || got.HealthyProxies != 2 || len(got.Proxies) != 2 {
		t.Fatalf("unexpected proxy pool summary: %#v", got)
	}

	statusByID := map[string]ProxyStatus{}
	for _, proxy := range got.Proxies {
		statusByID[proxy.ID] = proxy
	}

	east := statusByID["proxy-east"]
	if east.Region != "us-east" || east.SuccessCount != 1 || east.RequestCount != 1 || east.AvgLatencyMs != 120 {
		t.Fatalf("unexpected east proxy status: %#v", east)
	}

	west := statusByID["proxy-west"]
	if west.Region != "us-west" || west.FailureCount != 1 || west.RequestCount != 1 || west.ConsecutiveFails != 1 {
		t.Fatalf("unexpected west proxy status: %#v", west)
	}
}
