/**
 * Purpose: Render route-specific in-product help for the current major workflow surface.
 * Responsibilities: Show route guidance, relevant shortcuts, and discoverability actions that operators can reopen at any time.
 * Scope: Route help presentation only.
 * Usage: Render below each `RouteHeader` in `App.tsx` with the current route key and keyboard shortcut config.
 * Invariants/Assumptions: The panel must stay manually accessible after first visit, default expansion should reflect route-visit context supplied by the parent, and next actions stay route-specific.
 */

import { useEffect, useState } from "react";
import type { ShortcutConfig } from "../hooks/useKeyboard";
import {
  ROUTE_HELP_CONTENT,
  type OnboardingRouteKey,
  type RouteHelpAction,
} from "../lib/onboarding";
import { ShortcutHint } from "./ShortcutHint";

interface RouteHelpPanelProps {
  routeKey: OnboardingRouteKey;
  shortcuts: ShortcutConfig;
  isMac?: boolean;
  defaultExpanded?: boolean;
  onOpenCommandPalette: () => void;
  onOpenShortcuts: () => void;
  onRestartTour: () => void;
  onAction: (actionId: RouteHelpAction["id"]) => void;
}

export function RouteHelpPanel({
  routeKey,
  shortcuts,
  isMac = false,
  defaultExpanded = false,
  onOpenCommandPalette,
  onOpenShortcuts,
  onRestartTour,
  onAction,
}: RouteHelpPanelProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);
  const content = ROUTE_HELP_CONTENT[routeKey];

  useEffect(() => {
    setIsExpanded(defaultExpanded);
  }, [defaultExpanded]);

  return (
    <section
      className={`route-help panel${isExpanded ? " is-expanded" : ""}`}
      data-tour="route-help"
      aria-label={`${content.title} for this route`}
    >
      <div className="route-help__header">
        <div>
          <div className="route-help__eyebrow">Route help</div>
          <h2>{content.title}</h2>
          <p>{content.summary}</p>
        </div>

        <button
          type="button"
          className="secondary"
          data-tour="route-help-toggle"
          onClick={() => setIsExpanded((previous) => !previous)}
          aria-expanded={isExpanded}
        >
          {isExpanded ? "Hide help" : "What can I do here?"}
        </button>
      </div>

      {isExpanded ? (
        <div className="route-help__content">
          <div className="route-help__section">
            <h3>What you can do</h3>
            <ul>
              {content.whatYouCanDo.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>

          <div className="route-help__section">
            <h3>Shortcuts for this route</h3>
            <div className="route-help__shortcuts">
              {content.shortcuts.map((item) => (
                <div key={item.label} className="route-help__shortcut">
                  <span>{item.label}</span>
                  <ShortcutHint
                    shortcut={shortcuts[item.shortcut]}
                    isMac={isMac}
                  />
                </div>
              ))}
            </div>
          </div>

          <div className="route-help__section">
            <h3>Next actions</h3>
            <div className="route-help__actions">
              {content.nextActions.map((action) => (
                <button
                  key={action.id}
                  type="button"
                  className="secondary"
                  onClick={() => onAction(action.id)}
                >
                  {action.label}
                </button>
              ))}
              <button
                type="button"
                className="secondary"
                onClick={onOpenCommandPalette}
              >
                Open command palette
              </button>
              <button
                type="button"
                className="secondary"
                onClick={onOpenShortcuts}
              >
                Open shortcut help
              </button>
              <button
                type="button"
                className="secondary"
                onClick={onRestartTour}
              >
                Restart full tour
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  );
}
