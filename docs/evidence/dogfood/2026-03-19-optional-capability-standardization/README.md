# 2026-03-19 Optional capability standardization

## Goal

Finish the remaining optional-capability standardization pass so intentionally disabled AI helpers stay coherent, self-explanatory, and non-alarming across the Web UI and CLI.

## What changed

- Added a shared `AIUnavailableNotice` presentation and reused it across optional AI surfaces.
- Self-defended the export-shape and export-transform AI modals so they stay explanatory even if opened while AI is unavailable.
- Threaded AI capability state through the job, results, and template assistant rails so disabled AI keeps the rail usable but clearly blocks AI-only actions.
- Added explicit disabled reasons to `AIImageAttachments` so image uploads no longer silently gray out.
- Extended the same self-defending AI-off behavior to render-profile and pipeline-JS AI generator/debugger modals.
- Unified CLI recovery-action rendering so `spartan health`, `spartan proxy-pool status`, and `spartan retention status` all print the same indented bullet format.

## Visual verification

Validated against a fresh local runtime with AI unavailable.

| Surface | Expectation | Result | Evidence |
| --- | --- | --- | --- |
| New job assistant rail | AI preview controls should explain the disabled state and disable AI-only controls without breaking the main job workflow | Pass | `screenshots/01-new-job-ai-unavailable.png` |
| Settings optional AI surfaces | Render profile / pipeline-JS AI entry points should read as optional-off rather than broken | Pass | `screenshots/03-settings-ai-unavailable.png` |

## Automated verification

- `cd web && pnpm exec vitest run src/components/AIExportShapeAssistant.test.tsx src/components/AIExportTransformAssistant.test.tsx src/components/ai-assistant/__tests__/AIAssistantPanel.test.tsx src/components/ai-assistant/__tests__/JobSubmissionAssistantSection.test.tsx src/components/ai-assistant/__tests__/ResultsAssistantSection.test.tsx src/components/export-schedules/ExportScheduleForm.test.tsx src/components/TransformPreview.test.tsx src/components/render-profiles/RenderProfileEditor.test.tsx src/components/pipeline-js/PipelineJSEditor.test.tsx`
- `go test ./internal/cli/...`
- `make ci`

## Notes

- Template preview remains usable when AI is off, while the generate/debug assistant paths now show the same shared unavailable guidance.
- CLI output no longer mixes `Next step:` lines with bullet-style action lists.
