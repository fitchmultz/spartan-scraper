// Package captcha provides CAPTCHA detection and solving service integration.
//
// This file implements CAPTCHA detection logic for HTML and headless-rendered pages.
// It follows the same pattern as form_detect.go in the fetch package.
//
// Detection heuristics:
//   - reCAPTCHA v2: g-recaptcha class, data-sitekey attribute, google.com/recaptcha iframe
//   - reCAPTCHA v3: grecaptcha.execute() calls, data-action attribute
//   - hCaptcha: h-captcha class, data-sitekey attribute, hcaptcha.com iframe
//   - Turnstile: cf-turnstile class, challenges.cloudflare.com iframe
//   - Image CAPTCHA: <img> with specific URL patterns, input fields with "captcha" in name/id
//
// It does NOT execute JavaScript to detect dynamically loaded CAPTCHAs.
// For those, use DetectInChromedpPage or DetectInPlaywrightPage.
package captcha

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Detector analyzes HTML and pages to find CAPTCHA challenges.
type Detector struct {
	config DetectionConfig
}

// NewDetector creates a new CAPTCHA detector with default configuration.
func NewDetector() *Detector {
	return &Detector{
		config: DefaultDetectionConfig(),
	}
}

// NewDetectorWithConfig creates a new CAPTCHA detector with custom configuration.
func NewDetectorWithConfig(config DetectionConfig) *Detector {
	return &Detector{
		config: config,
	}
}

// Detect scans HTML for CAPTCHA indicators.
// Returns nil if no CAPTCHA is detected above the confidence threshold.
func (d *Detector) Detect(html string, pageURL string) (*CaptchaDetection, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var detections []*CaptchaDetection

	// Check for each CAPTCHA type
	if detection := d.detectReCAPTCHAV2(doc, pageURL); detection != nil {
		detections = append(detections, detection)
	}
	if detection := d.detectReCAPTCHAV3(doc, pageURL); detection != nil {
		detections = append(detections, detection)
	}
	if detection := d.detectHCaptcha(doc, pageURL); detection != nil {
		detections = append(detections, detection)
	}
	if detection := d.detectTurnstile(doc, pageURL); detection != nil {
		detections = append(detections, detection)
	}
	if detection := d.detectImageCaptcha(doc, pageURL); detection != nil {
		detections = append(detections, detection)
	}

	if len(detections) == 0 {
		return nil, nil
	}

	// Return the highest confidence detection
	highest := detections[0]
	for _, d := range detections[1:] {
		if d.Score > highest.Score {
			highest = d
		}
	}

	if highest.Score < d.config.MinConfidence {
		return nil, nil
	}

	return highest, nil
}

// detectReCAPTCHAV2 detects reCAPTCHA v2 challenges.
func (d *Detector) detectReCAPTCHAV2(doc *goquery.Document, pageURL string) *CaptchaDetection {
	var reasons []DetectionReason
	var siteKey string
	var selector string
	score := 0.0

	// Look for g-recaptcha class
	if doc.Find(".g-recaptcha").Length() > 0 {
		reasons = append(reasons, ReasonClassMatch)
		score += 0.5
		selector = ".g-recaptcha"

		// Extract site key
		doc.Find(".g-recaptcha").Each(func(i int, s *goquery.Selection) {
			if key, exists := s.Attr("data-sitekey"); exists && key != "" {
				siteKey = key
				reasons = append(reasons, ReasonAttributeMatch)
				score += 0.3
			}
		})
	}

	// Look for reCAPTCHA iframe
	doc.Find("iframe").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if strings.Contains(src, "google.com/recaptcha") || strings.Contains(src, "recaptcha.net") {
				reasons = append(reasons, ReasonIFrameMatch)
				score += 0.4
				if selector == "" {
					selector = fmt.Sprintf("iframe[src*=\"recaptcha\"]:nth-of-type(%d)", i+1)
				}
			}
		}
	})

	// Look for recaptcha script
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if strings.Contains(src, "recaptcha") {
				reasons = append(reasons, ReasonScriptMatch)
				score += 0.2
			}
		}
	})

	if score < d.config.MinConfidence {
		return nil
	}

	return &CaptchaDetection{
		Type:       CaptchaTypeReCAPTCHAV2,
		Selector:   selector,
		SiteKey:    siteKey,
		Score:      min(score, 1.0),
		Reasons:    reasons,
		PageURL:    pageURL,
		DetectedAt: time.Now(),
	}
}

// detectReCAPTCHAV3 detects reCAPTCHA v3 challenges (invisible).
func (d *Detector) detectReCAPTCHAV3(doc *goquery.Document, pageURL string) *CaptchaDetection {
	var reasons []DetectionReason
	var siteKey string
	var action string
	score := 0.0

	// Look for grecaptcha.execute in scripts
	html, _ := doc.Html()
	if strings.Contains(html, "grecaptcha.execute") {
		reasons = append(reasons, ReasonScriptMatch)
		score += 0.4
	}

	// Look for data-action attribute (common in reCAPTCHA v3)
	doc.Find("[data-action]").Each(func(i int, s *goquery.Selection) {
		if act, exists := s.Attr("data-action"); exists {
			// Check if it's a recaptcha action
			if act == "login" || act == "signup" || act == "submit" {
				action = act
				reasons = append(reasons, ReasonAttributeMatch)
				score += 0.2
			}
		}
	})

	// Look for invisible recaptcha badge
	if doc.Find(".grecaptcha-badge").Length() > 0 {
		reasons = append(reasons, ReasonClassMatch)
		score += 0.5
	}

	// Look for site key in scripts or data attributes
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		// Look for site key pattern in scripts
		if key := extractSiteKey(text); key != "" {
			siteKey = key
			reasons = append(reasons, ReasonAttributeMatch)
			score += 0.3
		}
	})

	if score < d.config.MinConfidence {
		return nil
	}

	return &CaptchaDetection{
		Type:       CaptchaTypeReCAPTCHAV3,
		SiteKey:    siteKey,
		Action:     action,
		Score:      min(score, 1.0),
		Reasons:    reasons,
		PageURL:    pageURL,
		DetectedAt: time.Now(),
	}
}

// detectHCaptcha detects hCaptcha challenges.
func (d *Detector) detectHCaptcha(doc *goquery.Document, pageURL string) *CaptchaDetection {
	var reasons []DetectionReason
	var siteKey string
	var selector string
	score := 0.0

	// Look for h-captcha class
	if doc.Find(".h-captcha").Length() > 0 {
		reasons = append(reasons, ReasonClassMatch)
		score += 0.5
		selector = ".h-captcha"

		// Extract site key
		doc.Find(".h-captcha").Each(func(i int, s *goquery.Selection) {
			if key, exists := s.Attr("data-sitekey"); exists && key != "" {
				siteKey = key
				reasons = append(reasons, ReasonAttributeMatch)
				score += 0.3
			}
		})
	}

	// Look for hCaptcha iframe
	doc.Find("iframe").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if strings.Contains(src, "hcaptcha.com") {
				reasons = append(reasons, ReasonIFrameMatch)
				score += 0.4
				if selector == "" {
					selector = fmt.Sprintf("iframe[src*=\"hcaptcha\"]:nth-of-type(%d)", i+1)
				}
			}
		}
	})

	// Look for hcaptcha script
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if strings.Contains(src, "hcaptcha") {
				reasons = append(reasons, ReasonScriptMatch)
				score += 0.2
			}
		}
	})

	if score < d.config.MinConfidence {
		return nil
	}

	return &CaptchaDetection{
		Type:       CaptchaTypeHCaptcha,
		Selector:   selector,
		SiteKey:    siteKey,
		Score:      min(score, 1.0),
		Reasons:    reasons,
		PageURL:    pageURL,
		DetectedAt: time.Now(),
	}
}

// detectTurnstile detects Cloudflare Turnstile challenges.
func (d *Detector) detectTurnstile(doc *goquery.Document, pageURL string) *CaptchaDetection {
	var reasons []DetectionReason
	var siteKey string
	var selector string
	score := 0.0

	// Look for cf-turnstile class
	if doc.Find(".cf-turnstile").Length() > 0 {
		reasons = append(reasons, ReasonClassMatch)
		score += 0.5
		selector = ".cf-turnstile"

		// Extract site key
		doc.Find(".cf-turnstile").Each(func(i int, s *goquery.Selection) {
			if key, exists := s.Attr("data-sitekey"); exists && key != "" {
				siteKey = key
				reasons = append(reasons, ReasonAttributeMatch)
				score += 0.3
			}
		})
	}

	// Look for Turnstile iframe
	doc.Find("iframe").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if strings.Contains(src, "challenges.cloudflare.com") {
				reasons = append(reasons, ReasonIFrameMatch)
				score += 0.4
				if selector == "" {
					selector = fmt.Sprintf("iframe[src*=\"cloudflare\"]:nth-of-type(%d)", i+1)
				}
			}
		}
	})

	// Look for turnstile script
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if strings.Contains(src, "turnstile") || strings.Contains(src, "challenges.cloudflare") {
				reasons = append(reasons, ReasonScriptMatch)
				score += 0.2
			}
		}
	})

	if score < d.config.MinConfidence {
		return nil
	}

	return &CaptchaDetection{
		Type:       CaptchaTypeTurnstile,
		Selector:   selector,
		SiteKey:    siteKey,
		Score:      min(score, 1.0),
		Reasons:    reasons,
		PageURL:    pageURL,
		DetectedAt: time.Now(),
	}
}

// detectImageCaptcha detects generic image-based CAPTCHAs.
func (d *Detector) detectImageCaptcha(doc *goquery.Document, pageURL string) *CaptchaDetection {
	var reasons []DetectionReason
	var selector string
	score := 0.0

	// Look for images with captcha in src or alt
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, hasSrc := s.Attr("src")
		alt, hasAlt := s.Attr("alt")

		if hasSrc && containsCaptchaKeyword(src) {
			reasons = append(reasons, ReasonAttributeMatch)
			score += 0.3
			if selector == "" {
				selector = fmt.Sprintf("img[src*=\"captcha\"]:nth-of-type(%d)", i+1)
			}
		}

		if hasAlt && containsCaptchaKeyword(alt) {
			reasons = append(reasons, ReasonAttributeMatch)
			score += 0.3
			if selector == "" {
				selector = fmt.Sprintf("img[alt*=\"captcha\"]:nth-of-type(%d)", i+1)
			}
		}
	})

	// Look for input fields with captcha in name/id
	doc.Find("input").Each(func(i int, s *goquery.Selection) {
		name, hasName := s.Attr("name")
		id, hasId := s.Attr("id")

		if hasName && containsCaptchaKeyword(name) {
			reasons = append(reasons, ReasonInputPattern)
			score += 0.4
			if selector == "" {
				selector = fmt.Sprintf("input[name*=\"captcha\"]:nth-of-type(%d)", i+1)
			}
		}

		if hasId && containsCaptchaKeyword(id) {
			reasons = append(reasons, ReasonInputPattern)
			score += 0.4
			if selector == "" {
				selector = fmt.Sprintf("input#%s", id)
			}
		}
	})

	// Look for labels containing "captcha"
	doc.Find("label").Each(func(i int, s *goquery.Selection) {
		text := strings.ToLower(s.Text())
		if strings.Contains(text, "captcha") {
			reasons = append(reasons, ReasonAttributeMatch)
			score += 0.2
		}
	})

	if score < d.config.MinConfidence {
		return nil
	}

	return &CaptchaDetection{
		Type:       CaptchaTypeImage,
		Selector:   selector,
		Score:      min(score, 1.0),
		Reasons:    reasons,
		PageURL:    pageURL,
		DetectedAt: time.Now(),
	}
}

// DetectInChromedpPage detects CAPTCHA in a chromedp-rendered page.
// This requires chromedp to be available.
func (d *Detector) DetectInChromedpPage(ctx context.Context) (*CaptchaDetection, error) {
	// This is a placeholder for chromedp-specific detection.
	// The actual implementation would use chromedp to evaluate JavaScript
	// and check for CAPTCHA-related elements in the DOM.
	//
	// Example:
	// var html string
	// if err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html)); err != nil {
	//     return nil, err
	// }
	// return d.Detect(html, "")
	return nil, nil
}

// containsCaptchaKeyword checks if a string contains CAPTCHA-related keywords.
func containsCaptchaKeyword(s string) bool {
	s = strings.ToLower(s)
	keywords := []string{"captcha", "verify", "challenge", "security code"}
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// extractSiteKey extracts a reCAPTCHA site key from JavaScript code.
func extractSiteKey(js string) string {
	// Common patterns for site key in JS
	patterns := []string{
		`sitekey["']?\s*[:=]\s*["']([a-zA-Z0-9_-]+)["']`,
		`data-sitekey=["']([a-zA-Z0-9_-]+)["']`,
		`grecaptcha\.render\(["']([a-zA-Z0-9_-]+)["']`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(js)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
