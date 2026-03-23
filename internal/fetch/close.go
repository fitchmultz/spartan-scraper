// Package fetch provides browser/tooling availability checks and fetcher lifecycle helpers.
//
// Purpose:
// - Centralize best-effort fetcher cleanup so callers do not leak repo-started browser automation.
//
// Responsibilities:
// - Detect whether a fetcher exposes a Close method.
// - Invoke Close safely for callers that create short-lived fetchers per request or test.
//
// Scope:
// - Fetcher lifecycle cleanup only; concrete fetch behavior lives in sibling files.
//
// Usage:
// - Call CloseFetcher(fetcher) in scrape/crawl teardown paths after constructing a fetch.Fetcher.
//
// Invariants/Assumptions:
// - Cleanup is best-effort and should be safe to call on nil or non-closable fetchers.
// - Close must not panic when the underlying fetcher has already been cleaned up.
package fetch

type fetcherCloser interface {
	Close() error
}

// CloseFetcher closes fetchers that expose a Close method.
func CloseFetcher(fetcher Fetcher) error {
	if fetcher == nil {
		return nil
	}
	closer, ok := fetcher.(fetcherCloser)
	if !ok {
		return nil
	}
	return closer.Close()
}
