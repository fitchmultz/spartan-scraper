/**
 * Purpose: Render the screenshot config UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

type ScreenshotFormat = "png" | "jpeg";

export interface ScreenshotConfigProps {
  enabled: boolean;
  setEnabled: (value: boolean) => void;
  fullPage: boolean;
  setFullPage: (value: boolean) => void;
  format: ScreenshotFormat;
  setFormat: (value: ScreenshotFormat) => void;
  quality: number;
  setQuality: (value: number) => void;
  width: number;
  setWidth: (value: number) => void;
  height: number;
  setHeight: (value: number) => void;
  disabled?: boolean;
  inputPrefix: string;
}

export function ScreenshotConfig({
  enabled,
  setEnabled,
  fullPage,
  setFullPage,
  format,
  setFormat,
  quality,
  setQuality,
  width,
  setWidth,
  height,
  setHeight,
  disabled = false,
  inputPrefix,
}: ScreenshotConfigProps) {
  return (
    <div
      style={{
        marginTop: 12,
        padding: 12,
        borderRadius: 12,
        background: "rgba(0, 0, 0, 0.15)",
        opacity: disabled ? 0.6 : 1,
      }}
    >
      <label
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          cursor: disabled ? "not-allowed" : "pointer",
          marginBottom: enabled ? 12 : 0,
        }}
      >
        <input
          type="checkbox"
          checked={enabled}
          onChange={(event) => setEnabled(event.target.checked)}
          disabled={disabled}
        />
        <span>Capture Screenshot</span>
      </label>

      {disabled ? (
        <small
          style={{
            display: "block",
            marginTop: 8,
            color: "var(--text-secondary)",
          }}
        >
          Screenshot capture requires headless mode to be enabled.
        </small>
      ) : null}

      {enabled && !disabled ? (
        <>
          <div className="row" style={{ marginTop: 12 }}>
            <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <input
                type="checkbox"
                checked={fullPage}
                onChange={(event) => setFullPage(event.target.checked)}
              />
              Full page
            </label>
            <label htmlFor={`${inputPrefix}-screenshot-format`}>
              Format
              <select
                id={`${inputPrefix}-screenshot-format`}
                value={format}
                onChange={(event) =>
                  setFormat(event.target.value as ScreenshotFormat)
                }
              >
                <option value="png">PNG</option>
                <option value="jpeg">JPEG</option>
              </select>
            </label>
            {format === "jpeg" ? (
              <label htmlFor={`${inputPrefix}-screenshot-quality`}>
                JPEG quality
                <input
                  id={`${inputPrefix}-screenshot-quality`}
                  type="number"
                  min={1}
                  max={100}
                  value={quality}
                  onChange={(event) => setQuality(Number(event.target.value))}
                />
              </label>
            ) : null}
          </div>

          <div className="row" style={{ marginTop: 12 }}>
            <label htmlFor={`${inputPrefix}-screenshot-width`}>
              Viewport width (0 = default)
              <input
                id={`${inputPrefix}-screenshot-width`}
                type="number"
                min={0}
                value={width}
                onChange={(event) => setWidth(Number(event.target.value))}
              />
            </label>
            <label htmlFor={`${inputPrefix}-screenshot-height`}>
              Viewport height (0 = default)
              <input
                id={`${inputPrefix}-screenshot-height`}
                type="number"
                min={0}
                value={height}
                onChange={(event) => setHeight(Number(event.target.value))}
              />
            </label>
          </div>
        </>
      ) : null}
    </div>
  );
}

export default ScreenshotConfig;
