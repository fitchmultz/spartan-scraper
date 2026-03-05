/**
 * Network Intercept Config Component
 *
 * Reusable network interception configuration UI shared across all job forms.
 * Handles enable/disable toggle, URL pattern filtering, resource type selection,
 * and body capture options for capturing network traffic during headless scraping.
 *
 * @module NetworkInterceptConfig
 */

export interface NetworkInterceptConfigProps {
  enabled: boolean;
  setEnabled: (value: boolean) => void;
  urlPatterns: string;
  setURLPatterns: (value: string) => void;
  resourceTypes: string[];
  setResourceTypes: (value: string[]) => void;
  captureRequestBody: boolean;
  setCaptureRequestBody: (value: boolean) => void;
  captureResponseBody: boolean;
  setCaptureResponseBody: (value: boolean) => void;
  maxBodySize: number;
  setMaxBodySize: (value: number) => void;
  maxEntries?: number;
  setMaxEntries?: (value: number) => void;
  disabled?: boolean;
  inputPrefix: string;
}

const RESOURCE_TYPES = [
  { value: "xhr", label: "XHR", description: "XMLHttpRequest" },
  { value: "fetch", label: "Fetch", description: "Fetch API" },
  { value: "document", label: "Document", description: "Main document" },
  { value: "script", label: "Script", description: "JavaScript files" },
  { value: "stylesheet", label: "Stylesheet", description: "CSS files" },
  { value: "image", label: "Image", description: "Image files" },
  { value: "media", label: "Media", description: "Audio/Video files" },
  { value: "font", label: "Font", description: "Font files" },
  {
    value: "websocket",
    label: "WebSocket",
    description: "WebSocket connections",
  },
  { value: "other", label: "Other", description: "Other resources" },
];

export function NetworkInterceptConfig({
  enabled,
  setEnabled,
  urlPatterns,
  setURLPatterns,
  resourceTypes,
  setResourceTypes,
  captureRequestBody,
  setCaptureRequestBody,
  captureResponseBody,
  setCaptureResponseBody,
  maxBodySize,
  setMaxBodySize,
  maxEntries,
  setMaxEntries,
  disabled = false,
  inputPrefix,
}: NetworkInterceptConfigProps) {
  const handleResourceTypeToggle = (type: string) => {
    if (resourceTypes.includes(type)) {
      setResourceTypes(resourceTypes.filter((t) => t !== type));
    } else {
      setResourceTypes([...resourceTypes, type]);
    }
  };

  const handleSelectAll = () => {
    setResourceTypes(RESOURCE_TYPES.map((t) => t.value));
  };

  const handleSelectNone = () => {
    setResourceTypes([]);
  };

  const handleSelectDefault = () => {
    setResourceTypes(["xhr", "fetch"]);
  };

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
          onChange={(e) => setEnabled(e.target.checked)}
          disabled={disabled}
        />
        <span>Enable Network Interception</span>
      </label>

      {disabled && (
        <small
          style={{
            display: "block",
            marginTop: 8,
            color: "var(--text-secondary)",
          }}
        >
          Network interception requires headless mode to be enabled.
        </small>
      )}

      {enabled && !disabled && (
        <>
          <label
            htmlFor={`${inputPrefix}-intercept-patterns`}
            style={{ marginTop: 12 }}
          >
            URL Patterns (optional)
          </label>
          <input
            id={`${inputPrefix}-intercept-patterns`}
            value={urlPatterns}
            onChange={(e) => setURLPatterns(e.target.value)}
            placeholder="**/api/**, *.json"
          />
          <small>
            Comma-separated glob patterns. Leave empty to intercept all URLs.
          </small>

          <div style={{ marginTop: 12 }}>
            <span
              style={{ marginBottom: 8, display: "block", fontWeight: 500 }}
            >
              Resource Types to Capture
            </span>
            <div
              style={{
                display: "flex",
                gap: 8,
                marginBottom: 8,
              }}
            >
              <button
                type="button"
                className="secondary"
                style={{ padding: "4px 12px", fontSize: "0.85rem" }}
                onClick={handleSelectAll}
              >
                All
              </button>
              <button
                type="button"
                className="secondary"
                style={{ padding: "4px 12px", fontSize: "0.85rem" }}
                onClick={handleSelectDefault}
              >
                Default (XHR/Fetch)
              </button>
              <button
                type="button"
                className="secondary"
                style={{ padding: "4px 12px", fontSize: "0.85rem" }}
                onClick={handleSelectNone}
              >
                None
              </button>
            </div>
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(2, 1fr)",
                gap: 8,
              }}
            >
              {RESOURCE_TYPES.map((type) => (
                <label
                  key={type.value}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 6,
                    cursor: "pointer",
                    padding: "4px 8px",
                    borderRadius: 6,
                    background: resourceTypes.includes(type.value)
                      ? "rgba(59, 130, 246, 0.2)"
                      : "transparent",
                  }}
                >
                  <input
                    type="checkbox"
                    checked={resourceTypes.includes(type.value)}
                    onChange={() => handleResourceTypeToggle(type.value)}
                  />
                  <span style={{ fontSize: "0.9rem" }}>
                    {type.label}
                    <small
                      style={{
                        display: "block",
                        color: "var(--text-secondary)",
                        fontSize: "0.75rem",
                      }}
                    >
                      {type.description}
                    </small>
                  </span>
                </label>
              ))}
            </div>
          </div>

          <div style={{ marginTop: 12 }}>
            <span
              style={{ marginBottom: 8, display: "block", fontWeight: 500 }}
            >
              Body Capture Options
            </span>
            <label
              style={{
                display: "flex",
                alignItems: "center",
                gap: 8,
                cursor: "pointer",
                marginBottom: 8,
              }}
            >
              <input
                type="checkbox"
                checked={captureRequestBody}
                onChange={(e) => setCaptureRequestBody(e.target.checked)}
              />
              Capture request bodies
            </label>
            <label
              style={{
                display: "flex",
                alignItems: "center",
                gap: 8,
                cursor: "pointer",
              }}
            >
              <input
                type="checkbox"
                checked={captureResponseBody}
                onChange={(e) => setCaptureResponseBody(e.target.checked)}
              />
              Capture response bodies
            </label>
          </div>

          <div className="row" style={{ marginTop: 12 }}>
            <label>
              Max body size (bytes)
              <input
                type="number"
                min={1024}
                max={10485760}
                value={maxBodySize}
                onChange={(e) => setMaxBodySize(Number(e.target.value))}
              />
            </label>
            {setMaxEntries && maxEntries !== undefined && (
              <label>
                Max entries
                <input
                  type="number"
                  min={10}
                  max={10000}
                  value={maxEntries}
                  onChange={(e) => setMaxEntries(Number(e.target.value))}
                />
              </label>
            )}
          </div>
        </>
      )}
    </div>
  );
}

export default NetworkInterceptConfig;
