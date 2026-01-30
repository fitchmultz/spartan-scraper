// Package extract provides HTML content extraction using selectors, JSON-LD, and regex.
// It handles template-based extraction, field normalization, and schema validation.
// It does NOT handle fetching or rendering HTML content.
package extract

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// VersionedDocument extends a NormalizedDocument with version metadata.
type VersionedDocument struct {
	NormalizedDocument
	SchemaVersion   string `json:"schemaVersion,omitempty"`
	TemplateVersion string `json:"templateVersion,omitempty"`
}

// MigrateDocument attempts to migrate a document from one schema version to another.
// It applies migration rules in sequence to transform the document.
func MigrateDocument(
	doc NormalizedDocument,
	fromVersion, toVersion string,
	rules []MigrationRule,
) (NormalizedDocument, error) {
	// If versions are the same, no migration needed
	if fromVersion == toVersion {
		return doc, nil
	}

	// If no rules provided, return error
	if len(rules) == 0 {
		return doc, fmt.Errorf("no migration rules available from %s to %s", fromVersion, toVersion)
	}

	// Find migration path
	path := findMigrationPath(fromVersion, toVersion, rules)
	if len(path) == 0 {
		return doc, fmt.Errorf("no migration path found from %s to %s", fromVersion, toVersion)
	}

	// Apply each migration rule in sequence
	currentDoc := doc
	for _, rule := range path {
		migratedDoc, err := applyMigrationRule(currentDoc, rule)
		if err != nil {
			return doc, fmt.Errorf("migration from %s to %s failed: %w", rule.FromVersion, rule.ToVersion, err)
		}
		currentDoc = migratedDoc
	}

	return currentDoc, nil
}

// findMigrationPath finds a sequence of migration rules to get from fromVersion to toVersion.
// It uses a simple breadth-first search to find the shortest path.
func findMigrationPath(fromVersion, toVersion string, rules []MigrationRule) []MigrationRule {
	// Build adjacency list
	graph := make(map[string][]MigrationRule)
	for _, rule := range rules {
		graph[rule.FromVersion] = append(graph[rule.FromVersion], rule)
	}

	// BFS to find shortest path
	type node struct {
		version string
		path    []MigrationRule
	}

	visited := make(map[string]bool)
	queue := []node{{version: fromVersion, path: []MigrationRule{}}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.version == toVersion {
			return current.path
		}

		if visited[current.version] {
			continue
		}
		visited[current.version] = true

		for _, rule := range graph[current.version] {
			newPath := make([]MigrationRule, len(current.path)+1)
			copy(newPath, current.path)
			newPath[len(current.path)] = rule
			queue = append(queue, node{version: rule.ToVersion, path: newPath})
		}
	}

	return nil // No path found
}

// applyMigrationRule applies a single migration rule to a document.
// Currently supports simple field renames and basic transformations.
func applyMigrationRule(doc NormalizedDocument, rule MigrationRule) (NormalizedDocument, error) {
	// For now, we only support simple field renames via transform syntax
	// Format: "oldField->newField" or "delete:fieldName"
	// In the future, this could support JavaScript or JSONata expressions

	if rule.Transform == "" {
		// No transform specified, return document as-is
		return doc, nil
	}

	result := doc
	result.Fields = make(map[string]FieldValue)

	// Copy existing fields
	maps.Copy(result.Fields, doc.Fields)

	// Parse and apply transform
	for t := range strings.SplitSeq(rule.Transform, ";") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}

		// Handle field rename: "old->new"
		if strings.Contains(t, "->") {
			parts := strings.SplitN(t, "->", 2)
			if len(parts) == 2 {
				oldName := strings.TrimSpace(parts[0])
				newName := strings.TrimSpace(parts[1])
				if val, ok := result.Fields[oldName]; ok {
					result.Fields[newName] = val
					delete(result.Fields, oldName)
				}
			}
			continue
		}

		// Handle field deletion: "delete:fieldName"
		if fieldName, ok := strings.CutPrefix(t, "delete:"); ok {
			fieldName = strings.TrimSpace(fieldName)
			delete(result.Fields, fieldName)
			continue
		}

		// Unknown transform, skip
	}

	return result, nil
}

// ValidateVersionCompatibility checks if a document version is compatible with a template version.
func ValidateVersionCompatibility(docVersion, templateVersion string) error {
	// If either version is empty, consider compatible
	if docVersion == "" || templateVersion == "" {
		return nil
	}

	// For now, require exact match
	// In the future, this could support semantic versioning comparison
	if docVersion != templateVersion {
		return fmt.Errorf("version mismatch: document version %s incompatible with template version %s", docVersion, templateVersion)
	}

	return nil
}

// IsVersionDeprecated checks if a given version is deprecated.
// This is a placeholder for future deprecation tracking.
func IsVersionDeprecated(version string, deprecatedVersions []string) bool {
	return slices.Contains(deprecatedVersions, version)
}

// GetLatestVersion returns the latest version from a list of versions.
// For now, it returns the last version in the list.
// In the future, this could use semantic versioning comparison.
func GetLatestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	return versions[len(versions)-1]
}

// VersionInfo contains version metadata for a template.
type VersionInfo struct {
	Version       string
	SchemaVersion string
	IsDeprecated  bool
}

// ParseVersionInfo extracts version information from a template.
func ParseVersionInfo(tmpl Template) VersionInfo {
	return VersionInfo{
		Version:       tmpl.Version,
		SchemaVersion: tmpl.SchemaVersion,
		IsDeprecated:  false, // Could be populated from a deprecation registry
	}
}
