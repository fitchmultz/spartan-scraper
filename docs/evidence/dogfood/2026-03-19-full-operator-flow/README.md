# 2026-03-19 Full operator flow dogfood

## Goal

Re-dogfood Spartan's fresh-start-to-daily-use operator path end to end: land on a blank workspace, discover navigation, create a real job, inspect saved output, verify CLI parity, and confirm first-run guidance gets out of the way once work has started.

## What changed

- Fixed command-palette navigation so route commands deep-link to the right path, including automation sections and job detail routes.
- Re-pointed the onboarding tour to stable New Job anchors so the guided flow lands on the wizard header and stepper reliably.
- Cut render-profile and pipeline-JS editors over to `getApiBaseUrl()` so Settings works correctly behind non-default dev proxies.
- Retired first-run guidance once real work exists: the onboarding nudge now auto-dismisses after the first job, and the Settings first-run overview hides once jobs are present.
- Standardized route-help actions away from stale `Create first job` copy on non-first-run surfaces.

## Visual verification

Validated against a fresh local runtime using a temporary `DATA_DIR`, backend on `127.0.0.1:8745`, and Vite on `127.0.0.1:5175`.

| Surface | Expectation | Result | Evidence |
| --- | --- | --- | --- |
| Jobs landing | Fresh workspace should orient the operator without blocking job creation | Pass | `screenshots/01-jobs-landing.png` |
| Onboarding tour | Tour should open and step through the new route model cleanly | Pass | `screenshots/02-onboarding-tour.png`, `screenshots/06-onboarding-new-job-step.png` |
| Command palette | Navigation commands should reach major routes and automation sections | Pass | `screenshots/03-command-palette-routes.png` |
| Guided job wizard | A scrape job should reach review and submit cleanly | Pass | `screenshots/04-job-wizard-review.png`, `screenshots/05-job-queued-toast.png` |
| Result inspection | Completed work should open on its own result route with saved output visible | Pass | `screenshots/07-results-view.png` |
| Templates + automation routes | Major secondary workspaces should remain reachable from the shell | Pass | `screenshots/09-templates-workspace.png`, `screenshots/10-automation-batches.png`, `screenshots/11-automation-chains.png`, `screenshots/12-automation-watches.png`, `screenshots/13-automation-exports.png`, `screenshots/14-automation-webhooks.png` |
| Post-first-job cleanup | First-run nudges should disappear once jobs exist and Settings should stop showing the first-run overview | Pass | `screenshots/16-settings-after-first-job.png`, `screenshots/17-jobs-after-first-job.png` |

## CLI and API verification

- Runtime health: `01-healthz.json`, `02-cli-health.txt`
- Direct CLI scrape against the same temp data dir: `05-cli-scrape.txt`, `05-cli-scrape.json`
- Persisted job inspection: `06-cli-jobs.txt`
- Markdown export inspection: `07-cli-export.txt`, `07-cli-export.md`

## Automated verification

- `cd web && CI=1 NODE_OPTIONS=--localstorage-file=.vitest-localstorage pnpm exec vitest run src/hooks/useOnboarding.test.ts src/lib/settings-overview.test.ts src/components/RouteHelpPanel.test.tsx src/components/SettingsOverviewPanel.test.tsx src/components/CommandPalette.test.tsx src/components/OnboardingFlow.test.tsx src/components/jobs/__tests__/JobSubmissionContainer.test.tsx src/components/render-profiles/RenderProfileEditor.test.tsx src/components/pipeline-js/PipelineJSEditor.test.tsx`
- `make ci`

## Notes

- For CLI parity against the alternate local server used in this dogfood run, `PORT=8745` and the temporary `DATA_DIR` must be supplied so CLI commands inspect the same runtime.
- The browser dogfood pass still used direct route opens occasionally because the automation snapshot tool was less reliable than URL navigation for SPA route confirmation during this session.
