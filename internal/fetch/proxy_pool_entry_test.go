// Package fetch provides HTTP and headless browser content fetching capabilities.
package fetch

import (
	"testing"
)

func TestProxyEntry_ToProxyConfig(t *testing.T) {
	entry := ProxyEntry{
		ID:       "test-proxy",
		URL:      "http://proxy.example.com:8080",
		Username: "user",
		Password: "pass",
		Region:   "us-east",
		Tags:     []string{"datacenter"},
		Weight:   10,
	}

	config := entry.ToProxyConfig()

	if config.URL != entry.URL {
		t.Errorf("URL mismatch: got %q, want %q", config.URL, entry.URL)
	}
	if config.Username != entry.Username {
		t.Errorf("Username mismatch: got %q, want %q", config.Username, entry.Username)
	}
	if config.Password != entry.Password {
		t.Errorf("Password mismatch: got %q, want %q", config.Password, entry.Password)
	}
}

func TestProxyStats_SuccessRate(t *testing.T) {
	tests := []struct {
		name      string
		stats     ProxyStats
		want      float64
		tolerance float64
	}{
		{
			name:      "no requests",
			stats:     ProxyStats{},
			want:      100.0,
			tolerance: 0.01,
		},
		{
			name: "all success",
			stats: ProxyStats{
				SuccessCount: 10,
				FailureCount: 0,
			},
			want:      100.0,
			tolerance: 0.01,
		},
		{
			name: "all failure",
			stats: ProxyStats{
				SuccessCount: 0,
				FailureCount: 10,
			},
			want:      0.0,
			tolerance: 0.01,
		},
		{
			name: "mixed",
			stats: ProxyStats{
				SuccessCount: 75,
				FailureCount: 25,
			},
			want:      75.0,
			tolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.stats.SuccessRate()
			if got < tt.want-tt.tolerance || got > tt.want+tt.tolerance {
				t.Errorf("SuccessRate() = %v, want %v (tolerance %v)", got, tt.want, tt.tolerance)
			}
		})
	}
}
