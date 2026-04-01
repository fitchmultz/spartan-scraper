/**
 * Purpose: Render the skeleton UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import type { CSSProperties } from "react";

export interface SkeletonProps {
  /** Width of the skeleton element */
  width?: string | number;
  /** Height of the skeleton element */
  height?: string | number;
  /** Render as a circle (for avatars, etc.) */
  circle?: boolean;
  /** Additional CSS class names */
  className?: string;
  /** Additional inline styles */
  style?: CSSProperties;
}

/**
 * Base skeleton placeholder with shimmer animation.
 *
 * @example
 * ```tsx
 * <Skeleton width="200px" height="24px" />
 * <Skeleton width={40} height={40} circle />
 * ```
 */
export function Skeleton({
  width = "100%",
  height = "16px",
  circle = false,
  className = "",
  style = {},
}: SkeletonProps) {
  const computedStyle: CSSProperties = {
    width,
    height,
    borderRadius: circle ? "50%" : "8px",
    ...style,
  };

  return (
    <div
      className={`skeleton ${className}`.trim()}
      style={computedStyle}
      aria-hidden="true"
    />
  );
}

/**
 * Skeleton for a job list item.
 */
export function JobItemSkeleton() {
  return (
    <div
      style={{
        padding: "12px 16px",
        borderRadius: "14px",
        background: "var(--bg-alt)",
        border: "1px solid var(--stroke)",
        display: "flex",
        flexDirection: "column",
        gap: "8px",
      }}
      aria-hidden="true"
    >
      <Skeleton width="60%" height="14px" />
      <Skeleton width="30%" height="12px" />
    </div>
  );
}

/**
 * Skeleton for a list of job items.
 */
export function JobListSkeleton({ count = 5 }: { count?: number }) {
  return (
    <output
      style={{
        display: "grid",
        gap: "12px",
      }}
      aria-label="Loading jobs"
    >
      {Array.from({ length: count }, (_, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: skeleton placeholders are static
        <JobItemSkeleton key={i} />
      ))}
    </output>
  );
}

/**
 * Skeleton for a panel/card component.
 */
export function PanelSkeleton() {
  return (
    <div className="panel" aria-hidden="true">
      <Skeleton width="40%" height="24px" style={{ marginBottom: "16px" }} />
      <Skeleton width="100%" height="40px" style={{ marginBottom: "12px" }} />
      <Skeleton width="100%" height="40px" style={{ marginBottom: "12px" }} />
      <Skeleton width="80%" height="40px" />
    </div>
  );
}

/**
 * Skeleton for the hero section.
 */
export function HeroSkeleton() {
  return (
    <section className="hero" aria-label="Loading...">
      <div className="hero-card">
        <Skeleton width="120px" height="12px" style={{ marginBottom: "8px" }} />
        <Skeleton width="80%" height="48px" style={{ marginBottom: "8px" }} />
        <Skeleton width="90%" height="20px" />
      </div>
      <div className="stats">
        <Skeleton width="100px" height="20px" style={{ marginBottom: "8px" }} />
        <Skeleton width="80%" height="16px" />
        <Skeleton width="60%" height="16px" />
        <Skeleton width="70%" height="16px" />
      </div>
    </section>
  );
}

/**
 * Skeleton for a result item.
 */
export function ResultItemSkeleton() {
  return (
    <div
      style={{
        padding: "10px 14px",
        borderRadius: "10px",
        background: "var(--bg-alt)",
        border: "1px solid var(--stroke)",
        display: "flex",
        flexDirection: "column",
        gap: "4px",
      }}
      aria-hidden="true"
    >
      <div
        style={{ display: "flex", justifyContent: "space-between", gap: "8px" }}
      >
        <Skeleton width="70%" height="14px" />
        <Skeleton width="50px" height="14px" />
      </div>
      <Skeleton width="40%" height="12px" />
    </div>
  );
}

/**
 * Skeleton for a list of result items.
 */
export function ResultListSkeleton({ count = 5 }: { count?: number }) {
  return (
    <output
      style={{
        display: "grid",
        gap: "8px",
      }}
      aria-label="Loading results"
    >
      {Array.from({ length: count }, (_, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: skeleton placeholders are static
        <ResultItemSkeleton key={i} />
      ))}
    </output>
  );
}

/**
 * Skeleton for the results viewer.
 */
export function ResultsViewerSkeleton() {
  return (
    <output className="panel" aria-label="Loading results">
      <Skeleton width="30%" height="24px" style={{ marginBottom: "16px" }} />
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr 1fr",
          gap: "16px",
        }}
      >
        <div>
          <Skeleton
            width="100%"
            height="300px"
            style={{ marginBottom: "12px" }}
          />
          <div
            style={{ display: "flex", justifyContent: "center", gap: "8px" }}
          >
            <Skeleton width="80px" height="32px" />
            <Skeleton width="80px" height="32px" />
          </div>
        </div>
        <div>
          <Skeleton
            width="100%"
            height="150px"
            style={{ marginBottom: "12px" }}
          />
          <Skeleton width="100%" height="150px" />
        </div>
      </div>
    </output>
  );
}

/**
 * Skeleton for form inputs.
 */
export function FormFieldSkeleton() {
  return (
    <div style={{ marginBottom: "12px" }} aria-hidden="true">
      <Skeleton width="80px" height="14px" style={{ marginBottom: "6px" }} />
      <Skeleton width="100%" height="40px" />
    </div>
  );
}

/**
 * Skeleton for a complete form panel.
 */
export function FormSkeleton({ fieldCount = 4 }: { fieldCount?: number }) {
  return (
    <output className="panel" aria-label="Loading form">
      <Skeleton width="40%" height="24px" style={{ marginBottom: "16px" }} />
      {Array.from({ length: fieldCount }, (_, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: skeleton placeholders are static
        <FormFieldSkeleton key={i} />
      ))}
      <Skeleton width="120px" height="40px" />
    </output>
  );
}

/**
 * Full page loading state with multiple skeleton sections.
 */
export function PageSkeleton() {
  return (
    <output className="app" aria-label="Loading page">
      <HeroSkeleton />
      <section
        className="grid"
        style={{
          display: "grid",
          gap: "20px",
          gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))",
        }}
      >
        <FormSkeleton fieldCount={3} />
        <FormSkeleton fieldCount={3} />
        <FormSkeleton fieldCount={3} />
      </section>
      <div className="panel">
        <Skeleton
          width="100px"
          height="24px"
          style={{ marginBottom: "16px" }}
        />
        <JobListSkeleton count={3} />
      </div>
    </output>
  );
}
