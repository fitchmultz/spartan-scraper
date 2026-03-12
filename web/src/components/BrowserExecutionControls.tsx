interface BrowserExecutionControlsProps {
  headless: boolean;
  setHeadless: (value: boolean) => void;
  usePlaywright: boolean;
  setUsePlaywright: (value: boolean) => void;
  timeoutSeconds: number;
  setTimeoutSeconds: (value: number) => void;
  timeoutLabel?: string;
  helperText?: string;
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
}: BrowserExecutionControlsProps) {
  const helperId = useId();

  return (
    <div className="browser-execution-controls">
      <div className="row browser-execution-controls__row">
        <label className="browser-execution-controls__toggle">
          <input
            type="checkbox"
            checked={headless}
            onChange={(event) => setHeadless(event.target.checked)}
          />{" "}
          Headless
        </label>
        <label
          className={`browser-execution-controls__toggle ${!headless ? "is-disabled" : ""}`}
        >
          <input
            type="checkbox"
            checked={usePlaywright}
            disabled={!headless}
            aria-describedby={!headless ? helperId : undefined}
            onChange={(event) => setUsePlaywright(event.target.checked)}
          />{" "}
          Playwright
        </label>
        <label className="browser-execution-controls__timeout">
          {timeoutLabel}
          <input
            type="number"
            min={5}
            value={timeoutSeconds}
            onChange={(event) => setTimeoutSeconds(Number(event.target.value))}
          />
        </label>
      </div>
      {!headless && (
        <p id={helperId} className="browser-execution-controls__helper">
          {helperText}
        </p>
      )}
    </div>
  );
}
import { useId } from "react";
