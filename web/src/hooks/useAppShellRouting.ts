/**
 * Purpose: Own the browser history and route parsing state for the app shell.
 * Responsibilities: Normalize and parse top-level paths, keep the shell synchronized with popstate, enforce canonical automation/settings URLs, and preserve promotion seeds across navigation.
 * Scope: App-shell route state only; route-local workflows and rendering stay in `App.tsx` and route containers.
 * Usage: Call from `App.tsx` once per shell render to obtain the parsed route, navigation helper, and promotion-seed accessors.
 * Invariants/Assumptions: The app runs in a browser context, supported routes match the shell contract, and promotion seed history state is a plain serializable object.
 */

import { useCallback, useEffect, useMemo, useState } from "react";

import {
  DEFAULT_AUTOMATION_SECTION,
  getAutomationPath,
  getAutomationSectionFromHash,
  getAutomationSectionFromPath,
  type AutomationSection,
} from "../components/automation/automationSections";
import {
  DEFAULT_SETTINGS_SECTION,
  getSettingsPath,
  getSettingsSectionFromPath,
  type SettingsSectionId,
} from "../components/settings/settingsSections";
import type { PromotionSeed } from "../types/promotion";

export type RouteKind =
  | "jobs"
  | "new-job"
  | "job-detail"
  | "templates"
  | "automation"
  | "settings";

export interface AppRoute {
  kind: RouteKind;
  path: string;
  jobId?: string;
  automationSection?: AutomationSection;
  settingsSection?: SettingsSectionId;
}

interface AppNavigationState {
  promotionSeed?: PromotionSeed | null;
}

export function normalizePath(pathname: string): string {
  if (!pathname || pathname === "/") {
    return "/jobs";
  }

  const trimmed = pathname.replace(/\/+$/, "");
  return trimmed === "" ? "/jobs" : trimmed;
}

export function parseRoute(pathname: string): AppRoute {
  const path = normalizePath(pathname);
  if (path === "/jobs") {
    return { kind: "jobs", path };
  }
  if (path === "/jobs/new") {
    return { kind: "new-job", path };
  }
  if (path.startsWith("/jobs/")) {
    const jobId = decodeURIComponent(path.slice("/jobs/".length));
    return { kind: "job-detail", path, jobId };
  }
  if (path === "/templates") {
    return { kind: "templates", path };
  }
  if (path === "/automation" || path.startsWith("/automation/")) {
    return {
      kind: "automation",
      path,
      automationSection:
        getAutomationSectionFromPath(path) ?? DEFAULT_AUTOMATION_SECTION,
    };
  }
  if (path === "/settings" || path.startsWith("/settings/")) {
    return {
      kind: "settings",
      path,
      settingsSection:
        getSettingsSectionFromPath(path) ?? DEFAULT_SETTINGS_SECTION,
    };
  }
  return { kind: "jobs", path: "/jobs" };
}

export function useAppShellRouting() {
  const [pathname, setPathname] = useState(() =>
    normalizePath(window.location.pathname),
  );
  const [navigationState, setNavigationState] = useState<AppNavigationState>(
    () => (window.history.state as AppNavigationState | null) ?? {},
  );

  const route = useMemo(() => parseRoute(pathname), [pathname]);

  const navigate = useCallback(
    (path: string, state?: AppNavigationState | null) => {
      const nextPath = normalizePath(path);
      const nextState = state ?? {};
      if (
        nextPath === pathname &&
        JSON.stringify(window.history.state ?? {}) === JSON.stringify(nextState)
      ) {
        return;
      }
      window.history.pushState(nextState, "", nextPath);
      setPathname(nextPath);
      setNavigationState(nextState);
    },
    [pathname],
  );

  useEffect(() => {
    const handlePopState = () => {
      setPathname(normalizePath(window.location.pathname));
      setNavigationState(
        (window.history.state as AppNavigationState | null) ?? {},
      );
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  useEffect(() => {
    if (route.kind !== "automation") {
      return;
    }

    const nextSection =
      pathname === "/automation"
        ? (getAutomationSectionFromHash(window.location.hash) ??
          route.automationSection ??
          DEFAULT_AUTOMATION_SECTION)
        : (route.automationSection ?? DEFAULT_AUTOMATION_SECTION);
    const canonicalPath = getAutomationPath(nextSection);

    if (pathname === canonicalPath) {
      return;
    }

    window.history.replaceState(window.history.state ?? {}, "", canonicalPath);
    setPathname(canonicalPath);
  }, [pathname, route.automationSection, route.kind]);

  useEffect(() => {
    if (route.kind !== "settings") {
      return;
    }

    const canonicalPath = getSettingsPath(
      route.settingsSection ?? DEFAULT_SETTINGS_SECTION,
    );

    if (pathname === canonicalPath) {
      return;
    }

    window.history.replaceState(window.history.state ?? {}, "", canonicalPath);
    setPathname(canonicalPath);
  }, [pathname, route.kind, route.settingsSection]);

  const routePromotionSeed = navigationState.promotionSeed ?? null;

  const clearPromotionSeed = useCallback(() => {
    if (!navigationState.promotionSeed) {
      return;
    }

    const nextState: AppNavigationState = {};
    window.history.replaceState(nextState, "", window.location.pathname);
    setNavigationState(nextState);
  }, [navigationState.promotionSeed]);

  return {
    route,
    navigate,
    routePromotionSeed,
    clearPromotionSeed,
  };
}
