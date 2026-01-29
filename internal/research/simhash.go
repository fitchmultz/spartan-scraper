// Package research provides simhash computation and deduplication utilities.
package research

import (
	"hash/fnv"
	"math/bits"
)

// computeSimHash generates a 64-bit simhash for the given text.
// Used for content similarity detection and deduplication.
func computeSimHash(text string) uint64 {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return 0
	}
	var weights [64]int
	for _, token := range tokens {
		h := fnv.New64a()
		_, _ = h.Write([]byte(token))
		hash := h.Sum64()
		for i := 0; i < 64; i++ {
			if hash&(1<<i) != 0 {
				weights[i]++
			} else {
				weights[i]--
			}
		}
	}
	var out uint64
	for i := 0; i < 64; i++ {
		if weights[i] >= 0 {
			out |= 1 << i
		}
	}
	return out
}

// hammingDistance calculates the number of differing bits between two 64-bit values.
func hammingDistance(a uint64, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// dedupEvidence removes duplicate evidence items based on simhash similarity.
// maxDistance is the maximum hamming distance allowed for items to be considered duplicates.
func dedupEvidence(items []Evidence, maxDistance int) []Evidence {
	if len(items) == 0 {
		return items
	}
	out := make([]Evidence, 0, len(items))
	for _, item := range items {
		duplicate := false
		for _, existing := range out {
			if hammingDistance(item.SimHash, existing.SimHash) <= maxDistance {
				duplicate = true
				break
			}
		}
		if !duplicate {
			out = append(out, item)
		}
	}
	return out
}
