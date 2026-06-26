package matcher

// IndexByte2 returns the index of the first occurrence of either c1 or c2 in b,
// or -1 if neither is present. It uses SIMD when available: AVX2 on amd64 and
// NEON (ASIMD) on arm64 (including Apple-silicon Macs). On every other platform,
// or when the input is too small to benefit from vectorization, it falls back to
// the pure-Go scalar implementation.
//
// The actual vector dispatch (and the minimum length threshold) lives in the
// architecture-specific indexByte2SIMD implementations so that the scalar
// fallback never reads out of bounds.
func IndexByte2(b []byte, c1, c2 byte) int {
	return indexByte2SIMD(b, c1, c2)
}

// IndexByte2Generic is the pure Go fallback implementation of IndexByte2.
func IndexByte2Generic(b []byte, c1, c2 byte) int {
	for i, c := range b {
		if c == c1 || c == c2 {
			return i
		}
	}
	return -1
}
