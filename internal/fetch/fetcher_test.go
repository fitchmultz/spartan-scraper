// Package fetch provides tests for fetcher construction and browser-tooling availability probes.
//
// Purpose:
// - Verify browser discovery plus cached and fresh Playwright readiness checks.
//
// Responsibilities:
// - Cover Chrome/Chromium lookup across supported platforms.
// - Assert browser availability helpers classify missing dependencies correctly.
// - Confirm cached checks can be refreshed for operator-facing diagnostics.
//
// Scope:
// - Availability probing only; end-to-end browser automation lives in higher-level tests.
//
// Usage:
// - Run with `go test ./internal/fetch`.
//
// Invariants/Assumptions:
// - Tests stub host lookups and Playwright bootstrap helpers instead of requiring local browser installs.
// - Fresh checks must invalidate the cached Playwright result.
package fetch

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

func init() {
	osGetenv = os.Getenv
}

func TestCheckBrowserAvailability_Chromedp(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func()
		teardownMock  func()
		wantErr       bool
		errorContains string
	}{
		{
			name: "Chrome found on PATH",
			setupMock: func() {
				lookPath = func(file string) (string, error) {
					if file == "google-chrome" {
						return "/usr/bin/google-chrome", nil
					}
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
			},
			teardownMock: func() {
				lookPath = exec.LookPath
			},
			wantErr: false,
		},
		{
			name: "Chrome not found on PATH",
			setupMock: func() {
				lookPath = func(file string) (string, error) {
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
				osStat = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
			},
			teardownMock: func() {
				lookPath = exec.LookPath
				osStat = os.Stat
			},
			wantErr:       true,
			errorContains: "chrome/chromium not found on PATH",
		},
		{
			name: "Chromium found on PATH",
			setupMock: func() {
				lookPath = func(file string) (string, error) {
					if file == "chromium" {
						return "/usr/bin/chromium", nil
					}
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
			},
			teardownMock: func() {
				lookPath = exec.LookPath
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			defer tt.teardownMock()

			err := CheckBrowserAvailability(false)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckBrowserAvailability() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorContains != "" {
				if err == nil {
					t.Errorf("CheckBrowserAvailability() expected error containing %q, got nil", tt.errorContains)
					return
				}
				if !errors.Is(err, ErrChromeNotFound) {
					t.Errorf("CheckBrowserAvailability() expected ErrChromeNotFound, got %v", err)
				}
			}
		})
	}
}

func TestCheckBrowserAvailability_Playwright(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Playwright integration test in short mode")
	}

	tests := []struct {
		name          string
		setupMock     func()
		teardownMock  func()
		wantErr       bool
		errorContains string
	}{
		{
			name: "Playwright drivers not installed",
			setupMock: func() {
				playwrightOnce = &sync.Once{}
				playwrightErr = nil
				playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
					return nil, errors.New("exec: playwright: executable file not found")
				}
			},
			teardownMock: func() {
				playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
					return playwright.Run(options...)
				}
			},
			wantErr:       true,
			errorContains: "playwright drivers not installed",
		},
		{
			name: "Playwright timeout",
			setupMock: func() {
				playwrightOnce = &sync.Once{}
				playwrightErr = nil
				playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
					time.Sleep(11 * time.Second)
					return nil, errors.New("should timeout first")
				}
			},
			teardownMock: func() {
				playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
					return playwright.Run(options...)
				}
			},
			wantErr:       true,
			errorContains: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			defer tt.teardownMock()

			err := CheckBrowserAvailability(true)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckBrowserAvailability() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorContains != "" {
				if err == nil {
					t.Errorf("CheckBrowserAvailability() expected error containing %q, got nil", tt.errorContains)
					return
				}
				if !errors.Is(err, ErrPlaywrightNotReady) {
					t.Errorf("CheckBrowserAvailability() expected ErrPlaywrightNotReady, got %v", err)
				}
			}
		})
	}
}

func TestFindChrome(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func()
		teardownMock func()
		wantPath     string
		wantErr      bool
	}{
		{
			name: "Finds google-chrome",
			setupMock: func() {
				lookPath = func(file string) (string, error) {
					if file == "google-chrome" {
						return "/usr/bin/google-chrome", nil
					}
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
				osStat = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
			},
			teardownMock: func() {
				lookPath = exec.LookPath
				osStat = os.Stat
			},
			wantPath: "/usr/bin/google-chrome",
			wantErr:  false,
		},
		{
			name: "Finds chromium-browser",
			setupMock: func() {
				lookPath = func(file string) (string, error) {
					if file == "chromium-browser" {
						return "/usr/bin/chromium-browser", nil
					}
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
				osStat = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
			},
			teardownMock: func() {
				lookPath = exec.LookPath
				osStat = os.Stat
			},
			wantPath: "/usr/bin/chromium-browser",
			wantErr:  false,
		},
		{
			name: "No Chrome found",
			setupMock: func() {
				lookPath = func(file string) (string, error) {
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
				osStat = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
			},
			teardownMock: func() {
				lookPath = exec.LookPath
				osStat = os.Stat
			},
			wantPath: "",
			wantErr:  true,
		},
		{
			name: "Finds Chrome on macOS",
			setupMock: func() {
				currentOS = "darwin"
				lookPath = func(file string) (string, error) {
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
				osStat = func(name string) (os.FileInfo, error) {
					if name == "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				}
			},
			teardownMock: func() {
				currentOS = runtime.GOOS
				lookPath = exec.LookPath
				osStat = os.Stat
			},
			wantPath: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			wantErr:  false,
		},
		{
			name: "Finds Chromium on macOS",
			setupMock: func() {
				currentOS = "darwin"
				lookPath = func(file string) (string, error) {
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
				osStat = func(name string) (os.FileInfo, error) {
					if name == "/Applications/Chromium.app/Contents/MacOS/Chromium" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				}
			},
			teardownMock: func() {
				currentOS = runtime.GOOS
				lookPath = exec.LookPath
				osStat = os.Stat
			},
			wantPath: "/Applications/Chromium.app/Contents/MacOS/Chromium",
			wantErr:  false,
		},
		{
			name: "Finds Chrome on Windows Program Files",
			setupMock: func() {
				currentOS = "windows"
				lookPath = func(file string) (string, error) {
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
				osStat = func(name string) (os.FileInfo, error) {
					if name == `C:\Program Files\Google\Chrome\Application\chrome.exe` {
						return nil, nil
					}
					return nil, os.ErrNotExist
				}
			},
			teardownMock: func() {
				currentOS = runtime.GOOS
				lookPath = exec.LookPath
				osStat = os.Stat
			},
			wantPath: `C:\Program Files\Google\Chrome\Application\chrome.exe`,
			wantErr:  false,
		},
		{
			name: "Finds Chrome on Windows Program Files (x86)",
			setupMock: func() {
				currentOS = "windows"
				lookPath = func(file string) (string, error) {
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
				osStat = func(name string) (os.FileInfo, error) {
					if name == `C:\Program Files (x86)\Google\Chrome\Application\chrome.exe` {
						return nil, nil
					}
					return nil, os.ErrNotExist
				}
			},
			teardownMock: func() {
				currentOS = runtime.GOOS
				lookPath = exec.LookPath
				osStat = os.Stat
			},
			wantPath: `C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			wantErr:  false,
		},
		{
			name: "Finds Chrome on Windows user profile",
			setupMock: func() {
				currentOS = "windows"
				lookPath = func(file string) (string, error) {
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
				osGetenv = func(key string) string {
					if key == "USERPROFILE" {
						return `C:\Users\testuser`
					}
					return ""
				}
				osStat = func(name string) (os.FileInfo, error) {
					if name == `C:\Users\testuser\AppData\Local\Google\Chrome\Application\chrome.exe` {
						return nil, nil
					}
					return nil, os.ErrNotExist
				}
			},
			teardownMock: func() {
				currentOS = runtime.GOOS
				lookPath = exec.LookPath
				osGetenv = os.Getenv
				osStat = os.Stat
			},
			wantPath: `C:\Users\testuser\AppData\Local\Google\Chrome\Application\chrome.exe`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			defer tt.teardownMock()

			gotPath, err := findChrome()

			if (err != nil) != tt.wantErr {
				t.Errorf("findChrome() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotPath != tt.wantPath {
				t.Errorf("findChrome() = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

func TestFindChrome_MacOSOnly(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test on non-macOS platform")
	}

	t.Run("macOS Chrome path check", func(t *testing.T) {
		lookPath = func(file string) (string, error) {
			return "", &exec.Error{Name: file, Err: errors.New("not found")}
		}
		defer func() {
			lookPath = exec.LookPath
		}()

		path, err := findChrome()

		if err == nil && path == "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" {
			t.Log("Successfully found Chrome on macOS")
		} else if err != nil {
			t.Logf("Chrome not found on macOS (may not be installed): %v", err)
		}
	})
}

func TestCheckChromeAvailability(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func()
		teardownMock func()
		wantErr      bool
	}{
		{
			name: "Chrome available",
			setupMock: func() {
				lookPath = func(file string) (string, error) {
					return "/usr/bin/google-chrome", nil
				}
			},
			teardownMock: func() {
				lookPath = exec.LookPath
			},
			wantErr: false,
		},
		{
			name: "Chrome not available",
			setupMock: func() {
				lookPath = func(file string) (string, error) {
					return "", &exec.Error{Name: file, Err: errors.New("not found")}
				}
				osStat = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
			},
			teardownMock: func() {
				lookPath = exec.LookPath
				osStat = os.Stat
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			defer tt.teardownMock()

			err := checkChromeAvailability()

			if (err != nil) != tt.wantErr {
				t.Errorf("checkChromeAvailability() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckPlaywrightAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Playwright integration test in short mode")
	}

	tests := []struct {
		name         string
		setupMock    func()
		teardownMock func()
		wantErr      bool
	}{
		{
			name: "Playwright not available",
			setupMock: func() {
				playwrightOnce = &sync.Once{}
				playwrightErr = nil
				playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
					return nil, errors.New("playwright: executable not found")
				}
			},
			teardownMock: func() {
				playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
					return playwright.Run(options...)
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			defer tt.teardownMock()

			err := checkPlaywrightAvailability()

			if (err != nil) != tt.wantErr {
				t.Errorf("checkPlaywrightAvailability() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckPlaywrightAvailability_Caching(t *testing.T) {
	callCount := 0
	playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
		callCount++
		return nil, errors.New("not found")
	}
	defer func() {
		playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
			return playwright.Run(options...)
		}
	}()

	// Reset cache
	playwrightOnce = &sync.Once{}
	playwrightErr = nil

	// First call
	err1 := checkPlaywrightAvailability()
	if err1 == nil {
		t.Error("expected error on first call")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call to playwrightRun, got %d", callCount)
	}

	// Second call
	err2 := checkPlaywrightAvailability()
	if err2 == nil {
		t.Error("expected error on second call")
	}
	if callCount != 1 {
		t.Errorf("expected call count to remain 1, got %d", callCount)
	}

	if err1 != err2 {
		t.Errorf("expected same error, got %v and %v", err1, err2)
	}
}

func TestCheckBrowserAvailabilityFresh_InvalidatesPlaywrightCache(t *testing.T) {
	callCount := 0
	playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
		callCount++
		if callCount == 1 {
			return nil, errors.New("not found")
		}
		return nil, nil
	}
	defer func() {
		playwrightRun = func(options ...*playwright.RunOptions) (*playwright.Playwright, error) {
			return playwright.Run(options...)
		}
		playwrightOnce = &sync.Once{}
		playwrightErr = nil
	}()

	playwrightOnce = &sync.Once{}
	playwrightErr = nil

	if err := CheckBrowserAvailability(true); err == nil {
		t.Fatal("expected cached playwright check to fail initially")
	}
	if callCount != 1 {
		t.Fatalf("expected initial playwright call count 1, got %d", callCount)
	}

	if err := CheckBrowserAvailabilityFresh(true); err != nil {
		t.Fatalf("expected fresh playwright check to succeed, got %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected fresh playwright call count 2, got %d", callCount)
	}

	if err := CheckBrowserAvailability(true); err != nil {
		t.Fatalf("expected refreshed cached playwright check to stay healthy, got %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected cached playwright call count to remain 2, got %d", callCount)
	}
}

func TestErrChromeNotFound(t *testing.T) {
	err := ErrChromeNotFound
	if err == nil {
		t.Error("ErrChromeNotFound should not be nil")
	}
	if err.Error() != "chrome/chromium not found on PATH" {
		t.Errorf("ErrChromeNotFound message = %v, want 'chrome/chromium not found on PATH'", err.Error())
	}
}

func TestErrPlaywrightNotReady(t *testing.T) {
	err := ErrPlaywrightNotReady
	if err == nil {
		t.Error("ErrPlaywrightNotReady should not be nil")
	}
	if err.Error() != "playwright drivers not installed or not found" {
		t.Errorf("ErrPlaywrightNotReady message = %v, want 'playwright drivers not installed or not found'", err.Error())
	}
}
