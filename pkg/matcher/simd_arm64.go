//go:build arm64

package matcher

//go:noescape
func indexByte2NEON(b []byte, c1, c2 byte) int

// indexByte2SIMD dispatches to the NEON/ASIMD implementation when the CPU
// supports Advanced SIMD (true on all arm64v8 cores, including Apple-silicon
// Macs) and the buffer is at least one 16-byte vector wide; otherwise it falls
// back to the scalar implementation. The length guard guarantees the assembly
// routine never reads past the end of b before entering its scalar trailing
// loop.
func indexByte2SIMD(b []byte, c1, c2 byte) int {
	if ActiveSIMD == SIMDNEON && len(b) >= 16 {
		return indexByte2NEON(b, c1, c2)
	}
	return IndexByte2Generic(b, c1, c2)
}
