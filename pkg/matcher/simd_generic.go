//go:build !amd64 && !arm64

package matcher

// indexByte2SIMD on platforms without a vectorized implementation simply uses
// the scalar fallback.
func indexByte2SIMD(b []byte, c1, c2 byte) int {
	return IndexByte2Generic(b, c1, c2)
}
