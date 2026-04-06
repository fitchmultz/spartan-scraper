# 2026-04-05 release readiness evidence

## Scope

This package captures the fresh release-readiness pass for the current workspace snapshot.

Because the day-to-day workspace already had unrelated in-flight edits, the exact release-candidate contents were copied into a clean temporary git repo before running the clean-tree-gated CI checks. That matches the release policy for validating a clean candidate tree without overwriting unrelated local work.

## Verification summary

| Checklist item | Result | Evidence |
| --- | --- | --- |
| `make audit-public` | Pass | `01-make-audit-public.txt` |
| `make ci-pr` | Pass | `02-make-ci-pr.txt` |
| `make ci` | Pass | `03-make-ci.txt` |
| `make ci-slow` | Pass | `04-make-ci-slow.txt` |
| `make secret-scan` (real repo history) | Pass | `05-make-secret-scan.txt` |
| CLI help/version smoke | Pass | `06-cli-help.txt`, `07-cli-version.txt` |
| API health smoke | Pass | `08-server.log`, `09-healthz.json` |
| WebSocket origin safety (`403`) | Pass | `10-ws-forbidden.txt` |
| Web UI shell loads in browser | Pass | `11-web-dev.log`, `12-web-index.html`, `13-web-title.txt`, `14-web-snapshot.txt`, `15-web-screenshot.png` |
| Guided create-job route loads | Pass | `16-new-job-snapshot.txt` |
| Automation batch route exposes all three batch modes | Pass | `17-batches-route-snapshot.txt` |

## Notes

- The public-hygiene sweep now covers `docs/evidence/**/*.json|txt|md` and catches temp/home-path leaks plus machine-local app paths such as `<local-chrome>` before release.
- `make secret-scan` was rerun against the real repository history rather than the temporary validation repo so the release evidence reflects the full git history requirement in `RELEASING.md`.
- The release pass also fixed the deterministic heavy-lane regression where `scripts/stress_test.sh` exported outside its command working directory.
- Fresh `make ci` / `make ci-slow` coverage includes the current batch-route regression suite (`web` Vitest + `internal/e2e`) in addition to the browser smoke artifacts above.
