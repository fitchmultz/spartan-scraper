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
	TokenOAuth2 TokenKind = "oauth2"
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

	// AutoDetect enables automatic form field detection.
	// When true, the system will attempt to detect login form fields
	// automatically without requiring manual CSS selector configuration.
	AutoDetect bool `json:"autoDetect,omitempty"`

	// ConfidenceScore indicates the confidence level of auto-detection (0.0-1.0).
	// Only populated when AutoDetect is true.
	ConfidenceScore float64 `json:"confidenceScore,omitempty"`
}

// OAuth2FlowType defines the OAuth 2.0 flow variant.
type OAuth2FlowType string

const (
	OAuth2FlowAuthorizationCode OAuth2FlowType = "authorization_code"
	OAuth2FlowClientCredentials OAuth2FlowType = "client_credentials"
	OAuth2FlowDeviceCode        OAuth2FlowType = "device_code"
)

// OAuth2Config defines OAuth 2.0 client configuration.
type OAuth2Config struct {
	FlowType     OAuth2FlowType `json:"flow_type"`
	ClientID     string         `json:"client_id"`
	ClientSecret string         `json:"client_secret,omitempty"`
	AuthorizeURL string         `json:"authorize_url,omitempty"` // For auth code flow
	TokenURL     string         `json:"token_url"`
	RevokeURL    string         `json:"revoke_url,omitempty"`
	Scopes       []string       `json:"scopes,omitempty"`
	UsePKCE      bool           `json:"use_pkce"` // Required for public clients
	RedirectURI  string         `json:"redirect_uri,omitempty"`
	// OIDC-specific
	DiscoveryURL string `json:"discovery_url,omitempty"` // .well-known/openid-configuration
	Issuer       string `json:"issuer,omitempty"`
}

// OAuth2Token represents an OAuth 2.0 token set with refresh capability.
type OAuth2Token struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	TokenType    string     `json:"token_type"` // "Bearer", etc.
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Scope        string     `json:"scope,omitempty"`
}

// OAuth2State represents an in-progress OAuth 2.0 authorization flow.
type OAuth2State struct {
	State        string    `json:"state"`         // CSRF protection token
	CodeVerifier string    `json:"code_verifier"` // PKCE verifier
	ProfileName  string    `json:"profile_name"`  // Associated profile
	RedirectURI  string    `json:"redirect_uri"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"` // State expires for security
}

// OIDCProviderMetadata represents OIDC discovery document.
type OIDCProviderMetadata struct {
	Issuer                 string   `json:"issuer"`
	AuthorizationEndpoint  string   `json:"authorization_endpoint"`
	TokenEndpoint          string   `json:"token_endpoint"`
	UserinfoEndpoint       string   `json:"userinfo_endpoint,omitempty"`
	RevocationEndpoint     string   `json:"revocation_endpoint,omitempty"`
	JWKSURI                string   `json:"jwks_uri,omitempty"`
	ScopesSupported        []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported []string `json:"response_types_supported,omitempty"`
	GrantTypesSupported    []string `json:"grant_types_supported,omitempty"`
	CodeChallengeMethods   []string `json:"code_challenge_methods_supported,omitempty"`
}

type Profile struct {
	Name    string         `json:"name"`
	Parents []string       `json:"parents,omitempty"`
	Headers []HeaderKV     `json:"headers,omitempty"`
	Cookies []Cookie       `json:"cookies,omitempty"`
	Tokens  []Token        `json:"tokens,omitempty"`
	Login   *LoginFlow     `json:"login,omitempty"`
	OAuth2  *OAuth2Config  `json:"oauth2,omitempty"`
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
