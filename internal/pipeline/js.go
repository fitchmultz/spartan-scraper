// Package pipeline provides a plugin system for extending scrape and crawl workflows.
// It handles plugin hooks at pre/post stages of fetch, extract, and output operations,
// plugin registration, and JavaScript plugin execution.
// It does NOT handle workflow execution or plugin implementations.
package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	EngineChromedp   = "chromedp"
	EnginePlaywright = "playwright"
	jsRegistryFile   = "pipeline_js.json"
)

type JSTargetScript struct {
	Name         string   `json:"name"`
	HostPatterns []string `json:"hostPatterns"`
	Engine       string   `json:"engine,omitempty"`
	PreNav       string   `json:"preNav,omitempty"`
	PostNav      string   `json:"postNav,omitempty"`
	Selectors    []string `json:"selectors,omitempty"`
}

type JSRegistry struct {
	Scripts []JSTargetScript `json:"scripts"`
}

type jsRegistryFilePayload struct {
	Scripts []JSTargetScript `json:"scripts"`
}

func LoadJSRegistry(dataDir string) (*JSRegistry, error) {
	path := filepath.Join(defaultDataDir(dataDir), jsRegistryFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &JSRegistry{Scripts: []JSTargetScript{}}, nil
		}
		return nil, err
	}
	var payload jsRegistryFilePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &JSRegistry{Scripts: payload.Scripts}, nil
}

func (r *JSRegistry) Match(rawURL string) []JSTargetScript {
	if r == nil || len(r.Scripts) == 0 {
		return nil
	}
	host := HostFromURL(rawURL)
	if host == "" {
		return nil
	}
	out := make([]JSTargetScript, 0)
	for _, script := range r.Scripts {
		if hostMatchesAnyPattern(host, script.HostPatterns) {
			out = append(out, script)
		}
	}
	return out
}

func SelectScripts(scripts []JSTargetScript, engine string) ([]string, []string, []string) {
	if len(scripts) == 0 {
		return nil, nil, nil
	}
	engine = strings.ToLower(strings.TrimSpace(engine))
	pre := make([]string, 0)
	post := make([]string, 0)
	selectors := make([]string, 0)
	for _, script := range scripts {
		if script.Engine != "" && strings.ToLower(strings.TrimSpace(script.Engine)) != engine {
			continue
		}
		if strings.TrimSpace(script.PreNav) != "" {
			pre = append(pre, script.PreNav)
		}
		if strings.TrimSpace(script.PostNav) != "" {
			post = append(post, script.PostNav)
		}
		if len(script.Selectors) > 0 {
			selectors = append(selectors, script.Selectors...)
		}
	}
	return pre, post, selectors
}

func hostMatchesAnyPattern(host string, patterns []string) bool {
	if host == "" || len(patterns) == 0 {
		return false
	}

	host = strings.ToLower(strings.TrimSpace(host))

	for _, pattern := range patterns {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if host == pattern {
			return true
		}
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*.")
			if strings.HasSuffix(host, "."+suffix) {
				return true
			}
			continue
		}
		if strings.HasSuffix(pattern, ".*") {
			prefix := strings.TrimSuffix(pattern, ".*")
			if strings.HasPrefix(host, prefix) {
				return true
			}
			continue
		}
	}
	return false
}

func defaultDataDir(dataDir string) string {
	base := strings.TrimSpace(dataDir)
	if base == "" {
		base = ".data"
	}
	return base
}
