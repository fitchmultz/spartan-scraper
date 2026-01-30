// Package common provides shared CLI flag registration.
//
// It does NOT run commands; it only defines types and registration helpers
// to keep flags consistent across command modules.
package common

import (
	"flag"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

type StringSliceFlag []string

func (s *StringSliceFlag) String() string { return stringsJoinComma(*s) }
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
		key = stringsTrimSpace(key)
		val = stringsTrimSpace(val)
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func splitOnce(s, sep string) (string, string, bool) {
	idx := indexOf(s, sep)
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
	ProxyURL      *string
	ProxyUsername *string
	ProxyPassword *string

	// HTTP method and body flags
	Method      *string // HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
	Body        *string // Request body (string or @file path)
	ContentType *string // Content-Type header for request body
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
		ProxyURL:      fs.String("proxy", "", "Proxy URL (http://, https://, socks5://)"),
		ProxyUsername: fs.String("proxy-username", "", "Proxy username"),
		ProxyPassword: fs.String("proxy-password", "", "Proxy password"),

		// HTTP method and body flags
		Method:      fs.String("method", "GET", "HTTP method (GET, POST, PUT, DELETE, PATCH)"),
		Body:        fs.String("body", "", "Request body (string or @file to read from file)"),
		ContentType: fs.String("content-type", "", "Content-Type header (auto-detected if not set)"),
	}

	fs.Var(&cf.PreProcessors, "pre-processor", "Pipeline pre-processor plugin name (repeatable)")
	fs.Var(&cf.PostProcessors, "post-processor", "Pipeline post-processor plugin name (repeatable)")
	fs.Var(&cf.Transformers, "transformer", "Output transformer name (repeatable)")
	fs.Var(&cf.TokenValues, "token", "Token value (repeatable)")
	fs.Var(&cf.Headers, "header", "Extra header (repeatable, Key: Value)")
	fs.Var(&cf.Cookies, "cookie", "Cookie value (repeatable, name=value)")

	return cf
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
	}

	fs.Var(&af.TokenValues, "token", "Token value (repeatable)")
	fs.Var(&af.Headers, "header", "Extra header (repeatable, Key: Value)")
	fs.Var(&af.Cookies, "cookie", "Cookie value (repeatable, name=value)")

	return af
}

/*
Small string helpers kept private to avoid importing strings repeatedly.
(We keep this file self-contained and test public behavior via cli_test.go.)
*/
func stringsJoinComma(items []string) string { return joinWithComma(items) }
func stringsTrimSpace(s string) string       { return trimSpace(s) }
func indexOf(s, sep string) int              { return stringsIndex(s, sep) }
