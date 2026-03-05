// Package fetch provides HTTP and headless browser content fetching capabilities.
// Device emulation types for mobile/responsive content.
package fetch

import "strings"

// DeviceCategory classifies devices by form factor.
type DeviceCategory string

const (
	DeviceCategoryMobile  DeviceCategory = "mobile"
	DeviceCategoryTablet  DeviceCategory = "tablet"
	DeviceCategoryDesktop DeviceCategory = "desktop"
)

// Orientation represents the device screen orientation.
type Orientation string

const (
	OrientationPortrait  Orientation = "portrait"
	OrientationLandscape Orientation = "landscape"
)

// DeviceEmulation defines device emulation settings for mobile/responsive content.
// Used by headless fetchers to emulate specific devices.
type DeviceEmulation struct {
	Name              string         `json:"name"`              // Device preset name (e.g., "iPhone 14", "Pixel 7")
	ViewportWidth     int            `json:"viewportWidth"`     // Viewport width in pixels
	ViewportHeight    int            `json:"viewportHeight"`    // Viewport height in pixels
	DeviceScaleFactor float64        `json:"deviceScaleFactor"` // Device pixel ratio (e.g., 2.0 for Retina)
	UserAgent         string         `json:"userAgent"`         // User agent string for the device
	IsMobile          bool           `json:"isMobile"`          // Whether to emulate mobile viewport
	HasTouch          bool           `json:"hasTouch"`          // Whether the device has touch capability
	Category          DeviceCategory `json:"category"`          // Device category (mobile, tablet, desktop)
	Orientation       Orientation    `json:"orientation"`       // Default orientation (portrait/landscape)
}

// Common device presets for mobile emulation.
var devicePresets = map[string]DeviceEmulation{
	// iPhone 15 series
	"iphone15": {
		Name:              "iPhone 15",
		ViewportWidth:     393,
		ViewportHeight:    852,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone15pro": {
		Name:              "iPhone 15 Pro",
		ViewportWidth:     393,
		ViewportHeight:    852,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone15promax": {
		Name:              "iPhone 15 Pro Max",
		ViewportWidth:     430,
		ViewportHeight:    932,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone15plus": {
		Name:              "iPhone 15 Plus",
		ViewportWidth:     430,
		ViewportHeight:    932,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	// iPhone 16 series
	"iphone16": {
		Name:              "iPhone 16",
		ViewportWidth:     393,
		ViewportHeight:    852,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone16pro": {
		Name:              "iPhone 16 Pro",
		ViewportWidth:     402,
		ViewportHeight:    874,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone16promax": {
		Name:              "iPhone 16 Pro Max",
		ViewportWidth:     440,
		ViewportHeight:    956,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphone16plus": {
		Name:              "iPhone 16 Plus",
		ViewportWidth:     430,
		ViewportHeight:    932,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	// Legacy iPhone
	"iphone14": {
		Name:              "iPhone 14",
		ViewportWidth:     390,
		ViewportHeight:    844,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"iphonemax": {
		Name:              "iPhone 14 Pro Max",
		ViewportWidth:     430,
		ViewportHeight:    932,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	// Pixel series
	"pixel7": {
		Name:              "Pixel 7",
		ViewportWidth:     412,
		ViewportHeight:    915,
		DeviceScaleFactor: 2.625,
		UserAgent:         "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"pixel8": {
		Name:              "Pixel 8",
		ViewportWidth:     412,
		ViewportHeight:    915,
		DeviceScaleFactor: 2.625,
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"pixel8pro": {
		Name:              "Pixel 8 Pro",
		ViewportWidth:     448,
		ViewportHeight:    998,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"pixel9": {
		Name:              "Pixel 9",
		ViewportWidth:     412,
		ViewportHeight:    915,
		DeviceScaleFactor: 2.625,
		UserAgent:         "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"pixel9pro": {
		Name:              "Pixel 9 Pro",
		ViewportWidth:     448,
		ViewportHeight:    998,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 15; Pixel 9 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	// Galaxy S series
	"galaxys23": {
		Name:              "Galaxy S23",
		ViewportWidth:     360,
		ViewportHeight:    780,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 13; SM-S911B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"galaxys24": {
		Name:              "Galaxy S24",
		ViewportWidth:     360,
		ViewportHeight:    780,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; SM-S921B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"galaxys24plus": {
		Name:              "Galaxy S24+",
		ViewportWidth:     384,
		ViewportHeight:    824,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; SM-S926B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	"galaxys24ultra": {
		Name:              "Galaxy S24 Ultra",
		ViewportWidth:     384,
		ViewportHeight:    824,
		DeviceScaleFactor: 3.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; SM-S928B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryMobile,
		Orientation:       OrientationPortrait,
	},
	// iPad series
	"ipad": {
		Name:              "iPad",
		ViewportWidth:     810,
		ViewportHeight:    1080,
		DeviceScaleFactor: 2.0,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryTablet,
		Orientation:       OrientationPortrait,
	},
	"ipadpro": {
		Name:              "iPad Pro 12.9\"",
		ViewportWidth:     1024,
		ViewportHeight:    1366,
		DeviceScaleFactor: 2.0,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryTablet,
		Orientation:       OrientationPortrait,
	},
	"ipadair": {
		Name:              "iPad Air",
		ViewportWidth:     820,
		ViewportHeight:    1180,
		DeviceScaleFactor: 2.0,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryTablet,
		Orientation:       OrientationPortrait,
	},
	"ipadmini": {
		Name:              "iPad Mini",
		ViewportWidth:     744,
		ViewportHeight:    1133,
		DeviceScaleFactor: 2.0,
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryTablet,
		Orientation:       OrientationPortrait,
	},
	// Android tablets
	"galaxytabs9": {
		Name:              "Galaxy Tab S9",
		ViewportWidth:     1600,
		ViewportHeight:    2560,
		DeviceScaleFactor: 2.0,
		UserAgent:         "Mozilla/5.0 (Linux; Android 13; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		IsMobile:          true,
		HasTouch:          true,
		Category:          DeviceCategoryTablet,
		Orientation:       OrientationPortrait,
	},
	// Desktop
	"desktop": {
		Name:              "Desktop",
		ViewportWidth:     1920,
		ViewportHeight:    1080,
		DeviceScaleFactor: 1.0,
		UserAgent:         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		IsMobile:          false,
		HasTouch:          false,
		Category:          DeviceCategoryDesktop,
		Orientation:       OrientationLandscape,
	},
	"laptop": {
		Name:              "Laptop",
		ViewportWidth:     1366,
		ViewportHeight:    768,
		DeviceScaleFactor: 1.0,
		UserAgent:         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		IsMobile:          false,
		HasTouch:          false,
		Category:          DeviceCategoryDesktop,
		Orientation:       OrientationLandscape,
	},
}

// GetDevicePreset returns a device emulation preset by name.
// Returns nil if the preset name is not recognized.
func GetDevicePreset(name string) *DeviceEmulation {
	if name == "" {
		return nil
	}
	// Normalize name to lowercase for case-insensitive lookup
	name = strings.ToLower(name)
	if preset, ok := devicePresets[name]; ok {
		// Return a copy to prevent modification of the original
		presetCopy := preset
		return &presetCopy
	}
	return nil
}

// GetDevicePresetsByCategory returns all device presets matching the given category.
func GetDevicePresetsByCategory(cat DeviceCategory) []DeviceEmulation {
	var result []DeviceEmulation
	for _, preset := range devicePresets {
		if preset.Category == cat {
			// Return a copy to prevent modification of the original
			presetCopy := preset
			result = append(result, presetCopy)
		}
	}
	return result
}

// ListDevicePresetNames returns all available device preset names.
func ListDevicePresetNames() []string {
	names := make([]string, 0, len(devicePresets))
	for name := range devicePresets {
		names = append(names, name)
	}
	return names
}

// GetDeviceCategories returns all available device categories.
func GetDeviceCategories() []DeviceCategory {
	return []DeviceCategory{
		DeviceCategoryMobile,
		DeviceCategoryTablet,
		DeviceCategoryDesktop,
	}
}

// ApplyOrientation applies the specified orientation to a device emulation.
// For landscape orientation on mobile/tablet devices, it swaps width and height.
func (d *DeviceEmulation) ApplyOrientation(orientation Orientation) *DeviceEmulation {
	if d == nil {
		return nil
	}
	// Create a copy to avoid modifying the original
	result := *d
	result.Orientation = orientation

	if orientation == OrientationLandscape && d.Category != DeviceCategoryDesktop {
		// Swap width and height for landscape orientation
		result.ViewportWidth = d.ViewportHeight
		result.ViewportHeight = d.ViewportWidth
	}

	return &result
}
