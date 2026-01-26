// Package common contains small internal shims to keep other files cleaner.
// This file exists to keep the rest of the package focused and reduce import noise.
//
// It does NOT provide any additional CLI behavior.
package common

import "strings"

func trimSpace(s string) string { return strings.TrimSpace(s) }
func joinWithComma(items []string) string {
	return strings.Join(items, ",")
}
func stringsIndex(s, sep string) int { return strings.Index(s, sep) }
