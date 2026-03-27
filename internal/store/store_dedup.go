/*
Purpose: Attach dedup content-index lifecycle to each Store instance.
Responsibilities: Lazily initialize the store-backed dedup index, expose it to callers, and release its resources during store shutdown.
Scope: Per-store dedup wiring only; query behavior lives in `internal/dedup` and API/runtime consumers live elsewhere.
Usage: Call `GetContentIndex()` when a store-backed dedup index is needed; shutdown happens through `Store.Close()`.
Invariants/Assumptions: A Store never shares its dedup index with another Store instance, and dedup state is initialized at most once per Store.
*/
package store

import "github.com/fitchmultz/spartan-scraper/internal/dedup"

// initContentIndex initializes the content index for this store.
// This is called lazily on first access to ensure the database is ready.
func (s *Store) initContentIndex() error {
	s.contentIndexOnce.Do(func() {
		sqliteIndex, err := dedup.NewSQLiteIndex(s.db)
		if err != nil {
			s.contentIndexErr = err
			return
		}
		s.contentIndex = sqliteIndex
	})
	return s.contentIndexErr
}

// GetContentIndex returns the content index for deduplication operations.
// Returns nil if the content index could not be initialized.
func (s *Store) GetContentIndex() dedup.ContentIndex {
	if err := s.initContentIndex(); err != nil {
		return nil
	}
	return s.contentIndex
}

func (s *Store) closeContentIndex() error {
	if s.contentIndex == nil {
		return nil
	}
	closer, ok := s.contentIndex.(interface{ Close() error })
	s.contentIndex = nil
	if !ok {
		return nil
	}
	return closer.Close()
}
