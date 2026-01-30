/**
 * Webhook Configuration Component
 *
 * Form for configuring webhook notifications for job completion.
 * Allows setting URL, event subscriptions, and HMAC secret.
 *
 * @module WebhookConfig
 */

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

export function WebhookConfig({
  webhookUrl,
  setWebhookUrl,
  webhookEvents,
  setWebhookEvents,
  webhookSecret,
  setWebhookSecret,
  inputPrefix = "webhook",
}: WebhookConfigProps) {
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
        onChange={(e) => setWebhookUrl(e.target.value)}
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
        type="password"
        value={webhookSecret}
        onChange={(e) => setWebhookSecret(e.target.value)}
        placeholder="Optional secret for signature verification"
      />
    </div>
  );
}
