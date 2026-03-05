// Package common provides shared CLI helpers used across command modules.
// It is responsible for flag value helpers and small parsing utilities.
//
// It does NOT implement command routing (that's internal/cli) or domain logic
// (jobs/auth/scheduler/etc live in their own internal packages).
package common

import "strings"

// SplitCSV splits a comma-separated string into trimmed, non-empty values.
func SplitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trim := strings.TrimSpace(part)
		if trim != "" {
			out = append(out, trim)
		}
	}
	return out
}
