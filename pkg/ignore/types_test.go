package ignore

import "testing"

func TestShouldIgnoreByType(t *testing.T) {
	tests := []struct {
		filename string
		types    []string
		typesNot []string
		expected bool
	}{
		// No filters
		{"main.go", nil, nil, false},
		// Positive filters
		{"main.go", []string{"go"}, nil, false},
		{"main.go", []string{"rust", "go"}, nil, false},
		{"main.go", []string{"rust"}, nil, true},
		// Negative filters
		{"main.go", nil, []string{"rust"}, false},
		{"main.go", nil, []string{"go"}, true},
		{"main.go", nil, []string{"go", "rust"}, true},
		// Combined filters
		{"main.go", []string{"go"}, []string{"rust"}, false},
		{"main.go", []string{"go"}, []string{"go"}, true},
	}

	for i, tc := range tests {
		got := ShouldIgnoreByType(tc.filename, tc.types, tc.typesNot)
		if got != tc.expected {
			t.Errorf("test %d (file=%s, types=%v, typesNot=%v) = %v, expected %v",
				i, tc.filename, tc.types, tc.typesNot, got, tc.expected)
		}
	}
}
