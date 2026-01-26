// Package server contains health CLI command wiring.
//
// It does NOT provide /healthz endpoint (internal/api does); this is a CLI check.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"spartan-scraper/internal/api"
	"spartan-scraper/internal/config"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/store"
)

func RunHealth(ctx context.Context, cfg config.Config, _ []string) int {
	// 1. Try local server healthz
	url := fmt.Sprintf("http://localhost:%s/healthz", cfg.Port)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		var health api.HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err == nil {
			payload, _ := json.MarshalIndent(health, "", "  ")
			fmt.Println(string(payload))
			if health.Status != "ok" {
				return 1
			}
			return 0
		}
	}

	// 2. Fallback local check
	fmt.Println("Local health check (server not responding):")

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Printf("Database: ERROR (%v)\n", err)
		return 1
	}
	defer st.Close()

	if err := st.Ping(ctx); err != nil {
		fmt.Printf("Database: ERROR (%v)\n", err)
	} else {
		fmt.Println("Database: OK")
	}

	if err := fetch.CheckBrowserAvailability(cfg.UsePlaywright); err != nil {
		fmt.Printf("Browser: ERROR (%v)\n", err)
	} else {
		fmt.Println("Browser: OK")
	}
	return 0
}
