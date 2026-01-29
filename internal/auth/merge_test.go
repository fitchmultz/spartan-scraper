package auth

import (
	"reflect"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestMergeProfiles(t *testing.T) {
	vault := Vault{
		Profiles: []Profile{
			{
				Name:    "base",
				Headers: []HeaderKV{{Key: "X-Base", Value: "1"}},
			},
			{
				Name:    "child",
				Parents: []string{"base"},
				Headers: []HeaderKV{{Key: "X-Child", Value: "2"}},
			},
			{
				Name:    "override",
				Parents: []string{"base"},
				Headers: []HeaderKV{{Key: "X-Base", Value: "overridden"}},
			},
			{
				Name:    "diamond-left",
				Parents: []string{"base"},
				Headers: []HeaderKV{{Key: "X-Side", Value: "left"}},
			},
			{
				Name:    "diamond-right",
				Parents: []string{"base"},
				Headers: []HeaderKV{{Key: "X-Side", Value: "right"}},
			},
			{
				Name:    "diamond",
				Parents: []string{"diamond-left", "diamond-right"},
			},
		},
	}

	tests := []struct {
		name     string
		profile  string
		expected map[string]string
		wantErr  bool
	}{
		{"base", "base", map[string]string{"X-Base": "1"}, false},
		{"child", "child", map[string]string{"X-Base": "1", "X-Child": "2"}, false},
		{"override", "override", map[string]string{"X-Base": "overridden"}, false},
		{"diamond", "diamond", map[string]string{"X-Base": "1", "X-Side": "right"}, false},
		{"missing", "no-such-profile", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := MergeProfiles(vault, tt.profile, map[string]bool{})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("MergeProfiles() expected error, got nil")
				}
				if !apperrors.IsKind(err, apperrors.KindValidation) && !apperrors.IsKind(err, apperrors.KindNotFound) {
					t.Fatalf("MergeProfiles() error kind = %v, expected Validation or NotFound, got: %v", apperrors.KindOf(err), err)
				}
			} else {
				if err != nil {
					t.Fatalf("MergeProfiles() unexpected error = %v", err)
				}
				headers := map[string]string{}
				for _, h := range merged.Headers {
					headers[h.Key] = h.Value
				}
				if !reflect.DeepEqual(headers, tt.expected) {
					t.Errorf("MergeProfiles() headers = %v, want %v", headers, tt.expected)
				}
			}
		})
	}
}

func TestMergeProfilesCycle(t *testing.T) {
	vault := Vault{
		Profiles: []Profile{
			{Name: "a", Parents: []string{"b"}},
			{Name: "b", Parents: []string{"a"}},
		},
	}
	_, err := MergeProfiles(vault, "a", map[string]bool{})
	if err == nil {
		t.Errorf("expected cycle error, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Errorf("expected Validation kind for cycle error, got %v", apperrors.KindOf(err))
	}
}

func TestMatchPreset(t *testing.T) {
	vault := Vault{
		Presets: []TargetPreset{
			{Name: "exact", HostPatterns: []string{"example.com"}},
			{Name: "wildcard", HostPatterns: []string{"*.test.com"}},
		},
	}

	tests := []struct {
		url      string
		wantName string
		wantOk   bool
	}{
		{"https://example.com/path", "exact", true},
		{"http://sub.test.com", "wildcard", true},
		{"https://other.com", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, ok := MatchPreset(vault, tt.url)
			if ok != tt.wantOk {
				t.Errorf("MatchPreset() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && got.Name != tt.wantName {
				t.Errorf("MatchPreset() name = %q, want %q", got.Name, tt.wantName)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	dataDir := t.TempDir()
	vault := Vault{
		Profiles: []Profile{
			{
				Name:    "p1",
				Headers: []HeaderKV{{Key: "X-P1", Value: "v1"}},
			},
		},
	}
	_ = SaveVault(dataDir, vault)

	in := ResolveInput{
		ProfileName: "p1",
		Headers:     []HeaderKV{{Key: "X-Extra", Value: "v2"}},
	}
	res, err := Resolve(dataDir, in)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if res.Headers["X-P1"] != "v1" || res.Headers["X-Extra"] != "v2" {
		t.Errorf("Resolve() unexpected headers: %v", res.Headers)
	}
}

func TestHostMatchesAnyPattern(t *testing.T) {
	tests := []struct {
		host     string
		patterns []string
		expected bool
	}{
		{"example.com", []string{"example.com"}, true},
		{"sub.example.com", []string{"*.example.com"}, true},
		{"example.com", []string{"*.example.com"}, false},
		{"foo.bar.com", []string{"foo.*"}, true},
		{"foo.bar.com", []string{"bar.*"}, false},
		{"example.com", []string{"EXAMPLE.COM"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := hostMatchesAnyPattern(tt.host, tt.patterns); got != tt.expected {
				t.Errorf("hostMatchesAnyPattern(%q, %v) = %v, want %v", tt.host, tt.patterns, got, tt.expected)
			}
		})
	}
}
