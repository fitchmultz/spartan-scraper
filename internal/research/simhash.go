// Package research provides simhash computation and deduplication utilities.
package research

import (
	"github.com/fitchmultz/spartan-scraper/internal/simhash"
)

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
			if simhash.HammingDistance(item.SimHash, existing.SimHash) <= maxDistance {
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
