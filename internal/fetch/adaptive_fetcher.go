// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"context"
	"log/slog"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

type AdaptiveFetcher struct {
	store *RenderProfileStore
	http  *HTTPFetcher
	cdp   *ChromedpFetcher
	pw    *PlaywrightFetcher
}

func NewAdaptiveFetcher() *AdaptiveFetcher {
	// Note: We don't have access to dataDir here easily unless passed in.
	// But the fetcher instance itself is transient-ish or stateless.
	// Actually, request has DataDir. We can load store on demand or cache it.
	// For efficiency, we should cache stores by DataDir or assume one global DataDir for the process?
	// The Request struct has DataDir. We will maintain a map of stores or just open one per request (store handles caching ideally).
	// RenderProfileStore is designed to be lightweight/cached.
	// For simplicity in this iteration: We'll create store on the fly or manage it inside Fetch if needed.
	// BUT, strict performance says don't re-read file every time.
	// We will use a shared global cache or similar if needed, but for now NewRenderProfileStore is fast enough if called once per batch.
	// Wait, NewFetcher is called once per job? No, scrape.Run calls it.
	// We'll trust RenderProfileStore's optimization (stat check).

	return &AdaptiveFetcher{
		http: &HTTPFetcher{},
		cdp:  &ChromedpFetcher{},
		pw:   &PlaywrightFetcher{},
	}
}

func (f *AdaptiveFetcher) Fetch(ctx context.Context, req Request) (Result, error) {
	slog.Debug("adaptive fetch start", "url", apperrors.SanitizeURL(req.URL))
	// 1. Load Profile
	store := NewRenderProfileStore(req.DataDir)
	profPtr, found, err := store.MatchURL(req.URL)
	if err != nil {
		slog.Error("failed to match URL in render profile store", "url", apperrors.SanitizeURL(req.URL), "error", err)
		return Result{}, err
	}
	prof := defaultRenderProfile()
	if found {
		slog.Debug("matched render profile", "url", apperrors.SanitizeURL(req.URL), "profile", profPtr.Name)
		prof = mergeRenderProfile(prof, profPtr)
	}

	// 2. Decision: Forced Headless?
	if prof.NeverHeadless {
		slog.Debug("profile forces HTTP (NeverHeadless)", "url", apperrors.SanitizeURL(req.URL))
		return f.http.Fetch(ctx, req)
	}
	if req.Headless || prof.PreferHeadless || prof.AssumeJSHeavy || prof.ForceEngine != "" {
		slog.Debug("profile or request forces headless", "url", apperrors.SanitizeURL(req.URL), "headless", req.Headless, "preferHeadless", prof.PreferHeadless, "assumeJSHeavy", prof.AssumeJSHeavy, "forceEngine", prof.ForceEngine)
		return f.fetchHeadless(ctx, req, prof)
	}

	// 3. HTTP Probe
	slog.Debug("probing with HTTP", "url", apperrors.SanitizeURL(req.URL))
	probeReq := req
	// Reduce timeout for probe if not specified, to save time on failure?
	// Actually, stick to configured timeout to avoid premature giving up.
	res, err := f.http.Fetch(ctx, probeReq)
	if err != nil {
		slog.Warn("HTTP probe failed", "url", apperrors.SanitizeURL(req.URL), "error", err)
		// If HTTP failed, depends on error.
		// If timeout, maybe headless won't help?
		// If network error, headless might not help.
		return Result{}, err
	}

	// 4. Analyze
	// Check status codes that suggest blocking or JS requirement
	if res.Status == 403 || res.Status == 401 || res.Status == 429 {
		slog.Debug("HTTP probe returned status suggesting bot detection or JS challenge", "url", apperrors.SanitizeURL(req.URL), "status", res.Status)
		// Potential bot detection. Headless might help if it's JS challenge?
		// Or maybe it won't. But worth a try if we are adaptive.
		// But 429 is rate limit.
		if res.Status != 429 {
			slog.Info("retrying with headless due to HTTP status", "url", apperrors.SanitizeURL(req.URL), "status", res.Status)
			return f.fetchHeadless(ctx, req, prof)
		}
	}

	js := DetectJSHeaviness(res.HTML)
	threshold := 0.5
	if prof.JSHeavyThreshold > 0 {
		threshold = prof.JSHeavyThreshold
	}

	if IsJSHeavy(js, threshold) {
		slog.Info("retrying with headless due to JS heaviness", "url", apperrors.SanitizeURL(req.URL), "jsScore", js, "threshold", threshold)
		return f.fetchHeadless(ctx, req, prof)
	}

	// 5. Return HTTP result if satisfied
	slog.Debug("satisfied with HTTP result", "url", apperrors.SanitizeURL(req.URL))
	return res, nil
}

func (f *AdaptiveFetcher) fetchHeadless(ctx context.Context, req Request, prof RenderProfile) (Result, error) {
	// Engine selection
	engine := RenderEngineChromedp
	if req.UsePlaywright {
		engine = RenderEnginePlaywright
	}
	if prof.ForceEngine == RenderEngineChromedp {
		engine = RenderEngineChromedp
	} else if prof.ForceEngine == RenderEnginePlaywright {
		engine = RenderEnginePlaywright
	}

	slog.Debug("fetching headless", "url", apperrors.SanitizeURL(req.URL), "engine", engine)

	// Primary attempt
	var res Result
	var err error

	if engine == RenderEnginePlaywright {
		res, err = f.pw.Fetch(ctx, req, prof)
	} else {
		res, err = f.cdp.Fetch(ctx, req, prof)
	}

	if err == nil {
		slog.Debug("headless fetch success", "url", apperrors.SanitizeURL(req.URL), "engine", engine)
		return res, nil
	}

	slog.Error("headless fetch failed", "url", apperrors.SanitizeURL(req.URL), "engine", engine, "error", err)

	// Fallback logic?
	// If Chromedp failed and UsePlaywright is explicitly allowed but not forced?
	// For now, simple: if error, error.
	return Result{}, err
}

func (f *AdaptiveFetcher) Close() error {
	// Only Playwright fetcher needs cleanup for its singleton browser
	if f.pw != nil {
		return f.pw.Close()
	}
	return nil
}

func defaultRenderProfile() RenderProfile {
	return RenderProfile{
		Wait: RenderWaitPolicy{
			Mode: RenderWaitModeDOMReady,
		},
		Timeouts: RenderTimeoutPolicy{
			MaxRenderMs:  30000,
			ScriptEvalMs: 5000,
			NavigationMs: 30000,
		},
	}
}

func mergeRenderProfile(base RenderProfile, override *RenderProfile) RenderProfile {
	// Simple overlay
	out := base
	out.Name = override.Name
	if len(override.HostPatterns) > 0 {
		out.HostPatterns = override.HostPatterns
	}
	if override.ForceEngine != "" {
		out.ForceEngine = override.ForceEngine
	}
	if override.PreferHeadless {
		out.PreferHeadless = true
	}
	if override.AssumeJSHeavy {
		out.AssumeJSHeavy = true
	}
	if override.NeverHeadless {
		out.NeverHeadless = true
	}
	if override.JSHeavyThreshold > 0 {
		out.JSHeavyThreshold = override.JSHeavyThreshold
	}
	// Policies: if override has them set, take them? Or merge deep?
	// Replacing full structs for block/wait/timeout is safer/simpler.
	if len(override.Block.ResourceTypes) > 0 || len(override.Block.URLPatterns) > 0 {
		out.Block = override.Block
	}
	if override.Wait.Mode != "" {
		out.Wait = override.Wait
	}
	if override.Timeouts.MaxRenderMs > 0 {
		out.Timeouts = override.Timeouts
	}
	return out
}
