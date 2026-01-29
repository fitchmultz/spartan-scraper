package common

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
)

func TestBuildLoginFlow_EmptyInput(t *testing.T) {
	input := LoginFlowInput{}
	got := BuildLoginFlow(input)

	if got != nil {
		t.Errorf("expected nil for empty input, got %v", got)
	}
}

func TestBuildLoginFlow_PartialInput(t *testing.T) {
	tests := []struct {
		name        string
		input       LoginFlowInput
		expectedURL string
		expectNil   bool
	}{
		{"only URL", LoginFlowInput{URL: "http://example.com/login"}, "http://example.com/login", false},
		{"only username", LoginFlowInput{Username: "user"}, "", false},
		{"URL and selector", LoginFlowInput{URL: "http://example.com/login", UserSelector: "#user"}, "http://example.com/login", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildLoginFlow(tt.input)
			if tt.expectNil {
				if got != nil {
					t.Errorf("expected nil for partial input, got %v", got)
				}
			} else {
				if got == nil {
					t.Fatalf("expected non-nil result, got nil")
				}
				if tt.expectedURL != "" && got.URL != tt.expectedURL {
					t.Errorf("expected URL %q, got %q", tt.expectedURL, got.URL)
				}
			}
		})
	}
}

func TestBuildLoginFlow_CompleteInput(t *testing.T) {
	input := LoginFlowInput{
		URL:            "http://example.com/login",
		UserSelector:   "#username",
		PassSelector:   "#password",
		SubmitSelector: "#submit",
		Username:       "testuser",
		Password:       "testpass",
	}

	got := BuildLoginFlow(input)

	if got == nil {
		t.Fatal("expected non-nil result, got nil")
	}

	if got.URL != input.URL {
		t.Errorf("expected URL %q, got %q", input.URL, got.URL)
	}
	if got.UserSelector != input.UserSelector {
		t.Errorf("expected UserSelector %q, got %q", input.UserSelector, got.UserSelector)
	}
	if got.PassSelector != input.PassSelector {
		t.Errorf("expected PassSelector %q, got %q", input.PassSelector, got.PassSelector)
	}
	if got.SubmitSelector != input.SubmitSelector {
		t.Errorf("expected SubmitSelector %q, got %q", input.SubmitSelector, got.SubmitSelector)
	}
	if got.Username != input.Username {
		t.Errorf("expected Username %q, got %q", input.Username, got.Username)
	}
	if got.Password != input.Password {
		t.Errorf("expected Password %q, got %q", input.Password, got.Password)
	}
}

func TestParseTokenKind_ValidValues(t *testing.T) {
	tests := []struct {
		input    string
		expected auth.TokenKind
	}{
		{"bearer", auth.TokenBearer},
		{"Bearer", auth.TokenBearer},
		{"BEARER", auth.TokenBearer},
		{" bearer ", auth.TokenBearer},
		{"basic", auth.TokenBasic},
		{"Basic", auth.TokenBasic},
		{"api_key", auth.TokenApiKey},
		{"api-key", auth.TokenApiKey},
		{"apiKey", auth.TokenApiKey},
		{"API_KEY", auth.TokenApiKey},
		{"api-key-with-dash", auth.TokenBearer},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseTokenKind(tt.input)
			if got != tt.expected {
				t.Errorf("ParseTokenKind(%q) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseTokenKind_InvalidReturnsDefault(t *testing.T) {
	tests := []string{"", "unknown", "custom", "jwt"}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got := ParseTokenKind(input)
			if got != auth.TokenBearer {
				t.Errorf("ParseTokenKind(%q) = %v; want TokenBearer", input, got)
			}
		})
	}
}

func TestBuildTokens_OnlyBasicAuth(t *testing.T) {
	got := BuildTokens("user:pass", nil, "bearer", "", "", "")

	if len(got) != 1 {
		t.Fatalf("expected 1 token, got %d", len(got))
	}

	if got[0].Kind != auth.TokenBasic {
		t.Errorf("expected TokenBasic, got %v", got[0].Kind)
	}
	if got[0].Value != "user:pass" {
		t.Errorf("expected value 'user:pass', got %q", got[0].Value)
	}
}

func TestBuildTokens_MultipleTokens(t *testing.T) {
	tokens := []string{"token1", "token2"}
	got := BuildTokens("", tokens, "api_key", "X-API-Key", "", "")

	if len(got) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(got))
	}

	if got[0].Kind != auth.TokenApiKey {
		t.Errorf("expected TokenApiKey, got %v", got[0].Kind)
	}
	if got[0].Value != "token1" {
		t.Errorf("expected value 'token1', got %q", got[0].Value)
	}
	if got[0].Header != "X-API-Key" {
		t.Errorf("expected header 'X-API-Key', got %q", got[0].Header)
	}
}

func TestBuildTokens_EmptyValuesSkipped(t *testing.T) {
	tokens := []string{"", " ", "valid", "\t"}
	got := BuildTokens("user:pass", tokens, "bearer", "", "", "")

	if len(got) != 2 {
		t.Fatalf("expected 2 tokens (basic + valid), got %d", len(got))
	}

	if got[0].Kind != auth.TokenBasic {
		t.Errorf("first should be TokenBasic, got %v", got[0].Kind)
	}
	if got[1].Value != "valid" {
		t.Errorf("second value should be 'valid', got %q", got[1].Value)
	}
}

func TestBuildTokens_AllOptionalFields(t *testing.T) {
	got := BuildTokens("", []string{"mytoken"}, "api_key", "X-Auth", "auth_param", "session")

	if len(got) != 1 {
		t.Fatalf("expected 1 token, got %d", len(got))
	}

	if got[0].Header != "X-Auth" {
		t.Errorf("expected header 'X-Auth', got %q", got[0].Header)
	}
	if got[0].Query != "auth_param" {
		t.Errorf("expected query 'auth_param', got %q", got[0].Query)
	}
	if got[0].Cookie != "session" {
		t.Errorf("expected cookie 'session', got %q", got[0].Cookie)
	}
}

func TestToHeaderKVs_EmptyMap(t *testing.T) {
	got := ToHeaderKVs(nil)
	if got != nil {
		t.Errorf("expected nil for empty map, got %v", got)
	}

	got = ToHeaderKVs(map[string]string{})
	if got != nil {
		t.Errorf("expected nil for empty map, got %v", got)
	}
}

func TestToHeaderKVs_ValidMap(t *testing.T) {
	input := map[string]string{
		"Authorization": "Bearer token",
		"Content-Type":  "application/json",
	}

	got := ToHeaderKVs(input)

	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}

	foundAuth := false
	foundContentType := false

	for _, kv := range got {
		if kv.Key == "Authorization" {
			foundAuth = true
			if kv.Value != "Bearer token" {
				t.Errorf("expected 'Bearer token', got %q", kv.Value)
			}
		}
		if kv.Key == "Content-Type" {
			foundContentType = true
			if kv.Value != "application/json" {
				t.Errorf("expected 'application/json', got %q", kv.Value)
			}
		}
	}

	if !foundAuth {
		t.Error("expected Authorization header not found")
	}
	if !foundContentType {
		t.Error("expected Content-Type header not found")
	}
}

func TestToHeaderKVs_EmptyKeysSkipped(t *testing.T) {
	input := map[string]string{
		"":          "should-be-skipped",
		"Valid-Key": "valid-value",
		"   ":       "whitespace-key",
	}

	got := ToHeaderKVs(input)

	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}

	if got[0].Key != "Valid-Key" {
		t.Errorf("expected key 'Valid-Key', got %q", got[0].Key)
	}
}

func TestToCookies_EmptySlice(t *testing.T) {
	got := ToCookies(nil)
	if got != nil {
		t.Errorf("expected nil for empty slice, got %v", got)
	}

	got = ToCookies([]string{})
	if got != nil {
		t.Errorf("expected nil for empty slice, got %v", got)
	}
}

func TestToCookies_ValidCookies(t *testing.T) {
	input := []string{
		"session=abc123",
		"user_id=456",
		" pref= val ",
	}

	got := ToCookies(input)

	if len(got) != 3 {
		t.Fatalf("expected 3 cookies, got %d", len(got))
	}

	if got[0].Name != "session" {
		t.Errorf("expected name 'session', got %q", got[0].Name)
	}
	if got[0].Value != "abc123" {
		t.Errorf("expected value 'abc123', got %q", got[0].Value)
	}
}

func TestToCookies_InvalidEntriesSkipped(t *testing.T) {
	input := []string{
		"valid=cookie",
		"no-equals-here",
		"=no-name",
		"  =  no-name  ",
		"",
		"multiple=equals=here",
	}

	got := ToCookies(input)

	if len(got) != 2 {
		t.Fatalf("expected 2 valid cookies, got %d", len(got))
	}

	if got[0].Name != "valid" {
		t.Errorf("expected first name 'valid', got %q", got[0].Name)
	}
	if got[1].Name != "multiple" {
		t.Errorf("expected second name 'multiple', got %q", got[1].Name)
	}
	if got[1].Value != "equals=here" {
		t.Errorf("expected second value 'equals=here', got %q", got[1].Value)
	}
}
