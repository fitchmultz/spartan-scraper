/**
 * Purpose: Provide the public route-container exports consumed by the application shell.
 * Responsibilities: Re-export the extracted route modules behind the stable `AppRoutes` entrypoint used by `App.tsx` and tests.
 * Scope: Route export wiring only; route implementations live in `JobRoutes.tsx` and `WorkspaceRoutes.tsx`.
 * Usage: Import route containers from this module to keep callers insulated from internal file layout changes.
 * Invariants/Assumptions: The exported route names remain stable even as the underlying route files are split into smaller units.
 */

export {
  JobDetailRoute,
  JobsRoute,
  NewJobRoute,
  TemplatesRoute,
} from "./JobRoutes";
export {
  AutomationRoute,
  SettingsRoute,
  SetupRequiredRoute,
} from "./WorkspaceRoutes";
