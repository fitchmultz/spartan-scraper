package fetch

import (
	"net/url"
	"testing"
)

func TestApplyAuthQuery(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		query    map[string]string
		want     string
		validate func(t *testing.T, got string)
	}{
		{
			name:   "empty query map returns original",
			rawURL: "https://example.com",
			query:  nil,
			want:   "https://example.com",
		},
		{
			name:   "empty query map returns original with empty map",
			rawURL: "https://example.com",
			query:  map[string]string{},
			want:   "https://example.com",
		},
		{
			name:   "adds params to URL without query",
			rawURL: "https://example.com",
			query:  map[string]string{"key": "value"},
			want:   "https://example.com?key=value",
		},
		{
			name:   "appends params to URL with existing query",
			rawURL: "https://example.com?existing=param",
			query:  map[string]string{"new": "value"},
			want:   "https://example.com?existing=param&new=value",
		},
		{
			name:   "preserves fragment",
			rawURL: "https://example.com#section",
			query:  map[string]string{"key": "value"},
			want:   "https://example.com?key=value#section",
		},
		{
			name:   "special characters are URL encoded",
			rawURL: "https://example.com",
			query:  map[string]string{"key": "value with spaces"},
			want:   "https://example.com?key=value+with+spaces",
		},
		{
			name:   "empty key is skipped",
			rawURL: "https://example.com",
			query:  map[string]string{"": "value", "valid": "data"},
			want:   "https://example.com?valid=data",
		},
		{
			name:   "multiple query params",
			rawURL: "https://example.com",
			query:  map[string]string{"a": "1", "b": "2", "c": "3"},
			want:   "https://example.com?a=1&b=2&c=3",
		},
		{
			name:   "invalid URL returns original",
			rawURL: "://not-a-url",
			query:  map[string]string{"key": "value"},
			want:   "://not-a-url",
		},
		{
			name:   "URL with duplicate params uses new value",
			rawURL: "https://example.com?key=old",
			query:  map[string]string{"key": "new"},
			want:   "https://example.com?key=new",
		},
		{
			name:   "complex URL with path and existing query",
			rawURL: "https://api.example.com/v1/users?page=2&limit=10",
			query:  map[string]string{"filter": "active"},
			validate: func(t *testing.T, got string) {
				parsed, err := url.Parse(got)
				if err != nil {
					t.Errorf("failed to parse result: %v", err)
					return
				}
				values := parsed.Query()
				if values.Get("page") != "2" || values.Get("limit") != "10" || values.Get("filter") != "active" {
					t.Errorf("URL missing or wrong params: %v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyAuthQuery(tt.rawURL, tt.query)
			if tt.validate != nil {
				tt.validate(t, got)
			} else if got != tt.want {
				t.Errorf("ApplyAuthQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
