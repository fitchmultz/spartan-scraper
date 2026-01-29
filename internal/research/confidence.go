// Package research provides confidence score calculations for evidence and clusters.
package research

import (
	"math"
)

// evidenceConfidence calculates confidence score for a single evidence item.
// Combines score relevance (70%) and content length (30%).
func evidenceConfidence(item Evidence, maxScore float64) float64 {
	if maxScore <= 0 {
		return 0
	}
	scoreFactor := math.Log1p(item.Score) / math.Log1p(maxScore)
	lengthFactor := 0.0
	if len(item.Snippet) > 0 {
		lengthFactor = math.Min(float64(len(item.Snippet))/300.0, 1.0)
	}
	return clamp01(0.7*scoreFactor + 0.3*lengthFactor)
}

// clusterConfidence calculates average confidence across all evidence in a cluster.
func clusterConfidence(items []Evidence) float64 {
	if len(items) == 0 {
		return 0
	}
	sum := 0.0
	for _, item := range items {
		sum += item.Confidence
	}
	return clamp01(sum / float64(len(items)))
}

// overallConfidence calculates overall research confidence combining evidence and clusters.
// Evidence confidence weighted 60%, cluster confidence weighted 40%.
func overallConfidence(items []Evidence, clusters []EvidenceCluster) float64 {
	if len(items) == 0 {
		return 0
	}
	sum := 0.0
	for _, item := range items {
		sum += item.Confidence
	}
	evidenceScore := sum / float64(len(items))

	clusterScore := 0.0
	if len(clusters) > 0 {
		for _, cluster := range clusters {
			clusterScore += cluster.Confidence
		}
		clusterScore /= float64(len(clusters))
	}

	return clamp01(0.6*evidenceScore + 0.4*clusterScore)
}

// clamp01 clamps a float64 value to the range [0, 1].
func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
