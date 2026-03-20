/**
 * Purpose: Decide when the Settings route should surface the first-run overview panel.
 * Responsibilities: Centralize the pristine-workspace gate so Settings guidance disappears once real operator work or reusable inventory exists.
 * Scope: Pure Settings overview visibility logic only; rendering stays in `SettingsOverviewPanel` and `App.tsx`.
 * Usage: Import `shouldShowSettingsOverviewPanel(...)` from the Settings route container or related tests.
 * Invariants/Assumptions: The overview is first-run guidance only, so any persisted jobs or saved Settings inventory should hide it immediately.
 */

interface SettingsOverviewStatusInput {
  isSettingsRoute: boolean;
  setupRequired: boolean;
  jobsTotal: number;
  profilesCount: number;
  schedulesCount: number;
  crawlStatesTotal: number;
  renderProfileCount: number | null;
  pipelineScriptCount: number | null;
  proxyStatus?: string;
  retentionStatus?: string;
}

export function shouldShowSettingsOverviewPanel({
  isSettingsRoute,
  setupRequired,
  jobsTotal,
  profilesCount,
  schedulesCount,
  crawlStatesTotal,
  renderProfileCount,
  pipelineScriptCount,
  proxyStatus,
  retentionStatus,
}: SettingsOverviewStatusInput): boolean {
  if (!isSettingsRoute || setupRequired) {
    return false;
  }

  if (renderProfileCount === null || pipelineScriptCount === null) {
    return false;
  }

  const optionalSubsystemsQuiet = [proxyStatus, retentionStatus].every(
    (status) =>
      status === undefined || status === "disabled" || status === "ok",
  );

  return (
    optionalSubsystemsQuiet &&
    jobsTotal === 0 &&
    profilesCount === 0 &&
    schedulesCount === 0 &&
    crawlStatesTotal === 0 &&
    renderProfileCount === 0 &&
    pipelineScriptCount === 0
  );
}
