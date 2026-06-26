package matcher

import (
	"testing"
)

func TestActiveSIMD(t *testing.T) {
	t.Logf("Detected Active SIMD level: %s", ActiveSIMD)
	
	// Test override
	original := ActiveSIMD
	defer func() { ActiveSIMD = original }()
	
	ForceSIMD(SIMDNone)
	if ActiveSIMD != SIMDNone {
		t.Errorf("expected SIMDNone, got %v", ActiveSIMD)
	}
}
