/**
 * Shared status pill component.
 *
 * Purpose:
 * - Render a consistent inline pill with an optional status dot.
 *
 * Responsibilities:
 * - Apply the shared status tone palette.
 * - Keep repeated status-chip markup out of feature components.
 *
 * Scope:
 * - Compact web UI status labels only.
 *
 * Usage:
 * - Import into feature components that need small inline status chips.
 *
 * Invariants/Assumptions:
 * - The component is presentation-only and does not own status-mapping logic.
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
