// Package crawl provides URL pattern matching for crawl filtering.
//
// This file implements glob-style pattern matching for include/exclude URL
// filtering during crawls. Patterns support * and ** wildcards and are
// anchored to match full URL paths.
//
// Pattern syntax:
//   - * matches any sequence of characters except /
//   - ** matches any sequence of characters including /
//   - Patterns are anchored (^...$) to match the full path
//
// Examples:
//   - /blog/** matches /blog/2024/post.html and /blog/index.html
//   - /products/* matches /products/item but not /products/cat/item
//   - /admin/* excludes all /admin/ paths when used as exclude pattern
//
// Precedence rules:
//  1. Exclude patterns take precedence over include patterns
//  2. If no include patterns are specified, all URLs are candidates
//  3. If include patterns are specified, only matching URLs are candidates
package crawl

import (
	"regexp"
	"strings"
)

// PatternMatcher compiles and matches glob patterns against URL paths.
// It supports include and exclude patterns with * and ** wildcards.
type PatternMatcher struct {
	include []*regexp.Regexp
	exclude []*regexp.Regexp
}

// NewPatternMatcher creates a matcher from glob patterns.
// Supports glob syntax: * matches any sequence except /, ** matches any sequence including /
// Returns an error if any pattern cannot be compiled.
func NewPatternMatcher(include, exclude []string) (*PatternMatcher, error) {
	pm := &PatternMatcher{}

	for _, pattern := range include {
		if pattern == "" {
			continue
		}
		re, err := globToRegex(pattern)
		if err != nil {
			return nil, err
		}
		pm.include = append(pm.include, re)
	}

	for _, pattern := range exclude {
		if pattern == "" {
			continue
		}
		re, err := globToRegex(pattern)
		if err != nil {
			return nil, err
		}
		pm.exclude = append(pm.exclude, re)
	}

	return pm, nil
}

// Matches checks if a URL path matches the patterns.
// Returns true if the URL should be crawled (passes include and exclude filters).
//
// Matching rules:
//  1. If any exclude pattern matches, return false
//  2. If no include patterns are specified, return true
//  3. If include patterns are specified, return true only if at least one matches
func (pm *PatternMatcher) Matches(urlPath string) bool {
	// Check exclude patterns first (they take precedence)
	for _, re := range pm.exclude {
		if re.MatchString(urlPath) {
			return false
		}
	}

	// If no include patterns specified, all URLs pass (except excluded)
	if len(pm.include) == 0 {
		return true
	}

	// Check include patterns - must match at least one
	for _, re := range pm.include {
		if re.MatchString(urlPath) {
			return true
		}
	}

	return false
}

// globToRegex converts a glob pattern to a regex pattern.
// The pattern is anchored with ^ and $ to match the full path.
// Supports * (matches any chars except /) and ** (matches any chars including /).
func globToRegex(glob string) (*regexp.Regexp, error) {
	var sb strings.Builder
	sb.WriteString("^")

	i := 0
	for i < len(glob) {
		c := glob[i]
		switch c {
		case '*':
			// Check for **
			if i+1 < len(glob) && glob[i+1] == '*' {
				// ** matches any sequence including /
				sb.WriteString(".*")
				i += 2
			} else {
				// * matches any sequence except /
				sb.WriteString("[^/]*")
				i++
			}
		case '?', '+', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\', '.':
			// Escape regex special characters
			sb.WriteByte('\\')
			sb.WriteByte(c)
			i++
		default:
			sb.WriteByte(c)
			i++
		}
	}

	sb.WriteString("$")

	return regexp.Compile(sb.String())
}
