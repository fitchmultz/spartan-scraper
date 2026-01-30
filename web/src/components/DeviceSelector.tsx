/**
 * Device Selector Component
 *
 * Provides a UI for selecting device emulation presets with category filtering,
 * orientation toggle, and custom device configuration. Used by scrape, crawl,
 * and research forms to configure mobile/responsive rendering.
 *
 * @module DeviceSelector
 */
import { useState, useMemo } from "react";
import type { DeviceEmulation, DeviceCategory, Orientation } from "../api";

type DevicePreset = {
  key: string;
  name: string;
  category: DeviceCategory;
  width: number;
  height: number;
  dpr: number;
  ua: string;
  isMobile: boolean;
  hasTouch: boolean;
  orientation: Orientation;
};

const DEVICE_PRESETS: DevicePreset[] = [
  // iPhone 15 series
  {
    key: "iphone15",
    name: "iPhone 15",
    category: "mobile",
    width: 393,
    height: 852,
    dpr: 3,
    ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "iphone15pro",
    name: "iPhone 15 Pro",
    category: "mobile",
    width: 393,
    height: 852,
    dpr: 3,
    ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "iphone15promax",
    name: "iPhone 15 Pro Max",
    category: "mobile",
    width: 430,
    height: 932,
    dpr: 3,
    ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "iphone15plus",
    name: "iPhone 15 Plus",
    category: "mobile",
    width: 430,
    height: 932,
    dpr: 3,
    ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  // iPhone 16 series
  {
    key: "iphone16",
    name: "iPhone 16",
    category: "mobile",
    width: 393,
    height: 852,
    dpr: 3,
    ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "iphone16pro",
    name: "iPhone 16 Pro",
    category: "mobile",
    width: 402,
    height: 874,
    dpr: 3,
    ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "iphone16promax",
    name: "iPhone 16 Pro Max",
    category: "mobile",
    width: 440,
    height: 956,
    dpr: 3,
    ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "iphone16plus",
    name: "iPhone 16 Plus",
    category: "mobile",
    width: 430,
    height: 932,
    dpr: 3,
    ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  // Legacy iPhone
  {
    key: "iphone14",
    name: "iPhone 14",
    category: "mobile",
    width: 390,
    height: 844,
    dpr: 3,
    ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "iphonemax",
    name: "iPhone 14 Pro Max",
    category: "mobile",
    width: 430,
    height: 932,
    dpr: 3,
    ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  // Pixel series
  {
    key: "pixel7",
    name: "Pixel 7",
    category: "mobile",
    width: 412,
    height: 915,
    dpr: 2.625,
    ua: "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "pixel8",
    name: "Pixel 8",
    category: "mobile",
    width: 412,
    height: 915,
    dpr: 2.625,
    ua: "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "pixel8pro",
    name: "Pixel 8 Pro",
    category: "mobile",
    width: 448,
    height: 998,
    dpr: 3,
    ua: "Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "pixel9",
    name: "Pixel 9",
    category: "mobile",
    width: 412,
    height: 915,
    dpr: 2.625,
    ua: "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "pixel9pro",
    name: "Pixel 9 Pro",
    category: "mobile",
    width: 448,
    height: 998,
    dpr: 3,
    ua: "Mozilla/5.0 (Linux; Android 15; Pixel 9 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  // Galaxy S series
  {
    key: "galaxys23",
    name: "Galaxy S23",
    category: "mobile",
    width: 360,
    height: 780,
    dpr: 3,
    ua: "Mozilla/5.0 (Linux; Android 13; SM-S911B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "galaxys24",
    name: "Galaxy S24",
    category: "mobile",
    width: 360,
    height: 780,
    dpr: 3,
    ua: "Mozilla/5.0 (Linux; Android 14; SM-S921B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "galaxys24plus",
    name: "Galaxy S24+",
    category: "mobile",
    width: 384,
    height: 824,
    dpr: 3,
    ua: "Mozilla/5.0 (Linux; Android 14; SM-S926B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "galaxys24ultra",
    name: "Galaxy S24 Ultra",
    category: "mobile",
    width: 384,
    height: 824,
    dpr: 3,
    ua: "Mozilla/5.0 (Linux; Android 14; SM-S928B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  // iPad series
  {
    key: "ipad",
    name: "iPad",
    category: "tablet",
    width: 810,
    height: 1080,
    dpr: 2,
    ua: "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "ipadpro",
    name: 'iPad Pro 12.9"',
    category: "tablet",
    width: 1024,
    height: 1366,
    dpr: 2,
    ua: "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "ipadair",
    name: "iPad Air",
    category: "tablet",
    width: 820,
    height: 1180,
    dpr: 2,
    ua: "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  {
    key: "ipadmini",
    name: "iPad Mini",
    category: "tablet",
    width: 744,
    height: 1133,
    dpr: 2,
    ua: "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  // Android tablets
  {
    key: "galaxytabs9",
    name: "Galaxy Tab S9",
    category: "tablet",
    width: 1600,
    height: 2560,
    dpr: 2,
    ua: "Mozilla/5.0 (Linux; Android 13; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
    isMobile: true,
    hasTouch: true,
    orientation: "portrait",
  },
  // Desktop
  {
    key: "desktop",
    name: "Desktop",
    category: "desktop",
    width: 1920,
    height: 1080,
    dpr: 1,
    ua: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
    isMobile: false,
    hasTouch: false,
    orientation: "landscape",
  },
  {
    key: "laptop",
    name: "Laptop",
    category: "desktop",
    width: 1366,
    height: 768,
    dpr: 1,
    ua: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
    isMobile: false,
    hasTouch: false,
    orientation: "landscape",
  },
];

interface DeviceSelectorProps {
  device: DeviceEmulation | null;
  onChange: (device: DeviceEmulation | null) => void;
  disabled?: boolean;
}

const CATEGORY_LABELS: Record<DeviceCategory | "all", string> = {
  all: "All",
  mobile: "Mobile",
  tablet: "Tablet",
  desktop: "Desktop",
};

const CATEGORY_ICONS: Record<DeviceCategory, string> = {
  mobile: "📱",
  tablet: "📲",
  desktop: "💻",
};

export function DeviceSelector({
  device,
  onChange,
  disabled = false,
}: DeviceSelectorProps) {
  const [category, setCategory] = useState<DeviceCategory | "all">("all");
  const [showCustom, setShowCustom] = useState(false);

  // Custom device state
  const [customName, setCustomName] = useState("");
  const [customWidth, setCustomWidth] = useState(390);
  const [customHeight, setCustomHeight] = useState(844);
  const [customDpr, setCustomDpr] = useState(2);
  const [customUa, setCustomUa] = useState("");
  const [customIsMobile, setCustomIsMobile] = useState(true);
  const [customHasTouch, setCustomHasTouch] = useState(true);
  const [customOrientation, setCustomOrientation] =
    useState<Orientation>("portrait");

  const filteredPresets = useMemo(() => {
    if (category === "all") return DEVICE_PRESETS;
    return DEVICE_PRESETS.filter((p) => p.category === category);
  }, [category]);

  const selectedPresetKey = useMemo(() => {
    if (!device || showCustom) return "";
    return (
      DEVICE_PRESETS.find(
        (p) =>
          p.name === device.name &&
          p.width === device.viewportWidth &&
          p.height === device.viewportHeight,
      )?.key || ""
    );
  }, [device, showCustom]);

  const handlePresetSelect = (key: string) => {
    if (key === "") {
      onChange(null);
      return;
    }
    const preset = DEVICE_PRESETS.find((p) => p.key === key);
    if (!preset) return;

    const newDevice: DeviceEmulation = {
      name: preset.name,
      viewportWidth: preset.width,
      viewportHeight: preset.height,
      deviceScaleFactor: preset.dpr,
      userAgent: preset.ua,
      isMobile: preset.isMobile,
      hasTouch: preset.hasTouch,
      category: preset.category,
      orientation: preset.orientation,
    };
    onChange(newDevice);
    setShowCustom(false);
  };

  const handleOrientationToggle = () => {
    if (!device || device.category === "desktop") return;
    const newOrientation: Orientation =
      device.orientation === "portrait" ? "landscape" : "portrait";
    const newDevice: DeviceEmulation = {
      ...device,
      orientation: newOrientation,
      viewportWidth: device.viewportHeight,
      viewportHeight: device.viewportWidth,
    };
    onChange(newDevice);
  };

  const handleCustomToggle = () => {
    if (showCustom) {
      setShowCustom(false);
      onChange(null);
      return;
    }
    setShowCustom(true);
    const customDevice: DeviceEmulation = {
      name: customName || "Custom Device",
      viewportWidth: customWidth,
      viewportHeight: customHeight,
      deviceScaleFactor: customDpr,
      userAgent: customUa,
      isMobile: customIsMobile,
      hasTouch: customHasTouch,
      category: customIsMobile ? "mobile" : "desktop",
      orientation: customOrientation,
    };
    onChange(customDevice);
  };

  const updateCustomDevice = () => {
    if (!showCustom) return;
    const customDevice: DeviceEmulation = {
      name: customName || "Custom Device",
      viewportWidth: customWidth,
      viewportHeight: customHeight,
      deviceScaleFactor: customDpr,
      userAgent: customUa,
      isMobile: customIsMobile,
      hasTouch: customHasTouch,
      category: customIsMobile ? "mobile" : "desktop",
      orientation: customOrientation,
    };
    onChange(customDevice);
  };

  return (
    <div
      className="device-selector"
      style={{
        opacity: disabled ? 0.5 : 1,
        pointerEvents: disabled ? "none" : "auto",
      }}
    >
      <div className="device-selector-header">
        <span className="device-selector-title">Device Emulation</span>
        <div className="device-category-filters">
          {(["all", "mobile", "tablet", "desktop"] as const).map((cat) => (
            <button
              key={cat}
              type="button"
              className={`category-btn ${category === cat ? "active" : ""}`}
              onClick={() => setCategory(cat)}
            >
              {cat !== "all" && (
                <span className="category-icon">{CATEGORY_ICONS[cat]}</span>
              )}
              {CATEGORY_LABELS[cat]}
            </button>
          ))}
        </div>
      </div>

      <div className="device-selector-row">
        <select
          value={showCustom ? "custom" : selectedPresetKey}
          onChange={(e) =>
            e.target.value === "custom"
              ? handleCustomToggle()
              : handlePresetSelect(e.target.value)
          }
          className="device-select"
        >
          <option value="">None (default viewport)</option>
          {filteredPresets.map((preset) => (
            <option key={preset.key} value={preset.key}>
              {CATEGORY_ICONS[preset.category]} {preset.name} ({preset.width}×
              {preset.height})
            </option>
          ))}
          <option value="custom">⚙️ Custom Device...</option>
        </select>

        {device && device.category !== "desktop" && !showCustom && (
          <button
            type="button"
            className="orientation-btn"
            onClick={handleOrientationToggle}
            title="Toggle orientation"
          >
            {device.orientation === "portrait" ? "📱 Portrait" : "📲 Landscape"}
          </button>
        )}
      </div>

      {showCustom && (
        <div className="custom-device-panel">
          <div className="custom-device-row">
            <label>
              Name
              <input
                type="text"
                value={customName}
                onChange={(e) => {
                  setCustomName(e.target.value);
                  setTimeout(updateCustomDevice, 0);
                }}
                placeholder="Custom Device"
              />
            </label>
          </div>
          <div className="custom-device-row">
            <label>
              Width
              <input
                type="number"
                min={100}
                max={4096}
                value={customWidth}
                onChange={(e) => {
                  setCustomWidth(Number(e.target.value));
                  setTimeout(updateCustomDevice, 0);
                }}
              />
            </label>
            <label>
              Height
              <input
                type="number"
                min={100}
                max={4096}
                value={customHeight}
                onChange={(e) => {
                  setCustomHeight(Number(e.target.value));
                  setTimeout(updateCustomDevice, 0);
                }}
              />
            </label>
            <label>
              DPR
              <input
                type="number"
                min={0.5}
                max={5}
                step={0.1}
                value={customDpr}
                onChange={(e) => {
                  setCustomDpr(Number(e.target.value));
                  setTimeout(updateCustomDevice, 0);
                }}
              />
            </label>
          </div>
          <div className="custom-device-row">
            <label>
              User Agent
              <input
                type="text"
                value={customUa}
                onChange={(e) => {
                  setCustomUa(e.target.value);
                  setTimeout(updateCustomDevice, 0);
                }}
                placeholder="Mozilla/5.0..."
              />
            </label>
          </div>
          <div className="custom-device-row">
            <label>
              <input
                type="checkbox"
                checked={customIsMobile}
                onChange={(e) => {
                  setCustomIsMobile(e.target.checked);
                  setTimeout(updateCustomDevice, 0);
                }}
              />{" "}
              Is Mobile
            </label>
            <label>
              <input
                type="checkbox"
                checked={customHasTouch}
                onChange={(e) => {
                  setCustomHasTouch(e.target.checked);
                  setTimeout(updateCustomDevice, 0);
                }}
              />{" "}
              Has Touch
            </label>
            <label>
              Orientation
              <select
                value={customOrientation}
                onChange={(e) => {
                  setCustomOrientation(e.target.value as Orientation);
                  setTimeout(updateCustomDevice, 0);
                }}
              >
                <option value="portrait">Portrait</option>
                <option value="landscape">Landscape</option>
              </select>
            </label>
          </div>
        </div>
      )}

      {device && !showCustom && (
        <div className="device-info">
          <span className="device-dimensions">
            {device.viewportWidth}×{device.viewportHeight}
          </span>
          <span className="device-dpr">DPR: {device.deviceScaleFactor}</span>
          <span className="device-category">
            {CATEGORY_ICONS[device.category || "desktop"]} {device.category}
          </span>
        </div>
      )}
    </div>
  );
}
