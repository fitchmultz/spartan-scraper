package buildinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/yaml.v3"
)

type openAPIInfo struct {
	Version string `yaml:"version"`
}

type openAPISpec struct {
	Info openAPIInfo `yaml:"info"`
}

func TestVersionConsistency(t *testing.T) {
	// Get the path to the root of the repository.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not get current file path")
	}
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	openapiPath := filepath.Join(repoRoot, "api", "openapi.yaml")

	data, err := os.ReadFile(openapiPath)
	if err != nil {
		t.Fatalf("failed to read openapi.yaml: %v", err)
	}

	var spec openAPISpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("failed to unmarshal openapi.yaml: %v", err)
	}

	if spec.Info.Version != Version {
		t.Errorf("version inconsistency: buildinfo.Version=%q, openapi.yaml info.version=%q", Version, spec.Info.Version)
	}
}
