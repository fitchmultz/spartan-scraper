package jobs

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestManagerRun_UsesProxyPoolForScrapeJobs(t *testing.T) {
	var proxyHits atomic.Int32
	proxyTransport := &http.Transport{Proxy: nil}

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<html><head><title>proxied page</title></head><body><h1>proxy ok</h1></body></html>")
	}))
	defer target.Close()

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits.Add(1)

		req, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		req.Header = r.Header.Clone()
		resp, err := proxyTransport.RoundTrip(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	defer proxyServer.Close()

	pool, err := fetch.NewProxyPool(fetch.ProxyPoolConfig{
		DefaultStrategy: "round_robin",
		HealthCheck:     fetch.HealthCheckConfig{Enabled: false},
		Proxies: []fetch.ProxyEntry{{
			ID:  "proxy-1",
			URL: proxyServer.URL,
		}},
	})
	if err != nil {
		t.Fatalf("NewProxyPool() failed: %v", err)
	}
	defer pool.Stop()

	m, st, cleanup := setupTestManager(t)
	defer cleanup()
	m.SetProxyPool(pool)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		m.Wait()
	}()
	m.Start(ctx)

	job, err := m.CreateScrapeJob(ctx, target.URL, http.MethodGet, nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob() failed: %v", err)
	}
	if err := m.Enqueue(job); err != nil {
		t.Fatalf("Enqueue() failed: %v", err)
	}

	for i := 0; i < 100; i++ {
		persisted, err := st.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}
		if persisted.Status == model.StatusSucceeded {
			if proxyHits.Load() == 0 {
				t.Fatal("expected scrape request to traverse proxy pool")
			}
			stats, ok := pool.GetProxyStats("proxy-1")
			if !ok {
				t.Fatal("expected proxy stats for proxy-1")
			}
			if stats.SuccessCount == 0 {
				t.Fatalf("expected proxy success stats to be recorded, got %#v", stats)
			}
			return
		}
		if persisted.Status == model.StatusFailed {
			t.Fatalf("job failed unexpectedly: %s", persisted.Error)
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("job did not complete within expected time")
}
