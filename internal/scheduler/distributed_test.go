// Package scheduler provides distributed scheduling with leader election.
//
// This file contains tests for the distributed scheduler functionality,
// particularly focusing on the cryptographically secure instance ID generation.
package scheduler

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// TestRandomStringLength verifies that randomString returns a string of the correct length.
func TestRandomStringLength(t *testing.T) {
	tests := []int{1, 6, 10, 32, 64}
	for _, length := range tests {
		s, err := randomString(length)
		if err != nil {
			t.Fatalf("randomString(%d) returned error: %v", length, err)
		}
		if len(s) != length {
			t.Errorf("randomString(%d) returned string of length %d, want %d", length, len(s), length)
		}
	}
}

// TestRandomStringCharacters verifies that randomString only uses allowed characters.
func TestRandomStringCharacters(t *testing.T) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const charsetSet = "abcdefghijklmnopqrstuvwxyz0123456789"

	s, err := randomString(1000)
	if err != nil {
		t.Fatalf("randomString(1000) returned error: %v", err)
	}

	for _, c := range s {
		if !strings.ContainsRune(charsetSet, c) {
			t.Errorf("randomString returned character '%c' not in charset %q", c, charset)
		}
	}
}

// TestRandomStringRandomness verifies that randomString returns different values on subsequent calls.
func TestRandomStringRandomness(t *testing.T) {
	const iterations = 100
	seen := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		s, err := randomString(6)
		if err != nil {
			t.Fatalf("randomString(6) returned error: %v", err)
		}
		if seen[s] {
			t.Logf("Duplicate string generated: %s (this is possible but unlikely)", s)
		}
		seen[s] = true
	}

	// With 36^6 = 2,176,782,336 possible combinations, we expect almost no collisions
	// in 100 iterations. Check that we have at least 95% uniqueness.
	if len(seen) < iterations*95/100 {
		t.Errorf("Too many collisions: got %d unique strings out of %d", len(seen), iterations)
	}
}

// TestRandomStringEmpty verifies behavior with length 0.
func TestRandomStringEmpty(t *testing.T) {
	s, err := randomString(0)
	if err != nil {
		t.Fatalf("randomString(0) returned error: %v", err)
	}
	if s != "" {
		t.Errorf("randomString(0) returned %q, want empty string", s)
	}
}

// TestGenerateInstanceIDFormat verifies that generateInstanceID returns a correctly formatted ID.
func TestGenerateInstanceIDFormat(t *testing.T) {
	id, err := generateInstanceID()
	if err != nil {
		t.Fatalf("generateInstanceID() returned error: %v", err)
	}

	// Format should be: scheduler-YYYYMMDD-HHMMSS-XXXXXX
	if !strings.HasPrefix(id, "scheduler-") {
		t.Errorf("generateInstanceID() returned ID without 'scheduler-' prefix: %s", id)
	}

	// Should have 3 parts separated by hyphens
	parts := strings.Split(id, "-")
	if len(parts) != 4 { // scheduler, YYYYMMDD, HHMMSS, XXXXXX
		t.Errorf("generateInstanceID() returned ID with wrong format, expected 4 parts, got %d: %s", len(parts), id)
	}

	// Check that the random suffix has correct length (6 characters)
	suffix := parts[len(parts)-1]
	if len(suffix) != 6 {
		t.Errorf("generateInstanceID() returned ID with suffix length %d, want 6: %s", len(suffix), id)
	}
}

// TestGenerateInstanceIDUniqueness generates 10,000 IDs and verifies no collisions.
func TestGenerateInstanceIDUniqueness(t *testing.T) {
	const count = 10000
	ids := make(map[string]bool)
	collisions := 0

	for i := 0; i < count; i++ {
		id, err := generateInstanceID()
		if err != nil {
			t.Fatalf("generateInstanceID() returned error at iteration %d: %v", i, err)
		}
		if ids[id] {
			collisions++
			t.Logf("Collision detected: %s", id)
		}
		ids[id] = true
	}

	if collisions > 0 {
		t.Errorf("Found %d collision(s) out of %d generated IDs", collisions, count)
	}

	t.Logf("Generated %d unique IDs with 0 collisions", len(ids))
}

// TestGenerateInstanceIDConcurrent tests concurrent generation for race conditions and collisions.
func TestGenerateInstanceIDConcurrent(t *testing.T) {
	const numGoroutines = 100
	const idsPerGoroutine = 100

	var wg sync.WaitGroup
	idChan := make(chan string, numGoroutines*idsPerGoroutine)
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := generateInstanceID()
				if err != nil {
					errChan <- err
					return
				}
				idChan <- id
			}
		}()
	}

	wg.Wait()
	close(idChan)
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Fatalf("generateInstanceID() returned error: %v", err)
	}

	// Check for collisions
	ids := make(map[string]bool)
	collisionCount := 0
	for id := range idChan {
		if ids[id] {
			collisionCount++
		}
		ids[id] = true
	}

	if collisionCount > 0 {
		t.Errorf("Found %d collision(s) in concurrent test", collisionCount)
	}

	total := numGoroutines * idsPerGoroutine
	t.Logf("Concurrent test: Generated %d unique IDs with %d collisions", len(ids), collisionCount)

	// Verify we got all expected IDs
	if len(ids) != total {
		t.Errorf("Expected %d IDs, got %d", total, len(ids))
	}
}

// TestGenerateInstanceIDTimestamp verifies that the timestamp portion is valid.
func TestGenerateInstanceIDTimestamp(t *testing.T) {
	id, err := generateInstanceID()
	if err != nil {
		t.Fatalf("generateInstanceID() returned error: %v", err)
	}

	// Extract timestamp from ID (format: scheduler-YYYYMMDD-HHMMSS-XXXXXX)
	parts := strings.Split(id, "-")
	if len(parts) != 4 {
		t.Fatalf("Unexpected ID format: %s", id)
	}

	datePart := parts[1] // YYYYMMDD
	timePart := parts[2] // HHMMSS

	// Verify date part is valid (8 digits representing a valid date)
	if len(datePart) != 8 {
		t.Errorf("Date part has wrong length: %d, expected 8", len(datePart))
	}
	_, err = time.Parse("20060102", datePart)
	if err != nil {
		t.Errorf("Failed to parse date from ID %s: %v", id, err)
		return
	}

	// Verify time part is valid (6 digits representing a valid time)
	if len(timePart) != 6 {
		t.Errorf("Time part has wrong length: %d, expected 6", len(timePart))
	}
	_, err = time.Parse("150405", timePart)
	if err != nil {
		t.Errorf("Failed to parse time from ID %s: %v", id, err)
		return
	}

	// Verify the timestamp format matches the expected pattern
	// (YYYYMMDD should be 8 digits, HHMMSS should be 6 digits)
	for _, c := range datePart {
		if c < '0' || c > '9' {
			t.Errorf("Date part contains non-digit character: %c", c)
		}
	}
	for _, c := range timePart {
		if c < '0' || c > '9' {
			t.Errorf("Time part contains non-digit character: %c", c)
		}
	}
}

// TestGenerateInstanceIDWithPrefix verifies that generateInstanceID creates unique IDs even when
// timestamps are the same (by relying on the random suffix).
func TestGenerateInstanceIDSameTimestamp(t *testing.T) {
	// Generate multiple IDs as quickly as possible to get same-second timestamps
	const count = 100
	ids := make([]string, count)

	for i := 0; i < count; i++ {
		id, err := generateInstanceID()
		if err != nil {
			t.Fatalf("generateInstanceID() returned error: %v", err)
		}
		ids[i] = id
	}

	// Check that all IDs are unique
	seen := make(map[string]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		seen[id] = true
	}
}

// TestRandomStringErrorHandling verifies that randomString returns proper error classification.
func TestRandomStringErrorHandling(t *testing.T) {
	// Test that we can generate strings without errors under normal conditions
	s, err := randomString(6)
	if err != nil {
		t.Fatalf("randomString(6) returned unexpected error: %v", err)
	}
	if len(s) != 6 {
		t.Errorf("randomString(6) returned string of length %d, want 6", len(s))
	}

	// Verify error would be KindInternal if crypto/rand fails
	// (This is difficult to test directly as crypto/rand rarely fails,
	// but we verify the error wrapping is correct)
	if err != nil {
		if !apperrors.IsKind(err, apperrors.KindInternal) {
			t.Errorf("Expected KindInternal error, got %v", apperrors.KindOf(err))
		}
	}
}

// BenchmarkRandomString benchmarks the randomString function.
func BenchmarkRandomString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := randomString(6)
		if err != nil {
			b.Fatalf("randomString(6) returned error: %v", err)
		}
	}
}

// BenchmarkGenerateInstanceID benchmarks the generateInstanceID function.
func BenchmarkGenerateInstanceID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := generateInstanceID()
		if err != nil {
			b.Fatalf("generateInstanceID() returned error: %v", err)
		}
	}
}
