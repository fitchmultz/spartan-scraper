// Package fetch provides HTTP and headless browser content fetching capabilities.
package fetch

import (
	"testing"
)

func TestGetDevicePreset(t *testing.T) {
	tests := []struct {
		name     string
		preset   string
		wantNil  bool
		wantName string
	}{
		{
			name:     "empty string returns nil",
			preset:   "",
			wantNil:  true,
			wantName: "",
		},
		{
			name:     "unknown preset returns nil",
			preset:   "unknown-device",
			wantNil:  true,
			wantName: "",
		},
		{
			name:     "iPhone 14 preset",
			preset:   "iphone14",
			wantNil:  false,
			wantName: "iPhone 14",
		},
		{
			name:     "iPhone 14 preset case insensitive",
			preset:   "IPHONE14",
			wantNil:  false,
			wantName: "iPhone 14",
		},
		{
			name:     "iPhone Max preset",
			preset:   "iphonemax",
			wantNil:  false,
			wantName: "iPhone 14 Pro Max",
		},
		{
			name:     "iPad preset",
			preset:   "ipad",
			wantNil:  false,
			wantName: "iPad",
		},
		{
			name:     "iPad Pro preset",
			preset:   "ipadpro",
			wantNil:  false,
			wantName: "iPad Pro",
		},
		{
			name:     "Pixel 7 preset",
			preset:   "pixel7",
			wantNil:  false,
			wantName: "Pixel 7",
		},
		{
			name:     "Galaxy S23 preset",
			preset:   "galaxys23",
			wantNil:  false,
			wantName: "Galaxy S23",
		},
		{
			name:     "Desktop preset",
			preset:   "desktop",
			wantNil:  false,
			wantName: "Desktop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDevicePreset(tt.preset)
			if tt.wantNil {
				if got != nil {
					t.Errorf("GetDevicePreset(%q) = %v, want nil", tt.preset, got)
				}
				return
			}
			if got == nil {
				t.Errorf("GetDevicePreset(%q) = nil, want non-nil", tt.preset)
				return
			}
			if got.Name != tt.wantName {
				t.Errorf("GetDevicePreset(%q).Name = %q, want %q", tt.preset, got.Name, tt.wantName)
			}
		})
	}
}

func TestGetDevicePresetReturnsCopy(t *testing.T) {
	// Ensure that modifying the returned device doesn't affect the original preset
	device1 := GetDevicePreset("iphone14")
	device2 := GetDevicePreset("iphone14")

	if device1 == device2 {
		t.Error("GetDevicePreset should return a copy, not the same pointer")
	}

	// Modify device1 and ensure it doesn't affect device2
	device1.ViewportWidth = 999
	if device2.ViewportWidth == 999 {
		t.Error("Modifying one device should not affect another")
	}
}

func TestDevicePresetsHaveRequiredFields(t *testing.T) {
	presets := []string{"iphone14", "iphonemax", "ipad", "ipadpro", "pixel7", "galaxys23", "desktop"}

	for _, preset := range presets {
		t.Run(preset, func(t *testing.T) {
			device := GetDevicePreset(preset)
			if device == nil {
				t.Fatalf("GetDevicePreset(%q) returned nil", preset)
			}

			if device.Name == "" {
				t.Errorf("%s: Name is empty", preset)
			}
			if device.ViewportWidth <= 0 {
				t.Errorf("%s: ViewportWidth must be positive, got %d", preset, device.ViewportWidth)
			}
			if device.ViewportHeight <= 0 {
				t.Errorf("%s: ViewportHeight must be positive, got %d", preset, device.ViewportHeight)
			}
			if device.DeviceScaleFactor <= 0 {
				t.Errorf("%s: DeviceScaleFactor must be positive, got %f", preset, device.DeviceScaleFactor)
			}
			if device.UserAgent == "" {
				t.Errorf("%s: UserAgent is empty", preset)
			}
		})
	}
}

func TestDeviceEmulationFields(t *testing.T) {
	device := DeviceEmulation{
		Name:              "Test Device",
		ViewportWidth:     390,
		ViewportHeight:    844,
		DeviceScaleFactor: 3.0,
		UserAgent:         "TestUserAgent/1.0",
		IsMobile:          true,
		HasTouch:          true,
	}

	if device.Name != "Test Device" {
		t.Errorf("Name = %q, want %q", device.Name, "Test Device")
	}
	if device.ViewportWidth != 390 {
		t.Errorf("ViewportWidth = %d, want %d", device.ViewportWidth, 390)
	}
	if device.ViewportHeight != 844 {
		t.Errorf("ViewportHeight = %d, want %d", device.ViewportHeight, 844)
	}
	if device.DeviceScaleFactor != 3.0 {
		t.Errorf("DeviceScaleFactor = %f, want %f", device.DeviceScaleFactor, 3.0)
	}
	if device.UserAgent != "TestUserAgent/1.0" {
		t.Errorf("UserAgent = %q, want %q", device.UserAgent, "TestUserAgent/1.0")
	}
	if !device.IsMobile {
		t.Error("IsMobile = false, want true")
	}
	if !device.HasTouch {
		t.Error("HasTouch = false, want true")
	}
}
