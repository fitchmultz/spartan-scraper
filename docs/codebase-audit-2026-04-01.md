# Codebase Audit — 2026-04-01

This audit scanned the repository source surfaces (`cmd`, `internal`, `web/src`, `scripts`, `api`, and supporting docs/contracts) with repository-wide search/metrics passes plus targeted file reads on the highest-risk hotspots.

## Executive Summary

No direct SQL injection, command injection, secrets-in-repo, or path-traversal critical was found in the current code paths I inspected. The main risks are correctness regressions in automation, backend fan-out/backpressure problems, and oversized frontend state/components that are already slowing safe iteration.

Top 10 priority findings:

1. 🟠 Export schedule tag filters are wired through API/Web/CLI but can never match jobs.
2. 🟠 Job event publication holds the subscriber lock while doing synchronous webhook/export side effects.
3. 🟠 Async webhook delivery spawns an unbounded goroutine per dispatch before concurrency control kicks in.
4. 🟡 Playwright availability checks can leak startup work/processes after timeout.
5. 🟡 Job deletion removes the database row before artifact cleanup, leaving orphaned files on failure.
6. 🟡 Analytics aggregation holds the collector mutex during store I/O and daily rollups.
7. 🟠 `useFormState` is a 443-line god hook with 40+ fields and setters spanning too many concerns.
8. 🟡 `TransformPreview` has out-of-order async validation races with no cancellation/sequence guard.
9. 🟡 `DeviceSelector` custom editing schedules stale callbacks, making parent state lag behind edits.
10. 🟡 `useBatches` allows overlapping refresh/poll/page requests to overwrite newer UI state.

## Metrics

- Source files scanned: **1,032** code files (`.go`, `.ts`, `.tsx`, `.js`, `.mjs`, `.sh`)
- Production files >300 lines (excluding generated/test files): **161**
- Production functions/components >50 lines (heuristic, excluding generated/test files): **411**
- Inline `style={{...}}` blocks in `web/src`: **861** across **59** files
- Security findings by type:
  - Injection/path traversal/secrets criticals: **0** found
  - Availability / self-DoS risks: **2** high-confidence findings
  - Permission / local restore hardening gaps: **0** high-confidence criticals found

## Top 10 Largest Production Files

(Generated files and tests excluded for maintainability focus.)

| Lines | File |
| ---: | --- |
| 1021 | `tools/pi-bridge/src/sdk-backend.ts` |
| 903 | `web/src/lib/diff-utils.ts` |
| 732 | `web/src/components/DeviceSelector.tsx` |
| 719 | `web/src/components/VisualSelectorBuilder.tsx` |
| 708 | `internal/api/diagnostic_status.go` |
| 702 | `web/src/components/render-profiles/RenderProfileForm.tsx` |
| 683 | `web/src/hooks/useBatches.ts` |
| 678 | `web/src/components/AuthConfig.tsx` |
| 660 | `web/src/components/WatchDetail.tsx` |
| 660 | `internal/fetch/playwright_fetcher.go` |

## Top 10 Most Complex Functions

(Production Go functions measured with `gocyclo`; TS/TSX complexity hotspots are covered separately in findings.)

| Cyclomatic | Function |
| ---: | --- |
| 84 | `internal/fetch/http_fetcher.go:57` — `(*HTTPFetcher).Fetch` |
| 77 | `internal/fetch/playwright_fetcher.go:203` — `(*PlaywrightFetcher).fetchOnce` |
| 72 | `internal/ui/tui/update.go:12` — `(appModel).Update` |
| 67 | `internal/fetch/chromedp_fetcher.go:90` — `(*ChromedpFetcher).doFetch` |
| 49 | `internal/cli/manage/auth.go:19` — `RunAuth` |
| 44 | `internal/crawl/orchestrator.go:47` — `Run` |
| 43 | `internal/fetch/form_detect_classify.go:241` — `(*FormDetector).classifyFieldType` |
| 42 | `internal/watch/watch.go:77` — `(*Watcher).Check` |
| 36 | `internal/cli/manage/schedule.go:21` — `RunSchedule` |
| 34 | `internal/crawl/worker.go:40` — `processPage` |

## Full Findings

### Architecture & Design

🟠 **[God Hook / Missing Separation] `web/src/hooks/useFormState.ts:14-125,192-415`**
- Violation:
  ```ts
  export interface FormState {
    headless: boolean;
    usePlaywright: boolean;
    ...
    interceptMaxEntries: number;
  }

  export function useFormState(): FormController {
    const [state, setState] = useState<FormState>(INITIAL_STATE);
    const setHeadless = useCallback(...)
    const setUsePlaywright = useCallback(...)
    ...
  }
  ```
- Problem: one hook owns browser runtime, auth, proxy, login, AI extract, crawl, screenshot, and interception concerns behind dozens of nearly identical setters.
- Impact: every job-creation change fans out through one file, one state shape, and one rerender path; regressions become harder to localize.
- Blast Radius: `web/src/App.tsx` and all job-creation surfaces that consume the shared form controller.
- Fix: split into domain hooks/reducers (`useRuntimeFormState`, `useAuthFormState`, `useAIExtractState`, `useInterceptState`) and compose them behind a thinner controller.

🟡 **[Presentation Logic Sprawl] `web/src/components/Skeleton.tsx:37-58,75-104` and `web/src/components/WatchDetail.tsx` / peers**
- Violation:
  ```tsx
  <div
    style={{
      padding: "12px 16px",
      borderRadius: "14px",
      background: "var(--bg-alt)",
      border: "1px solid var(--stroke)",
    }}
  >
  ```
- Problem: the web app currently has **861** inline style blocks across **59** files. Styling, layout, and component logic are tightly mixed.
- Impact: theming, visual consistency, and future layout changes require editing many component files instead of shared tokens/classes.
- Blast Radius: especially watch/export/auth/result surfaces, where several files have 20-80 inline style blocks each.
- Fix: move repeated layout/color primitives into shared CSS classes or tokenized component wrappers, starting with watch/export/auth surfaces.

### Complexity & Cognitive Load

🟡 **[Large Multi-Responsibility Hook] `web/src/hooks/useBatches.ts:495-663`**
- Violation:
  ```ts
  const refreshBatches = useCallback(async (nextOffset = offset) => {
    setLoading(true);
    ...
    setBatches(nextBatches);
    setLimit(resolvedLimit);
    setOffset(resolvedOffset);
    setTotal(resolvedTotal);
  }, [getBatchList, offset]);

  useEffect(() => {
    if (hasProcessing) {
      intervalRef.current = setInterval(() => {
        void refreshBatches();
      }, POLL_INTERVAL_MS);
    }
  }, [hasProcessing, refreshBatches]);
  ```
- Problem: list loading, pagination, polling, submit flows, localStorage persistence, and detail refresh all live in one hook with no request sequencing.
- Impact: overlapping poll/manual/page requests can let older responses overwrite newer state.
- Blast Radius: `web/src/components/batches/BatchContainer.tsx` and all batch list/detail UX.
- Fix: separate list-query state from mutation state and add request IDs / `AbortController` so only the latest response commits.

### Code Quality

🟡 **[Configuration/Docs Drift] `scripts/stress_test.sh:79-84`**
- Violation:
  ```bash
  Prerequisites:
    - go (1.25+)
  ```
- Problem: the script help still advertises Go 1.25+, while the repo pins Go 1.26.1 in `.tool-versions` and `go.mod` tooling guidance.
- Impact: clean-machine setup can drift away from the real reproducible contract.
- Blast Radius: contributors and operators using `scripts/stress_test.sh` as authoritative setup guidance.
- Fix: derive displayed tool versions from `.tool-versions` or update the help text in the same change whenever the toolchain moves.

### Security

No direct critical injection, hardcoded-secret, or path-traversal vulnerability stood out in the audited paths. In particular:
- webhook SSRF validation and IP pinning in `internal/webhook/delivery_target.go` are strong,
- restore path traversal defenses in `internal/cli/manage/restore.go` are present,
- exported diff HTML is escaped before reaching `dangerouslySetInnerHTML`.

The highest-confidence security-adjacent issues are availability/self-DoS risks:

🟠 **[Availability / Self-DoS] `internal/webhook/dispatcher.go:242-265`**
- Violation:
  ```go
  func (d *Dispatcher) Dispatch(ctx context.Context, url string, payload Payload, secret string) {
      go func() {
          _ = d.Deliver(ctx, url, payload, secret)
      }()
  }
  ```
- Problem: concurrency limits are enforced inside delivery, but each async dispatch still creates a goroutine first.
- Impact: large crawls or many watch/export events can accumulate parked goroutines and memory pressure before semaphore admission.
- Blast Radius: crawl webhooks, watch notifications, export notifications, and any future async webhook sender.
- Fix: replace per-call goroutine spawning with a bounded worker queue or dispatcher-owned worker pool.

### Error Handling

🟡 **[Out-of-Order Async State] `web/src/components/TransformPreview.tsx:170-197`**
- Violation:
  ```tsx
  useEffect(() => {
    const timer = setTimeout(async () => {
      const result = await validateTransform(expression, language);
      setValidationResult({ valid: result.valid, message: result.message });
    }, 500);
    return () => clearTimeout(timer);
  }, [expression, language]);
  ```
- Problem: the debounce timer is cleaned up, but in-flight requests are not cancelled or versioned. Slower older validations can overwrite newer results.
- Impact: operators can see stale “valid/invalid” status while typing quickly or toggling language.
- Blast Radius: results-route transform authoring and any export flow that trusts this validation state.
- Fix: add `AbortController` or request sequence IDs so only the latest validation result commits.

🟡 **[Stale Callback State] `web/src/components/DeviceSelector.tsx:521-535,613-707`**
- Violation:
  ```tsx
  const updateCustomDevice = () => {
    if (!showCustom) return;
    onChange({ name: customName, viewportWidth: customWidth, ... });
  };

  onChange={(e) => {
    setCustomWidth(Number(e.target.value));
    setTimeout(updateCustomDevice, 0);
  }}
  ```
- Problem: each `setTimeout(updateCustomDevice, 0)` captures the old render’s state, so upstream `onChange` can lag one edit behind.
- Impact: parent forms can submit stale device dimensions/UA/touch flags.
- Blast Radius: scrape, crawl, research, and batch flows that use custom device emulation.
- Fix: compute the next device from the event value immediately, or move synchronization into a `useEffect` keyed on the custom state fields.

### Resource Management

🟡 **[Leaked Startup Work] `internal/fetch/fetcher.go:153-170`**
- Violation:
  ```go
  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  ...
  go func() {
      pw, err := playwrightRun()
      resultChan <- result{pw: pw, err: err}
  }()
  select {
  case <-ctx.Done():
      return fmt.Errorf("%w: timeout ...", ErrPlaywrightNotReady)
  ```
- Problem: timeout returns without cancelling/joining the Playwright startup goroutine or force-stopping a process that wins the race later.
- Impact: repeated diagnostics/preflight checks can leak work and leave browser startup artifacts behind.
- Blast Radius: browser diagnostics, startup health checks, and any flow that probes Playwright readiness.
- Fix: make startup itself context-aware or add a cleanup path that drains/stops late completions.

🟡 **[Delete Order Can Orphan Artifacts] `internal/store/store_jobs.go:420-434`**
- Violation:
  ```go
  if err := s.Delete(ctx, id); err != nil {
      return err
  }
  ...
  return os.RemoveAll(cleanPath)
  ```
- Problem: the DB row is deleted before filesystem cleanup. If `RemoveAll` fails, the system loses the index entry but keeps artifacts on disk.
- Impact: orphaned artifacts, storage drift, and weaker retention/reporting guarantees.
- Blast Radius: API/CLI delete paths and retention/manual cleanup workflows.
- Fix: delete artifacts first, then remove the row, or use a transactional tombstone + cleanup worker.

### Performance

🟠 **[Locked Fan-Out + Synchronous Side Effects] `internal/jobs/manager.go:346-365`**
- Violation:
  ```go
  m.subscribersMu.RLock()
  defer m.subscribersMu.RUnlock()
  ...
  m.dispatchWebhook(event, cfg)
  ...
  m.exportTrigger.HandleJobEvent(event)
  ```
- Problem: event publication keeps the subscriber lock held while doing synchronous webhook delivery and export-trigger work.
- Impact: one slow webhook or export match blocks all concurrent subscribe/unsubscribe work and lengthens the hot path for every job state transition.
- Blast Radius: all job lifecycle updates from `manager.go`, `job_run.go`, and chain execution.
- Fix: snapshot subscribers under lock, unlock immediately, then run side effects outside the lock with bounded async fan-out.

🟡 **[Mutex Held Across Store I/O] `internal/analytics/collector.go:129-160`**
- Violation:
  ```go
  c.mu.Lock()
  defer c.mu.Unlock()
  ...
  if err := c.store.RecordHourlyMetrics(ctx, c.hourlyMetrics); err != nil { ... }
  for _, hm := range c.hostMetrics {
      if err := c.store.RecordHostMetrics(ctx, hm); err != nil { ... }
  }
  ```
- Problem: analytics persistence and daily rollups happen while the collector mutex is held.
- Impact: metrics updates serialize on DB latency and can stall producer paths during slow storage periods.
- Blast Radius: analytics rollups and any code path recording metrics into the collector.
- Fix: copy the snapshot under lock, release the mutex, then persist/store-roll up outside the critical section.

### Testing

🟠 **[Broken Behavior With No Regression Test] `internal/scheduler/export_trigger.go:229-231` + `internal/scheduler/export_trigger_test.go:43-220`**
- Violation:
  ```go
  // Check tags (all specified tags must be present)
  if len(filters.Tags) > 0 {
      return false
  }
  ```
- Problem: export-schedule tags are surfaced in API/Web/CLI (`web/src/lib/export-schedule-utils.ts`, `web/src/components/export-schedules/*`, `internal/cli/manage/export_schedule.go`) but the matcher rejects every tagged schedule.
- Impact: a saved tagged export schedule silently never triggers.
- Blast Radius: export schedule creation, editing, persistence, and automated export execution across API/Web/CLI.
- Fix: implement actual tag matching against job metadata or remove the tag filter from every surface until the model can support it. Add regression coverage for matching and non-matching tag cases.

🟡 **[Coverage Misses Async/State Races] `web/src/components/TransformPreview.test.tsx:14-146`, `web/src/components/DeviceSelector.test.tsx:13-49`, `web/src/hooks/useBatches.test.ts:232-475`**
- Violation:
  ```ts
  // TransformPreview tests cover AI/manual availability.
  // DeviceSelector tests cover label/chip/icon behavior.
  // useBatches tests cover hydrate/submit/detail happy paths.
  ```
- Problem: the current tests do not exercise out-of-order async responses, custom device editing, or overlapping refresh/poll/page changes.
- Impact: the current correctness bugs can ship without failing CI.
- Blast Radius: transform authoring, custom device emulation, and batch monitoring.
- Fix: add race-oriented tests with deferred promises/fake timers and explicit stale-response assertions.

🟡 **[Dispatcher Tests Miss Goroutine Pressure] `internal/webhook/dispatcher_concurrency_test.go:29-157`**
- Violation:
  ```go
  // Launch 20 concurrent dispatches
  for i := 0; i < 20; i++ {
      d.Dispatch(context.Background(), server.URL, payload, "")
  }
  ```
- Problem: current tests assert active HTTP concurrency and dropped counts, but not parked goroutine growth before semaphore acquisition.
- Impact: the availability risk can remain invisible while tests still pass.
- Blast Radius: webhook delivery under bursty production workloads.
- Fix: add tests that inspect goroutine growth or, preferably, refactor to a bounded queue whose depth is directly testable.

### Observability

🟡 **[Backpressure Is Hard To Attribute] `internal/jobs/manager.go:346-365`**
- Violation:
  ```go
  m.dispatchWebhook(event, cfg)
  m.exportTrigger.HandleJobEvent(event)
  ```
- Problem: queue/event slowdowns are coupled to webhook/export side effects but the publication path does not emit dedicated queue-latency/backpressure metrics.
- Impact: operators see slower job progress without an immediate “events blocked on webhook/export side effects” signal.
- Blast Radius: all job lifecycle observability.
- Fix: expose event publish latency / queued side-effect metrics when decoupling this path.

### Configuration

No severe startup-validation gap was found in the main app config path; toolchain pinning and local CI entrypoints are strong. The most visible config drift is the `scripts/stress_test.sh` prerequisite text noted above.

## Remediation Roadmap

### 🔥 Quick wins

1. **Repair export schedule tag filters end-to-end** — implement real matching or remove the filter from every surface until supported; add regression tests.
2. **Fix stale frontend async/state bugs** — sequence/cancel `TransformPreview`, remove stale `setTimeout(updateCustomDevice, 0)` callbacks, add race tests.
3. **Reorder job artifact deletion** — stop deleting the DB row before artifact cleanup succeeds.

### 📋 Short-term

1. **Decouple job-event publication from webhook/export side effects** — snapshot subscribers under lock, then fan out outside the lock with bounded async work.
2. **Bound webhook dispatch fan-out** — replace goroutine-per-dispatch with a queue/worker model and add queue-depth observability.
3. **Make Playwright readiness checks cancellable/cleaned up** — eliminate timeout leaks.
4. **Take store I/O out of analytics collector critical sections** — persist snapshots after unlocking.

### 🏗️ Long-term

1. **Split `useFormState` into feature-scoped state modules** with a narrower orchestration layer.
2. **Break `useBatches` into query/mutation/polling primitives** with explicit request ownership.
3. **Reduce web inline-style sprawl** by migrating high-churn feature surfaces to shared CSS classes/tokens.
4. **Shrink the biggest modules** (`sdk-backend.ts`, `diff-utils.ts`, `DeviceSelector.tsx`, `diagnostic_status.go`, `playwright_fetcher.go`) into narrower files with clearer ownership.

## Categories Checked

- Architecture & design: checked
- Complexity & cognitive load: checked
- Code quality: checked
- Security: checked
- Error handling: checked
- Resource management: checked
- Performance: checked
- Testing: checked
- Observability: checked
- Configuration: checked
- Go-specific checks: checked
- TypeScript/React-specific checks: checked
