// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

// ProxyPoolFromConfig creates a proxy pool from the global config.
// Returns nil if no proxy pool is configured.
func ProxyPoolFromConfig(dataDir string) (*ProxyPool, error) {
	if dataDir == "" {
		dataDir = ".data"
	}

	path := filepath.Join(dataDir, "proxy_pool.json")

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	return LoadProxyPoolFromFile(path)
}
