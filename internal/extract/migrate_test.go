// Package extract provides HTML content extraction using selectors, JSON-LD, and regex.
// It handles template-based extraction, field normalization, and schema validation.
// It does NOT handle fetching or rendering HTML content.
package extract

import (
	"testing"
)

func TestMigrateDocument(t *testing.T) {
	tests := []struct {
		name        string
		doc         NormalizedDocument
		fromVersion string
		toVersion   string
		rules       []MigrationRule
		wantErr     bool
		wantFields  map[string]FieldValue
	}{
		{
			name: "same version - no migration needed",
			doc: NormalizedDocument{
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			fromVersion: "1.0.0",
			toVersion:   "1.0.0",
			rules:       []MigrationRule{},
			wantErr:     false,
			wantFields: map[string]FieldValue{
				"name": {Values: []string{"test"}, Source: FieldSourceSelector},
			},
		},
		{
			name: "simple field rename",
			doc: NormalizedDocument{
				Fields: map[string]FieldValue{
					"old_name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			fromVersion: "1.0.0",
			toVersion:   "2.0.0",
			rules: []MigrationRule{
				{
					FromVersion: "1.0.0",
					ToVersion:   "2.0.0",
					Transform:   "old_name->new_name",
				},
			},
			wantErr: false,
			wantFields: map[string]FieldValue{
				"new_name": {Values: []string{"test"}, Source: FieldSourceSelector},
			},
		},
		{
			name: "field deletion",
			doc: NormalizedDocument{
				Fields: map[string]FieldValue{
					"name":     {Values: []string{"test"}, Source: FieldSourceSelector},
					"obsolete": {Values: []string{"old"}, Source: FieldSourceSelector},
				},
			},
			fromVersion: "1.0.0",
			toVersion:   "2.0.0",
			rules: []MigrationRule{
				{
					FromVersion: "1.0.0",
					ToVersion:   "2.0.0",
					Transform:   "delete:obsolete",
				},
			},
			wantErr: false,
			wantFields: map[string]FieldValue{
				"name": {Values: []string{"test"}, Source: FieldSourceSelector},
			},
		},
		{
			name: "multiple transforms",
			doc: NormalizedDocument{
				Fields: map[string]FieldValue{
					"old_name": {Values: []string{"test"}, Source: FieldSourceSelector},
					"obsolete": {Values: []string{"old"}, Source: FieldSourceSelector},
				},
			},
			fromVersion: "1.0.0",
			toVersion:   "2.0.0",
			rules: []MigrationRule{
				{
					FromVersion: "1.0.0",
					ToVersion:   "2.0.0",
					Transform:   "old_name->new_name; delete:obsolete",
				},
			},
			wantErr: false,
			wantFields: map[string]FieldValue{
				"new_name": {Values: []string{"test"}, Source: FieldSourceSelector},
			},
		},
		{
			name: "multi-step migration",
			doc: NormalizedDocument{
				Fields: map[string]FieldValue{
					"v1_field": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			fromVersion: "1.0.0",
			toVersion:   "3.0.0",
			rules: []MigrationRule{
				{
					FromVersion: "1.0.0",
					ToVersion:   "2.0.0",
					Transform:   "v1_field->v2_field",
				},
				{
					FromVersion: "2.0.0",
					ToVersion:   "3.0.0",
					Transform:   "v2_field->v3_field",
				},
			},
			wantErr: false,
			wantFields: map[string]FieldValue{
				"v3_field": {Values: []string{"test"}, Source: FieldSourceSelector},
			},
		},
		{
			name: "no migration rules available",
			doc: NormalizedDocument{
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			fromVersion: "1.0.0",
			toVersion:   "2.0.0",
			rules:       []MigrationRule{},
			wantErr:     true,
		},
		{
			name: "no migration path found",
			doc: NormalizedDocument{
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			fromVersion: "1.0.0",
			toVersion:   "3.0.0",
			rules: []MigrationRule{
				{
					FromVersion: "1.0.0",
					ToVersion:   "2.0.0",
					Transform:   "name->new_name",
				},
				// No rule from 2.0.0 to 3.0.0
			},
			wantErr: true,
		},
		{
			name: "empty transform - no changes",
			doc: NormalizedDocument{
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			fromVersion: "1.0.0",
			toVersion:   "2.0.0",
			rules: []MigrationRule{
				{
					FromVersion: "1.0.0",
					ToVersion:   "2.0.0",
					Transform:   "",
				},
			},
			wantErr: false,
			wantFields: map[string]FieldValue{
				"name": {Values: []string{"test"}, Source: FieldSourceSelector},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MigrateDocument(tt.doc, tt.fromVersion, tt.toVersion, tt.rules)
			if (err != nil) != tt.wantErr {
				t.Errorf("MigrateDocument() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if tt.wantFields != nil {
				if len(got.Fields) != len(tt.wantFields) {
					t.Errorf("MigrateDocument() Fields length = %d, want %d", len(got.Fields), len(tt.wantFields))
				}
				for key, wantVal := range tt.wantFields {
					gotVal, ok := got.Fields[key]
					if !ok {
						t.Errorf("MigrateDocument() missing field %s", key)
						continue
					}
					if len(gotVal.Values) != len(wantVal.Values) {
						t.Errorf("MigrateDocument() field %s Values length = %d, want %d", key, len(gotVal.Values), len(wantVal.Values))
					}
				}
			}
		})
	}
}

func TestValidateVersionCompatibility(t *testing.T) {
	tests := []struct {
		name            string
		docVersion      string
		templateVersion string
		wantErr         bool
	}{
		{
			name:            "same version",
			docVersion:      "1.0.0",
			templateVersion: "1.0.0",
			wantErr:         false,
		},
		{
			name:            "different versions",
			docVersion:      "1.0.0",
			templateVersion: "2.0.0",
			wantErr:         true,
		},
		{
			name:            "empty doc version",
			docVersion:      "",
			templateVersion: "1.0.0",
			wantErr:         false,
		},
		{
			name:            "empty template version",
			docVersion:      "1.0.0",
			templateVersion: "",
			wantErr:         false,
		},
		{
			name:            "both empty",
			docVersion:      "",
			templateVersion: "",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVersionCompatibility(tt.docVersion, tt.templateVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVersionCompatibility() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsVersionDeprecated(t *testing.T) {
	tests := []struct {
		name               string
		version            string
		deprecatedVersions []string
		want               bool
	}{
		{
			name:               "version is deprecated",
			version:            "1.0.0",
			deprecatedVersions: []string{"1.0.0", "1.1.0"},
			want:               true,
		},
		{
			name:               "version is not deprecated",
			version:            "2.0.0",
			deprecatedVersions: []string{"1.0.0", "1.1.0"},
			want:               false,
		},
		{
			name:               "empty deprecated list",
			version:            "1.0.0",
			deprecatedVersions: []string{},
			want:               false,
		},
		{
			name:               "nil deprecated list",
			version:            "1.0.0",
			deprecatedVersions: nil,
			want:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVersionDeprecated(tt.version, tt.deprecatedVersions)
			if got != tt.want {
				t.Errorf("IsVersionDeprecated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLatestVersion(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		want     string
	}{
		{
			name:     "multiple versions",
			versions: []string{"1.0.0", "1.1.0", "2.0.0"},
			want:     "2.0.0",
		},
		{
			name:     "single version",
			versions: []string{"1.0.0"},
			want:     "1.0.0",
		},
		{
			name:     "empty list",
			versions: []string{},
			want:     "",
		},
		{
			name:     "nil list",
			versions: nil,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetLatestVersion(tt.versions)
			if got != tt.want {
				t.Errorf("GetLatestVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseVersionInfo(t *testing.T) {
	tests := []struct {
		name string
		tmpl Template
		want VersionInfo
	}{
		{
			name: "full version info",
			tmpl: Template{
				Name:          "test-template",
				Version:       "1.0.0",
				SchemaVersion: "2020-12",
			},
			want: VersionInfo{
				Version:       "1.0.0",
				SchemaVersion: "2020-12",
				IsDeprecated:  false,
			},
		},
		{
			name: "empty version info",
			tmpl: Template{
				Name: "test-template",
			},
			want: VersionInfo{
				Version:       "",
				SchemaVersion: "",
				IsDeprecated:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseVersionInfo(tt.tmpl)
			if got.Version != tt.want.Version {
				t.Errorf("ParseVersionInfo() Version = %v, want %v", got.Version, tt.want.Version)
			}
			if got.SchemaVersion != tt.want.SchemaVersion {
				t.Errorf("ParseVersionInfo() SchemaVersion = %v, want %v", got.SchemaVersion, tt.want.SchemaVersion)
			}
			if got.IsDeprecated != tt.want.IsDeprecated {
				t.Errorf("ParseVersionInfo() IsDeprecated = %v, want %v", got.IsDeprecated, tt.want.IsDeprecated)
			}
		})
	}
}

func TestVersionedDocument(t *testing.T) {
	doc := VersionedDocument{
		NormalizedDocument: NormalizedDocument{
			URL:   "https://example.com",
			Title: "Test",
		},
		SchemaVersion:   "2020-12",
		TemplateVersion: "1.0.0",
	}

	if doc.URL != "https://example.com" {
		t.Errorf("VersionedDocument.URL = %v, want %v", doc.URL, "https://example.com")
	}
	if doc.SchemaVersion != "2020-12" {
		t.Errorf("VersionedDocument.SchemaVersion = %v, want %v", doc.SchemaVersion, "2020-12")
	}
	if doc.TemplateVersion != "1.0.0" {
		t.Errorf("VersionedDocument.TemplateVersion = %v, want %v", doc.TemplateVersion, "1.0.0")
	}
}

func TestMigrationRule_Struct(t *testing.T) {
	rule := MigrationRule{
		FromVersion: "1.0.0",
		ToVersion:   "2.0.0",
		Transform:   "old->new",
	}

	if rule.FromVersion != "1.0.0" {
		t.Errorf("MigrationRule.FromVersion = %v, want %v", rule.FromVersion, "1.0.0")
	}
	if rule.ToVersion != "2.0.0" {
		t.Errorf("MigrationRule.ToVersion = %v, want %v", rule.ToVersion, "2.0.0")
	}
	if rule.Transform != "old->new" {
		t.Errorf("MigrationRule.Transform = %v, want %v", rule.Transform, "old->new")
	}
}

func TestSchemaVersionInfo_Struct(t *testing.T) {
	info := SchemaVersionInfo{
		Version:       "1.0.0",
		SchemaVersion: "2020-12",
		MigrationRules: []MigrationRule{
			{
				FromVersion: "1.0.0",
				ToVersion:   "2.0.0",
				Transform:   "old->new",
			},
		},
	}

	if info.Version != "1.0.0" {
		t.Errorf("SchemaVersionInfo.Version = %v, want %v", info.Version, "1.0.0")
	}
	if info.SchemaVersion != "2020-12" {
		t.Errorf("SchemaVersionInfo.SchemaVersion = %v, want %v", info.SchemaVersion, "2020-12")
	}
	if len(info.MigrationRules) != 1 {
		t.Errorf("SchemaVersionInfo.MigrationRules length = %v, want %v", len(info.MigrationRules), 1)
	}
}
