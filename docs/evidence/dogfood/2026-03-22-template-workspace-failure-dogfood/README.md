# Template workspace failure-path dogfood

Date: 2026-03-22

## Goal

Validate `/templates` close, discard, duplicate, builder, AI-apply, and save-failure flows in a real browser now that draft recovery is tab-resilient.

## Environment

- Backend: `env -u PROXY_POOL_FILE DATA_DIR=TMPDIR/spartan-template-dogfood.BciHFv/data PORT=8761 PI_ENABLED=true PI_CONFIG_PATH=TMPDIR/spartan-template-dogfood.BciHFv/pi-routes.json ./bin/spartan server`
- Frontend: `cd web && DEV_API_PROXY_TARGET=http://127.0.0.1:8761 pnpm exec vite --host 127.0.0.1 --port 5182`
- Fixture site: `python3 -m http.server 8772 --bind 127.0.0.1 -d TMPDIR/spartan-template-dogfood.BciHFv/site`
- Web URL: `http://127.0.0.1:5182/templates`
- API URL: `http://127.0.0.1:8761`
- AI mode: fixture (`PI_CONFIG_PATH=TMPDIR/spartan-template-dogfood.BciHFv/pi-routes.json`)

## Scenarios

| Scenario | Expected outcome | Result | Evidence |
| --- | --- | --- | --- |
| Close → hidden draft → resume | Closed drafts stay available in-tab and restore their unsaved edits | Pass | `screenshots/02-close-resume-before-close.png`, `screenshots/03-close-resume-hidden-draft.png`, `screenshots/04-close-resume-restored.png` |
| Hidden draft discard | Explicit discard removes the hidden draft instead of silently restoring it later | Pass | `screenshots/05-discard-confirm.png`, `screenshots/06-discard-complete.png` |
| Duplicate built-in template | Built-ins duplicate into an editable draft without destructive in-place actions | Pass after fix | `screenshots/07-duplicate-built-in.png`, `screenshots/23-duplicate-no-delete.png` |
| Dirty draft replacement when switching saved templates | Operators get a keep/discard confirmation before another saved template replaces local edits | Pass | `screenshots/08-switch-dirty-confirm.png`, `screenshots/09-switch-dirty-keep-draft.png`, `screenshots/10-switch-dirty-discarded.png` |
| Visual builder cancel/save | Builder opens from the current draft, cancel returns to the same draft, and builder save returns to the workspace with the saved template selected | Pass | `screenshots/11-builder-open.png`, `screenshots/12-builder-cancel-restores-draft.png`, `screenshots/13-builder-save-return.png`, `03-docs-template.json` |
| AI generate validation failure | Invalid AI-generated selectors stay non-destructive and surface the validation error inline | Pass | `screenshots/15-ai-generated-result.png`, `screenshots/15b-ai-generate-error-visible.png` |
| AI apply over dirty local draft | AI apply warns before replacing local edits, supports keep draft, and can intentionally replace the draft | Pass | `screenshots/16-ai-generated-success.png`, `screenshots/17-ai-apply-confirm.png`, `screenshots/18-ai-apply-keep-draft.png`, `screenshots/19-ai-apply-discarded.png` |
| Save failure retry path | Name-conflict save failures preserve the current draft for retry | Pass | `screenshots/20-save-failure-retains-draft.png` |
| New unsaved draft actions | New/create/AI-applied drafts should not expose Delete for the previously selected saved template | Fixed | `screenshots/22-new-draft-no-delete.png`, `screenshots/23-duplicate-no-delete.png` |
| Browser health | No browser console or page errors were reported during the verification pass | Pass | `24-browser-console.txt`, `25-browser-errors.txt` |

## Notes

- Dogfooding found one real route-level regression: unsaved create/duplicate drafts still exposed the destructive `Delete` action for the previously selected saved template. That was misleading and potentially destructive, so it was fixed before closing the pass.
- The AI fixture intentionally generates `[data-field="..."]` selectors. `article.html` was useful for the validation-failure path; `ai-article.html` was used to validate successful apply behavior.
- Builder save persisted the updated `docs` selectors through the real API, confirmed by `03-docs-template.json`.
