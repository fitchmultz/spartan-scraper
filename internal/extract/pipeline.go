package extract

import (
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func ApplyTemplate(url string, html string, template Template) (Extracted, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return Extracted{}, err
	}

	result := Extracted{
		URL:         url,
		Fields:      make(map[string]FieldValue),
		Metadata:    make(map[string]string),
		Links:       make([]string, 0),
		Raw:         make(map[string][]string),
		Template:    template.Name,
		ExtractedAt: time.Now(),
	}

	// 1. Base Extraction (Title, Text, Links) - always done for convenience
	// These are "default" extractions if not overridden by rules, but we populate them here.
	// NOTE: The legacy extract.FromHTML logic did this. We replicate it here as a baseline.
	result.Title = strings.TrimSpace(doc.Find("title").First().Text())

	// Remove noise for text extraction
	clone := doc.Clone()
	clone.Find("script,style,noscript").Remove()
	bodyText := strings.TrimSpace(clone.Find("body").Text())
	result.Text = strings.Join(strings.Fields(bodyText), " ")

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			result.Links = append(result.Links, href)
		}
	})

	// 2. Selectors
	for _, rule := range template.Selectors {
		var values []string
		doc.Find(rule.Selector).Each(func(_ int, s *goquery.Selection) {
			if !rule.All && len(values) > 0 {
				return
			}
			val := ""
			if rule.Attr == "" || rule.Attr == "text" {
				val = s.Text()
			} else if rule.Attr == "html" {
				val, _ = s.Html()
			} else {
				val, _ = s.Attr(rule.Attr)
			}

			if rule.Trim {
				val = strings.TrimSpace(val)
			}
			if val != "" {
				values = append(values, val)
			}
		})

		if len(values) > 0 {
			if rule.Join != "" {
				values = []string{strings.Join(values, rule.Join)}
			}
			result.Fields[rule.Name] = FieldValue{Values: values, Source: FieldSourceSelector}
			result.Raw[rule.Name] = values
		}
	}

	// 3. JSON-LD
	if len(template.JSONLD) > 0 {
		jsonldObjects, err := ExtractJSONLD(html)
		if err == nil {
			result.JSONLD = jsonldObjects
			for _, rule := range template.JSONLD {
				matches := MatchJSONLD(jsonldObjects, rule)
				if len(matches) > 0 {
					result.Fields[rule.Name] = FieldValue{Values: matches, Source: FieldSourceJSONLD}
					result.Raw[rule.Name] = matches
				}
			}
		}
	}

	// 4. Regex
	for _, rule := range template.Regex {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			continue
		}

		var input string
		switch rule.Source {
		case RegexSourceText:
			input = result.Text
		case RegexSourceHTML:
			input = html
		case RegexSourceURL:
			input = url
		default:
			input = result.Text
		}

		var values []string
		if rule.All {
			matches := re.FindAllStringSubmatch(input, -1)
			for _, match := range matches {
				if rule.Group < len(match) {
					values = append(values, match[rule.Group])
				} else if len(match) > 0 {
					values = append(values, match[0])
				}
			}
		} else {
			match := re.FindStringSubmatch(input)
			if match != nil {
				if rule.Group < len(match) {
					values = append(values, match[rule.Group])
				} else if len(match) > 0 {
					values = append(values, match[0])
				}
			}
		}

		if len(values) > 0 {
			result.Fields[rule.Name] = FieldValue{Values: values, Source: FieldSourceRegex}
			result.Raw[rule.Name] = values
		}
	}

	return result, nil
}
