// Package pipeline provides JavaScript script management utilities.
// This file implements CRUD operations for pipeline JS scripts stored in DATA_DIR/pipeline_js.json.
//
// Responsibilities:
// - Load and save JS registry with strict validation
// - CRUD operations: List, Get, Upsert, Delete
// - Atomic file writes to prevent corruption
// - Validation of script fields (name uniqueness, host patterns, engine enum)
//
// This file does NOT:
// - Execute JavaScript code
// - Handle script matching at runtime (see js.go)
//
// Invariants:
// - Script names must be unique (case-sensitive)
// - Host patterns must be non-empty and pass hostmatch.ValidateHostPatterns
// - Engine must be one of: chromedp, playwright (if set)
// - File writes are atomic (temp file + rename)
package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/hostmatch"
)

// LoadJSRegistryStrict loads the JS registry from disk with strict validation.
// If the file doesn't exist, returns an empty registry.
// Uses strict JSON decoding - unknown fields cause a validation error.
func LoadJSRegistryStrict(dataDir string) (JSRegistry, error) {
	path := jsRegistryPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return JSRegistry{Scripts: []JSTargetScript{}}, nil
		}
		return JSRegistry{}, fmt.Errorf("failed to read JS registry: %w", err)
	}

	var registry JSRegistry
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&registry); err != nil {
		return JSRegistry{}, apperrors.Validation(fmt.Sprintf("invalid JS registry JSON: %v", err))
	}

	// Validate the loaded registry
	if err := ValidateJSRegistry(registry); err != nil {
		return JSRegistry{}, err
	}

	return registry, nil
}

// SaveJSRegistry saves the JS registry to disk atomically.
// Validates before writing. Creates parent directories if needed.
func SaveJSRegistry(dataDir string, registry JSRegistry) error {
	if err := ValidateJSRegistry(registry); err != nil {
		return err
	}

	path := jsRegistryPath(dataDir)
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JS registry: %w", err)
	}
	data = append(data, '\n') // Trailing newline for POSIX compliance

	if err := fsutil.WriteFileAtomic(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to save JS registry: %w", err)
	}

	return nil
}

// ListJSScriptNames returns a sorted list of all script names.
func ListJSScriptNames(dataDir string) ([]string, error) {
	registry, err := LoadJSRegistryStrict(dataDir)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(registry.Scripts))
	for i, s := range registry.Scripts {
		names[i] = s.Name
	}
	sort.Strings(names)
	return names, nil
}

// GetJSScript retrieves a single script by name.
// Returns (script, true, nil) if found, (zero, false, nil) if not found.
func GetJSScript(dataDir, name string) (JSTargetScript, bool, error) {
	registry, err := LoadJSRegistryStrict(dataDir)
	if err != nil {
		return JSTargetScript{}, false, err
	}

	for _, s := range registry.Scripts {
		if s.Name == name {
			return s, true, nil
		}
	}
	return JSTargetScript{}, false, nil
}

// UpsertJSScript creates or updates a JS script.
// If a script with the same name exists, it is replaced in-place (preserving order).
// If not found, the script is appended to the end.
func UpsertJSScript(dataDir string, script JSTargetScript) error {
	if err := ValidateJSTargetScript(script); err != nil {
		return err
	}

	registry, err := LoadJSRegistryStrict(dataDir)
	if err != nil {
		return err
	}

	// Check for name conflicts with other scripts
	found := false
	for i, s := range registry.Scripts {
		if s.Name == script.Name {
			registry.Scripts[i] = script
			found = true
			break
		}
	}

	if !found {
		registry.Scripts = append(registry.Scripts, script)
	}

	return SaveJSRegistry(dataDir, registry)
}

// DeleteJSScript removes a script by name.
// Returns apperrors.NotFound if the script doesn't exist.
func DeleteJSScript(dataDir, name string) error {
	registry, err := LoadJSRegistryStrict(dataDir)
	if err != nil {
		return err
	}

	found := false
	newScripts := make([]JSTargetScript, 0, len(registry.Scripts))
	for _, s := range registry.Scripts {
		if s.Name == name {
			found = true
			continue
		}
		newScripts = append(newScripts, s)
	}

	if !found {
		return apperrors.NotFound(fmt.Sprintf("JS script not found: %s", name))
	}

	registry.Scripts = newScripts
	return SaveJSRegistry(dataDir, registry)
}

// ValidateJSRegistry validates an entire JS registry.
func ValidateJSRegistry(registry JSRegistry) error {
	names := make(map[string]int) // name -> index
	for i, s := range registry.Scripts {
		if err := ValidateJSTargetScript(s); err != nil {
			return fmt.Errorf("script[%d]: %w", i, err)
		}
		if prevIdx, exists := names[s.Name]; exists {
			return apperrors.Validation(fmt.Sprintf("duplicate script name %q at indices %d and %d", s.Name, prevIdx, i))
		}
		names[s.Name] = i
	}
	return nil
}

// ValidateJSTargetScript validates a single JS target script.
func ValidateJSTargetScript(s JSTargetScript) error {
	if strings.TrimSpace(s.Name) == "" {
		return apperrors.Validation("script name is required")
	}

	if len(s.HostPatterns) == 0 {
		return apperrors.Validation("hostPatterns is required")
	}

	if err := hostmatch.ValidateHostPatterns(s.HostPatterns); err != nil {
		return apperrors.Validation(fmt.Sprintf("invalid hostPatterns: %v", err))
	}

	// Validate engine enum if set
	if s.Engine != "" {
		engine := strings.ToLower(strings.TrimSpace(s.Engine))
		switch engine {
		case EngineChromedp, EnginePlaywright:
			// valid
		default:
			return apperrors.Validation(fmt.Sprintf("invalid engine %q, must be one of: chromedp, playwright", s.Engine))
		}
	}

	return nil
}

// jsRegistryPath returns the full path to the JS registry file.
func jsRegistryPath(dataDir string) string {
	base := strings.TrimSpace(dataDir)
	if base == "" {
		base = ".data"
	}
	return filepath.Join(base, jsRegistryFile)
}
