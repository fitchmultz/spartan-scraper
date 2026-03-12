/**
 * Device Selector Component
 *
 * Provides a UI for selecting device emulation presets with category filtering,
 * orientation toggle, and custom device configuration. Used by scrape, crawl,
 * and research forms to configure mobile/responsive rendering.
 *
 * @module DeviceSelector
 */
import { useMemo, useState, type SVGProps } from "react";
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

function AllDevicesIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false" {...props}>
      <rect x="3.5" y="5" width="6" height="11" rx="1.8" />
      <rect x="10.75" y="4" width="7" height="12.5" rx="2.2" />
      <path d="M18.5 7.5h2.25a1.75 1.75 0 0 1 1.75 1.75v7a1.75 1.75 0 0 1-1.75 1.75H7.25" />
    </svg>
  );
}

function MobileIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false" {...props}>
      <rect x="7" y="2.75" width="10" height="18.5" rx="2.5" />
      <path d="M10.25 6h3.5" />
      <circle cx="12" cy="17.75" r="0.9" fill="currentColor" stroke="none" />
    </svg>
  );
}

function TabletIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false" {...props}>
      <rect x="4" y="4.5" width="16" height="15" rx="2.5" />
      <circle cx="12" cy="16.75" r="0.85" fill="currentColor" stroke="none" />
    </svg>
  );
}

function DesktopIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false" {...props}>
      <rect x="3" y="4.5" width="18" height="12" rx="2.5" />
      <path d="M9 20h6M12 16.5V20" />
    </svg>
  );
}

function OrientationIcon({
  orientation,
  ...props
}: SVGProps<SVGSVGElement> & { orientation: Orientation }) {
  if (orientation === "portrait") {
    return <MobileIcon {...props} />;
  }

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false" {...props}>
      <rect x="2.75" y="7" width="18.5" height="10" rx="2.5" />
      <path d="M18 10.25v3.5" />
      <circle cx="6.25" cy="12" r="0.9" fill="currentColor" stroke="none" />
    </svg>
  );
}

const CATEGORY_ICONS: Record<DeviceCategory | "all", typeof AllDevicesIcon> = {
  all: AllDevicesIcon,
  mobile: MobileIcon,
  tablet: TabletIcon,
  desktop: DesktopIcon,
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

  const activeCategory = category;
  const SelectedCategoryIcon = device
    ? CATEGORY_ICONS[device.category || "desktop"]
    : DesktopIcon;

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
      className={`device-selector ${disabled ? "is-disabled" : ""}`}
      aria-disabled={disabled}
    >
      <div className="device-selector-header">
        <span className="device-selector-title">Device Emulation</span>
        <div className="device-category-filters">
          {(["all", "mobile", "tablet", "desktop"] as const).map((cat) => {
            const Icon = CATEGORY_ICONS[cat];

            return (
              <button
                key={cat}
                type="button"
                className={`category-btn ${activeCategory === cat ? "active" : ""}`}
                onClick={() => setCategory(cat)}
                aria-pressed={activeCategory === cat}
              >
                <span className="category-icon" aria-hidden="true">
                  <Icon className="device-selector-icon" />
                </span>
                {CATEGORY_LABELS[cat]}
              </button>
            );
          })}
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
              {preset.name} ({preset.width}×{preset.height})
            </option>
          ))}
          <option value="custom">Custom Device...</option>
        </select>

        {device && device.category !== "desktop" && !showCustom && (
          <button
            type="button"
            className="orientation-btn"
            onClick={handleOrientationToggle}
            title="Toggle orientation"
          >
            <OrientationIcon
              className="device-selector-icon"
              orientation={device.orientation ?? "portrait"}
            />
            {(device.orientation ?? "portrait") === "portrait"
              ? "Portrait"
              : "Landscape"}
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
            <SelectedCategoryIcon className="device-selector-icon" />
            {CATEGORY_LABELS[device.category || "desktop"]}
          </span>
        </div>
      )}
    </div>
  );
}
