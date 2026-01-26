/**
 * Hero Component
 *
 * Displays the main header and live status overview. Shows the application title,
 * description, and real-time metrics including loading status, job queue counts,
 * total jobs, and headless/playwright mode configuration.
 *
 * @module Hero
 */

interface HeroProps {
  loading: boolean;
  managerStatus: { queued: number; active: number } | null;
  jobsCount: number;
  headless: boolean;
  usePlaywright: boolean;
}

export function Hero({
  loading,
  managerStatus,
  jobsCount,
  headless,
  usePlaywright,
}: HeroProps) {
  return (
    <section className="hero">
      <div className="hero-card">
        <div className="kicker">Operation Spartan</div>
        <h1>Spartan Scraper Command Center</h1>
        <p>
          Unified scraping and automation. Single pages, site-wide crawls,
          headless login flows, and durable job tracking.
        </p>
      </div>
      <div className="stats">
        <h3>Live Signals</h3>
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
