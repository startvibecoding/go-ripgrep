package globset

import (
	"testing"
)

func TestGlobSetMatch(t *testing.T) {
	tests := []struct {
		patterns  []string
		path      string
		matched   bool
		isIgnored bool
	}{
		// Simple suffix match
		{[]string{"*.log"}, "error.log", true, true},
		{[]string{"*.log"}, "dir/error.log", true, true},
		{[]string{"*.log"}, "dir/subdir/error.log", true, true},
		{[]string{"*.log"}, "error.txt", false, false},

		// Slash pattern (anchored to root)
		{[]string{"/bin/"}, "bin/exe", true, true},
		{[]string{"bin/"}, "dir/bin/exe", true, true}, // trailing slash matches directories or nested directories

		// Double star pattern
		{[]string{"src/**/*.go"}, "src/main.go", true, true},
		{[]string{"src/**/*.go"}, "src/pkg/helper.go", true, true},
		{[]string{"src/**/*.go"}, "pkg/helper.go", false, false},

		// Negated patterns (override ignore)
		{[]string{"*.go", "!main.go"}, "main.go", true, false},
		{[]string{"*.go", "!main.go"}, "helper.go", true, true},
		{[]string{"!main.go", "*.go"}, "main.go", true, true}, // order matters: last match wins
	}

	for i, tc := range tests {
		gs, err := NewGlobSet(tc.patterns)
		if err != nil {
			t.Fatalf("test case %d: unexpected error: %v", i, err)
		}
		matched, isIgnored := gs.MatchPath(tc.path)
		if matched != tc.matched || isIgnored != tc.isIgnored {
			t.Errorf("test case %d: patterns %v on path %q: expected (matched=%v, ignored=%v), got (matched=%v, ignored=%v)",
				i, tc.patterns, tc.path, tc.matched, tc.isIgnored, matched, isIgnored)
		}
	}
}
