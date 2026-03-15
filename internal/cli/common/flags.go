// Package common provides shared CLI flag registration.
//
// It does NOT run commands; it only defines types and registration helpers
// to keep flags consistent across command modules.
package common

import (
	"flag"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

type StringSliceFlag []string

func (s *StringSliceFlag) String() string { return strings.Join(*s, ",") }
func (s *StringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// ToMap parses entries in "Key: Value" form into a map.
// Invalid entries are ignored.
func (s *StringSliceFlag) ToMap() map[string]string {
	if len(*s) == 0 {
		return nil
	}
	out := map[string]string{}
	for _, item := range *s {
		key, val, ok := splitOnce(item, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func splitOnce(s, sep string) (string, string, bool) {
	idx := strings.Index(s, sep)
	if idx < 0 {
		return "", "", false
	}
	return s[:idx], s[idx+len(sep):], true
}

// CommonFlags are shared across scrape/crawl/research.
type CommonFlags struct {
	// Extraction flags
	ExtractTemplate *string
	ExtractConfig   *string
	ExtractValidate *bool

	// AI extraction flags
	AIExtract       *bool
	AIExtractMode   *string
	AIExtractPrompt *string
	AIExtractSchema *string
	AIExtractFields *string

	// Agentic research flags
	AgenticResearch             *bool
	AgenticResearchInstructions *string
	AgenticResearchMaxRounds    *int
	AgenticResearchMaxFollowUps *int

	// Pipeline flags
	PreProcessors  StringSliceFlag
	PostProcessors StringSliceFlag
	Transformers   StringSliceFlag

	// Auth flags
	AuthBasic   *string
	TokenKind   *string
	TokenHeader *string
	TokenQuery  *string
	TokenCookie *string
	TokenValues StringSliceFlag
	Headers     StringSliceFlag
	Cookies     StringSliceFlag

	// Login flow flags
	LoginURL            *string
	LoginUserSelector   *string
	LoginPassSelector   *string
	LoginSubmitSelector *string
	LoginUser           *string
	LoginPass           *string
	LoginAutoDetect     *bool

	// Browser flags
	Headless   *bool
	Playwright *bool
	Timeout    *int
	Device     *string // Device preset for mobile emulation

	// Output flags
	Out         *string
	Wait        *bool
	WaitTimeout *int

	// Profile flag
	ProfileName *string

	// Incremental flag
	Incremental *bool

	// Proxy flags
	ProxyURL            *string
	ProxyUsername       *string
	ProxyPassword       *string
	ProxyRegion         *string
	ProxyRequiredTags   StringSliceFlag
	ProxyExcludeProxyID StringSliceFlag

	// HTTP method and body flags
	Method      *string // HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
	Body        *string // Request body (string or @file path)
	ContentType *string // Content-Type header for request body

	// CAPTCHA flags
	CaptchaEnabled   *bool
	CaptchaAutoSolve *bool
	CaptchaService   *string
	CaptchaAPIKey    *string

	// Network interception flags
	InterceptEnabled         *bool
	InterceptURLPatterns     StringSliceFlag
	InterceptResourceTypes   StringSliceFlag
	InterceptCaptureRequest  *bool
	InterceptCaptureResponse *bool
	InterceptMaxBodySize     *int
	InterceptMaxEntries      *int
}

// BrowserFlags are used by schedule add.
type BrowserFlags struct {
	Headless   *bool
	Playwright *bool
	Timeout    *int
}

// PipelineFlags are used by schedule add.
type PipelineFlags struct {
	PreProcessors  StringSliceFlag
	PostProcessors StringSliceFlag
	Transformers   StringSliceFlag
}

// ExtractFlags are used by schedule add.
type ExtractFlags struct {
	ExtractTemplate *string
	ExtractConfig   *string
	ExtractValidate *bool
}

// AuthFlags are used by schedule add.
type AuthFlags struct {
	AuthBasic   *string
	TokenKind   *string
	TokenHeader *string
	TokenQuery  *string
	TokenCookie *string
	TokenValues StringSliceFlag
	Headers     StringSliceFlag
	Cookies     StringSliceFlag

	// Login flow flags
	LoginURL            *string
	LoginUserSelector   *string
	LoginPassSelector   *string
	LoginSubmitSelector *string
	LoginUser           *string
	LoginPass           *string
	LoginAutoDetect     *bool

	// Proxy flags
	ProxyURL            *string
	ProxyUsername       *string
	ProxyPassword       *string
	ProxyRegion         *string
	ProxyRequiredTags   StringSliceFlag
	ProxyExcludeProxyID StringSliceFlag
}

func RegisterCommonFlags(fs *flag.FlagSet, cfg config.Config) *CommonFlags {
	cf := &CommonFlags{
		ExtractTemplate: fs.String("extract-template", "", "Extraction template name"),
		ExtractConfig:   fs.String("extract-config", "", "Path to inline template JSON"),
		ExtractValidate: fs.Bool("extract-validate", false, "Validate extraction against schema"),

		PreProcessors:  StringSliceFlag{},
		PostProcessors: StringSliceFlag{},
		Transformers:   StringSliceFlag{},

		AuthBasic:   fs.String("auth-basic", "", "Basic auth user:pass"),
		TokenKind:   fs.String("token-kind", "bearer", "Token kind: bearer|basic|api_key"),
		TokenHeader: fs.String("token-header", "", "Token header name (api_key or bearer override)"),
		TokenQuery:  fs.String("token-query", "", "Token query param name (api_key)"),
		TokenCookie: fs.String("token-cookie", "", "Token cookie name (api_key)"),
		TokenValues: StringSliceFlag{},
		Headers:     StringSliceFlag{},
		Cookies:     StringSliceFlag{},

		LoginURL:            fs.String("login-url", "", "Login URL for headless auth"),
		LoginUserSelector:   fs.String("login-user-selector", "", "CSS selector for username input"),
		LoginPassSelector:   fs.String("login-pass-selector", "", "CSS selector for password input"),
		LoginSubmitSelector: fs.String("login-submit-selector", "", "CSS selector for submit button"),
		LoginUser:           fs.String("login-user", "", "Username for login"),
		LoginPass:           fs.String("login-pass", "", "Password for login"),
		LoginAutoDetect:     fs.Bool("login-auto-detect", false, "Auto-detect login form fields (requires --login-url)"),

		Headless:   fs.Bool("headless", false, "Use headless browser"),
		Playwright: fs.Bool("playwright", cfg.UsePlaywright, "Use Playwright for headless pages"),
		Timeout:    fs.Int("timeout", cfg.RequestTimeoutSecs, "Request timeout in seconds"),
		Device:     fs.String("device", "", "Device preset for mobile emulation (iphone15, iphone15pro, iphone15promax, iphone15plus, iphone16, iphone16pro, iphone16promax, iphone16plus, iphone14, iphonemax, pixel7, pixel8, pixel8pro, pixel9, pixel9pro, galaxys23, galaxys24, galaxys24plus, galaxys24ultra, ipad, ipadpro, ipadair, ipadmini, galaxytabs9, desktop, laptop)"),

		Out:         fs.String("out", "", "Output file (JSON)"),
		Wait:        fs.Bool("wait", false, "Wait for completion and write output"),
		WaitTimeout: fs.Int("wait-timeout", 0, "Max wait time in seconds (0 = no timeout)"),

		ProfileName: fs.String("auth-profile", "", "Auth profile name"),

		Incremental: fs.Bool("incremental", false, "Use incremental crawling (ETag/Hash)"),

		// Proxy flags
		ProxyURL:            fs.String("proxy", "", "Proxy URL (http://, https://, socks5://)"),
		ProxyUsername:       fs.String("proxy-username", "", "Proxy username"),
		ProxyPassword:       fs.String("proxy-password", "", "Proxy password"),
		ProxyRegion:         fs.String("proxy-region", "", "Preferred region when selecting from the loaded proxy pool"),
		ProxyRequiredTags:   StringSliceFlag{},
		ProxyExcludeProxyID: StringSliceFlag{},

		// HTTP method and body flags
		Method:      fs.String("method", "GET", "HTTP method (GET, POST, PUT, DELETE, PATCH)"),
		Body:        fs.String("body", "", "Request body (string or @file to read from file)"),
		ContentType: fs.String("content-type", "", "Content-Type header (auto-detected if not set)"),

		// CAPTCHA flags
		CaptchaEnabled:   fs.Bool("captcha-enabled", false, "Enable CAPTCHA detection"),
		CaptchaAutoSolve: fs.Bool("captcha-auto-solve", false, "Automatically solve detected CAPTCHAs (requires --captcha-service and --captcha-api-key)"),
		CaptchaService:   fs.String("captcha-service", "", "CAPTCHA solving service (2captcha|anticaptcha)"),
		CaptchaAPIKey:    fs.String("captcha-api-key", "", "API key for CAPTCHA solving service"),

		// Network interception flags
		InterceptEnabled:         fs.Bool("intercept-enabled", false, "Enable network interception (requires --headless)"),
		InterceptCaptureRequest:  fs.Bool("intercept-request-body", true, "Capture request bodies during interception"),
		InterceptCaptureResponse: fs.Bool("intercept-response-body", true, "Capture response bodies during interception"),
		InterceptMaxBodySize:     fs.Int("intercept-max-body-size", 1048576, "Maximum bytes to capture per body (default 1MB)"),
		InterceptMaxEntries:      fs.Int("intercept-max-entries", 1000, "Maximum number of entries to capture (default 1000)"),

		// AI extraction flags
		AIExtract:       fs.Bool("ai-extract", false, "Enable AI-powered intelligent extraction (requires PI_ENABLED bridge config)"),
		AIExtractMode:   fs.String("ai-mode", "natural_language", "AI extraction mode: natural_language|schema_guided"),
		AIExtractPrompt: fs.String("ai-prompt", "", "Natural language instructions for AI extraction"),
		AIExtractSchema: fs.String("ai-schema", "", "Schema-guided extraction example as a JSON object string"),
		AIExtractFields: fs.String("ai-fields", "", "Comma-separated list of fields to extract with AI"),
	}

	fs.Var(&cf.PreProcessors, "pre-processor", "Pipeline pre-processor plugin name (repeatable)")
	fs.Var(&cf.PostProcessors, "post-processor", "Pipeline post-processor plugin name (repeatable)")
	fs.Var(&cf.Transformers, "transformer", "Output transformer name (repeatable)")
	fs.Var(&cf.TokenValues, "token", "Token value (repeatable)")
	fs.Var(&cf.Headers, "header", "Extra header (repeatable, Key: Value)")
	fs.Var(&cf.Cookies, "cookie", "Cookie value (repeatable, name=value)")
	fs.Var(&cf.ProxyRequiredTags, "proxy-tag", "Required proxy tag when selecting from the loaded proxy pool (repeatable)")
	fs.Var(&cf.ProxyExcludeProxyID, "exclude-proxy-id", "Proxy ID to exclude from proxy-pool selection (repeatable)")
	fs.Var(&cf.InterceptURLPatterns, "intercept-pattern", "URL pattern to intercept (repeatable, glob syntax, e.g., '**/api/**')")
	fs.Var(&cf.InterceptResourceTypes, "intercept-resource-type", "Resource type to intercept (repeatable: xhr,fetch,document,script,stylesheet,image,media,font,websocket,other)")

	return cf
}

func RegisterResearchAgenticFlags(fs *flag.FlagSet, cf *CommonFlags) {
	if cf == nil {
		return
	}
	cf.AgenticResearch = fs.Bool("agentic", false, "Enable bounded pi-powered follow-up and synthesis for research jobs")
	cf.AgenticResearchInstructions = fs.String("agentic-instructions", "", "Additional instructions for agentic research synthesis")
	cf.AgenticResearchMaxRounds = fs.Int("agentic-max-rounds", 1, "Maximum bounded follow-up rounds for agentic research (1-3)")
	cf.AgenticResearchMaxFollowUps = fs.Int("agentic-max-follow-up-urls", 3, "Maximum follow-up URLs selected per agentic research round (1-10)")
}

func RegisterBrowserFlags(fs *flag.FlagSet, cfg config.Config) *BrowserFlags {
	return &BrowserFlags{
		Headless:   fs.Bool("headless", false, "Use headless browser"),
		Playwright: fs.Bool("playwright", cfg.UsePlaywright, "Use Playwright for headless pages"),
		Timeout:    fs.Int("timeout", cfg.RequestTimeoutSecs, "Request timeout in seconds"),
	}
}

func RegisterPipelineFlags(fs *flag.FlagSet) *PipelineFlags {
	pf := &PipelineFlags{
		PreProcessors:  StringSliceFlag{},
		PostProcessors: StringSliceFlag{},
		Transformers:   StringSliceFlag{},
	}
	fs.Var(&pf.PreProcessors, "pre-processor", "Pipeline pre-processor plugin name (repeatable)")
	fs.Var(&pf.PostProcessors, "post-processor", "Pipeline post-processor plugin name (repeatable)")
	fs.Var(&pf.Transformers, "transformer", "Output transformer name (repeatable)")
	return pf
}

func RegisterExtractFlags(fs *flag.FlagSet) *ExtractFlags {
	return &ExtractFlags{
		ExtractTemplate: fs.String("extract-template", "", "Extraction template name"),
		ExtractConfig:   fs.String("extract-config", "", "Path to inline template JSON"),
		ExtractValidate: fs.Bool("extract-validate", false, "Validate extraction against schema"),
	}
}

func RegisterAuthFlags(fs *flag.FlagSet) *AuthFlags {
	af := &AuthFlags{
		AuthBasic:   fs.String("auth-basic", "", "Basic auth user:pass"),
		TokenKind:   fs.String("token-kind", "bearer", "Token kind: bearer|basic|api_key"),
		TokenHeader: fs.String("token-header", "", "Token header name (api_key or bearer override)"),
		TokenQuery:  fs.String("token-query", "", "Token query param name (api_key)"),
		TokenCookie: fs.String("token-cookie", "", "Token cookie name (api_key)"),
		TokenValues: StringSliceFlag{},
		Headers:     StringSliceFlag{},
		Cookies:     StringSliceFlag{},

		LoginURL:            fs.String("login-url", "", "Login URL for headless auth"),
		LoginUserSelector:   fs.String("login-user-selector", "", "CSS selector for username input"),
		LoginPassSelector:   fs.String("login-pass-selector", "", "CSS selector for password input"),
		LoginSubmitSelector: fs.String("login-submit-selector", "", "CSS selector for submit button"),
		LoginUser:           fs.String("login-user", "", "Username for login"),
		LoginPass:           fs.String("login-pass", "", "Password for login"),
		LoginAutoDetect:     fs.Bool("login-auto-detect", false, "Auto-detect login form fields (requires --login-url)"),
		ProxyURL:            fs.String("proxy", "", "Proxy URL (http://, https://, socks5://)"),
		ProxyUsername:       fs.String("proxy-username", "", "Proxy username"),
		ProxyPassword:       fs.String("proxy-password", "", "Proxy password"),
		ProxyRegion:         fs.String("proxy-region", "", "Preferred region when selecting from the loaded proxy pool"),
		ProxyRequiredTags:   StringSliceFlag{},
		ProxyExcludeProxyID: StringSliceFlag{},
	}

	fs.Var(&af.TokenValues, "token", "Token value (repeatable)")
	fs.Var(&af.Headers, "header", "Extra header (repeatable, Key: Value)")
	fs.Var(&af.Cookies, "cookie", "Cookie value (repeatable, name=value)")
	fs.Var(&af.ProxyRequiredTags, "proxy-tag", "Required proxy tag when selecting from the loaded proxy pool (repeatable)")
	fs.Var(&af.ProxyExcludeProxyID, "exclude-proxy-id", "Proxy ID to exclude from proxy-pool selection (repeatable)")

	return af
}

/*
Small string helpers kept private to avoid importing strings repeatedly.
(We keep this file self-contained and test public behavior via cli_test.go.)
*/
