package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

func TestHostFromURL_Empty(t *testing.T) {
	host := HostFromURL("")
	if host != "" {
		t.Errorf("expected empty string, got '%s'", host)
	}
}

func TestHostFromURL_ValidWithScheme(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com", "example.com"},
		{"http://example.com", "example.com"},
		{"HTTPS://EXAMPLE.COM", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			host := HostFromURL(tt.url)
			if host != tt.want {
				t.Errorf("expected '%s', got '%s'", tt.want, host)
			}
		})
	}
}

func TestHostFromURL_ValidWithoutScheme(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"example.com", "example.com"},
		{"EXAMPLE.COM", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			host := HostFromURL(tt.url)
			if host != tt.want {
				t.Errorf("expected '%s', got '%s'", tt.want, host)
			}
		})
	}
}

func TestHostFromURL_WithPort(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com:8080", "example.com"},
		{"http://example.com:443", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			host := HostFromURL(tt.url)
			if host != tt.want {
				t.Errorf("expected '%s', got '%s'", tt.want, host)
			}
		})
	}
}

func TestHostFromURL_WithPath(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/path/to/page", "example.com"},
		{"https://example.com/", "example.com"},
		{"https://example.com?query=1", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			host := HostFromURL(tt.url)
			if host != tt.want {
				t.Errorf("expected '%s', got '%s'", tt.want, host)
			}
		})
	}
}

func TestHostFromURL_WithSubdomain(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://sub.example.com", "sub.example.com"},
		{"https://api.service.example.com", "api.service.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			host := HostFromURL(tt.url)
			if host != tt.want {
				t.Errorf("expected '%s', got '%s'", tt.want, host)
			}
		})
	}
}

func TestHostFromURL_InvalidURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"not a url", "not a url"},
		{"http://", "http://"},
		{"://example.com", "://example.com"},
		{"EXAMPLE.COM", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			host := HostFromURL(tt.url)
			if host != tt.want {
				t.Errorf("expected '%s', got '%s'", tt.want, host)
			}
		})
	}
}

func TestHostFromURL_WithSchemeParsingFailure(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"http://invalid<>url.com", "invalid<>url.com"},
		{"https://invalid url.com", "https://invalid url.com"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			host := HostFromURL(tt.url)
			if host != tt.want {
				t.Errorf("expected '%s', got '%s'", tt.want, host)
			}
		})
	}
}

func TestNewTarget(t *testing.T) {
	tests := []struct {
		name string
		url  string
		kind string
		want Target
	}{
		{
			name: "basic target",
			url:  "https://example.com",
			kind: "scrape",
			want: Target{URL: "https://example.com", Kind: "scrape", Host: "example.com"},
		},
		{
			name: "with subdomain",
			url:  "https://sub.example.com/path",
			kind: "crawl",
			want: Target{URL: "https://sub.example.com/path", Kind: "crawl", Host: "sub.example.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := NewTarget(tt.url, tt.kind)
			if target.URL != tt.want.URL {
				t.Errorf("expected URL '%s', got '%s'", tt.want.URL, target.URL)
			}
			if target.Kind != tt.want.Kind {
				t.Errorf("expected Kind '%s', got '%s'", tt.want.Kind, target.Kind)
			}
			if target.Host != tt.want.Host {
				t.Errorf("expected Host '%s', got '%s'", tt.want.Host, target.Host)
			}
		})
	}
}

func TestAllStages(t *testing.T) {
	stages := AllStages()
	expected := []Stage{
		StagePreFetch,
		StagePostFetch,
		StagePreExtract,
		StagePostExtract,
		StagePreOutput,
		StagePostOutput,
	}
	if len(stages) != len(expected) {
		t.Fatalf("expected %d stages, got %d", len(expected), len(stages))
	}
	for i, stage := range expected {
		if stages[i] != stage {
			t.Errorf("expected stage %d to be %s, got %s", i, stage, stages[i])
		}
	}
}

func TestStageConstants(t *testing.T) {
	tests := []struct {
		name  string
		stage Stage
		value string
	}{
		{"StagePreFetch", StagePreFetch, "pre_fetch"},
		{"StagePostFetch", StagePostFetch, "post_fetch"},
		{"StagePreExtract", StagePreExtract, "pre_extract"},
		{"StagePostExtract", StagePostExtract, "post_extract"},
		{"StagePreOutput", StagePreOutput, "pre_output"},
		{"StagePostOutput", StagePostOutput, "post_output"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.stage) != tt.value {
				t.Errorf("expected '%s', got '%s'", tt.value, tt.stage)
			}
		})
	}
}

func TestOptions_JSONSerialization(t *testing.T) {
	opts := Options{
		PreProcessors:  []string{"plugin1", "plugin2"},
		PostProcessors: []string{"plugin3"},
		Transformers:   []string{"transformer1"},
	}
	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	var decoded Options
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(decoded.PreProcessors) != 2 || decoded.PreProcessors[0] != "plugin1" || decoded.PreProcessors[1] != "plugin2" {
		t.Errorf("PreProcessors not preserved: got %v", decoded.PreProcessors)
	}
	if len(decoded.PostProcessors) != 1 || decoded.PostProcessors[0] != "plugin3" {
		t.Errorf("PostProcessors not preserved: got %v", decoded.PostProcessors)
	}
	if len(decoded.Transformers) != 1 || decoded.Transformers[0] != "transformer1" {
		t.Errorf("Transformers not preserved: got %v", decoded.Transformers)
	}
}

func TestOptions_Empty(t *testing.T) {
	opts := Options{}
	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	var decoded Options
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.PreProcessors != nil || decoded.PostProcessors != nil || decoded.Transformers != nil {
		t.Errorf("expected nil slices, got PreProcessors=%v, PostProcessors=%v, Transformers=%v",
			decoded.PreProcessors, decoded.PostProcessors, decoded.Transformers)
	}
}

func TestFetchInput_Fields(t *testing.T) {
	input := FetchInput{
		Target:     NewTarget("https://example.com", "scrape"),
		Request:    fetch.Request{},
		Auth:       fetch.AuthOptions{},
		Timeout:    30,
		UserAgent:  "test-agent",
		Headless:   true,
		Playwright: false,
		DataDir:    "/tmp/data",
	}
	if input.Target.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got '%s'", input.Target.URL)
	}
	if input.UserAgent != "test-agent" {
		t.Errorf("expected UserAgent 'test-agent', got '%s'", input.UserAgent)
	}
	if !input.Headless {
		t.Error("expected Headless true")
	}
}

func TestExtractInput_Fields(t *testing.T) {
	input := ExtractInput{
		Target:  NewTarget("https://example.com", "scrape"),
		HTML:    "<html><body>test</body></html>",
		Options: extract.ExtractOptions{},
		DataDir: "/tmp/data",
	}
	if input.HTML != "<html><body>test</body></html>" {
		t.Errorf("expected HTML '<html><body>test</body></html>', got '%s'", input.HTML)
	}
}

func TestOutputInput_Fields(t *testing.T) {
	input := OutputInput{
		Target:     NewTarget("https://example.com", "scrape"),
		Kind:       "json",
		Raw:        []byte("test"),
		Structured: map[string]any{"key": "value"},
	}
	if input.Kind != "json" {
		t.Errorf("expected Kind 'json', got '%s'", input.Kind)
	}
	if string(input.Raw) != "test" {
		t.Errorf("expected Raw 'test', got '%s'", string(input.Raw))
	}
	if input.Structured == nil {
		t.Error("expected non-nil Structured")
	}
}

func TestHookContext_Fields(t *testing.T) {
	ctx := HookContext{
		Context:     nil,
		RequestID:   "req-123",
		Stage:       StagePreFetch,
		Target:      NewTarget("https://example.com", "scrape"),
		DataDir:     "/tmp/data",
		Options:     Options{},
		Attributes:  map[string]string{"key": "value"},
		Diagnostics: map[string]any{"count": 42},
	}
	if ctx.RequestID != "req-123" {
		t.Errorf("expected RequestID 'req-123', got '%s'", ctx.RequestID)
	}
	if ctx.Stage != StagePreFetch {
		t.Errorf("expected Stage StagePreFetch, got '%s'", ctx.Stage)
	}
	if ctx.Attributes["key"] != "value" {
		t.Errorf("expected attribute key 'value', got '%v'", ctx.Attributes["key"])
	}
}
