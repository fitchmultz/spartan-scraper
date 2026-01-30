// Package research provides evidence clustering by simhash similarity.
package research

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/simhash"
)

// clusterEvidence clusters evidence items by simhash similarity.
// Returns clusters and evidence with cluster IDs assigned.
func clusterEvidence(items []Evidence, maxDistance int, minSize int) ([]EvidenceCluster, []Evidence) {
	if len(items) == 0 {
		return []EvidenceCluster{}, items
	}
	type cluster struct {
		id       string
		evidence []Evidence
	}
	clusters := make([]cluster, 0)

	for _, item := range items {
		placed := false
		for i := range clusters {
			for _, member := range clusters[i].evidence {
				if simhash.HammingDistance(item.SimHash, member.SimHash) <= maxDistance {
					clusters[i].evidence = append(clusters[i].evidence, item)
					placed = true
					break
				}
			}
			if placed {
				break
			}
		}
		if !placed {
			clusters = append(clusters, cluster{
				id:       fmtClusterID(len(clusters) + 1),
				evidence: []Evidence{item},
			})
		}
	}

	enriched := make([]Evidence, 0, len(items))
	finalClusters := make([]EvidenceCluster, 0, len(clusters))
	for _, c := range clusters {
		for i := range c.evidence {
			c.evidence[i].ClusterID = c.id
			enriched = append(enriched, c.evidence[i])
		}
		label := clusterLabel(c.evidence)
		conf := clusterConfidence(c.evidence)
		if minSize <= 1 || len(c.evidence) >= minSize {
			finalClusters = append(finalClusters, EvidenceCluster{
				ID:         c.id,
				Label:      label,
				Evidence:   c.evidence,
				Confidence: conf,
			})
		}
	}

	sort.Slice(finalClusters, func(i, j int) bool {
		return finalClusters[i].Confidence > finalClusters[j].Confidence
	})

	return finalClusters, enriched
}

// fmtClusterID formats a cluster ID from an index.
func fmtClusterID(index int) string {
	return "cluster-" + strconv.Itoa(index)
}

// clusterLabel generates a label for a cluster from its evidence.
func clusterLabel(items []Evidence) string {
	if len(items) == 0 {
		return ""
	}
	if strings.TrimSpace(items[0].Title) != "" {
		return items[0].Title
	}
	return hostFromURL(items[0].URL)
}

// hostFromURL extracts the host from a URL string.
func hostFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return raw
	}
	return parsed.Host
}
