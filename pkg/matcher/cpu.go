package matcher

import (
	"golang.org/x/sys/cpu"
)

// SIMDLevel represents the level of SIMD support detected at runtime.
type SIMDLevel int

const (
	SIMDNone SIMDLevel = iota
	SIMDSSE42
	SIMDAVX2
	SIMDNEON
)

func (s SIMDLevel) String() string {
	switch s {
	case SIMDAVX2:
		return "AVX2"
	case SIMDSSE42:
		return "SSE4.2"
	case SIMDNEON:
		return "NEON"
	default:
		return "None (Scalar Fallback)"
	}
}

// ActiveSIMD holds the detected SIMD capability of the current CPU.
var ActiveSIMD SIMDLevel

func init() {
	// Detect CPU capabilities at startup
	if cpu.X86.HasAVX2 {
		ActiveSIMD = SIMDAVX2
	} else if cpu.X86.HasSSE42 {
		ActiveSIMD = SIMDSSE42
	} else if cpu.ARM64.HasASIMD {
		ActiveSIMD = SIMDNEON
	} else {
		ActiveSIMD = SIMDNone
	}
}

// ForceSIMD overrides the active SIMD level (primarily for testing/benchmarking).
func ForceSIMD(level SIMDLevel) {
	ActiveSIMD = level
}
