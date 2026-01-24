package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportVaultPathValidation(t *testing.T) {
	dataDir := t.TempDir()

	vault := Vault{
		Version:  "1",
		Profiles: []Profile{{Name: "test"}},
	}
	if err := SaveVault(dataDir, vault); err != nil {
		t.Fatalf("failed to save test vault: %v", err)
	}

	testData := []byte(`{"version":"1","profiles":[]}`)
	if err := os.WriteFile(filepath.Join(dataDir, "test_vault.json"), testData, 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "vault-backup-2024.json"), testData, 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "a.json"), testData, 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		shouldErr bool
		errMsg    string
	}{
		{"valid filename", "test_vault.json", false, ""},
		{"empty string", "", true, "path is required"},
		{"absolute path", "/tmp/test.json", true, "invalid path"},
		{"path traversal", "../test.json", true, "invalid path"},
		{"with directory", "subdir/test.json", true, "invalid path"},
		{"backslash", "subdir\\test.json", true, "invalid path"},
		{"double slash", "sub//test.json", true, "invalid path"},
		{"valid with extension", "vault-backup-2024.json", false, ""},
		{"valid simple", "a.json", false, ""},
		{"current directory", ".", true, "invalid path"},
		{"parent directory", "..", true, "invalid path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ImportVault(dataDir, tt.path)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExportVaultPathValidation(t *testing.T) {
	dataDir := t.TempDir()

	vault := Vault{
		Version:  "1",
		Profiles: []Profile{{Name: "test"}},
	}
	if err := SaveVault(dataDir, vault); err != nil {
		t.Fatalf("failed to save test vault: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		shouldErr bool
		errMsg    string
	}{
		{"valid filename", "export.json", false, ""},
		{"empty string", "", true, "path is required"},
		{"absolute path", "/tmp/export.json", true, "invalid path"},
		{"path traversal", "../export.json", true, "invalid path"},
		{"with directory", "subdir/export.json", true, "invalid path"},
		{"backslash", "subdir\\export.json", true, "invalid path"},
		{"double slash", "sub//export.json", true, "invalid path"},
		{"valid with extension", "backup-2024.json", false, ""},
		{"valid simple", "a.json", false, ""},
		{"current directory", ".", true, "invalid path"},
		{"parent directory", "..", true, "invalid path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExportVault(dataDir, tt.path)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				expectedPath := filepath.Join(dataDir, tt.path)
				if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
					t.Errorf("expected file %q to exist", expectedPath)
				}
			}
		})
	}
}
