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
		// iPhone 15 series
		{
			name:     "iPhone 15 preset",
			preset:   "iphone15",
			wantNil:  false,
			wantName: "iPhone 15",
		},
		{
			name:     "iPhone 15 Pro preset",
			preset:   "iphone15pro",
			wantNil:  false,
			wantName: "iPhone 15 Pro",
		},
		{
			name:     "iPhone 15 Pro Max preset",
			preset:   "iphone15promax",
			wantNil:  false,
			wantName: "iPhone 15 Pro Max",
		},
		{
			name:     "iPhone 15 Plus preset",
			preset:   "iphone15plus",
			wantNil:  false,
			wantName: "iPhone 15 Plus",
		},
		// iPhone 16 series
		{
			name:     "iPhone 16 preset",
			preset:   "iphone16",
			wantNil:  false,
			wantName: "iPhone 16",
		},
		{
			name:     "iPhone 16 Pro preset",
			preset:   "iphone16pro",
			wantNil:  false,
			wantName: "iPhone 16 Pro",
		},
		{
			name:     "iPhone 16 Pro Max preset",
			preset:   "iphone16promax",
			wantNil:  false,
			wantName: "iPhone 16 Pro Max",
		},
		{
			name:     "iPhone 16 Plus preset",
			preset:   "iphone16plus",
			wantNil:  false,
			wantName: "iPhone 16 Plus",
		},
		// Legacy iPhone
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
		// Pixel series
		{
			name:     "Pixel 7 preset",
			preset:   "pixel7",
			wantNil:  false,
			wantName: "Pixel 7",
		},
		{
			name:     "Pixel 8 preset",
			preset:   "pixel8",
			wantNil:  false,
			wantName: "Pixel 8",
		},
		{
			name:     "Pixel 8 Pro preset",
			preset:   "pixel8pro",
			wantNil:  false,
			wantName: "Pixel 8 Pro",
		},
		{
			name:     "Pixel 9 preset",
			preset:   "pixel9",
			wantNil:  false,
			wantName: "Pixel 9",
		},
		{
			name:     "Pixel 9 Pro preset",
			preset:   "pixel9pro",
			wantNil:  false,
			wantName: "Pixel 9 Pro",
		},
		// Galaxy S series
		{
			name:     "Galaxy S23 preset",
			preset:   "galaxys23",
			wantNil:  false,
			wantName: "Galaxy S23",
		},
		{
			name:     "Galaxy S24 preset",
			preset:   "galaxys24",
			wantNil:  false,
			wantName: "Galaxy S24",
		},
		{
			name:     "Galaxy S24+ preset",
			preset:   "galaxys24plus",
			wantNil:  false,
			wantName: "Galaxy S24+",
		},
		{
			name:     "Galaxy S24 Ultra preset",
			preset:   "galaxys24ultra",
			wantNil:  false,
			wantName: "Galaxy S24 Ultra",
		},
		// iPad series
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
			wantName: "iPad Pro 12.9\"",
		},
		{
			name:     "iPad Air preset",
			preset:   "ipadair",
			wantNil:  false,
			wantName: "iPad Air",
		},
		{
			name:     "iPad Mini preset",
			preset:   "ipadmini",
			wantNil:  false,
			wantName: "iPad Mini",
		},
		// Android tablets
		{
			name:     "Galaxy Tab S9 preset",
			preset:   "galaxytabs9",
			wantNil:  false,
			wantName: "Galaxy Tab S9",
		},
		// Desktop
		{
			name:     "Desktop preset",
			preset:   "desktop",
			wantNil:  false,
			wantName: "Desktop",
		},
		{
			name:     "Laptop preset",
			preset:   "laptop",
			wantNil:  false,
			wantName: "Laptop",
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
	presets := ListDevicePresetNames()

	// Ensure we have at least 20 device presets
	if len(presets) < 20 {
		t.Errorf("Expected at least 20 device presets, got %d", len(presets))
	}

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
			// Check new fields
			if device.Category == "" {
				t.Errorf("%s: Category is empty", preset)
			}
			if device.Orientation == "" {
				t.Errorf("%s: Orientation is empty", preset)
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
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
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
	if device.Category != DeviceCategoryMobile {
		t.Errorf("Category = %q, want %q", device.Category, DeviceCategoryMobile)
	}
	if device.Orientation != OrientationPortrait {
		t.Errorf("Orientation = %q, want %q", device.Orientation, OrientationPortrait)
	}
}

func TestGetDevicePresetsByCategory(t *testing.T) {
	tests := []struct {
		name     string
		category DeviceCategory
		minCount int
	}{
		{
			name:     "mobile devices",
			category: DeviceCategoryMobile,
			minCount: 14, // iPhone 15/16 series + legacy + Pixel + Galaxy
		},
		{
			name:     "tablet devices",
			category: DeviceCategoryTablet,
			minCount: 5, // iPad, iPad Pro, iPad Air, iPad Mini, Galaxy Tab S9
		},
		{
			name:     "desktop devices",
			category: DeviceCategoryDesktop,
			minCount: 2, // Desktop, Laptop
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devices := GetDevicePresetsByCategory(tt.category)
			if len(devices) < tt.minCount {
				t.Errorf("GetDevicePresetsByCategory(%q) returned %d devices, want at least %d", tt.category, len(devices), tt.minCount)
			}
			// Verify all returned devices have the correct category
			for _, device := range devices {
				if device.Category != tt.category {
					t.Errorf("Device %q has category %q, want %q", device.Name, device.Category, tt.category)
				}
			}
		})
	}
}

func TestListDevicePresetNames(t *testing.T) {
	names := ListDevicePresetNames()
	if len(names) < 20 {
		t.Errorf("ListDevicePresetNames() returned %d names, want at least 20", len(names))
	}
	// Check that some expected presets are present
	expectedPresets := []string{"iphone15", "iphone16", "pixel9", "galaxys24", "ipadair", "desktop"}
	for _, preset := range expectedPresets {
		found := false
		for _, name := range names {
			if name == preset {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ListDevicePresetNames() missing expected preset %q", preset)
		}
	}
}

func TestGetDeviceCategories(t *testing.T) {
	categories := GetDeviceCategories()
	if len(categories) != 3 {
		t.Errorf("GetDeviceCategories() returned %d categories, want 3", len(categories))
	}
	expected := []DeviceCategory{DeviceCategoryMobile, DeviceCategoryTablet, DeviceCategoryDesktop}
	for i, cat := range categories {
		if cat != expected[i] {
			t.Errorf("GetDeviceCategories()[%d] = %q, want %q", i, cat, expected[i])
		}
	}
}

func TestApplyOrientation(t *testing.T) {
	tests := []struct {
		name            string
		device          DeviceEmulation
		orientation     Orientation
		wantWidth       int
		wantHeight      int
		wantOrientation Orientation
		shouldSwap      bool
	}{
		{
			name: "mobile portrait stays same",
			device: DeviceEmulation{
				ViewportWidth:  390,
				ViewportHeight: 844,
				Category:       DeviceCategoryMobile,
			},
			orientation:     OrientationPortrait,
			wantWidth:       390,
			wantHeight:      844,
			wantOrientation: OrientationPortrait,
			shouldSwap:      false,
		},
		{
			name: "mobile landscape swaps dimensions",
			device: DeviceEmulation{
				ViewportWidth:  390,
				ViewportHeight: 844,
				Category:       DeviceCategoryMobile,
			},
			orientation:     OrientationLandscape,
			wantWidth:       844,
			wantHeight:      390,
			wantOrientation: OrientationLandscape,
			shouldSwap:      true,
		},
		{
			name: "desktop landscape does not swap",
			device: DeviceEmulation{
				ViewportWidth:  1920,
				ViewportHeight: 1080,
				Category:       DeviceCategoryDesktop,
			},
			orientation:     OrientationLandscape,
			wantWidth:       1920,
			wantHeight:      1080,
			wantOrientation: OrientationLandscape,
			shouldSwap:      false,
		},
		{
			name:            "nil device returns nil",
			device:          DeviceEmulation{},
			orientation:     OrientationPortrait,
			wantWidth:       0,
			wantHeight:      0,
			wantOrientation: OrientationPortrait,
			shouldSwap:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var device *DeviceEmulation
			if tt.name != "nil device returns nil" {
				d := tt.device
				device = &d
			}
			result := device.ApplyOrientation(tt.orientation)

			if tt.name == "nil device returns nil" {
				if result != nil {
					t.Errorf("ApplyOrientation() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("ApplyOrientation() returned nil for non-nil device")
			}
			if result.ViewportWidth != tt.wantWidth {
				t.Errorf("ViewportWidth = %d, want %d", result.ViewportWidth, tt.wantWidth)
			}
			if result.ViewportHeight != tt.wantHeight {
				t.Errorf("ViewportHeight = %d, want %d", result.ViewportHeight, tt.wantHeight)
			}
			if result.Orientation != tt.wantOrientation {
				t.Errorf("Orientation = %q, want %q", result.Orientation, tt.wantOrientation)
			}
			// Verify original device is not modified
			if device.ViewportWidth != tt.device.ViewportWidth {
				t.Error("ApplyOrientation modified the original device width")
			}
		})
	}
}
