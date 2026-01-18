package pipeline

import (
	"context"
	"net/url"
	"strings"
	"time"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
)

type Stage string

const (
	StagePreFetch    Stage = "pre_fetch"
	StagePostFetch   Stage = "post_fetch"
	StagePreExtract  Stage = "pre_extract"
	StagePostExtract Stage = "post_extract"
	StagePreOutput   Stage = "pre_output"
	StagePostOutput  Stage = "post_output"
)

type Options struct {
	PreProcessors  []string `json:"preProcessors,omitempty"`
	PostProcessors []string `json:"postProcessors,omitempty"`
	Transformers   []string `json:"transformers,omitempty"`
}

type Target struct {
	URL         string
	Kind        string
	Host        string
	ProfileName string
	Tags        []string
}

func NewTarget(rawURL string, kind string) Target {
	return Target{
		URL:  rawURL,
		Kind: kind,
		Host: HostFromURL(rawURL),
	}
}

func HostFromURL(rawURL string) string {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		if !strings.Contains(raw, "://") {
			parsed, err = url.Parse("https://" + raw)
			if err == nil && parsed.Hostname() != "" {
				return strings.ToLower(parsed.Hostname())
			}
		}
		return strings.ToLower(raw)
	}
	return strings.ToLower(parsed.Hostname())
}

func AllStages() []Stage {
	return []Stage{
		StagePreFetch,
		StagePostFetch,
		StagePreExtract,
		StagePostExtract,
		StagePreOutput,
		StagePostOutput,
	}
}

type FetchInput struct {
	Target     Target
	Request    fetch.Request
	Auth       fetch.AuthOptions
	Timeout    time.Duration
	UserAgent  string
	Headless   bool
	Playwright bool
	DataDir    string
}

type FetchOutput struct {
	Result fetch.Result
}

type ExtractInput struct {
	Target  Target
	HTML    string
	Options extract.ExtractOptions
	DataDir string
}

type ExtractOutput struct {
	Extracted  extract.Extracted
	Normalized extract.NormalizedDocument
}

type OutputInput struct {
	Target     Target
	Kind       string
	Raw        []byte
	Structured any
}

type OutputOutput struct {
	Raw        []byte
	Structured any
}

type HookContext struct {
	Context     context.Context
	Stage       Stage
	Target      Target
	Now         time.Time
	DataDir     string
	Options     Options
	Attributes  map[string]string
	Diagnostics map[string]any
}
