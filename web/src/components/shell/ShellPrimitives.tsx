/**
 * Purpose: Provide compact shared shell primitives for the web app's global top bar, route headers, and route-local signals.
 * Responsibilities: Render the simplified product shell, keep route framing consistent, and isolate shell markup from App.tsx.
 * Scope: Shared shell primitives only; route content and business logic remain owned by the application shell and route containers.
 * Usage: Import `AppTopBar`, `RouteHeader`, and `RouteSignals` from the root application shell or future route layouts.
 * Invariants/Assumptions: The shell stays compact, route context belongs to each route, and signal rows are optional per route.
 */

import type { ReactNode } from "react";

export interface AppShellNavItem<TKind extends string = string> {
  kind: TKind;
  label: string;
  path: string;
  description: string;
}

export interface RouteSignal {
  label: string;
  value: string | number;
}

interface AppTopBarProps<TKind extends string> {
  activeRoute: TKind;
  navItems: ReadonlyArray<AppShellNavItem<TKind>>;
  onNavigate: (path: string) => void;
  globalAction?: ReactNode;
  utilities?: ReactNode;
}

export function AppTopBar<TKind extends string>({
  activeRoute,
  navItems,
  onNavigate,
  globalAction,
  utilities,
}: AppTopBarProps<TKind>) {
  return (
    <header className="app-shell">
      <div className="app-shell__topbar">
        <button
          type="button"
          className="app-brand"
          onClick={() => onNavigate("/jobs")}
          aria-label="Go to Jobs"
        >
          <span className="app-brand__mark" aria-hidden="true">
            S
          </span>
          <span className="app-brand__copy">
            <strong className="app-brand__title">Spartan Scraper</strong>
            <span className="app-brand__meta">Local-first workbench</span>
          </span>
        </button>

        <nav className="app-nav" aria-label="Primary">
          <div className="app-nav__items">
            {navItems.map((item) => {
              const isActive = item.kind === activeRoute;

              return (
                <button
                  key={item.path}
                  type="button"
                  className={
                    isActive ? "app-nav__button is-active" : "app-nav__button"
                  }
                  onClick={() => onNavigate(item.path)}
                  aria-current={isActive ? "page" : undefined}
                  title={item.description}
                >
                  {item.label}
                </button>
              );
            })}
          </div>
        </nav>

        <div className="app-shell__toolbar" data-tour="command-palette">
          {globalAction}
          {utilities}
        </div>
      </div>
    </header>
  );
}

export function RouteHeader({
  title,
  description,
  actions,
  subnav,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
  subnav?: ReactNode;
}) {
  return (
    <section className="route-header" aria-label={`${title} overview`}>
      <div className="route-header__title-row">
        <div className="route-header__heading">
          <h1>{title}</h1>
          {description ? <p>{description}</p> : null}
        </div>

        {actions ? (
          <div className="route-header__actions">{actions}</div>
        ) : null}
      </div>

      {subnav ? <div className="route-header__subnav">{subnav}</div> : null}
    </section>
  );
}

export function RouteSignals({
  items,
  ariaLabel = "Route signals",
}: {
  items: RouteSignal[];
  ariaLabel?: string;
}) {
  if (items.length === 0) {
    return null;
  }

  return (
    <section className="route-signals" aria-label={ariaLabel}>
      {items.map((item) => (
        <SignalPill key={item.label} label={item.label} value={item.value} />
      ))}
    </section>
  );
}

export function SignalPill({
  label,
  value,
}: {
  label: string;
  value: string | number;
}) {
  return (
    <div className="signal-pill">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}
