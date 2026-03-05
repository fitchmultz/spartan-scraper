// Package auth provides authentication profile management and credential resolution.
// It handles profile inheritance, preset matching, environment variable overrides,
// profile persistence (Load/Save vault), and CRUD operations.
// It does NOT handle authentication execution.
package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
)

const (
	vaultVersion  = "1"
	vaultFilename = "auth_vault.json"
	legacyFile    = "profiles.json"
)

var (
	ErrInvalidPath = apperrors.ErrInvalidPath
)

func LoadVault(dataDir string) (Vault, error) {
	path := vaultPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			migrated, ok, migErr := migrateLegacyProfiles(dataDir)
			if migErr != nil {
				return Vault{}, migErr
			}
			if ok {
				if saveErr := SaveVault(dataDir, migrated); saveErr != nil {
					return Vault{}, saveErr
				}
				return migrated, nil
			}
			return Vault{Version: vaultVersion, Profiles: []Profile{}, Presets: []TargetPreset{}}, nil
		}
		return Vault{}, err
	}
	var vault Vault
	if err := json.Unmarshal(data, &vault); err != nil {
		return Vault{}, err
	}
	if vault.Version == "" {
		vault.Version = vaultVersion
	}
	if vault.Profiles == nil {
		vault.Profiles = []Profile{}
	}
	return vault, nil
}

func SaveVault(dataDir string, vault Vault) error {
	if vault.Version == "" {
		vault.Version = vaultVersion
	}
	if err := fsutil.EnsureDataDir(dataDirOrDefault(dataDir)); err != nil {
		return err
	}
	path := vaultPath(dataDir)
	payload, err := json.MarshalIndent(vault, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

func UpsertProfile(dataDir string, profile Profile) error {
	if strings.TrimSpace(profile.Name) == "" {
		return apperrors.Validation("profile name is required")
	}
	vault, err := LoadVault(dataDir)
	if err != nil {
		return err
	}
	found := false
	for i := range vault.Profiles {
		if vault.Profiles[i].Name == profile.Name {
			vault.Profiles[i] = profile
			found = true
			break
		}
	}
	if !found {
		vault.Profiles = append(vault.Profiles, profile)
	}
	return SaveVault(dataDir, vault)
}

func DeleteProfile(dataDir string, name string) error {
	vault, err := LoadVault(dataDir)
	if err != nil {
		return err
	}
	filtered := make([]Profile, 0, len(vault.Profiles))
	for _, profile := range vault.Profiles {
		if profile.Name != name {
			filtered = append(filtered, profile)
		}
	}
	vault.Profiles = filtered
	return SaveVault(dataDir, vault)
}

func GetProfile(dataDir, name string) (Profile, bool, error) {
	vault, err := LoadVault(dataDir)
	if err != nil {
		return Profile{}, false, err
	}
	for _, profile := range vault.Profiles {
		if profile.Name == name {
			return profile, true, nil
		}
	}
	return Profile{}, false, nil
}

func ListProfileNames(dataDir string) ([]string, error) {
	vault, err := LoadVault(dataDir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(vault.Profiles))
	for _, profile := range vault.Profiles {
		out = append(out, profile.Name)
	}
	sort.Strings(out)
	return out, nil
}

func ImportVault(dataDir string, path string) error {
	if err := validateVaultPath(path); err != nil {
		return err
	}
	fullPath := filepath.Join(dataDirOrDefault(dataDir), path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	var vault Vault
	if err := json.Unmarshal(data, &vault); err != nil {
		return err
	}
	if vault.Version == "" {
		vault.Version = vaultVersion
	}
	return SaveVault(dataDir, vault)
}

func ExportVault(dataDir string, path string) error {
	if err := validateVaultPath(path); err != nil {
		return err
	}
	vault, err := LoadVault(dataDir)
	if err != nil {
		return err
	}
	fullPath := filepath.Join(dataDirOrDefault(dataDir), path)
	payload, err := json.MarshalIndent(vault, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fullPath, payload, 0o600)
}

func UpsertPreset(dataDir string, preset TargetPreset) error {
	if strings.TrimSpace(preset.Name) == "" {
		return apperrors.Validation("preset name is required")
	}
	vault, err := LoadVault(dataDir)
	if err != nil {
		return err
	}
	found := false
	for i := range vault.Presets {
		if vault.Presets[i].Name == preset.Name {
			vault.Presets[i] = preset
			found = true
			break
		}
	}
	if !found {
		vault.Presets = append(vault.Presets, preset)
	}
	return SaveVault(dataDir, vault)
}

func DeletePreset(dataDir string, name string) error {
	vault, err := LoadVault(dataDir)
	if err != nil {
		return err
	}
	filtered := make([]TargetPreset, 0, len(vault.Presets))
	for _, preset := range vault.Presets {
		if preset.Name != name {
			filtered = append(filtered, preset)
		}
	}
	vault.Presets = filtered
	return SaveVault(dataDir, vault)
}

func vaultPath(dataDir string) string {
	return filepath.Join(dataDirOrDefault(dataDir), vaultFilename)
}

func legacyProfilesPath(dataDir string) string {
	return filepath.Join(dataDirOrDefault(dataDir), legacyFile)
}

func dataDirOrDefault(dataDir string) string {
	base := strings.TrimSpace(dataDir)
	if base == "" {
		base = ".data"
	}
	return base
}

func validateVaultPath(path string) error {
	if path == "" {
		return apperrors.Validation("path is required")
	}
	if path == "." || path == ".." {
		return apperrors.WithKind(apperrors.KindValidation, ErrInvalidPath)
	}
	if strings.ContainsAny(path, "/\\") {
		return apperrors.WithKind(apperrors.KindValidation, ErrInvalidPath)
	}
	return nil
}

func migrateLegacyProfiles(dataDir string) (Vault, bool, error) {
	path := legacyProfilesPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Vault{}, false, nil
		}
		return Vault{}, false, err
	}

	var legacy struct {
		Profiles []struct {
			Name string            `json:"name"`
			Auth fetch.AuthOptions `json:"auth"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return Vault{}, false, err
	}

	profiles := make([]Profile, 0, len(legacy.Profiles))
	for _, p := range legacy.Profiles {
		profile := Profile{Name: p.Name}
		profile.Headers = headersFromMap(p.Auth.Headers)
		profile.Cookies = cookiesFromStrings(p.Auth.Cookies)
		if p.Auth.Basic != "" {
			profile.Tokens = append(profile.Tokens, Token{
				Kind:  TokenBasic,
				Value: p.Auth.Basic,
			})
		}
		if len(p.Auth.Query) > 0 {
			for key, value := range p.Auth.Query {
				if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
					continue
				}
				profile.Tokens = append(profile.Tokens, Token{
					Kind:  TokenApiKey,
					Value: value,
					Query: key,
				})
			}
		}
		if login := loginFromLegacy(p.Auth); login != nil {
			profile.Login = login
		}
		profiles = append(profiles, profile)
	}

	return Vault{
		Version:  vaultVersion,
		Profiles: profiles,
		Presets:  []TargetPreset{},
	}, true, nil
}

func headersFromMap(headers map[string]string) []HeaderKV {
	if len(headers) == 0 {
		return nil
	}
	out := make([]HeaderKV, 0, len(headers))
	for k, v := range headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		out = append(out, HeaderKV{Key: k, Value: v})
	}
	return out
}

func cookiesFromStrings(cookies []string) []Cookie {
	if len(cookies) == 0 {
		return nil
	}
	out := make([]Cookie, 0, len(cookies))
	for _, raw := range cookies {
		if c, ok := parseCookieString(raw); ok {
			out = append(out, c)
		}
	}
	return out
}

func parseCookieString(raw string) (Cookie, bool) {
	parts := strings.SplitN(strings.TrimSpace(raw), "=", 2)
	if len(parts) != 2 {
		return Cookie{}, false
	}
	name := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if name == "" {
		return Cookie{}, false
	}
	return Cookie{Name: name, Value: value}, true
}

func loginFromLegacy(auth fetch.AuthOptions) *LoginFlow {
	if auth.LoginURL == "" && auth.LoginUserSelector == "" && auth.LoginPassSelector == "" && auth.LoginSubmitSelector == "" && auth.LoginUser == "" && auth.LoginPass == "" {
		return nil
	}
	return &LoginFlow{
		URL:            auth.LoginURL,
		UserSelector:   auth.LoginUserSelector,
		PassSelector:   auth.LoginPassSelector,
		SubmitSelector: auth.LoginSubmitSelector,
		Username:       auth.LoginUser,
		Password:       auth.LoginPass,
	}
}
