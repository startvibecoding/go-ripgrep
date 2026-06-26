//go:build amd64

package matcher

//go:noescape
func indexByte2AVX2(b []byte, c1, c2 byte) int

// indexByte2SIMD dispatches to the AVX2 implementation when the CPU supports it
// and the buffer is at least one 32-byte vector wide; otherwise it falls back to
// the scalar implementation. The length guard guarantees the assembly routine
// never reads past the end of b before entering its scalar trailing loop.
func indexByte2SIMD(b []byte, c1, c2 byte) int {
	if ActiveSIMD == SIMDAVX2 && len(b) >= 32 {
		return indexByte2AVX2(b, c1, c2)
	}
	return IndexByte2Generic(b, c1, c2)
}
