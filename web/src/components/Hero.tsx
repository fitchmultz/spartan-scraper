/**
 * Purpose: Render the hero UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import { ThemeToggle } from "./ThemeToggle";
import type { Theme, ResolvedTheme } from "../hooks/useTheme";

interface HeroProps {
  loading: boolean;
  managerStatus: { queued: number; active: number } | null;
  jobsCount: number;
  headless: boolean;
  usePlaywright: boolean;
  theme: Theme;
  resolvedTheme: ResolvedTheme;
  onThemeChange: (theme: Theme) => void;
  onThemeToggle: () => void;
}

export function Hero({
  loading,
  managerStatus,
  jobsCount,
  headless,
  usePlaywright,
  theme,
  resolvedTheme,
  onThemeChange,
  onThemeToggle,
}: HeroProps) {
  return (
    <section className="hero">
      <div className="hero-card" data-tour="hero">
        <div className="kicker">Operation Spartan</div>
        <h1>Spartan Scraper Command Center</h1>
        <p>
          Unified scraping and automation. Single pages, site-wide crawls,
          headless login flows, and durable job tracking.
        </p>
      </div>
      <div className="stats" data-tour="fetcher-options">
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: "12px",
          }}
          data-tour="command-palette"
        >
          <h3 style={{ margin: 0 }}>Live Signals</h3>
          <ThemeToggle
            theme={theme}
            resolvedTheme={resolvedTheme}
            onThemeChange={onThemeChange}
            onToggle={onThemeToggle}
          />
        </div>
        <div>{loading ? "Refreshing…" : "Standing by"}</div>
        {managerStatus !== null ? (
          <>
            <div>Queued: {managerStatus.queued}</div>
            <div>Active: {managerStatus.active}</div>
          </>
        ) : null}
        <div>Total jobs: {jobsCount}</div>
        <div>Headless mode: {headless ? "Enabled" : "Disabled"}</div>
        <div>Playwright: {usePlaywright ? "Enabled" : "Disabled"}</div>
      </div>
    </section>
  );
}
