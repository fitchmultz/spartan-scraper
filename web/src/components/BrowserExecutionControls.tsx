/*
Purpose: Render the shared browser-runtime control cluster used across operator-facing authoring and submission flows.
Responsibilities: Present consistent headless/Playwright toggles, optional timeout controls, and dependency guidance for headless-gated browser features.
Scope: UI controls only; call sites own state transitions and any additional runtime toggles such as screenshot context.
Usage: Mount in forms and modals, wiring the controlled props to local state helpers.
Invariants/Assumptions: Playwright is always headless-gated, helper text is shown only while headless is off, and timeout controls are optional for surfaces that do not expose them.
*/
import { useId } from "react";

interface BrowserExecutionControlsProps {
  headless: boolean;
  setHeadless: (value: boolean) => void;
  usePlaywright: boolean;
  setUsePlaywright: (value: boolean) => void;
  timeoutSeconds?: number;
  setTimeoutSeconds?: (value: number) => void;
  timeoutLabel?: string;
  helperText?: string;
  headlessLabel?: string;
  playwrightLabel?: string;
  showTimeout?: boolean;
  disabled?: boolean;
}

const DEFAULT_HELPER_TEXT =
  "Enable Headless to unlock Playwright, device emulation, and browser-only diagnostics.";

export function BrowserExecutionControls({
  headless,
  setHeadless,
  usePlaywright,
  setUsePlaywright,
  timeoutSeconds,
  setTimeoutSeconds,
  timeoutLabel = "Timeout (s)",
  helperText = DEFAULT_HELPER_TEXT,
  headlessLabel = "Headless",
  playwrightLabel = "Playwright",
  showTimeout = timeoutSeconds !== undefined && setTimeoutSeconds !== undefined,
  disabled = false,
}: BrowserExecutionControlsProps) {
  const helperId = useId();

  return (
    <div className="browser-execution-controls">
      <div className="row browser-execution-controls__row">
        <label className="browser-execution-controls__toggle">
          <input
            type="checkbox"
            checked={headless}
            disabled={disabled}
            onChange={(event) => setHeadless(event.target.checked)}
          />{" "}
          {headlessLabel}
        </label>
        <label
          className={`browser-execution-controls__toggle ${!headless ? "is-disabled" : ""}`}
        >
          <input
            type="checkbox"
            checked={usePlaywright}
            disabled={!headless || disabled}
            aria-describedby={!headless ? helperId : undefined}
            onChange={(event) => setUsePlaywright(event.target.checked)}
          />{" "}
          {playwrightLabel}
        </label>
        {showTimeout && timeoutSeconds !== undefined && setTimeoutSeconds ? (
          <label className="browser-execution-controls__timeout">
            {timeoutLabel}
            <input
              type="number"
              min={5}
              value={timeoutSeconds}
              disabled={disabled}
              onChange={(event) =>
                setTimeoutSeconds(Number(event.target.value))
              }
            />
          </label>
        ) : null}
      </div>
      {!headless && (
        <p id={helperId} className="browser-execution-controls__helper">
          {helperText}
        </p>
      )}
    </div>
  );
}
