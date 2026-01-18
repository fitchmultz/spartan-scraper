package extract

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	jsonLDGraphKey = "@graph"
	jsonLDTypeKey  = "@type"
)

func ExtractJSONLD(html string) ([]map[string]any, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var results []map[string]any

	doc.Find("script[type='application/ld+json']").Each(func(_ int, s *goquery.Selection) {
		text := s.Text()
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return
		}

		// Handle array of objects or single object
		// Try parsing as single object
		var single map[string]any
		if err := json.Unmarshal([]byte(trimmed), &single); err == nil {
			processJSONLDObject(single, &results)
			return
		}

		// Try parsing as array
		var array []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &array); err == nil {
			for _, item := range array {
				processJSONLDObject(item, &results)
			}
			return
		}
	})

	return results, nil
}

func processJSONLDObject(obj map[string]any, results *[]map[string]any) {
	// Check for @graph key
	if graph, ok := obj[jsonLDGraphKey]; ok {
		if graphArray, ok := graph.([]any); ok {
			for _, item := range graphArray {
				if itemMap, ok := item.(map[string]any); ok {
					*results = append(*results, itemMap)
				}
			}
		}
		// Some implementations use @graph as the main container
		return
	}

	*results = append(*results, obj)
}

func MatchJSONLD(documents []map[string]any, rule JSONLDRule) []string {
	var matches []string

	for _, doc := range documents {
		if rule.Type != "" {
			typeVal, ok := doc[jsonLDTypeKey]
			if !ok {
				continue
			}
			// @type can be string or array of strings
			matchedType := false
			switch t := typeVal.(type) {
			case string:
				if strings.EqualFold(t, rule.Type) {
					matchedType = true
				}
			case []any:
				for _, v := range t {
					if s, ok := v.(string); ok && strings.EqualFold(s, rule.Type) {
						matchedType = true
						break
					}
				}
			}
			if !matchedType {
				continue
			}
		}

		// Traverse path
		val := getPath(doc, rule.Path)
		if val != nil {
			extractStrings(val, &matches)
		}
	}

	return matches
}

func getPath(obj any, path string) any {
	parts := strings.Split(path, ".")
	current := obj

	for _, part := range parts {
		if current == nil {
			return nil
		}

		// If current is a map, simple lookup
		if m, ok := current.(map[string]any); ok {
			val, found := m[part]
			if !found {
				return nil
			}
			current = val
			continue
		}

		// If current is a slice, apply path part to each element (flatten)
		if s, ok := current.([]any); ok {
			var next []any
			for _, item := range s {
				if m, ok := item.(map[string]any); ok {
					if val, found := m[part]; found {
						next = append(next, val)
					}
				}
			}
			if len(next) == 0 {
				return nil
			}
			current = next
			continue
		}

		return nil
	}
	return current
}

func extractStrings(val any, out *[]string) {
	switch v := val.(type) {
	case string:
		*out = append(*out, v)
	case float64:
		*out = append(*out, fmt.Sprintf("%v", v))
	case int:
		*out = append(*out, fmt.Sprintf("%d", v))
	case bool:
		*out = append(*out, fmt.Sprintf("%v", v))
	case []any:
		for _, item := range v {
			extractStrings(item, out)
		}
	}
}
