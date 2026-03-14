package aiauthoring

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

type pageContext struct {
	URL               string
	HTML              string
	Images            []extract.AIImageInput
	VisualContextUsed bool
	FetchStatus       int
	FetchEngine       string
	JSHeaviness       fetch.JSHeaviness
}

func (s *Service) resolvePageContext(ctx context.Context, pageURL string, html string, directImages []extract.AIImageInput, headless bool, usePlaywright bool, visual bool) (pageContext, error) {
	if strings.TrimSpace(html) != "" {
		trimmedURL := strings.TrimSpace(pageURL)
		images := appendAIImages(directImages)
		return pageContext{
			URL:               trimmedURL,
			HTML:              html,
			Images:            images,
			VisualContextUsed: len(images) > 0,
			JSHeaviness:       fetch.DetectJSHeaviness(html),
		}, nil
	}

	result, err := s.fetchPage(ctx, pageURL, headless, usePlaywright, visual)
	if err != nil {
		return pageContext{}, err
	}

	ctxResult := pageContext{
		URL:         strings.TrimSpace(pageURL),
		HTML:        result.HTML,
		Images:      appendAIImages(directImages),
		FetchStatus: result.Status,
		FetchEngine: string(result.Engine),
		JSHeaviness: fetch.DetectJSHeaviness(result.HTML),
	}
	if visual {
		image, err := loadScreenshotImage(result.ScreenshotPath)
		if err != nil {
			return pageContext{}, err
		}
		if image != nil {
			ctxResult.Images = appendAIImages(ctxResult.Images, []extract.AIImageInput{*image})
		}
	}
	ctxResult.VisualContextUsed = len(ctxResult.Images) > 0
	return ctxResult, nil
}

func (s *Service) fetchPage(ctx context.Context, pageURL string, headless bool, usePlaywright bool, visual bool) (fetch.Result, error) {
	if err := webhook.ValidateURL(pageURL, s.allowInternal); err != nil {
		return fetch.Result{}, err
	}

	fetcher := fetch.NewAdaptiveFetcher(s.cfg.DataDir)
	defer func() {
		_ = fetcher.Close()
	}()

	request := fetch.Request{
		URL:           pageURL,
		Method:        http.MethodGet,
		Timeout:       time.Duration(s.cfg.RequestTimeoutSecs) * time.Second,
		UserAgent:     s.cfg.UserAgent,
		Headless:      headless || visual,
		UsePlaywright: usePlaywright,
		DataDir:       s.cfg.DataDir,
	}
	if visual {
		request.Screenshot = &fetch.ScreenshotConfig{
			Enabled:  true,
			FullPage: true,
			Format:   fetch.ScreenshotFormatPNG,
		}
	}

	result, err := fetcher.Fetch(ctx, request)
	if err != nil {
		return fetch.Result{}, apperrors.Wrap(apperrors.KindInternal, "failed to fetch page", err)
	}
	return result, nil
}

func (s *Service) recheckAutomationPage(ctx context.Context, pageURL string, profile *fetch.RenderProfile, script *pipeline.JSTargetScript) (pageContext, error) {
	if err := webhook.ValidateURL(pageURL, s.allowInternal); err != nil {
		return pageContext{}, err
	}

	tmpDir, err := os.MkdirTemp("", "spartan-ai-authoring-*")
	if err != nil {
		return pageContext{}, apperrors.Wrap(apperrors.KindInternal, "failed to create automation recheck workspace", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	if profile != nil {
		if err := fetch.SaveRenderProfilesFile(tmpDir, fetch.RenderProfilesFile{Profiles: []fetch.RenderProfile{*profile}}); err != nil {
			return pageContext{}, err
		}
	}
	if script != nil {
		if err := pipeline.SaveJSRegistry(tmpDir, pipeline.JSRegistry{Scripts: []pipeline.JSTargetScript{*script}}); err != nil {
			return pageContext{}, err
		}
	}

	fetcher := fetch.NewAdaptiveFetcher(tmpDir)
	defer func() {
		_ = fetcher.Close()
	}()

	request := fetch.Request{
		URL:       pageURL,
		Method:    http.MethodGet,
		Timeout:   time.Duration(s.cfg.RequestTimeoutSecs) * time.Second,
		UserAgent: s.cfg.UserAgent,
		DataDir:   tmpDir,
	}
	if script != nil {
		applyScriptToRequest(&request, *script)
	}

	result, err := fetcher.Fetch(ctx, request)
	if err != nil {
		return pageContext{}, apperrors.Wrap(apperrors.KindInternal, "failed to recheck page with current automation config", err)
	}

	resolvedURL := strings.TrimSpace(result.URL)
	if resolvedURL == "" {
		resolvedURL = strings.TrimSpace(pageURL)
	}
	return pageContext{
		URL:         resolvedURL,
		HTML:        result.HTML,
		FetchStatus: result.Status,
		FetchEngine: string(result.Engine),
		JSHeaviness: fetch.DetectJSHeaviness(result.HTML),
	}, nil
}

func applyScriptToRequest(request *fetch.Request, script pipeline.JSTargetScript) {
	if request == nil {
		return
	}
	request.Headless = true
	switch strings.ToLower(strings.TrimSpace(script.Engine)) {
	case pipeline.EnginePlaywright:
		request.UsePlaywright = true
	case pipeline.EngineChromedp:
		request.UsePlaywright = false
	}
	if strings.TrimSpace(script.PreNav) != "" {
		request.PreNavJS = []string{script.PreNav}
	}
	if strings.TrimSpace(script.PostNav) != "" {
		request.PostNavJS = []string{script.PostNav}
	}
	if len(script.Selectors) > 0 {
		request.WaitSelectors = append([]string(nil), script.Selectors...)
	}
}

func validateHTTPURL(raw string) error {
	parsedURL, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return apperrors.Validation("invalid URL format")
	}
	return nil
}

func loadScreenshotImage(path string) (*extract.AIImageInput, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read captured screenshot", err)
	}
	_ = os.Remove(path)
	mimeType := detectScreenshotMimeType(path, data)
	return &extract.AIImageInput{
		Data:     base64.StdEncoding.EncodeToString(data),
		MimeType: mimeType,
	}, nil
}

func detectScreenshotMimeType(path string, data []byte) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".png":
		return "image/png"
	}
	if detected := http.DetectContentType(data); strings.HasPrefix(detected, "image/") {
		return detected
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if ext == "" {
		return "image/png"
	}
	return fmt.Sprintf("image/%s", ext)
}
