// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// Purpose:
//   - Load and validate persisted proxy-pool configuration.
//
// Responsibilities:
//   - Read proxy-pool JSON files.
//   - Distinguish optional default absence from explicit user misconfiguration.
//
// Scope:
//   - Proxy-pool persistence helpers only.
//
// Usage:
//   - LoadProxyPoolFromFile(path) for strict loading.
//   - ProxyPoolFromConfig(path, explicit) for optional startup loading.
//
// Invariants/Assumptions:
//   - Missing default optional proxy-pool files are silent.
//   - Explicit proxy-pool paths still surface errors.
package fetch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// LoadProxyPoolFromFile loads a proxy pool from a JSON configuration file.
func LoadProxyPoolFromFile(path string) (*ProxyPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apperrors.NotFound(fmt.Sprintf("proxy pool config file not found: %s", path))
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read proxy pool config", err)
	}

	var config ProxyPoolConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, apperrors.Wrap(apperrors.KindValidation, "invalid proxy pool config JSON", err)
	}

	return NewProxyPool(config)
}

// ProxyPoolFromConfig creates a proxy pool from configured startup settings.
// Missing files are silent only when the default optional path is implied.
func ProxyPoolFromConfig(path string, explicit bool) (*ProxyPool, error) {
	if path == "" {
		return nil, nil
	}

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if !explicit {
				return nil, nil
			}
			return nil, apperrors.NotFound(fmt.Sprintf("proxy pool config file not found: %s", path))
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to stat proxy pool config", err)
	}

	return LoadProxyPoolFromFile(path)
}
