package matcher

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

func TestIndexByte2(t *testing.T) {
	// Simple tests
	tests := []struct {
		b        []byte
		c1, c2   byte
		expected int
	}{
		{[]byte("hello"), 'e', 'o', 1},
		{[]byte("hello"), 'o', 'e', 1},
		{[]byte("hello"), 'x', 'y', -1},
		{[]byte(""), 'a', 'b', -1},
		{[]byte("a"), 'a', 'b', 0},
		{[]byte("b"), 'a', 'b', 0},
		{[]byte("x"), 'a', 'b', -1},
	}

	for _, tc := range tests {
		res := IndexByte2(tc.b, tc.c1, tc.c2)
		if res != tc.expected {
			t.Errorf("IndexByte2(%q, %q, %q) = %d, expected %d", tc.b, tc.c1, tc.c2, res, tc.expected)
		}
	}

	// Randomized property tests for long slices to trigger AVX2 and trailing fallback loops
	rng := rand.New(rand.NewSource(42))
	for length := 1; length <= 256; length++ {
		for i := 0; i < 20; i++ {
			b := make([]byte, length)
			for j := range b {
				// Fill with letters except 'a' and 'b'
				b[j] = byte('c' + rng.Intn(20))
			}
			c1, c2 := byte('a'), byte('b')

			// Verify no-match case
			resNone := IndexByte2(b, c1, c2)
			resGenNone := IndexByte2Generic(b, c1, c2)
			if resNone != resGenNone {
				t.Fatalf("len %d (no match): IndexByte2=%d, Generic=%d", length, resNone, resGenNone)
			}

			// Insert a match at a random position
			matchPos := rng.Intn(length)
			if rng.Float32() < 0.5 {
				b[matchPos] = c1
			} else {
				b[matchPos] = c2
			}

			res := IndexByte2(b, c1, c2)
			resGen := IndexByte2Generic(b, c1, c2)
			if res != resGen {
				t.Fatalf("len %d, matchPos %d: IndexByte2=%d, Generic=%d (data: %s)", length, matchPos, res, resGen, b)
			}
		}
	}
}

func BenchmarkIndexByte2(b *testing.B) {
	// Prepare a large buffer
	size := 1024 * 1024 // 1 MB
	buf := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz012345"), size/32)
	c1, c2 := byte('X'), byte('Y') // Not found to measure worst-case performance

	b.Run("Generic", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = IndexByte2Generic(buf, c1, c2)
		}
	})

	b.Run("SIMD", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = IndexByte2(buf, c1, c2)
		}
	})
}

// BenchmarkIndexByte2Ops measures throughput in operations per second
// (how many IndexByte2 calls complete per second) for both the generic
// and the active SIMD implementation (AVX2 on amd64, NEON on arm64) over a
// 1 MB buffer.
func BenchmarkIndexByte2Ops(b *testing.B) {
	size := 1024 * 1024 // 1 MB
	buf := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz012345"), size/32)
	c1, c2 := byte('X'), byte('Y') // Not found to measure worst-case performance

	b.Run("Generic", func(b *testing.B) {
		start := time.Now()
		for i := 0; i < b.N; i++ {
			_ = IndexByte2Generic(buf, c1, c2)
		}
		elapsed := time.Since(start).Seconds()
		if elapsed > 0 {
			b.ReportMetric(float64(b.N)/elapsed, "ops/s")
		}
	})

	b.Run("SIMD", func(b *testing.B) {
		start := time.Now()
		for i := 0; i < b.N; i++ {
			_ = IndexByte2(buf, c1, c2)
		}
		elapsed := time.Since(start).Seconds()
		if elapsed > 0 {
			b.ReportMetric(float64(b.N)/elapsed, "ops/s")
		}
	})
}
