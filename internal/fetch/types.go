package fetch

import "time"

type RenderEngine string

const (
	RenderEngineHTTP       RenderEngine = "http"
	RenderEngineChromedp   RenderEngine = "chromedp"
	RenderEnginePlaywright RenderEngine = "playwright"
)

type BlockedResourceType string

const (
	BlockedResourceImage      BlockedResourceType = "image"
	BlockedResourceMedia      BlockedResourceType = "media"
	BlockedResourceFont       BlockedResourceType = "font"
	BlockedResourceStylesheet BlockedResourceType = "stylesheet"
	BlockedResourceOther      BlockedResourceType = "other"
)

type RenderWaitMode string

const (
	RenderWaitModeDOMReady    RenderWaitMode = "dom_ready"    // DOMContentLoaded + body present
	RenderWaitModeNetworkIdle RenderWaitMode = "network_idle" // inflight==0 for quiet window
	RenderWaitModeStability   RenderWaitMode = "stability"    // body.innerText length stabilizes
	RenderWaitModeSelector    RenderWaitMode = "selector"     // selector appears (and optional stability)
)

type RenderBlockPolicy struct {
	ResourceTypes []BlockedResourceType `json:"resourceTypes,omitempty"`
	URLPatterns   []string              `json:"urlPatterns,omitempty"` // glob-style patterns
}

type RenderTimeoutPolicy struct {
	// Absolute cap for the entire render phase (headless only).
	MaxRenderMs int `json:"maxRenderMs,omitempty"`
	// Cap for in-page script evaluation/wait-for-function loops.
	ScriptEvalMs int `json:"scriptEvalMs,omitempty"`
	// Cap for navigation (goto) only.
	NavigationMs int `json:"navigationMs,omitempty"`
}

type RenderWaitPolicy struct {
	Mode RenderWaitMode `json:"mode,omitempty"`

	// RenderWaitModeSelector
	Selector string `json:"selector,omitempty"`

	// RenderWaitModeNetworkIdle
	NetworkIdleQuietMs int `json:"networkIdleQuietMs,omitempty"`

	// RenderWaitModeStability
	MinTextLength       int `json:"minTextLength,omitempty"`
	StabilityPollMs     int `json:"stabilityPollMs,omitempty"`
	StabilityIterations int `json:"stabilityIterations,omitempty"`

	// Always applied after wait mode completes (final settle).
	ExtraSleepMs int `json:"extraSleepMs,omitempty"`
}

type RenderProfile struct {
	Name         string   `json:"name"`
	HostPatterns []string `json:"hostPatterns"` // match against URL host, glob-style ("example.com", "*.example.com")

	// If set, overrides engine selection entirely.
	ForceEngine RenderEngine `json:"forceEngine,omitempty"`

	// If true, skip HTTP probe and go straight to headless engine selection.
	PreferHeadless bool `json:"preferHeadless,omitempty"`

	// If true, treat every page on this host as JS-heavy (forces escalation if not forced to HTTP).
	AssumeJSHeavy bool `json:"assumeJsHeavy,omitempty"`

	// If true, never escalate (forces HTTP).
	NeverHeadless bool `json:"neverHeadless,omitempty"`

	// Overrides default JS-heavy threshold for this host (0..1). 0 means use global default.
	JSHeavyThreshold float64 `json:"jsHeavyThreshold,omitempty"`

	Block    RenderBlockPolicy   `json:"block,omitempty"`
	Wait     RenderWaitPolicy    `json:"wait,omitempty"`
	Timeouts RenderTimeoutPolicy `json:"timeouts,omitempty"`
}

type RenderProfilesFile struct {
	Profiles []RenderProfile `json:"profiles"`
}

type JSHeaviness struct {
	Score   float64  `json:"score"`
	Reasons []string `json:"reasons"`

	ScriptTagCount   int `json:"scriptTagCount"`
	BodyTextLength   int `json:"bodyTextLength"`
	RootDivSignals   int `json:"rootDivSignals"`
	FrameworkSignals int `json:"frameworkSignals"`
}

type AuthOptions struct {
	Basic               string            `json:"basic"`
	Headers             map[string]string `json:"headers"`
	Cookies             []string          `json:"cookies"`
	LoginURL            string            `json:"loginUrl"`
	LoginUserSelector   string            `json:"loginUserSelector"`
	LoginPassSelector   string            `json:"loginPassSelector"`
	LoginSubmitSelector string            `json:"loginSubmitSelector"`
	LoginUser           string            `json:"loginUser"`
	LoginPass           string            `json:"loginPass"`
}

type Request struct {
	URL             string
	Timeout         time.Duration
	UserAgent       string
	Headless        bool
	UsePlaywright   bool
	Auth            AuthOptions
	Limiter         *HostLimiter
	MaxRetries      int
	RetryBaseDelay  time.Duration
	IfNoneMatch     string `json:"-"`
	IfModifiedSince string `json:"-"`
	DataDir         string `json:"-"` // New field for profiles
}

type Result struct {
	URL          string
	Status       int
	HTML         string
	FetchedAt    time.Time
	ETag         string       `json:"-"`
	LastModified string       `json:"-"`
	Engine       RenderEngine `json:"-"` // New field
}
