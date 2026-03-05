// Package simhash provides content similarity detection using simhash algorithm.
// It implements locality-sensitive hashing to detect near-duplicate content.
//
// This package is used by:
// - internal/crawl: for deduplicating pages during crawling
// - internal/research: for clustering and deduplicating evidence
//
// The simhash algorithm produces a 64-bit fingerprint where similar content
// produces hashes with small Hamming distances. This allows efficient
// near-duplicate detection without storing full content.
//
// Invariants:
// - Empty or whitespace-only text produces hash 0
// - Identical text produces identical hashes
// - Similar text produces hashes with small Hamming distance (typically < 5)
// - The algorithm is case-insensitive and ignores punctuation
package simhash

import (
	"hash/fnv"
	"math/bits"
	"regexp"
	"strings"
)

// Compute generates a 64-bit simhash fingerprint for the given text.
// The hash is computed by tokenizing the text, hashing each token,
// and combining the hashes using a weighted voting scheme.
//
// Similar texts will produce hashes with small Hamming distances.
// Returns 0 for empty or whitespace-only input.
func Compute(text string) uint64 {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return 0
	}

	var weights [64]int
	for _, token := range tokens {
		h := fnv.New64a()
		_, _ = h.Write([]byte(token))
		hash := h.Sum64()
		for i := range 64 {
			if hash&(1<<i) != 0 {
				weights[i]++
			} else {
				weights[i]--
			}
		}
	}

	var out uint64
	for i := range 64 {
		if weights[i] >= 0 {
			out |= 1 << i
		}
	}
	return out
}

// HammingDistance calculates the number of differing bits between two 64-bit values.
// This is used to compare simhash fingerprints - a small distance indicates
// similar content.
//
// Distance thresholds:
// - 0: identical content
// - 1-3: near-duplicates (minor changes)
// - 5-8: similar content
// - >10: distinct content
func HammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// IsDuplicate returns true if two simhash values are within the threshold distance.
// A threshold of 3 is recommended for detecting near-duplicates.
func IsDuplicate(a, b uint64, threshold int) bool {
	return HammingDistance(a, b) <= threshold
}

// tokenize converts text into a list of unique, normalized tokens.
// Removes punctuation, converts to lowercase, and deduplicates.
func tokenize(text string) []string {
	clean := strings.ToLower(strings.TrimSpace(text))
	if clean == "" {
		return nil
	}

	// Remove punctuation and special characters, keep alphanumeric and spaces
	re := regexp.MustCompile(`[^a-z0-9\s]+`)
	clean = re.ReplaceAllString(clean, " ")

	parts := strings.Fields(clean)
	if len(parts) == 0 {
		return nil
	}

	// Deduplicate while preserving order
	seen := make(map[string]bool, len(parts))
	uniq := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		uniq = append(uniq, part)
	}
	return uniq
}
