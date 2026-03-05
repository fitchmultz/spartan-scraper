// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"context"
	"log/slog"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/captcha"
)

type AdaptiveFetcher struct {
	store           *RenderProfileStore
	http            *HTTPFetcher
	cdp             *ChromedpFetcher
	pw              *PlaywrightFetcher
	metricsCallback MetricsCallback
	proxyPool       *ProxyPool
}

func NewAdaptiveFetcher(dataDir string) *AdaptiveFetcher {
	return &AdaptiveFetcher{
		store: NewRenderProfileStore(dataDir),
		http:  &HTTPFetcher{},
		cdp:   &ChromedpFetcher{},
		pw:    &PlaywrightFetcher{},
	}
}

// SetProxyPool sets the proxy pool for all underlying fetchers.
func (f *AdaptiveFetcher) SetProxyPool(pool *ProxyPool) {
	f.proxyPool = pool
	f.http.SetProxyPool(pool)
	f.cdp.SetProxyPool(pool)
	f.pw.SetProxyPool(pool)
}

func (f *AdaptiveFetcher) Fetch(ctx context.Context, req Request) (Result, error) {
	slog.Debug("adaptive fetch start", "url", apperrors.SanitizeURL(req.URL))

	// Start metrics tracking
	if f.metricsCallback != nil {
		f.metricsCallback(0, true, "http", req.URL) // Start marker
	}
	start := time.Now()
	var result Result
	var err error
	var fetcherType string
	defer func() {
		if f.metricsCallback != nil {
			duration := time.Since(start)
			f.metricsCallback(duration, err == nil, fetcherType, req.URL)
		}
	}()

	// 1. Reload profiles if file changed (cache invalidation)
	if err = f.store.ReloadIfChanged(); err != nil {
		slog.Error("failed to reload render profile store", "url", apperrors.SanitizeURL(req.URL), "error", err)
		return Result{}, err
	}

	// 2. Match URL against profiles
	profPtr, found, err := f.store.MatchURL(req.URL)
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
		fetcherType = "http"
		result, err = f.http.Fetch(ctx, req)
		return result, err
	}
	if req.Headless || prof.PreferHeadless || prof.AssumeJSHeavy || prof.ForceEngine != "" {
		slog.Debug("profile or request forces headless", "url", apperrors.SanitizeURL(req.URL), "headless", req.Headless, "preferHeadless", prof.PreferHeadless, "assumeJSHeavy", prof.AssumeJSHeavy, "forceEngine", prof.ForceEngine)
		fetcherType, result, err = f.fetchHeadlessWithType(ctx, req, prof)
		return result, err
	}

	// 3. HTTP Probe
	slog.Debug("probing with HTTP", "url", apperrors.SanitizeURL(req.URL))
	probeReq := req
	// Reduce timeout for probe if not specified, to save time on failure?
	// Actually, stick to configured timeout to avoid premature giving up.
	res, fetchErr := f.http.Fetch(ctx, probeReq)
	if fetchErr != nil {
		slog.Warn("HTTP probe failed", "url", apperrors.SanitizeURL(req.URL), "error", fetchErr)
		// If HTTP failed, depends on error.
		// If timeout, maybe headless won't help?
		// If network error, headless might not help.
		err = fetchErr
		fetcherType = "http"
		return Result{}, err
	}

	// 4. Analyze
	// Check status codes that suggest blocking or JS requirement
	if res.Status == 403 || res.Status == 401 || res.Status == 429 {
		slog.Debug("HTTP probe returned status suggesting bot detection or JS challenge", "url", apperrors.SanitizeURL(req.URL), "status", res.Status)

		// Check for CAPTCHA if detection is enabled in profile
		if prof.CaptchaConfig != nil && prof.CaptchaConfig.Enabled {
			detector := captcha.NewDetector()
			detection, detectErr := detector.Detect(res.HTML, req.URL)
			if detectErr == nil && detection != nil {
				slog.Info("CAPTCHA detected", "url", apperrors.SanitizeURL(req.URL), "type", detection.Type, "score", detection.Score)

				if prof.CaptchaConfig.AutoSolve {
					solution, solveErr := f.solveCaptcha(ctx, detection, prof.CaptchaConfig)
					if solveErr == nil && solution != "" {
						slog.Info("CAPTCHA solved, retrying request", "url", apperrors.SanitizeURL(req.URL))
						// Retry with the solution (would need to inject it)
						// For now, escalate to headless which can use the solution
					}
				}
			}
		}

		// Potential bot detection. Headless might help if it's JS challenge?
		// Or maybe it won't. But worth a try if we are adaptive.
		// But 429 is rate limit.
		if res.Status != 429 {
			slog.Info("retrying with headless due to HTTP status", "url", apperrors.SanitizeURL(req.URL), "status", res.Status)
			fetcherType, result, err = f.fetchHeadlessWithType(ctx, req, prof)
			return result, err
		}
	}

	// Also check for CAPTCHA in successful responses (some sites return 200 with CAPTCHA)
	if prof.CaptchaConfig != nil && prof.CaptchaConfig.Enabled {
		detector := captcha.NewDetector()
		detection, detectErr := detector.Detect(res.HTML, req.URL)
		if detectErr == nil && detection != nil {
			slog.Info("CAPTCHA detected in response", "url", apperrors.SanitizeURL(req.URL), "type", detection.Type, "score", detection.Score)

			if prof.CaptchaConfig.AutoSolve {
				solution, solveErr := f.solveCaptcha(ctx, detection, prof.CaptchaConfig)
				if solveErr == nil && solution != "" {
					slog.Info("CAPTCHA solved", "url", apperrors.SanitizeURL(req.URL))
					// Note: The solution would need to be injected into the page
					// For now, we escalate to headless which can handle this
				}
			}

			// Escalate to headless to handle the CAPTCHA
			slog.Info("escalating to headless due to CAPTCHA", "url", apperrors.SanitizeURL(req.URL), "type", detection.Type)
			fetcherType, result, err = f.fetchHeadlessWithType(ctx, req, prof)
			return result, err
		}
	}

	js := DetectJSHeaviness(res.HTML)
	threshold := 0.5
	if prof.JSHeavyThreshold > 0 {
		threshold = prof.JSHeavyThreshold
	}

	if IsJSHeavy(js, threshold) {
		slog.Info("retrying with headless due to JS heaviness", "url", apperrors.SanitizeURL(req.URL), "jsScore", js, "threshold", threshold)
		fetcherType, result, err = f.fetchHeadlessWithType(ctx, req, prof)
		return result, err
	}

	// 5. Return HTTP result if satisfied
	slog.Debug("satisfied with HTTP result", "url", apperrors.SanitizeURL(req.URL))
	fetcherType = "http"
	result = res
	err = nil
	return res, nil
}

// SetMetricsCallback sets the callback function for metrics collection
func (f *AdaptiveFetcher) SetMetricsCallback(cb MetricsCallback) {
	f.metricsCallback = cb
}

// fetchHeadless performs headless fetching and returns the result
func (f *AdaptiveFetcher) fetchHeadless(ctx context.Context, req Request, prof RenderProfile) (Result, error) {
	_, res, err := f.fetchHeadlessWithType(ctx, req, prof)
	return res, err
}

// fetchHeadlessWithType performs headless fetching and returns the fetcher type along with the result
func (f *AdaptiveFetcher) fetchHeadlessWithType(ctx context.Context, req Request, prof RenderProfile) (string, Result, error) {
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
	var fetcherType string

	if engine == RenderEnginePlaywright {
		fetcherType = "playwright"
		res, err = f.pw.Fetch(ctx, req, prof)
	} else {
		fetcherType = "chromedp"
		res, err = f.cdp.Fetch(ctx, req, prof)
	}

	if err == nil {
		slog.Debug("headless fetch success", "url", apperrors.SanitizeURL(req.URL), "engine", engine)
		return fetcherType, res, nil
	}

	slog.Error("headless fetch failed", "url", apperrors.SanitizeURL(req.URL), "engine", engine, "error", err)

	// Fallback logic?
	// If Chromedp failed and UsePlaywright is explicitly allowed but not forced?
	// For now, simple: if error, error.
	return fetcherType, Result{}, err
}

func (f *AdaptiveFetcher) Close() error {
	// Only Playwright fetcher needs cleanup for its singleton browser
	if f.pw != nil {
		return f.pw.Close()
	}
	return nil
}

// solveCaptcha attempts to solve a CAPTCHA using the configured service.
func (f *AdaptiveFetcher) solveCaptcha(ctx context.Context, detection *captcha.CaptchaDetection, config *captcha.CaptchaConfig) (string, error) {
	solver, err := captcha.SolverFactory(*config)
	if err != nil {
		return "", err
	}

	solution, err := solver.Solve(ctx, *detection, detection.PageURL)
	if err != nil {
		slog.Warn("CAPTCHA solving failed", "type", detection.Type, "error", err)
		return "", err
	}

	return solution, nil
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
	if override.RateLimitQPS > 0 {
		out.RateLimitQPS = override.RateLimitQPS
	}
	if override.RateLimitBurst > 0 {
		out.RateLimitBurst = override.RateLimitBurst
	}
	return out
}
