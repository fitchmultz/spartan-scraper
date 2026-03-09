/**
 * WatchForm Component
 *
 * Renders the create/edit form for watches in a modal dialog.
 * Includes all form fields: URL, selector, interval, diff format,
 * extraction settings, screenshot configuration, and webhook settings.
 *
 * This component does NOT handle:
 * - API calls for saving watches (parent handles via onSubmit)
 * - Form state management (controlled via props)
 * - Modal visibility state
 *
 * @module components/watches/WatchForm
 */

import type { WatchFormProps } from "../../types/watch";
import { formatSecondsAsDuration } from "../../lib/formatting";

/**
 * Form component for creating or editing a watch
 */
export function WatchForm({
  formData,
  formError,
  formSubmitting,
  isEditing,
  onChange,
  onSubmit,
  onCancel,
}: WatchFormProps) {
  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        backgroundColor: "rgba(0, 0, 0, 0.7)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
        padding: 20,
      }}
    >
      <div
        className="panel"
        style={{
          maxWidth: 600,
          width: "100%",
          maxHeight: "90vh",
          overflow: "auto",
        }}
      >
        <div
          className="row"
          style={{
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: 16,
          }}
        >
          <h3 style={{ margin: 0 }}>
            {isEditing ? "Edit Watch" : "Create Watch"}
          </h3>
          <button type="button" onClick={onCancel} className="secondary">
            Cancel
          </button>
        </div>

        {formError && (
          <div
            style={{
              padding: 12,
              backgroundColor: "rgba(239, 68, 68, 0.1)",
              borderRadius: 8,
              color: "#ef4444",
              marginBottom: 16,
            }}
          >
            {formError}
          </div>
        )}

        <form
          onSubmit={(e) => {
            e.preventDefault();
            onSubmit();
          }}
        >
          <div style={{ marginBottom: 16 }}>
            <label
              htmlFor="watch-url"
              style={{ display: "block", marginBottom: 4 }}
            >
              URL <span style={{ color: "#ef4444" }}>*</span>
            </label>
            <input
              id="watch-url"
              type="url"
              value={formData.url}
              onChange={(e) => onChange({ url: e.target.value })}
              placeholder="https://example.com/page"
              required
              style={{ width: "100%" }}
            />
          </div>

          <div style={{ marginBottom: 16 }}>
            <label
              htmlFor="watch-selector"
              style={{ display: "block", marginBottom: 4 }}
            >
              CSS Selector (optional)
            </label>
            <input
              id="watch-selector"
              type="text"
              value={formData.selector}
              onChange={(e) => onChange({ selector: e.target.value })}
              placeholder=".content, #main, article"
              style={{ width: "100%" }}
            />
            <small style={{ color: "var(--muted)" }}>
              Extract only content matching this selector
            </small>
          </div>

          <div className="row" style={{ gap: 16, marginBottom: 16 }}>
            <div style={{ flex: 1 }}>
              <label
                htmlFor="watch-interval"
                style={{ display: "block", marginBottom: 4 }}
              >
                Check Interval (seconds){" "}
                <span style={{ color: "#ef4444" }}>*</span>
              </label>
              <input
                id="watch-interval"
                type="number"
                min={60}
                step={60}
                value={formData.intervalSeconds}
                onChange={(e) =>
                  onChange({
                    intervalSeconds: parseInt(e.target.value, 10) || 60,
                  })
                }
                required
                style={{ width: "100%" }}
              />
              <small style={{ color: "var(--muted)" }}>
                Minimum 60 seconds (
                {formatSecondsAsDuration(formData.intervalSeconds)})
              </small>
            </div>

            <div style={{ flex: 1 }}>
              <label
                htmlFor="watch-diff-format"
                style={{ display: "block", marginBottom: 4 }}
              >
                Diff Format
              </label>
              <select
                id="watch-diff-format"
                value={formData.diffFormat}
                onChange={(e) =>
                  onChange({
                    diffFormat: e.target.value as typeof formData.diffFormat,
                  })
                }
                style={{ width: "100%" }}
              >
                <option value="unified">Unified Text</option>
                <option value="html-side-by-side">HTML Side-by-Side</option>
                <option value="html-inline">HTML Inline</option>
              </select>
            </div>
          </div>

          <div className="row" style={{ gap: 16, marginBottom: 16 }}>
            <div style={{ flex: 1 }}>
              <label
                htmlFor="watch-extract-mode"
                style={{ display: "block", marginBottom: 4 }}
              >
                Extract Mode
              </label>
              <select
                id="watch-extract-mode"
                value={formData.extractMode}
                onChange={(e) =>
                  onChange({
                    extractMode: e.target.value as typeof formData.extractMode,
                  })
                }
                style={{ width: "100%" }}
              >
                <option value="">HTML (default)</option>
                <option value="text">Plain Text</option>
                <option value="markdown">Markdown</option>
              </select>
            </div>

            <div style={{ flex: 1 }}>
              <label
                htmlFor="watch-min-size"
                style={{ display: "block", marginBottom: 4 }}
              >
                Min Change Size (bytes)
              </label>
              <input
                id="watch-min-size"
                type="number"
                min={0}
                value={formData.minChangeSize}
                onChange={(e) => onChange({ minChangeSize: e.target.value })}
                placeholder="0"
                style={{ width: "100%" }}
              />
            </div>
          </div>

          <div style={{ marginBottom: 16 }}>
            <label
              htmlFor="watch-ignore-patterns"
              style={{ display: "block", marginBottom: 4 }}
            >
              Ignore Patterns (one regex per line)
            </label>
            <textarea
              id="watch-ignore-patterns"
              value={formData.ignorePatterns}
              onChange={(e) => onChange({ ignorePatterns: e.target.value })}
              placeholder="\d{4}-\d{2}-\d{2}  # Date patterns&#10;timestamp: \d+  # Timestamps"
              rows={3}
              style={{
                width: "100%",
                fontFamily: "monospace",
                fontSize: 12,
              }}
            />
          </div>

          <div
            className="row"
            style={{ gap: 16, marginBottom: 16, alignItems: "center" }}
          >
            <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <input
                type="checkbox"
                checked={formData.enabled}
                onChange={(e) => onChange({ enabled: e.target.checked })}
              />
              Enabled
            </label>

            <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <input
                type="checkbox"
                checked={formData.headless}
                onChange={(e) => onChange({ headless: e.target.checked })}
              />
              Use Headless Browser
            </label>

            <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <input
                type="checkbox"
                checked={formData.usePlaywright}
                onChange={(e) => onChange({ usePlaywright: e.target.checked })}
              />
              Use Playwright
            </label>
          </div>

          <div style={{ marginBottom: 16 }}>
            <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <input
                type="checkbox"
                checked={formData.screenshotEnabled}
                onChange={(e) =>
                  onChange({
                    screenshotEnabled: e.target.checked,
                    // Auto-enable headless when screenshots are enabled
                    headless: e.target.checked ? true : formData.headless,
                  })
                }
              />
              Enable Visual Change Detection (Screenshots)
            </label>
            <small
              style={{
                color: "var(--muted)",
                display: "block",
                marginTop: 4,
              }}
            >
              Captures screenshots to detect visual/layout changes
            </small>
          </div>

          {formData.screenshotEnabled && (
            <div
              style={{
                marginBottom: 16,
                padding: 16,
                backgroundColor: "var(--bg-alt)",
                borderRadius: 8,
              }}
            >
              <h4 style={{ margin: "0 0 12px 0" }}>Screenshot Configuration</h4>
              <div className="row" style={{ gap: 16, marginBottom: 12 }}>
                <label
                  style={{ display: "flex", alignItems: "center", gap: 8 }}
                >
                  <input
                    type="checkbox"
                    checked={formData.screenshotFullPage}
                    onChange={(e) =>
                      onChange({ screenshotFullPage: e.target.checked })
                    }
                  />
                  Full Page Screenshot
                </label>
                <div style={{ flex: 1 }}>
                  <label
                    htmlFor="screenshot-format"
                    style={{ display: "block", marginBottom: 4 }}
                  >
                    Format
                  </label>
                  <select
                    id="screenshot-format"
                    value={formData.screenshotFormat}
                    onChange={(e) =>
                      onChange({
                        screenshotFormat: e.target.value as "png" | "jpeg",
                      })
                    }
                    style={{ width: "100%" }}
                  >
                    <option value="png">PNG</option>
                    <option value="jpeg">JPEG</option>
                  </select>
                </div>
                <div style={{ flex: 1 }}>
                  <label
                    htmlFor="visual-threshold"
                    style={{ display: "block", marginBottom: 4 }}
                  >
                    Diff Threshold (0-1)
                  </label>
                  <input
                    id="visual-threshold"
                    type="number"
                    min={0}
                    max={1}
                    step={0.05}
                    value={formData.visualDiffThreshold}
                    onChange={(e) =>
                      onChange({ visualDiffThreshold: e.target.value })
                    }
                    style={{ width: "100%" }}
                  />
                </div>
              </div>
            </div>
          )}

          <div style={{ marginBottom: 16 }}>
            <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <input
                type="checkbox"
                checked={formData.notifyOnChange}
                onChange={(e) => onChange({ notifyOnChange: e.target.checked })}
              />
              Send Webhook Notification on Change
            </label>
          </div>

          {formData.notifyOnChange && (
            <div
              style={{
                marginBottom: 16,
                padding: 16,
                backgroundColor: "var(--bg-alt)",
                borderRadius: 8,
              }}
            >
              <div style={{ marginBottom: 12 }}>
                <label
                  htmlFor="watch-webhook-url"
                  style={{ display: "block", marginBottom: 4 }}
                >
                  Webhook URL
                </label>
                <input
                  id="watch-webhook-url"
                  type="url"
                  value={formData.webhookUrl}
                  onChange={(e) => onChange({ webhookUrl: e.target.value })}
                  placeholder="https://hooks.example.com/webhook"
                  style={{ width: "100%" }}
                />
              </div>
              <div>
                <label
                  htmlFor="watch-webhook-secret"
                  style={{ display: "block", marginBottom: 4 }}
                >
                  Webhook Secret (optional)
                </label>
                <input
                  id="watch-webhook-secret"
                  type="password"
                  value={formData.webhookSecret}
                  onChange={(e) => onChange({ webhookSecret: e.target.value })}
                  placeholder="secret-for-hmac-signature"
                  style={{ width: "100%" }}
                />
              </div>
            </div>
          )}

          <div className="row" style={{ gap: 8, justifyContent: "flex-end" }}>
            <button
              type="button"
              onClick={onCancel}
              className="secondary"
              disabled={formSubmitting}
            >
              Cancel
            </button>
            <button type="submit" disabled={formSubmitting}>
              {formSubmitting
                ? "Saving..."
                : isEditing
                  ? "Update Watch"
                  : "Create Watch"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
