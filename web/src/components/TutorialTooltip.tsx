/**
 * Purpose: Render contextual onboarding tooltips and beacons for shell-level hints without hijacking primary actions.
 * Responsibilities: Position tooltip overlays against target elements, manage hover/focus visibility for passive hints, and expose a controlled backdrop mode for guided onboarding.
 * Scope: Reusable tooltip and beacon primitives used by the web shell onboarding surfaces.
 * Usage: Render `TutorialTooltip` for passive contextual help or `TutorialBeacon` to highlight a target element.
 * Invariants/Assumptions: Tooltip positioning is viewport-clamped, uncontrolled hints should not open from pointer-originated focus events, and controlled tooltips own their own dismiss backdrop.
 */

import { useState, useEffect, useCallback, useRef } from "react";

export interface TutorialTooltipProps {
  /** Target element selector or ref */
  target?: string;
  /** Tooltip title */
  title: string;
  /** Tooltip content/description */
  content: string;
  /** Tooltip position relative to target */
  position?: "top" | "bottom" | "left" | "right";
  /** Whether to show the pulsing beacon */
  showBeacon?: boolean;
  /** Whether the tooltip is visible (controlled mode) */
  isOpen?: boolean;
  /** Callback when tooltip is dismissed */
  onDismiss?: () => void;
  /** Delay before showing tooltip on hover (ms) */
  showDelay?: number;
  /** Whether to show on hover (uncontrolled mode) */
  showOnHover?: boolean;
  /** Additional CSS class */
  className?: string;
}

/**
 * Tutorial Tooltip Component
 *
 * Displays contextual help information for UI elements.
 * Can operate in controlled or uncontrolled mode.
 *
 * @example
 * ```tsx
 * // Uncontrolled (hover to show)
 * <TutorialTooltip
 *   target="#auth-section"
 *   title="Auth Profiles"
 *   content="Select a saved auth profile or configure custom credentials"
 *   position="bottom"
 * />
 *
 * // Controlled (programmatic)
 * <TutorialTooltip
 *   title="Step 3"
 *   content="Configure your extraction template"
 *   isOpen={currentStep === 3}
 *   position="right"
 * />
 * ```
 */
export function TutorialTooltip({
  target,
  title,
  content,
  position = "bottom",
  showBeacon = false,
  isOpen: controlledIsOpen,
  onDismiss,
  showDelay = 300,
  showOnHover = true,
  className = "",
}: TutorialTooltipProps) {
  const [isVisible, setIsVisible] = useState(false);
  const [tooltipPosition, setTooltipPosition] = useState({ top: 0, left: 0 });
  const tooltipRef = useRef<HTMLDivElement>(null);
  const targetRef = useRef<HTMLElement | null>(null);
  const hoverTimeoutRef = useRef<number | null>(null);
  const pointerFocusRef = useRef(false);

  const isControlled = controlledIsOpen !== undefined;
  const shouldShow = isControlled ? controlledIsOpen : isVisible;

  /**
   * Calculate tooltip position based on target element and desired position.
   */
  const calculatePosition = useCallback(() => {
    if (!targetRef.current || !tooltipRef.current) return;

    const targetRect = targetRef.current.getBoundingClientRect();
    const tooltipRect = tooltipRef.current.getBoundingClientRect();
    const margin = 12;

    let top = 0;
    let left = 0;

    switch (position) {
      case "top":
        top = targetRect.top - tooltipRect.height - margin;
        left = targetRect.left + targetRect.width / 2 - tooltipRect.width / 2;
        break;
      case "bottom":
        top = targetRect.bottom + margin;
        left = targetRect.left + targetRect.width / 2 - tooltipRect.width / 2;
        break;
      case "left":
        top = targetRect.top + targetRect.height / 2 - tooltipRect.height / 2;
        left = targetRect.left - tooltipRect.width - margin;
        break;
      case "right":
        top = targetRect.top + targetRect.height / 2 - tooltipRect.height / 2;
        left = targetRect.right + margin;
        break;
    }

    // Keep within viewport
    const padding = 8;
    top = Math.max(
      padding,
      Math.min(top, window.innerHeight - tooltipRect.height - padding),
    );
    left = Math.max(
      padding,
      Math.min(left, window.innerWidth - tooltipRect.width - padding),
    );

    setTooltipPosition({ top, left });
  }, [position]);

  /**
   * Find and track target element.
   */
  useEffect(() => {
    if (!target) return;

    const findTarget = () => {
      const element = document.querySelector(target) as HTMLElement;
      if (element) {
        targetRef.current = element;
        calculatePosition();
      }
    };

    findTarget();

    // Re-calculate on resize
    window.addEventListener("resize", calculatePosition);
    return () => window.removeEventListener("resize", calculatePosition);
  }, [target, calculatePosition]);

  /**
   * Update position when visibility changes.
   */
  useEffect(() => {
    if (shouldShow) {
      // Small delay to allow tooltip to render before calculating position
      const timer = setTimeout(calculatePosition, 0);
      return () => clearTimeout(timer);
    }
  }, [shouldShow, calculatePosition]);

  /**
   * Handle hover events on target element.
   */
  useEffect(() => {
    if (!target || isControlled || !showOnHover) return;

    const element = document.querySelector(target);
    if (!element) return;

    const handleMouseEnter = () => {
      if (hoverTimeoutRef.current) {
        window.clearTimeout(hoverTimeoutRef.current);
      }
      hoverTimeoutRef.current = window.setTimeout(() => {
        setIsVisible(true);
      }, showDelay);
    };

    const handleMouseLeave = () => {
      if (hoverTimeoutRef.current) {
        window.clearTimeout(hoverTimeoutRef.current);
        hoverTimeoutRef.current = null;
      }
      setIsVisible(false);
    };

    const handlePointerDown = () => {
      pointerFocusRef.current = true;
    };

    const handleFocus = () => {
      if (pointerFocusRef.current) {
        pointerFocusRef.current = false;
        return;
      }
      setIsVisible(true);
    };

    const handleBlur = () => {
      pointerFocusRef.current = false;
      setIsVisible(false);
    };

    element.addEventListener("mouseenter", handleMouseEnter);
    element.addEventListener("mouseleave", handleMouseLeave);
    element.addEventListener("pointerdown", handlePointerDown, true);
    element.addEventListener("focus", handleFocus, true);
    element.addEventListener("blur", handleBlur, true);

    return () => {
      element.removeEventListener("mouseenter", handleMouseEnter);
      element.removeEventListener("mouseleave", handleMouseLeave);
      element.removeEventListener("pointerdown", handlePointerDown, true);
      element.removeEventListener("focus", handleFocus, true);
      element.removeEventListener("blur", handleBlur, true);
      pointerFocusRef.current = false;
      if (hoverTimeoutRef.current) {
        window.clearTimeout(hoverTimeoutRef.current);
      }
    };
  }, [target, isControlled, showOnHover, showDelay]);

  /**
   * Handle dismiss via Escape key.
   */
  useEffect(() => {
    if (!shouldShow) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        if (!isControlled) {
          setIsVisible(false);
        }
        onDismiss?.();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [shouldShow, isControlled, onDismiss]);

  if (!shouldShow) {
    // Render beacon if enabled but tooltip not visible
    if (showBeacon && target) {
      const element = document.querySelector(target);
      if (element) {
        const rect = element.getBoundingClientRect();
        return (
          <span
            style={{
              position: "fixed",
              top: rect.top + rect.height / 2 - 8,
              left: rect.right - 8,
              width: 16,
              height: 16,
              borderRadius: "50%",
              background: "var(--accent, #ffb700)",
              animation: "tutorial-beacon-pulse 2s infinite",
              zIndex: 1000,
              pointerEvents: "none",
              display: "inline-block",
            }}
          />
        );
      }
    }
    return null;
  }

  return (
    <>
      {/* Backdrop for controlled mode */}
      {isControlled && (
        <button
          type="button"
          style={{
            position: "fixed",
            inset: 0,
            background: "transparent",
            zIndex: 1000,
            border: "none",
            padding: 0,
            cursor: "default",
          }}
          onClick={() => {
            if (!isControlled) setIsVisible(false);
            onDismiss?.();
          }}
          aria-label="Close tooltip"
        />
      )}

      {/* Tooltip */}
      <div
        ref={tooltipRef}
        className={`tutorial-tooltip ${className}`}
        role="tooltip"
        style={{
          position: "fixed",
          top: tooltipPosition.top,
          left: tooltipPosition.left,
          zIndex: 1001,
          background: "var(--panel, #1a1a24)",
          border: "1px solid var(--stroke, rgba(255,255,255,0.1))",
          borderRadius: 12,
          padding: 16,
          maxWidth: 320,
          boxShadow: "0 8px 32px var(--shadow, rgba(0,0,0,0.4))",
          animation: "tutorial-tooltip-fade-in 0.2s ease",
        }}
      >
        {/* Arrow */}
        <div
          style={{
            position: "absolute",
            width: 0,
            height: 0,
            ...(position === "bottom" && {
              top: -8,
              left: "50%",
              transform: "translateX(-50%)",
              borderLeft: "8px solid transparent",
              borderRight: "8px solid transparent",
              borderBottom: "8px solid var(--stroke, rgba(255,255,255,0.1))",
            }),
            ...(position === "top" && {
              bottom: -8,
              left: "50%",
              transform: "translateX(-50%)",
              borderLeft: "8px solid transparent",
              borderRight: "8px solid transparent",
              borderTop: "8px solid var(--stroke, rgba(255,255,255,0.1))",
            }),
            ...(position === "right" && {
              left: -8,
              top: "50%",
              transform: "translateY(-50%)",
              borderTop: "8px solid transparent",
              borderBottom: "8px solid transparent",
              borderRight: "8px solid var(--stroke, rgba(255,255,255,0.1))",
            }),
            ...(position === "left" && {
              right: -8,
              top: "50%",
              transform: "translateY(-50%)",
              borderTop: "8px solid transparent",
              borderBottom: "8px solid transparent",
              borderLeft: "8px solid var(--stroke, rgba(255,255,255,0.1))",
            }),
          }}
        />

        {/* Title */}
        <div
          style={{
            fontWeight: 600,
            fontSize: "0.95rem",
            color: "var(--text, #fff)",
            marginBottom: 8,
            display: "flex",
            alignItems: "center",
            gap: 8,
          }}
        >
          {showBeacon && (
            <span
              style={{
                width: 8,
                height: 8,
                borderRadius: "50%",
                background: "var(--accent, #ffb700)",
              }}
            />
          )}
          {title}
        </div>

        {/* Content */}
        <div
          style={{
            fontSize: "0.85rem",
            color: "var(--muted, #a0a0a8)",
            lineHeight: 1.5,
          }}
        >
          {content}
        </div>

        {/* Dismiss hint */}
        {!isControlled && (
          <div
            style={{
              marginTop: 12,
              fontSize: "0.7rem",
              color: "var(--muted, #a0a0a8)",
              opacity: 0.7,
            }}
          >
            Press{" "}
            <kbd
              style={{
                background: "rgba(255,255,255,0.1)",
                padding: "2px 6px",
                borderRadius: 4,
                fontFamily: "inherit",
              }}
            >
              Esc
            </kbd>{" "}
            to dismiss
          </div>
        )}
      </div>
    </>
  );
}

/**
 * Standalone beacon component for highlighting elements.
 */
export function TutorialBeacon({
  target,
  onClick,
}: {
  target: string;
  onClick?: () => void;
}) {
  const [position, setPosition] = useState({ top: 0, left: 0 });

  useEffect(() => {
    const updatePosition = () => {
      const element = document.querySelector(target);
      if (element) {
        const rect = element.getBoundingClientRect();
        setPosition({
          top: rect.top + rect.height / 2 - 8,
          left: rect.right - 8,
        });
      }
    };

    updatePosition();
    window.addEventListener("resize", updatePosition);
    return () => window.removeEventListener("resize", updatePosition);
  }, [target]);

  return (
    <button
      type="button"
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          onClick?.();
        }
      }}
      style={{
        position: "fixed",
        top: position.top,
        left: position.left,
        width: 16,
        height: 16,
        borderRadius: "50%",
        background: "var(--accent, #ffb700)",
        border: "none",
        cursor: onClick ? "pointer" : "default",
        animation: "tutorial-beacon-pulse 2s infinite",
        zIndex: 1000,
        padding: 0,
      }}
      aria-label="Tutorial hint"
    />
  );
}
