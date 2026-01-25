package cli

import (
	"reflect"
	"testing"

	"spartan-scraper/internal/auth"
)

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{"a,,c", []string{"a", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitCSV(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("splitCSV(%q) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSplitCSVEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{",", []string{}},                   // Only comma
		{" , ", []string{}},                 // Only comma with whitespace
		{",,", []string{}},                  // Multiple commas
		{", ,", []string{}},                 // Multiple commas with whitespace
		{"a,", []string{"a"}},               // Trailing comma
		{",a", []string{"a"}},               // Leading comma
		{",a,", []string{"a"}},              // Leading and trailing comma
		{"a,,b", []string{"a", "b"}},        // Empty between
		{" , a , , b ", []string{"a", "b"}}, // Mixed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitCSV(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("splitCSV(%q) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToCookies(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []auth.Cookie
	}{
		{"empty", []string{}, nil},
		{"valid", []string{"a=b", "c=d"}, []auth.Cookie{{Name: "a", Value: "b"}, {Name: "c", Value: "d"}}},
		{"invalid", []string{"invalid", "a=b"}, []auth.Cookie{{Name: "a", Value: "b"}}},
		{"whitespace", []string{" a = b "}, []auth.Cookie{{Name: "a", Value: "b"}}},
		{"missing_name", []string{"=value"}, []auth.Cookie{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toCookies(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("toCookies(%v) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToHeaderKVs(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected []auth.HeaderKV
	}{
		{"empty", nil, nil},
		{"valid", map[string]string{"Key": "Value"}, []auth.HeaderKV{{Key: "Key", Value: "Value"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toHeaderKVs(tt.input)
			// map iteration is random, so we just check length and presence if more than 1
			if len(got) != len(tt.expected) {
				t.Errorf("toHeaderKVs(%v) len = %d; want %d", tt.input, len(got), len(tt.expected))
			}
			for _, want := range tt.expected {
				found := false
				for _, g := range got {
					if g.Key == want.Key && g.Value == want.Value {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("toHeaderKVs(%v) missing %v", tt.input, want)
				}
			}
		})
	}
}

func TestParseTokenKind(t *testing.T) {
	tests := []struct {
		input    string
		expected auth.TokenKind
	}{
		{"bearer", auth.TokenBearer},
		{"basic", auth.TokenBasic},
		{"api_key", auth.TokenApiKey},
		{"API-KEY", auth.TokenApiKey},
		{"apikey", auth.TokenApiKey},
		{"unknown", auth.TokenBearer},
		{"", auth.TokenBearer},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseTokenKind(tt.input)
			if got != tt.expected {
				t.Errorf("parseTokenKind(%q) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildTokens(t *testing.T) {
	tests := []struct {
		name        string
		basic       string
		tokens      []string
		kind        string
		header      string
		query       string
		cookie      string
		expectedLen int
	}{
		{"empty", "", nil, "bearer", "", "", "", 0},
		{"basic_only", "user:pass", nil, "bearer", "", "", "", 1},
		{"tokens_only", "", []string{"t1", "t2"}, "bearer", "", "", "", 2},
		{"both", "user:pass", []string{"t1"}, "bearer", "", "", "", 2},
		{"with_fields", "", []string{"val"}, "api_key", "X-Key", "k", "c", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTokens(tt.basic, tt.tokens, tt.kind, tt.header, tt.query, tt.cookie)
			if len(got) != tt.expectedLen {
				t.Errorf("buildTokens() len = %d; want %d", len(got), tt.expectedLen)
			}
			if tt.name == "with_fields" && len(got) > 0 {
				if got[0].Header != tt.header || got[0].Query != tt.query || got[0].Cookie != tt.cookie {
					t.Errorf("buildTokens() fields mismatch: %+v", got[0])
				}
			}
		})
	}
}

func TestBuildLoginFlow(t *testing.T) {
	tests := []struct {
		name     string
		input    loginFlowInput
		expected *auth.LoginFlow
	}{
		{"empty", loginFlowInput{}, nil},
		{"full", loginFlowInput{URL: "http://login", Username: "user"}, &auth.LoginFlow{URL: "http://login", Username: "user"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLoginFlow(tt.input)
			if (got == nil) != (tt.expected == nil) {
				t.Fatalf("buildLoginFlow() nil mismatch: got %v, want %v", got == nil, tt.expected == nil)
			}
			if got != nil && tt.expected != nil {
				if got.URL != tt.expected.URL || got.Username != tt.expected.Username {
					t.Errorf("buildLoginFlow() = %+v; want %+v", got, tt.expected)
				}
			}
		})
	}
}

func TestStringSliceFlag(t *testing.T) {
	var s stringSliceFlag
	if s.String() != "" {
		t.Errorf("empty stringSliceFlag.String() = %q; want \"\"", s.String())
	}

	if err := s.Set("a:b"); err != nil {
		t.Errorf("Set() error: %v", err)
	}
	if err := s.Set("c:d"); err != nil {
		t.Errorf("Set() error: %v", err)
	}

	if s.String() != "a:b,c:d" {
		t.Errorf("stringSliceFlag.String() = %q; want \"a:b,c:d\"", s.String())
	}

	m := s.ToMap()
	expected := map[string]string{"a": "b", "c": "d"}
	if !reflect.DeepEqual(m, expected) {
		t.Errorf("ToMap() = %v; want %v", m, expected)
	}

	// Test invalid entries in ToMap
	var s2 stringSliceFlag
	_ = s2.Set("invalid")
	_ = s2.Set(" : ")
	if len(s2.ToMap()) != 0 {
		t.Errorf("ToMap() should be empty for invalid entries, got %v", s2.ToMap())
	}
}
