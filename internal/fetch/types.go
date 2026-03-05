// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/captcha"
)

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

type ScreenshotFormat string

const (
	ScreenshotFormatPNG  ScreenshotFormat = "png"
	ScreenshotFormatJPEG ScreenshotFormat = "jpeg"
)

// ScreenshotConfig defines screenshot capture options for headless fetchers.
// Screenshots are only applicable to chromedp and playwright engines, not HTTP fetcher.
type ScreenshotConfig struct {
	Enabled  bool             `json:"enabled"`           // Whether to capture screenshot
	FullPage bool             `json:"fullPage"`          // Capture full page or just viewport
	Format   ScreenshotFormat `json:"format"`            // png or jpeg
	Quality  int              `json:"quality,omitempty"` // JPEG quality (1-100), ignored for PNG
	Width    int              `json:"width,omitempty"`   // Viewport width (0 = default)
	Height   int              `json:"height,omitempty"`  // Viewport height (0 = default)
	Device   *DeviceEmulation `json:"device,omitempty"`  // Device emulation settings
}

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

	// Rate limiting configuration for this profile (0 = use global defaults).
	RateLimitQPS   int `json:"rateLimitQPS,omitempty"`
	RateLimitBurst int `json:"rateLimitBurst,omitempty"`

	Block      RenderBlockPolicy   `json:"block,omitempty"`
	Wait       RenderWaitPolicy    `json:"wait,omitempty"`
	Timeouts   RenderTimeoutPolicy `json:"timeouts,omitempty"`
	Screenshot ScreenshotConfig    `json:"screenshot,omitempty"`
	Device     *DeviceEmulation    `json:"device,omitempty"` // Device emulation for this profile

	// CaptchaConfig defines CAPTCHA handling for this profile.
	CaptchaConfig *captcha.CaptchaConfig `json:"captchaConfig,omitempty"`
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

type Request struct {
	URL              string
	Method           string // HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
	Body             []byte // Request body for POST/PUT/PATCH
	ContentType      string // Content-Type header for request body
	Timeout          time.Duration
	UserAgent        string
	Headless         bool
	UsePlaywright    bool
	Auth             AuthOptions
	SessionID        string // Reference to persisted session for cookie reuse
	Limiter          *HostLimiter
	MaxRetries       int
	RetryBaseDelay   time.Duration
	MaxResponseBytes int64                   `json:"maxResponseBytes,omitempty"`
	IfNoneMatch      string                  `json:"-"`
	IfModifiedSince  string                  `json:"-"`
	DataDir          string                  `json:"-"`
	PreNavJS         []string                `json:"-"`
	PostNavJS        []string                `json:"-"`
	WaitSelectors    []string                `json:"-"`
	Screenshot       *ScreenshotConfig       `json:"screenshot"`
	Device           *DeviceEmulation        `json:"device,omitempty"`           // Device emulation settings
	NetworkIntercept *NetworkInterceptConfig `json:"networkIntercept,omitempty"` // Network interception config
}

type Result struct {
	URL             string             `json:"url"`
	Status          int                `json:"status"`
	HTML            string             `json:"html"`
	FetchedAt       time.Time          `json:"fetchedAt"`
	ETag            string             `json:"-"`
	LastModified    string             `json:"-"`
	Engine          RenderEngine       `json:"-"`
	ScreenshotPath  string             `json:"screenshotPath,omitempty"`  // Path to saved screenshot file
	InterceptedData []InterceptedEntry `json:"interceptedData,omitempty"` // Captured network activity
	// RateLimit contains parsed rate limit information from response headers.
	// Populated when the server returns RateLimit (RFC 9440) or X-RateLimit-* headers.
	RateLimit *RateLimitInfo `json:"rateLimit,omitempty"`
}
