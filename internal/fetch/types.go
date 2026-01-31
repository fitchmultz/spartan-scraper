// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"net/url"
	"strings"
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

// DeviceCategory classifies devices by form factor.
type DeviceCategory string

const (
	DeviceCategoryMobile  DeviceCategory = "mobile"
	DeviceCategoryTablet  DeviceCategory = "tablet"
	DeviceCategoryDesktop DeviceCategory = "desktop"
)

// Orientation represents the device screen orientation.
type Orientation string

const (
	OrientationPortrait  Orientation = "portrait"
	OrientationLandscape Orientation = "landscape"
)

// DeviceEmulation defines device emulation settings for mobile/responsive content.
// Used by headless fetchers to emulate specific devices.
type DeviceEmulation struct {
	Name              string         `json:"name"`              // Device preset name (e.g., "iPhone 14", "Pixel 7")
	ViewportWidth     int            `json:"viewportWidth"`     // Viewport width in pixels
	ViewportHeight    int            `json:"viewportHeight"`    // Viewport height in pixels
	DeviceScaleFactor float64        `json:"deviceScaleFactor"` // Device pixel ratio (e.g., 2.0 for Retina)
	UserAgent         string         `json:"userAgent"`         // User agent string for the device
	IsMobile          bool           `json:"isMobile"`          // Whether to emulate mobile viewport
	HasTouch          bool           `json:"hasTouch"`          // Whether the device has touch capability
	Category          DeviceCategory `json:"category"`          // Device category (mobile, tablet, desktop)
	Orientation       Orientation    `json:"orientation"`       // Default orientation (portrait/landscape)
}

// Common device presets for mobile emulation.
var devicePresets = map[string]DeviceEmulation{
	// iPhone 15 series
	"iphone15": {
		Name:              "iPhone 15",
		ViewportWidth:     393,
		ViewportHeight:    852,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone15pro": {
		Name:              "iPhone 15 Pro",
		ViewportWidth:     393,
		ViewportHeight:    852,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone15promax": {
		Name:              "iPhone 15 Pro Max",
		ViewportWidth:     430,
		ViewportHeight:    932,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone15plus": {
		Name:              "iPhone 15 Plus",
		ViewportWidth:     430,
		ViewportHeight:    932,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	// iPhone 16 series
	"iphone16": {
		Name:              "iPhone 16",
		ViewportWidth:     393,
		ViewportHeight:    852,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone16pro": {
		Name:              "iPhone 16 Pro",
		ViewportWidth:     402,
		ViewportHeight:    874,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone16promax": {
		Name:              "iPhone 16 Pro Max",
		ViewportWidth:     440,
		ViewportHeight:    956,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone16plus": {
		Name:              "iPhone 16 Plus",
		ViewportWidth:     430,
		ViewportHeight:    932,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	// Legacy iPhone
	"iphone14": {
		Name:              "iPhone 14",
		ViewportWidth:     390,
		ViewportHeight:    844,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphonemax": {
		Name:              "iPhone 14 Pro Max",
		ViewportWidth:     430,
		ViewportHeight:    932,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	// Pixel series
	"pixel7": {
		Name:              "Pixel 7",
		ViewportWidth:     412,
		ViewportHeight:    915,
		DeviceScaleFactor: 2.625,
		UserAgent:         "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"pixel8": {
		Name:              "Pixel 8",
		ViewportWidth:     412,
		ViewportHeight:    915,
		DeviceScaleFactor: 2.625,
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"pixel8pro": {
		Name:              "Pixel 8 Pro",
		ViewportWidth:     448,
		ViewportHeight:    998,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"pixel9": {
		Name:              "Pixel 9",
		ViewportWidth:     412,
		ViewportHeight:    915,
		DeviceScaleFactor: 2.625,
		UserAgent:         "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"pixel9pro": {
		Name:              "Pixel 9 Pro",
		ViewportWidth:     448,
		ViewportHeight:    998,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 15; Pixel 9 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	// Galaxy S series
	"galaxys23": {
		Name:              "Galaxy S23",
		ViewportWidth:     360,
		ViewportHeight:    780,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 13; SM-S911B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"galaxys24": {
		Name:              "Galaxy S24",
		ViewportWidth:     360,
		ViewportHeight:    780,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; SM-S921B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"galaxys24plus": {
		Name:              "Galaxy S24+",
		ViewportWidth:     384,
		ViewportHeight:    824,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; SM-S926B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"galaxys24ultra": {
		Name:              "Galaxy S24 Ultra",
		ViewportWidth:     384,
		ViewportHeight:    824,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; SM-S928B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	// iPad series
	"ipad": {
		Name:              "iPad",
		ViewportWidth:     810,
		ViewportHeight:    1080,
		DeviceScaleFactor: 2.0,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryTablet,
		Orientation:       OrientationPortrait,
	},
	"ipadpro": {
		Name:              "iPad Pro 12.9\"",
		ViewportWidth:     1024,
		ViewportHeight:    1366,
		DeviceScaleFactor: 2.0,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryTablet,
		Orientation:       OrientationPortrait,
	},
	"ipadair": {
		Name:              "iPad Air",
		ViewportWidth:     820,
		ViewportHeight:    1180,
		DeviceScaleFactor: 2.0,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryTablet,
		Orientation:       OrientationPortrait,
	},
	"ipadmini": {
		Name:              "iPad Mini",
		ViewportWidth:     744,
		ViewportHeight:    1133,
		DeviceScaleFactor: 2.0,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryTablet,
		Orientation:       OrientationPortrait,
	},
	// Android tablets
	"galaxytabs9": {
		Name:              "Galaxy Tab S9",
		ViewportWidth:     1600,
		ViewportHeight:    2560,
		DeviceScaleFactor: 2.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 13; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryTablet,
		Orientation:       OrientationPortrait,
	},
	// Desktop
	"desktop": {
		Name:              "Desktop",
		ViewportWidth:     1920,
		ViewportHeight:    1080,
		DeviceScaleFactor: 1.0,
		UserAgent:         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		IsMobile:          false,
		HasTouch:          false,
		Category:          DeviceCategoryDesktop,
		Orientation:       OrientationLandscape,
	},
	"laptop": {
		Name:              "Laptop",
		ViewportWidth:     1366,
		ViewportHeight:    768,
		DeviceScaleFactor: 1.0,
		UserAgent:         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		IsMobile:          false,
		HasTouch:          false,
		Category:          DeviceCategoryDesktop,
		Orientation:       OrientationLandscape,
	},
}

// GetDevicePreset returns a device emulation preset by name.
// Returns nil if the preset name is not recognized.
func GetDevicePreset(name string) *DeviceEmulation {
	if name == "" {
		return nil
	}
	// Normalize name to lowercase for case-insensitive lookup
	name = strings.ToLower(name)
	if preset, ok := devicePresets[name]; ok {
		// Return a copy to prevent modification of the original
		presetCopy := preset
		return &presetCopy
	}
	return nil
}

// GetDevicePresetsByCategory returns all device presets matching the given category.
func GetDevicePresetsByCategory(cat DeviceCategory) []DeviceEmulation {
	var result []DeviceEmulation
	for _, preset := range devicePresets {
		if preset.Category == cat {
			// Return a copy to prevent modification of the original
			presetCopy := preset
			result = append(result, presetCopy)
		}
	}
	return result
}

// ListDevicePresetNames returns all available device preset names.
func ListDevicePresetNames() []string {
	names := make([]string, 0, len(devicePresets))
	for name := range devicePresets {
		names = append(names, name)
	}
	return names
}

// GetDeviceCategories returns all available device categories.
func GetDeviceCategories() []DeviceCategory {
	return []DeviceCategory{
		DeviceCategoryMobile,
		DeviceCategoryTablet,
		DeviceCategoryDesktop,
	}
}

// ApplyOrientation applies the specified orientation to a device emulation.
// For landscape orientation on mobile/tablet devices, it swaps width and height.
func (d *DeviceEmulation) ApplyOrientation(orientation Orientation) *DeviceEmulation {
	if d == nil {
		return nil
	}
	// Create a copy to avoid modifying the original
	result := *d
	result.Orientation = orientation

	if orientation == OrientationLandscape && d.Category != DeviceCategoryDesktop {
		// Swap width and height for landscape orientation
		result.ViewportWidth = d.ViewportHeight
		result.ViewportHeight = d.ViewportWidth
	}

	return &result
}

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

// ProxyConfig defines proxy settings for fetch operations.
type ProxyConfig struct {
	URL      string `json:"url,omitempty"`      // Proxy URL (http://, https://, socks5://)
	Username string `json:"username,omitempty"` // Username for proxy authentication
	Password string `json:"password,omitempty"` // Password for proxy authentication
}

type AuthOptions struct {
	Basic               string            `json:"basic,omitempty"`
	Headers             map[string]string `json:"headers,omitempty"`
	Cookies             []string          `json:"cookies,omitempty"`
	Query               map[string]string `json:"query,omitempty"`
	LoginURL            string            `json:"loginUrl,omitempty"`
	LoginUserSelector   string            `json:"loginUserSelector,omitempty"`
	LoginPassSelector   string            `json:"loginPassSelector,omitempty"`
	LoginSubmitSelector string            `json:"loginSubmitSelector,omitempty"`
	LoginUser           string            `json:"loginUser,omitempty"`
	LoginPass           string            `json:"loginPass,omitempty"`
	LoginAutoDetect     bool              `json:"loginAutoDetect,omitempty"`
	Proxy               *ProxyConfig      `json:"proxy,omitempty"`
	// ProxyPool enables proxy pool selection. When set, the pool will be used
	// to select a proxy based on the configured rotation strategy.
	ProxyPool string `json:"proxyPool,omitempty"`
	// ProxyHints provides hints for proxy selection when using a proxy pool.
	ProxyHints *ProxySelectionHints `json:"proxyHints,omitempty"`
	// OAuth2 contains OAuth 2.0 configuration for automatic token management.
	// When set, the fetcher will use OAuth transport with automatic token refresh.
	OAuth2 *OAuth2AuthConfig `json:"oauth2,omitempty"`
}

// OAuth2AuthConfig defines OAuth 2.0 authentication configuration for fetch operations.
type OAuth2AuthConfig struct {
	// ProfileName is the name of the auth profile with OAuth2 configuration
	ProfileName string `json:"profileName,omitempty"`
	// AccessToken is a static access token (optional - if not set, will be loaded from store)
	AccessToken string `json:"accessToken,omitempty"`
	// TokenType is the token type (e.g., "Bearer"). Defaults to "Bearer" if not set.
	TokenType string `json:"tokenType,omitempty"`
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

// InterceptedResourceType represents the type of network resource.
type InterceptedResourceType string

const (
	ResourceTypeXHR        InterceptedResourceType = "xhr"
	ResourceTypeFetch      InterceptedResourceType = "fetch"
	ResourceTypeDocument   InterceptedResourceType = "document"
	ResourceTypeScript     InterceptedResourceType = "script"
	ResourceTypeStylesheet InterceptedResourceType = "stylesheet"
	ResourceTypeImage      InterceptedResourceType = "image"
	ResourceTypeMedia      InterceptedResourceType = "media"
	ResourceTypeFont       InterceptedResourceType = "font"
	ResourceTypeWebSocket  InterceptedResourceType = "websocket"
	ResourceTypeOther      InterceptedResourceType = "other"
)

// NetworkInterceptConfig defines configuration for network request/response interception.
// Used to capture XHR/Fetch API traffic from SPAs for API scraping.
type NetworkInterceptConfig struct {
	Enabled             bool                      `json:"enabled"`             // Toggle interception
	URLPatterns         []string                  `json:"urlPatterns"`         // Glob patterns for URLs to intercept (e.g., "**/api/**", "*.json")
	ResourceTypes       []InterceptedResourceType `json:"resourceTypes"`       // Resource types to capture
	CaptureRequestBody  bool                      `json:"captureRequestBody"`  // Whether to capture request bodies
	CaptureResponseBody bool                      `json:"captureResponseBody"` // Whether to capture response bodies
	MaxBodySize         int64                     `json:"maxBodySize"`         // Max bytes to capture per body (default 1MB)
	MaxEntries          int                       `json:"maxEntries"`          // Max number of entries to capture (default 1000)
}

// DefaultNetworkInterceptConfig returns a default configuration with sensible limits.
func DefaultNetworkInterceptConfig() NetworkInterceptConfig {
	return NetworkInterceptConfig{
		Enabled:             false,
		URLPatterns:         []string{},
		ResourceTypes:       []InterceptedResourceType{ResourceTypeXHR, ResourceTypeFetch},
		CaptureRequestBody:  true,
		CaptureResponseBody: true,
		MaxBodySize:         1024 * 1024, // 1MB
		MaxEntries:          1000,
	}
}

// InterceptedRequest represents a captured network request.
type InterceptedRequest struct {
	RequestID    string                  `json:"requestId"`    // Unique identifier
	URL          string                  `json:"url"`          // Request URL
	Method       string                  `json:"method"`       // HTTP method
	Headers      map[string]string       `json:"headers"`      // Request headers
	Body         string                  `json:"body"`         // Request body (base64 if binary)
	BodySize     int64                   `json:"bodySize"`     // Original body size
	Timestamp    time.Time               `json:"timestamp"`    // When request was sent
	ResourceType InterceptedResourceType `json:"resourceType"` // Type of resource
}

// InterceptedResponse represents a captured network response.
type InterceptedResponse struct {
	RequestID  string            `json:"requestId"`  // Matches request
	Status     int               `json:"status"`     // HTTP status code
	StatusText string            `json:"statusText"` // HTTP status text
	Headers    map[string]string `json:"headers"`    // Response headers
	Body       string            `json:"body"`       // Response body (base64 if binary)
	BodySize   int64             `json:"bodySize"`   // Size of response body
	Timestamp  time.Time         `json:"timestamp"`  // When response received
}

// InterceptedEntry combines a request/response pair with timing data.
type InterceptedEntry struct {
	Request  InterceptedRequest   `json:"request"`
	Response *InterceptedResponse `json:"response,omitempty"` // nil if response not received
	Duration time.Duration        `json:"duration"`           // Time between request and response
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

// ApplyAuthQuery applies authentication query parameters to a URL.
// If the query map is empty, the original URL is returned unchanged.
func ApplyAuthQuery(rawURL string, query map[string]string) string {
	if len(query) == 0 {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	values := parsed.Query()
	for key, value := range query {
		if key == "" {
			continue
		}
		values.Set(key, value)
	}
	parsed.RawQuery = values.Encode()
	return parsed.String()
}
