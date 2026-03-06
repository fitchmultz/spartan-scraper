# Dogfood Evidence

This directory is the curated UI verification record for Spartan Scraper. It exists to show a reviewer two things quickly:

- issues were found in a real browser, not guessed from static review
- fixes were re-validated with concrete receipts after remediation

## Start Here

- [`2026-03-05-postfix-ui/report.md`](./2026-03-05-postfix-ui/report.md)
  - Best single read for current UI confidence.
  - Shows the post-fix verification pass with linked artifacts and clean-browser receipts.

- [`2026-03-05-focused-ui/report.md`](./2026-03-05-focused-ui/report.md)
  - Best read for understanding what was caught during the focused UI bug bash.
  - Includes concrete repro steps, issue framing, and evidence captured before fixes landed.

## How To Read These Packs

- Start with the postfix report if you want current-state confidence.
- Use the focused report only if you want the before/after trail for important UI regressions.
- Ignore raw snapshots unless you are validating a specific claim; they are stored as supporting evidence, not primary documentation.
