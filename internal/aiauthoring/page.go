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
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

type pageContext struct {
	URL               string
	HTML              string
	Images            []extract.AIImageInput
	VisualContextUsed bool
}

func (s *Service) resolvePageContext(ctx context.Context, pageURL string, html string, headless bool, usePlaywright bool, visual bool) (pageContext, error) {
	if strings.TrimSpace(html) != "" {
		return pageContext{
			URL:  strings.TrimSpace(pageURL),
			HTML: html,
		}, nil
	}

	result, err := s.fetchPage(ctx, pageURL, headless, usePlaywright, visual)
	if err != nil {
		return pageContext{}, err
	}

	ctxResult := pageContext{
		URL:  strings.TrimSpace(pageURL),
		HTML: result.HTML,
	}
	if visual {
		image, err := loadScreenshotImage(result.ScreenshotPath)
		if err != nil {
			return pageContext{}, err
		}
		if image != nil {
			ctxResult.Images = []extract.AIImageInput{*image}
			ctxResult.VisualContextUsed = true
		}
	}
	return ctxResult, nil
}

func (s *Service) fetchPage(ctx context.Context, pageURL string, headless bool, usePlaywright bool, visual bool) (fetch.Result, error) {
	if err := webhook.ValidateURL(pageURL, s.allowInternal); err != nil {
		return fetch.Result{}, err
	}

	fetcher := fetch.NewFetcher(s.cfg.DataDir)
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
