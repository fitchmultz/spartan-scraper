// Package simhash provides tests for simhash computation.
package simhash

import (
	"testing"
)

func TestCompute(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantZero bool
	}{
		{
			name:     "empty string returns zero",
			text:     "",
			wantZero: true,
		},
		{
			name:     "whitespace only returns zero",
			text:     "   \t\n  ",
			wantZero: true,
		},
		{
			name:     "punctuation only returns zero",
			text:     "!@#$%^&*()",
			wantZero: true,
		},
		{
			name:     "simple text produces non-zero hash",
			text:     "hello world",
			wantZero: false,
		},
		{
			name:     "long text produces non-zero hash",
			text:     "The quick brown fox jumps over the lazy dog multiple times for testing",
			wantZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compute(tt.text)
			if tt.wantZero && got != 0 {
				t.Errorf("Compute(%q) = %d, want 0", tt.text, got)
			}
			if !tt.wantZero && got == 0 {
				t.Errorf("Compute(%q) = 0, want non-zero", tt.text)
			}
		})
	}
}

func TestComputeConsistency(t *testing.T) {
	text := "This is a test of the simhash algorithm for content deduplication"

	// Same text should produce same hash
	hash1 := Compute(text)
	hash2 := Compute(text)
	if hash1 != hash2 {
		t.Errorf("Compute not consistent: %d != %d", hash1, hash2)
	}

	// Case insensitive - different case should produce same hash
	hash3 := Compute("HELLO WORLD TEST")
	hash4 := Compute("hello world test")
	if hash3 != hash4 {
		t.Errorf("Compute not case insensitive: %d != %d", hash3, hash4)
	}

	// Punctuation differences should not affect hash significantly
	hash5 := Compute("hello, world! test.")
	hash6 := Compute("hello world test")
	if hash5 != hash6 {
		t.Errorf("Compute affected by punctuation: %d != %d", hash5, hash6)
	}
}

func TestHammingDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        uint64
		b        uint64
		expected int
	}{
		{
			name:     "identical values have distance 0",
			a:        0x1234567890ABCDEF,
			b:        0x1234567890ABCDEF,
			expected: 0,
		},
		{
			name:     "one bit different",
			a:        0x0000000000000000,
			b:        0x0000000000000001,
			expected: 1,
		},
		{
			name:     "all bits different",
			a:        0x0000000000000000,
			b:        0xFFFFFFFFFFFFFFFF,
			expected: 64,
		},
		{
			name:     "alternating bits",
			a:        0xAAAAAAAAAAAAAAAA,
			b:        0x5555555555555555,
			expected: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HammingDistance(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("HammingDistance(%x, %x) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestIsDuplicate(t *testing.T) {
	tests := []struct {
		name      string
		a         uint64
		b         uint64
		threshold int
		want      bool
	}{
		{
			name:      "identical within threshold 0",
			a:         0x1234,
			b:         0x1234,
			threshold: 0,
			want:      true,
		},
		{
			name:      "different within threshold",
			a:         0x0000,
			b:         0x0007, // 3 bits different
			threshold: 3,
			want:      true,
		},
		{
			name:      "different outside threshold",
			a:         0x0000,
			b:         0x000F, // 4 bits different
			threshold: 3,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDuplicate(tt.a, tt.b, tt.threshold)
			if got != tt.want {
				t.Errorf("IsDuplicate(%x, %x, %d) = %v, want %v", tt.a, tt.b, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestSimilarContentDetection(t *testing.T) {
	// Test that similar content produces hashes with small Hamming distance
	text1 := "The quick brown fox jumps over the lazy dog"
	text2 := "The quick brown fox jumps over the lazy dog" // identical
	text3 := "The quick brown fox jumps over a lazy dog"   // one word different
	text4 := "A completely different sentence about cats"  // very different

	hash1 := Compute(text1)
	hash2 := Compute(text2)
	hash3 := Compute(text3)
	hash4 := Compute(text4)

	// Identical text should have distance 0
	if d := HammingDistance(hash1, hash2); d != 0 {
		t.Errorf("Identical text should have distance 0, got %d", d)
	}

	// Similar text should have small distance
	distSimilar := HammingDistance(hash1, hash3)
	if distSimilar > 8 {
		t.Errorf("Similar text should have small distance, got %d", distSimilar)
	}

	// Different text should have larger distance
	distDifferent := HammingDistance(hash1, hash4)
	if distDifferent < 10 {
		t.Errorf("Different text should have larger distance, got %d", distDifferent)
	}
}
