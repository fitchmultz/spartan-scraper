/**
 * Purpose: Render the status pill UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import type { CSSProperties, ReactNode } from "react";

import { getStatusToneColors, type StatusTone } from "../lib/status-display";

type StatusPillProps = {
  label: ReactNode;
  tone: StatusTone;
  showDot?: boolean;
  style?: CSSProperties;
};

export function StatusPill({
  label,
  tone,
  showDot = true,
  style,
}: StatusPillProps) {
  const colors = getStatusToneColors(tone);

  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 6,
        padding: "4px 10px",
        borderRadius: 12,
        fontSize: 12,
        fontWeight: 500,
        backgroundColor: colors.backgroundColor,
        color: colors.color,
        ...style,
      }}
    >
      {showDot ? (
        <span
          aria-hidden="true"
          style={{
            width: 6,
            height: 6,
            borderRadius: "50%",
            backgroundColor: colors.color,
          }}
        />
      ) : null}
      {label}
    </span>
  );
}
