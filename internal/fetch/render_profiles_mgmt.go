// Package fetch provides render profile management utilities.
// This file implements CRUD operations for render profiles stored in DATA_DIR/render_profiles.json.
//
// Responsibilities:
// - Load and save render profiles with strict validation
// - CRUD operations: List, Get, Upsert, Delete
// - Atomic file writes to prevent corruption
// - Validation of profile fields (name uniqueness, host patterns, engine enum)
//
// This file does NOT:
// - Handle runtime profile matching (see render_profiles_store.go)
// - Execute fetches or apply profiles to requests
//
// Invariants:
// - Profile names must be unique (case-sensitive)
// - Host patterns must be non-empty and pass hostmatch.ValidateHostPatterns
// - Engine must be one of: http, chromedp, playwright (if set)
// - File writes are atomic (temp file + rename)
package fetch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/hostmatch"
)

// LoadRenderProfilesFile loads the render profiles file from disk.
// If the file doesn't exist, returns an empty RenderProfilesFile.
// Uses strict JSON decoding - unknown fields cause a validation error.
func LoadRenderProfilesFile(dataDir string) (RenderProfilesFile, error) {
	path := RenderProfilesPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RenderProfilesFile{Profiles: []RenderProfile{}}, nil
		}
		return RenderProfilesFile{}, fmt.Errorf("failed to read render profiles file: %w", err)
	}

	var file RenderProfilesFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return RenderProfilesFile{}, apperrors.Validation(fmt.Sprintf("invalid render profiles JSON: %v", err))
	}

	// Validate the loaded file
	if err := ValidateRenderProfilesFile(file); err != nil {
		return RenderProfilesFile{}, err
	}

	return file, nil
}

// SaveRenderProfilesFile saves the render profiles file to disk atomically.
// Validates before writing. Creates parent directories if needed.
func SaveRenderProfilesFile(dataDir string, file RenderProfilesFile) error {
	if err := ValidateRenderProfilesFile(file); err != nil {
		return err
	}

	path := RenderProfilesPath(dataDir)
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal render profiles: %w", err)
	}
	data = append(data, '\n') // Trailing newline for POSIX compliance

	if err := fsutil.WriteFileAtomic(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to save render profiles: %w", err)
	}

	return nil
}

// ListRenderProfileNames returns a sorted list of all profile names.
func ListRenderProfileNames(dataDir string) ([]string, error) {
	file, err := LoadRenderProfilesFile(dataDir)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(file.Profiles))
	for i, p := range file.Profiles {
		names[i] = p.Name
	}
	sort.Strings(names)
	return names, nil
}

// GetRenderProfile retrieves a single profile by name.
// Returns (profile, true, nil) if found, (zero, false, nil) if not found.
func GetRenderProfile(dataDir, name string) (RenderProfile, bool, error) {
	file, err := LoadRenderProfilesFile(dataDir)
	if err != nil {
		return RenderProfile{}, false, err
	}

	for _, p := range file.Profiles {
		if p.Name == name {
			return p, true, nil
		}
	}
	return RenderProfile{}, false, nil
}

// UpsertRenderProfile creates or updates a render profile.
// If a profile with the same name exists, it is replaced in-place (preserving order).
// If not found, the profile is appended to the end.
func UpsertRenderProfile(dataDir string, profile RenderProfile) error {
	if err := ValidateRenderProfile(profile); err != nil {
		return err
	}

	file, err := LoadRenderProfilesFile(dataDir)
	if err != nil {
		return err
	}

	// Check for name conflicts with other profiles
	found := false
	for i, p := range file.Profiles {
		if p.Name == profile.Name {
			file.Profiles[i] = profile
			found = true
			break
		}
	}

	if !found {
		file.Profiles = append(file.Profiles, profile)
	}

	return SaveRenderProfilesFile(dataDir, file)
}

// DeleteRenderProfile removes a profile by name.
// Returns apperrors.NotFound if the profile doesn't exist.
func DeleteRenderProfile(dataDir, name string) error {
	file, err := LoadRenderProfilesFile(dataDir)
	if err != nil {
		return err
	}

	found := false
	newProfiles := make([]RenderProfile, 0, len(file.Profiles))
	for _, p := range file.Profiles {
		if p.Name == name {
			found = true
			continue
		}
		newProfiles = append(newProfiles, p)
	}

	if !found {
		return apperrors.NotFound(fmt.Sprintf("render profile not found: %s", name))
	}

	file.Profiles = newProfiles
	return SaveRenderProfilesFile(dataDir, file)
}

// ValidateRenderProfilesFile validates an entire render profiles file.
func ValidateRenderProfilesFile(file RenderProfilesFile) error {
	names := make(map[string]int) // name -> index
	for i, p := range file.Profiles {
		if err := ValidateRenderProfile(p); err != nil {
			return fmt.Errorf("profile[%d]: %w", i, err)
		}
		if prevIdx, exists := names[p.Name]; exists {
			return apperrors.Validation(fmt.Sprintf("duplicate profile name %q at indices %d and %d", p.Name, prevIdx, i))
		}
		names[p.Name] = i
	}
	return nil
}

// ValidateRenderProfile validates a single render profile.
func ValidateRenderProfile(p RenderProfile) error {
	if strings.TrimSpace(p.Name) == "" {
		return apperrors.Validation("profile name is required")
	}

	if len(p.HostPatterns) == 0 {
		return apperrors.Validation("hostPatterns is required")
	}

	if err := hostmatch.ValidateHostPatterns(p.HostPatterns); err != nil {
		return apperrors.Validation(fmt.Sprintf("invalid hostPatterns: %v", err))
	}

	// Validate engine enum if set
	if p.ForceEngine != "" {
		switch p.ForceEngine {
		case RenderEngineHTTP, RenderEngineChromedp, RenderEnginePlaywright:
			// valid
		default:
			return apperrors.Validation(fmt.Sprintf("invalid forceEngine %q, must be one of: http, chromedp, playwright", p.ForceEngine))
		}
	}

	// Guardrails for numeric fields
	if p.JSHeavyThreshold < 0 || p.JSHeavyThreshold > 1 {
		return apperrors.Validation("jsHeavyThreshold must be between 0 and 1")
	}

	if p.RateLimitQPS < 0 {
		return apperrors.Validation("rateLimitQPS must be non-negative")
	}

	if p.RateLimitBurst < 0 {
		return apperrors.Validation("rateLimitBurst must be non-negative")
	}

	return nil
}
