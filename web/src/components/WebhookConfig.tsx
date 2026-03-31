/**
 * Purpose: Render the shared webhook notification fields used across job and automation authoring surfaces.
 * Responsibilities: Keep webhook URL/events/secret inputs controlled, enforce explicit field-level URL validation copy, and preserve the shared event-toggle behavior.
 * Scope: Webhook form controls only.
 * Usage: Mount from authoring forms that need optional job or batch webhook configuration.
 * Invariants/Assumptions: Webhook URLs are optional, malformed URLs should identify the webhook field explicitly, and selecting `all` supersedes individual event checkboxes.
 */

import type { CSSProperties, ChangeEvent, InvalidEvent } from "react";

import { WEBHOOK_URL_INVALID_MESSAGE } from "../lib/form-utils";

interface WebhookConfigProps {
  webhookUrl: string;
  setWebhookUrl: (value: string) => void;
  webhookEvents: string[];
  setWebhookEvents: (value: string[]) => void;
  webhookSecret: string;
  setWebhookSecret: (value: string) => void;
  inputPrefix?: string;
}

const AVAILABLE_EVENTS = [
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
  { value: "canceled", label: "Canceled" },
  { value: "started", label: "Started" },
  { value: "all", label: "All Events" },
];

const maskedSecretStyle = {
  WebkitTextSecurity: "disc",
} as CSSProperties;

export function WebhookConfig({
  webhookUrl,
  setWebhookUrl,
  webhookEvents,
  setWebhookEvents,
  webhookSecret,
  setWebhookSecret,
  inputPrefix = "webhook",
}: WebhookConfigProps) {
  const handleWebhookUrlChange = (event: ChangeEvent<HTMLInputElement>) => {
    event.currentTarget.setCustomValidity("");
    setWebhookUrl(event.target.value);
  };

  const handleWebhookUrlInvalid = (event: InvalidEvent<HTMLInputElement>) => {
    if (event.currentTarget.validity.typeMismatch) {
      event.currentTarget.setCustomValidity(WEBHOOK_URL_INVALID_MESSAGE);
    }
  };

  const toggleEvent = (event: string) => {
    if (event === "all") {
      // If "all" is selected, clear other selections
      if (webhookEvents.includes("all")) {
        setWebhookEvents([]);
      } else {
        setWebhookEvents(["all"]);
      }
      return;
    }

    // If selecting a specific event, remove "all"
    const newEvents = webhookEvents.includes(event)
      ? webhookEvents.filter((e) => e !== event)
      : [...webhookEvents.filter((e) => e !== "all"), event];

    setWebhookEvents(newEvents);
  };

  return (
    <div className="panel">
      <h3>Webhook Notifications</h3>
      <label htmlFor={`${inputPrefix}-url`}>Webhook URL</label>
      <input
        id={`${inputPrefix}-url`}
        type="url"
        value={webhookUrl}
        onChange={handleWebhookUrlChange}
        onInvalid={handleWebhookUrlInvalid}
        placeholder="https://example.com/webhook"
      />

      <span id={`${inputPrefix}-events-label`}>Events</span>
      <fieldset
        className="row"
        style={{
          flexWrap: "wrap",
          gap: "8px 16px",
          border: "none",
          padding: 0,
          margin: 0,
        }}
        aria-labelledby={`${inputPrefix}-events-label`}
      >
        {AVAILABLE_EVENTS.map((event) => (
          <label
            key={event.value}
            style={{ display: "flex", alignItems: "center", gap: 4 }}
          >
            <input
              type="checkbox"
              checked={webhookEvents.includes(event.value)}
              onChange={() => toggleEvent(event.value)}
            />
            {event.label}
          </label>
        ))}
      </fieldset>

      <label htmlFor={`${inputPrefix}-secret`}>
        Secret (for HMAC signature)
      </label>
      <input
        id={`${inputPrefix}-secret`}
        type="text"
        autoComplete="off"
        name={`${inputPrefix}-secret`}
        value={webhookSecret}
        onChange={(e) => setWebhookSecret(e.target.value)}
        placeholder="Optional secret for signature verification"
        spellCheck={false}
        style={maskedSecretStyle}
      />
    </div>
  );
}
