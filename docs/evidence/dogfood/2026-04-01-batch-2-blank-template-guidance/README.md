# 2026-04-01 Batch 2 blank template promotion guidance

## Goal

Validate the Batch 2 template-promotion cutover: start from a plain scrape job with no reusable template rules, promote that job into `/templates`, and reach a successful template save without trial-and-error at a `1280×720` viewport.

## Environment

- Backend: `DATA_DIR=<temp> PORT=8759 ./bin/spartan server`
- Frontend: `DEV_API_PROXY_TARGET=http://127.0.0.1:8759 pnpm exec vite --host 127.0.0.1 --port 5179`
- Web URL: `http://127.0.0.1:5179`
- Viewport: `1280×720`
- Source URL used for the plain scrape: `https://example.com`

## Flow covered

1. Create a plain scrape job for `https://example.com` without adding extraction rules.
2. Open the successful job detail page.
3. Promote the job to a template draft.
4. Confirm the blank-draft guidance explains why no selectors were carried forward and why save is disabled.
5. Use the new `Use title starter` action to seed the first reusable rule.
6. Save the promoted template successfully as `example-com-template`.

## Acceptance summary

| Surface | Expectation | Result | Evidence |
| --- | --- | --- | --- |
| Job creation review | Plain scrape job can still be submitted normally before the promotion test starts | Pass | `screenshots/03-job-review.png`, `screenshots/04-jobs-after-submit.png` |
| Blank template promotion entry | Promoted `/templates` draft explains that no reusable selectors were recovered and presents an obvious next action above the fold | Pass | `screenshots/05-job-detail.png`, `screenshots/06-template-guidance.png` |
| Save guardrails | Save stays blocked until the first reusable selector rule is complete, with inline blocker copy instead of sequential trial-and-error | Pass | `screenshots/06-template-guidance.png` |
| Starter action | A starter fills the first reusable rule directly from the seeded draft surface | Pass | `screenshots/07-template-starter-applied.png` |
| Final save | The operator can save the promoted draft successfully without leaving the workspace | Pass | `screenshots/08-template-saved.png` |
| Browser health | No browser errors during the pass; only expected Vite/React development console noise | Pass | `09-browser-console.txt`, `10-browser-errors.txt` |

## Notes

- The promoted draft now keeps the source-job lineage while explicitly telling the operator that Spartan did **not** infer reusable selectors from the plain scrape.
- The new starter actions make the first reusable rule authorable directly from the seeded draft without hiding the Visual Builder path.
- The inline blocker list removes the prior one-error-at-a-time save loop; the operator sees the remaining save requirements together before clicking anything.
