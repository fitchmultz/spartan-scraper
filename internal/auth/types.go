package auth

type TokenKind string

const (
	TokenBearer TokenKind = "bearer"
	TokenBasic  TokenKind = "basic"
	TokenApiKey TokenKind = "api_key"
)

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
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain,omitempty"`
	Path   string `json:"path,omitempty"`
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
