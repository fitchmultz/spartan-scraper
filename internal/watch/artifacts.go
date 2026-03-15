// Package watch manages persisted watch-check artifacts.
//
// Purpose:
//   - Persist stable per-watch screenshot and visual-diff artifacts without exposing
//     host-local filesystem paths through public interfaces.
//
// Responsibilities:
// - Rotate current screenshots into previous screenshots for each watch.
// - Persist deterministic current/previous/diff artifact files per watch.
// - Resolve artifact metadata for API responses and download handlers.
// - Remove watch artifact directories when a watch is deleted.
//
// Scope:
// - Watch-owned artifact files under DATA_DIR/watch_artifacts/<watch-id>/.
//
// Usage:
//   - Used by watch checks to stage screenshots and by API handlers to serve
//     artifact downloads.
//
// Invariants/Assumptions:
//   - Artifact files live only under DATA_DIR/watch_artifacts/<watch-id>/.
//   - Artifact kinds map to deterministic filenames with no host-local path
//     exposure in public contracts.
//   - The current screenshot becomes the previous screenshot before a new current
//     screenshot is persisted.
package watch

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
)

// ArtifactKind identifies a persisted watch artifact.
type ArtifactKind string

const (
	ArtifactKindCurrentScreenshot  ArtifactKind = "current-screenshot"
	ArtifactKindPreviousScreenshot ArtifactKind = "previous-screenshot"
	ArtifactKindVisualDiff         ArtifactKind = "visual-diff"
)

// Artifact describes a persisted watch artifact.
type Artifact struct {
	Kind        ArtifactKind `json:"kind"`
	Filename    string       `json:"filename,omitempty"`
	ContentType string       `json:"contentType,omitempty"`
	ByteSize    int64        `json:"byteSize,omitempty"`
	Path        string       `json:"-"`
}

// ArtifactStore manages persisted watch-check artifacts.
type ArtifactStore struct {
	dataDir string
}

// NewArtifactStore creates a new artifact store rooted at the data directory.
func NewArtifactStore(dataDir string) *ArtifactStore {
	return &ArtifactStore{dataDir: dataDir}
}

// ParseArtifactKind validates a public artifact kind string.
func ParseArtifactKind(raw string) (ArtifactKind, bool) {
	switch ArtifactKind(raw) {
	case ArtifactKindCurrentScreenshot, ArtifactKindPreviousScreenshot, ArtifactKindVisualDiff:
		return ArtifactKind(raw), true
	default:
		return "", false
	}
}

// List returns the currently available artifacts for a watch in stable display order.
func (s *ArtifactStore) List(watchID string) ([]Artifact, error) {
	orderedKinds := []ArtifactKind{
		ArtifactKindCurrentScreenshot,
		ArtifactKindPreviousScreenshot,
		ArtifactKindVisualDiff,
	}
	artifacts := make([]Artifact, 0, len(orderedKinds))
	for _, kind := range orderedKinds {
		artifact, err := s.Resolve(watchID, kind)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

// Resolve returns metadata for a single watch artifact.
func (s *ArtifactStore) Resolve(watchID string, kind ArtifactKind) (Artifact, error) {
	path := s.pathFor(watchID, kind)
	info, err := os.Stat(path)
	if err != nil {
		return Artifact{}, err
	}
	contentType, err := detectContentType(path)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:        kind,
		Filename:    filenameFor(watchID, kind, contentType),
		ContentType: contentType,
		ByteSize:    info.Size(),
		Path:        path,
	}, nil
}

// ReplaceCurrent rotates the current screenshot into the previous slot and writes
// a new current screenshot from the supplied source file.
func (s *ArtifactStore) ReplaceCurrent(watchID string, sourcePath string) (Artifact, *Artifact, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return Artifact{}, nil, fmt.Errorf("source path is required")
	}
	if err := fsutil.MkdirAllSecure(s.watchDir(watchID)); err != nil {
		return Artifact{}, nil, err
	}

	currentPath := s.pathFor(watchID, ArtifactKindCurrentScreenshot)
	previousPath := s.pathFor(watchID, ArtifactKindPreviousScreenshot)
	if _, err := os.Stat(currentPath); err == nil {
		if err := copyFileAtomic(currentPath, previousPath); err != nil {
			return Artifact{}, nil, err
		}
	} else if !os.IsNotExist(err) {
		return Artifact{}, nil, err
	}

	if err := copyFileAtomic(sourcePath, currentPath); err != nil {
		return Artifact{}, nil, err
	}

	currentArtifact, err := s.Resolve(watchID, ArtifactKindCurrentScreenshot)
	if err != nil {
		return Artifact{}, nil, err
	}

	previousArtifact, err := s.Resolve(watchID, ArtifactKindPreviousScreenshot)
	if err != nil {
		if os.IsNotExist(err) {
			return currentArtifact, nil, nil
		}
		return Artifact{}, nil, err
	}
	return currentArtifact, &previousArtifact, nil
}

// ReplaceVisualDiff writes the visual-diff artifact from the supplied source file.
func (s *ArtifactStore) ReplaceVisualDiff(watchID string, sourcePath string) (Artifact, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return Artifact{}, fmt.Errorf("source path is required")
	}
	if err := fsutil.MkdirAllSecure(s.watchDir(watchID)); err != nil {
		return Artifact{}, err
	}
	if err := copyFileAtomic(sourcePath, s.pathFor(watchID, ArtifactKindVisualDiff)); err != nil {
		return Artifact{}, err
	}
	return s.Resolve(watchID, ArtifactKindVisualDiff)
}

// ClearVisualDiff removes the persisted visual-diff artifact if it exists.
func (s *ArtifactStore) ClearVisualDiff(watchID string) error {
	return removeIfExists(s.pathFor(watchID, ArtifactKindVisualDiff))
}

// RemoveAll removes the entire watch artifact directory.
func (s *ArtifactStore) RemoveAll(watchID string) error {
	return os.RemoveAll(s.watchDir(watchID))
}

func (s *ArtifactStore) watchDir(watchID string) string {
	base := s.dataDir
	if strings.TrimSpace(base) == "" {
		base = ".data"
	}
	return filepath.Join(base, "watch_artifacts", watchID)
}

func (s *ArtifactStore) pathFor(watchID string, kind ArtifactKind) string {
	return filepath.Join(s.watchDir(watchID), string(kind))
}

func copyFileAtomic(sourcePath string, destPath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(destPath, data, fsutil.FileMode)
}

func detectContentType(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	if n == 0 {
		return "application/octet-stream", nil
	}
	return http.DetectContentType(buf[:n]), nil
}

func filenameFor(watchID string, kind ArtifactKind, contentType string) string {
	ext := extensionForContentType(contentType)
	return fmt.Sprintf("watch-%s-%s%s", watchID, kind, ext)
}

func extensionForContentType(contentType string) string {
	switch strings.TrimSpace(strings.Split(contentType, ";")[0]) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".bin"
	}
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
