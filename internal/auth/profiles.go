package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"spartan-scraper/internal/fetch"
)

type Profile struct {
	Name string            `json:"name"`
	Auth fetch.AuthOptions `json:"auth"`
}

type store struct {
	Profiles []Profile `json:"profiles"`
}

func LoadAll(dataDir string) ([]Profile, error) {
	path := profilesPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Profile{}, nil
		}
		return nil, err
	}
	var s store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return s.Profiles, nil
}

func SaveAll(dataDir string, profiles []Profile) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	path := profilesPath(dataDir)
	payload, err := json.MarshalIndent(store{Profiles: profiles}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

func Get(dataDir, name string) (Profile, bool, error) {
	profiles, err := LoadAll(dataDir)
	if err != nil {
		return Profile{}, false, err
	}
	for _, profile := range profiles {
		if profile.Name == name {
			return profile, true, nil
		}
	}
	return Profile{}, false, nil
}

func Upsert(dataDir string, profile Profile) error {
	if profile.Name == "" {
		return errors.New("profile name is required")
	}
	profiles, err := LoadAll(dataDir)
	if err != nil {
		return err
	}
	found := false
	for i := range profiles {
		if profiles[i].Name == profile.Name {
			profiles[i] = profile
			found = true
			break
		}
	}
	if !found {
		profiles = append(profiles, profile)
	}
	return SaveAll(dataDir, profiles)
}

func Delete(dataDir, name string) error {
	profiles, err := LoadAll(dataDir)
	if err != nil {
		return err
	}
	filtered := make([]Profile, 0, len(profiles))
	for _, profile := range profiles {
		if profile.Name != name {
			filtered = append(filtered, profile)
		}
	}
	return SaveAll(dataDir, filtered)
}

func ListNames(dataDir string) ([]string, error) {
	profiles, err := LoadAll(dataDir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		out = append(out, profile.Name)
	}
	sort.Strings(out)
	return out, nil
}

func profilesPath(dataDir string) string {
	base := strings.TrimSpace(dataDir)
	if base == "" {
		base = ".data"
	}
	return filepath.Join(base, "profiles.json")
}
