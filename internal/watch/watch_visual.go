// Package watch provides visual comparison helpers for watch change detection.
//
// Purpose:
// - Compute perceptual hashes and generate visual diffs between screenshot snapshots.
//
// Responsibilities:
// - Compute visual hashes from screenshot image files.
// - Compare screenshots and produce similarity scores.
// - Persist visual diff artifacts through the artifact store.
//
// Scope:
// - Visual comparison logic only; watch execution and scheduling live in sibling files.
//
// Usage:
// - Called by Watcher.Check when visual change detection is enabled.
//
// Invariants/Assumptions:
// - Both current and previous screenshots must exist on disk before comparison.
// - Similarity is a byte-level proxy metric, not a true perceptual hash.
package watch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// computeVisualHash computes a simple perceptual hash for an image file.
// Uses a hash of file contents for basic perceptual similarity.
func computeVisualHash(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read screenshot: %w", err)
	}
	// Compute hash of file contents
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:16]), nil // Use first 16 bytes for shorter hash
}

// generateVisualDiff creates a visual diff between two screenshots.
// Returns the persisted diff artifact and similarity score.
func (w *Watcher) generateVisualDiff(watchID, currentPath, previousPath string, threshold float64) (*Artifact, float64, error) {
	if currentPath == "" || previousPath == "" {
		return nil, 0, nil
	}

	// Check if both files exist
	if _, err := os.Stat(currentPath); os.IsNotExist(err) {
		return nil, 0, fmt.Errorf("current screenshot not found")
	}
	if _, err := os.Stat(previousPath); os.IsNotExist(err) {
		return nil, 0, fmt.Errorf("previous screenshot not found")
	}

	// Read both files for comparison
	currentData, err := os.ReadFile(currentPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read current screenshot: %w", err)
	}
	previousData, err := os.ReadFile(previousPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read previous screenshot: %w", err)
	}

	// Compute similarity based on content comparison
	// This is a simplified similarity metric - in production would use image processing
	var similarity float64
	if len(previousData) > 0 {
		// Use simple byte-level comparison as proxy
		minLen := len(currentData)
		if len(previousData) < minLen {
			minLen = len(previousData)
		}
		if minLen > 0 {
			diffCount := 0
			for i := 0; i < minLen; i++ {
				if currentData[i] != previousData[i] {
					diffCount++
				}
			}
			// Account for length difference
			lengthDiff := len(currentData) - len(previousData)
			if lengthDiff < 0 {
				lengthDiff = -lengthDiff
			}
			diffCount += lengthDiff

			maxDiff := len(currentData) + len(previousData)
			if maxDiff > 0 {
				similarity = 1.0 - float64(diffCount)/float64(maxDiff)
			}
		}
	}

	artifact, err := NewArtifactStore(w.dataDir).ReplaceVisualDiff(watchID, currentPath)
	if err != nil {
		return nil, similarity, fmt.Errorf("failed to write visual diff artifact: %w", err)
	}
	return &artifact, similarity, nil
}
