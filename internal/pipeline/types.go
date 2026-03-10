// Package pipeline provides a plugin system for extending scrape and crawl workflows.
// It handles plugin hooks at pre/post stages of fetch, extract, and output operations,
// plugin registration, and JavaScript plugin execution.
// It does NOT handle workflow execution or plugin implementations.
package pipeline

import (
	"context"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/hostmatch"
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
	PreProcessors  []string       `json:"preProcessors,omitempty"`
	PostProcessors []string       `json:"postProcessors,omitempty"`
	Transformers   []string       `json:"transformers,omitempty"`
	JMESPath       string         `json:"jmesPath,omitempty"`      // JMESPath expression for data transformation
	JSONata        string         `json:"jsonata,omitempty"`       // JSONata expression for data transformation
	TransformVars  map[string]any `json:"transformVars,omitempty"` // Variables available to expressions
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
		Host: hostmatch.HostFromURL(rawURL),
	}
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
	RequestID   string
	Stage       Stage
	Target      Target
	Now         time.Time
	DataDir     string
	Options     Options
	Attributes  map[string]string
	Diagnostics map[string]any
}
