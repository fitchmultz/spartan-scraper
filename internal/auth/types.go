// Package auth provides authentication profile management and credential resolution.
// It handles profile inheritance, preset matching, environment variable overrides,
// profile persistence (Load/Save vault), and CRUD operations.
// It does NOT handle authentication execution.
package auth

import "time"

type TokenKind string

const (
	TokenBearer TokenKind = "bearer"
	TokenBasic  TokenKind = "basic"
	TokenApiKey TokenKind = "api_key"
)

// APIKeyPermission defines the access level for an API key
type APIKeyPermission string

const (
	APIKeyPermissionReadOnly  APIKeyPermission = "read_only"
	APIKeyPermissionReadWrite APIKeyPermission = "read_write"
)

// APIKey represents a server-side API key for authenticating API requests
type APIKey struct {
	Key         string           `json:"key"`         // The actual API key (prefixed with "ss_" for identification)
	Name        string           `json:"name"`        // Human-readable name for the key
	Permissions APIKeyPermission `json:"permissions"` // read_only or read_write
	CreatedAt   time.Time        `json:"created_at"`
	ExpiresAt   *time.Time       `json:"expires_at,omitempty"` // nil means no expiration
	LastUsedAt  *time.Time       `json:"last_used_at,omitempty"`
}

type Token struct {
	Kind   TokenKind `json:"kind"`
	Value  string    `json:"value"`
	Header string    `json:"header,omitempty"`
	Query  string    `json:"query,omitempty"`
	Cookie string    `json:"cookie,omitempty"`
}

type HeaderKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Cookie struct {
	Name     string     `json:"name"`
	Value    string     `json:"value"`
	Domain   string     `json:"domain,omitempty"`
	Path     string     `json:"path,omitempty"`
	Expires  *time.Time `json:"expires,omitempty"`
	Secure   bool       `json:"secure,omitempty"`
	HttpOnly bool       `json:"httpOnly,omitempty"`
	SameSite string     `json:"sameSite,omitempty"`
}

// Session represents a persisted cookie session for a domain.
// Sessions store cookies from successful logins and can be reused across requests.
type Session struct {
	ID        string     `json:"id"`      // Unique session identifier (user-defined or auto-generated)
	Name      string     `json:"name"`    // Human-readable name
	Domain    string     `json:"domain"`  // Domain this session is for (e.g., "example.com")
	Cookies   []Cookie   `json:"cookies"` // Persisted cookies
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // Optional session expiration
}

type LoginFlow struct {
	URL            string `json:"url,omitempty"`
	UserSelector   string `json:"userSelector,omitempty"`
	PassSelector   string `json:"passSelector,omitempty"`
	SubmitSelector string `json:"submitSelector,omitempty"`
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
}

type Profile struct {
	Name    string         `json:"name"`
	Parents []string       `json:"parents,omitempty"`
	Headers []HeaderKV     `json:"headers,omitempty"`
	Cookies []Cookie       `json:"cookies,omitempty"`
	Tokens  []Token        `json:"tokens,omitempty"`
	Login   *LoginFlow     `json:"login,omitempty"`
	Presets []TargetPreset `json:"presets,omitempty"`
}

type TargetPreset struct {
	Name         string     `json:"name"`
	HostPatterns []string   `json:"hostPatterns,omitempty"`
	Profile      string     `json:"profile,omitempty"`
	Headers      []HeaderKV `json:"headers,omitempty"`
	Cookies      []Cookie   `json:"cookies,omitempty"`
	Tokens       []Token    `json:"tokens,omitempty"`
}

type Vault struct {
	Version  string         `json:"version"`
	Profiles []Profile      `json:"profiles"`
	Presets  []TargetPreset `json:"presets,omitempty"`
}

type EnvOverrides struct {
	Basic        string            `json:"basic,omitempty"`
	Bearer       string            `json:"bearer,omitempty"`
	APIKey       string            `json:"apiKey,omitempty"`
	APIKeyHeader string            `json:"apiKeyHeader,omitempty"`
	APIKeyQuery  string            `json:"apiKeyQuery,omitempty"`
	APIKeyCookie string            `json:"apiKeyCookie,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Cookies      map[string]string `json:"cookies,omitempty"`
}

type ResolveInput struct {
	ProfileName string
	URL         string
	Headers     []HeaderKV
	Cookies     []Cookie
	Tokens      []Token
	Login       *LoginFlow
	Env         *EnvOverrides
}

type ResolvedAuth struct {
	Headers map[string]string `json:"headers,omitempty"`
	Cookies []string          `json:"cookies,omitempty"`
	Query   map[string]string `json:"query,omitempty"`
	Basic   string            `json:"basic,omitempty"`
	Login   *LoginFlow        `json:"login,omitempty"`
}
