// Package store provides deduplication-related store methods.
//
// This file extends the Store with content deduplication capabilities
// by integrating with the internal/dedup package.
package store

import (
	"github.com/fitchmultz/spartan-scraper/internal/dedup"
	"sync"
)

var (
	contentIndex     dedup.ContentIndex
	contentIndexOnce sync.Once
	contentIndexErr  error
)

// initContentIndex initializes the content index singleton.
// This is called lazily on first access to ensure the database is ready.
func (s *Store) initContentIndex() error {
	contentIndexOnce.Do(func() {
		sqliteIndex, err := dedup.NewSQLiteIndex(s.db)
		if err != nil {
			contentIndexErr = err
			return
		}
		contentIndex = sqliteIndex
	})
	return contentIndexErr
}

// GetContentIndex returns the content index for deduplication operations.
// Returns nil if the content index could not be initialized.
func (s *Store) GetContentIndex() dedup.ContentIndex {
	if err := s.initContentIndex(); err != nil {
		return nil
	}
	return contentIndex
}

// CloseContentIndex closes the content index and releases resources.
// This should be called when the store is closed.
func (s *Store) CloseContentIndex() error {
	if contentIndex != nil {
		if idx, ok := contentIndex.(*dedup.SQLiteIndex); ok {
			return idx.Close()
		}
	}
	return nil
}
