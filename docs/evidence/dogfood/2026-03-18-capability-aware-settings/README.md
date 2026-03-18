# Capability-aware settings follow-through dogfood

Date: 2026-03-18

## Goal

Manually validate the new guided settings/recovery experience in the Web UI and confirm a real website workflow still succeeds after the cutover.

## Environment

- Backend: `./bin/spartan server`
- Frontend: `make web-dev`
- Web URL: `http://localhost:5173`
- Real website exercised through the product: `https://example.com`

## Scenarios

| Scenario | Expected outcome | Result | Evidence |
| --- | --- | --- | --- |
| Settings route on a fresh-ish workspace | Empty sections explain when auth profiles, schedules, and crawl states matter instead of disappearing | Pass | `screenshots/01-settings-overview.png` |
| Proxy pool capability guidance | Proxy pool panel explains the missing file state and keeps inline re-check actions visible | Pass | `screenshots/01-settings-overview.png`, `screenshots/02-proxy-inline-diagnostic.png` |
| Retention capability guidance | Disabled retention explains what it does, shows env/CLI next steps, and allows preview cleanup from the panel | Pass | `screenshots/01-settings-overview.png`, `screenshots/03-retention-preview.png` |
| Real-site scrape submission | Guided wizard can still submit a live scrape successfully | Pass | `screenshots/04-real-site-scrape-success.png` |
| Result inspection after live scrape | Results view shows the extracted `https://example.com` payload without regressions | Pass | `screenshots/05-example-result.png` |
| Cross-surface guided status output | CLI and MCP surfaces surface the same guided proxy/retention/health framing after the cutover | Pass | `06-cli-proxy-pool-status.txt`, `07-cli-retention-status.txt`, `08-mcp-health-status.json` |

## Notes

- The settings route now reads as a control center instead of an implementation dump. Auth Profiles, Schedules, and Crawl States remain visible even when empty, with clear “when to care” guidance.
- Proxy pool recovery is easier to understand in-product: the panel explains the exact missing file problem, offers the intentional disable path, and supports inline re-checks without leaving Settings.
- Retention now presents “disabled” as an intentional optional state, not a failure, and the preview action returns inline results immediately.
- A live scrape against `https://example.com` completed successfully and the result viewer showed the expected Example Domain content.

## Issues found

None blocking during this validation pass.
