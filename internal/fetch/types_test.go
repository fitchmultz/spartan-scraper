// Package fetch provides tests for fetch types and utilities.
// Tests cover authentication query parameter application to URLs.
// Does NOT test request execution or response handling.
package fetch

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestApplyAuthQuery(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		query    map[string]string
		want     string
		validate func(t *testing.T, got string)
	}{
		{
			name:   "empty query map returns original",
			rawURL: "https://example.com",
			query:  nil,
			want:   "https://example.com",
		},
		{
			name:   "empty query map returns original with empty map",
			rawURL: "https://example.com",
			query:  map[string]string{},
			want:   "https://example.com",
		},
		{
			name:   "adds params to URL without query",
			rawURL: "https://example.com",
			query:  map[string]string{"key": "value"},
			want:   "https://example.com?key=value",
		},
		{
			name:   "appends params to URL with existing query",
			rawURL: "https://example.com?existing=param",
			query:  map[string]string{"new": "value"},
			want:   "https://example.com?existing=param&new=value",
		},
		{
			name:   "preserves fragment",
			rawURL: "https://example.com#section",
			query:  map[string]string{"key": "value"},
			want:   "https://example.com?key=value#section",
		},
		{
			name:   "special characters are URL encoded",
			rawURL: "https://example.com",
			query:  map[string]string{"key": "value with spaces"},
			want:   "https://example.com?key=value+with+spaces",
		},
		{
			name:   "empty key is skipped",
			rawURL: "https://example.com",
			query:  map[string]string{"": "value", "valid": "data"},
			want:   "https://example.com?valid=data",
		},
		{
			name:   "multiple query params",
			rawURL: "https://example.com",
			query:  map[string]string{"a": "1", "b": "2", "c": "3"},
			want:   "https://example.com?a=1&b=2&c=3",
		},
		{
			name:   "invalid URL returns original",
			rawURL: "://not-a-url",
			query:  map[string]string{"key": "value"},
			want:   "://not-a-url",
		},
		{
			name:   "URL with duplicate params uses new value",
			rawURL: "https://example.com?key=old",
			query:  map[string]string{"key": "new"},
			want:   "https://example.com?key=new",
		},
		{
			name:   "complex URL with path and existing query",
			rawURL: "https://api.example.com/v1/users?page=2&limit=10",
			query:  map[string]string{"filter": "active"},
			validate: func(t *testing.T, got string) {
				parsed, err := url.Parse(got)
				if err != nil {
					t.Errorf("failed to parse result: %v", err)
					return
				}
				values := parsed.Query()
				if values.Get("page") != "2" || values.Get("limit") != "10" || values.Get("filter") != "active" {
					t.Errorf("URL missing or wrong params: %v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyAuthQuery(tt.rawURL, tt.query)
			if tt.validate != nil {
				tt.validate(t, got)
			} else if got != tt.want {
				t.Errorf("ApplyAuthQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScreenshotConfig_JSON(t *testing.T) {
	tests := []struct {
		name   string
		config ScreenshotConfig
		want   string
	}{
		{
			name:   "default config",
			config: ScreenshotConfig{},
			want:   `{"enabled":false,"fullPage":false,"format":""}`,
		},
		{
			name: "enabled PNG screenshot",
			config: ScreenshotConfig{
				Enabled:  true,
				FullPage: false,
				Format:   ScreenshotFormatPNG,
			},
			want: `{"enabled":true,"fullPage":false,"format":"png"}`,
		},
		{
			name: "full page JPEG screenshot with quality",
			config: ScreenshotConfig{
				Enabled:  true,
				FullPage: true,
				Format:   ScreenshotFormatJPEG,
				Quality:  85,
			},
			want: `{"enabled":true,"fullPage":true,"format":"jpeg","quality":85}`,
		},
		{
			name: "custom viewport dimensions",
			config: ScreenshotConfig{
				Enabled: true,
				Format:  ScreenshotFormatPNG,
				Width:   1920,
				Height:  1080,
			},
			want: `{"enabled":true,"fullPage":false,"format":"png","width":1920,"height":1080}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("json.Marshal() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestScreenshotConfig_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    ScreenshotConfig
		wantErr bool
	}{
		{
			name: "valid PNG config",
			json: `{"enabled":true,"fullPage":true,"format":"png"}`,
			want: ScreenshotConfig{
				Enabled:  true,
				FullPage: true,
				Format:   ScreenshotFormatPNG,
			},
		},
		{
			name: "valid JPEG config with quality",
			json: `{"enabled":true,"format":"jpeg","quality":90}`,
			want: ScreenshotConfig{
				Enabled: true,
				Format:  ScreenshotFormatJPEG,
				Quality: 90,
			},
		},
		{
			name:    "invalid JSON",
			json:    `{"enabled":true,}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ScreenshotConfig
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("json.Unmarshal() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestScreenshotFormat_Values(t *testing.T) {
	if ScreenshotFormatPNG != "png" {
		t.Errorf("ScreenshotFormatPNG = %v, want png", ScreenshotFormatPNG)
	}
	if ScreenshotFormatJPEG != "jpeg" {
		t.Errorf("ScreenshotFormatJPEG = %v, want jpeg", ScreenshotFormatJPEG)
	}
}

func TestResult_WithScreenshotPath(t *testing.T) {
	result := Result{
		URL:            "https://example.com",
		Status:         200,
		HTML:           "<html></html>",
		ScreenshotPath: "/data/screenshots/test.png",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Verify ScreenshotPath is included in JSON
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded["screenshotPath"] != "/data/screenshots/test.png" {
		t.Errorf("screenshotPath = %v, want /data/screenshots/test.png", decoded["screenshotPath"])
	}

	// Verify omitempty works when empty
	result2 := Result{
		URL:    "https://example.com",
		Status: 200,
		HTML:   "<html></html>",
	}

	data2, err := json.Marshal(result2)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded2 map[string]interface{}
	if err := json.Unmarshal(data2, &decoded2); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, exists := decoded2["screenshotPath"]; exists {
		t.Error("screenshotPath should be omitted when empty")
	}
}
