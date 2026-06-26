package main

import (
	"reflect"
	"testing"
)

func TestParseArgsOverrides(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected *CliArgs
	}{
		{
			name: "case-sensitivity override: -i -s",
			args: []string{"-i", "-s", "pattern"},
			expected: &CliArgs{
				Pattern:         "pattern",
				Color:           "auto",
				CaseInsensitive: false,
				LineNumber:      true,
			},
		},
		{
			name: "case-sensitivity override: -s -i",
			args: []string{"-s", "-i", "pattern"},
			expected: &CliArgs{
				Pattern:         "pattern",
				Color:           "auto",
				CaseInsensitive: true,
				LineNumber:      true,
			},
		},
		{
			name: "line-number override: -n -N",
			args: []string{"-n", "-N", "pattern"},
			expected: &CliArgs{
				Pattern:      "pattern",
				Color:        "auto",
				LineNumber:   false,
				NoLineNumber: true,
			},
		},
		{
			name: "line-number override: -N -n",
			args: []string{"-N", "-n", "pattern"},
			expected: &CliArgs{
				Pattern:      "pattern",
				Color:        "auto",
				LineNumber:   true,
				NoLineNumber: false,
			},
		},
		{
			name: "filename override: -H -I",
			args: []string{"-H", "-I", "pattern"},
			expected: &CliArgs{
				Pattern:      "pattern",
				Color:        "auto",
				WithFilename: false,
				NoFilename:   true,
				LineNumber:   true,
			},
		},
		{
			name: "filename override: -I -H",
			args: []string{"-I", "-H", "pattern"},
			expected: &CliArgs{
				Pattern:      "pattern",
				Color:        "auto",
				WithFilename: true,
				NoFilename:   false,
				LineNumber:   true,
			},
		},
		{
			name: "context override: -A 1 -C 3",
			args: []string{"-A", "1", "-C", "3", "pattern"},
			expected: &CliArgs{
				Pattern:       "pattern",
				Color:         "auto",
				BeforeContext: 3,
				AfterContext:  3,
				LineNumber:    true,
			},
		},
		{
			name: "context override: -C 3 -A 1",
			args: []string{"-C", "3", "-A", "1", "pattern"},
			expected: &CliArgs{
				Pattern:       "pattern",
				Color:         "auto",
				BeforeContext: 3,
				AfterContext:  1,
				LineNumber:    true,
			},
		},
		{
			name: "combination of short flags",
			args: []string{"-iwF", "pattern", "path1", "path2"},
			expected: &CliArgs{
				Pattern:         "pattern",
				Paths:           []string{"path1", "path2"},
				Color:           "auto",
				CaseInsensitive: true,
				WordRegexp:      true,
				FixedStrings:    true,
				LineNumber:      true,
			},
		},
		{
			name: "glob patterns accumulated",
			args: []string{"-g", "*.log", "--glob", "!error.log", "pattern"},
			expected: &CliArgs{
				Pattern:    "pattern",
				Color:      "auto",
				Globs:      []string{"*.log", "!error.log"},
				LineNumber: true,
			},
		},
		{
			name: "only-matching short and long flags",
			args: []string{"-o", "--only-matching", "pattern"},
			expected: &CliArgs{
				Pattern:      "pattern",
				Color:        "auto",
				OnlyMatching: true,
				LineNumber:   true,
			},
		},
		{
			name: "count short and long flags",
			args: []string{"-c", "--count", "pattern"},
			expected: &CliArgs{
				Pattern:    "pattern",
				Color:      "auto",
				Count:      true,
				LineNumber: true,
			},
		},
		{
			name: "quiet short and long flags",
			args: []string{"-q", "--quiet", "pattern"},
			expected: &CliArgs{
				Pattern:    "pattern",
				Color:      "auto",
				Quiet:      true,
				LineNumber: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseArgs(tc.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Compare relevant fields
			if !reflect.DeepEqual(got.Pattern, tc.expected.Pattern) {
				t.Errorf("Pattern mismatch: got %v, expected %v", got.Pattern, tc.expected.Pattern)
			}
			if !reflect.DeepEqual(got.Paths, tc.expected.Paths) {
				t.Errorf("Paths mismatch: got %v, expected %v", got.Paths, tc.expected.Paths)
			}
			if got.CaseInsensitive != tc.expected.CaseInsensitive {
				t.Errorf("CaseInsensitive mismatch: got %v, expected %v", got.CaseInsensitive, tc.expected.CaseInsensitive)
			}
			if got.LineNumber != tc.expected.LineNumber || got.NoLineNumber != tc.expected.NoLineNumber {
				t.Errorf("LineNumber settings mismatch: got (LineNumber=%v, NoLineNumber=%v), expected (LineNumber=%v, NoLineNumber=%v)",
					got.LineNumber, got.NoLineNumber, tc.expected.LineNumber, tc.expected.NoLineNumber)
			}
			if got.WithFilename != tc.expected.WithFilename || got.NoFilename != tc.expected.NoFilename {
				t.Errorf("Filename settings mismatch: got (WithFilename=%v, NoFilename=%v), expected (WithFilename=%v, NoFilename=%v)",
					got.WithFilename, got.NoFilename, tc.expected.WithFilename, tc.expected.NoFilename)
			}
			if got.BeforeContext != tc.expected.BeforeContext || got.AfterContext != tc.expected.AfterContext {
				t.Errorf("Context mismatch: got (B=%d, A=%d), expected (B=%d, A=%d)",
					got.BeforeContext, got.AfterContext, tc.expected.BeforeContext, tc.expected.AfterContext)
			}
			if !reflect.DeepEqual(got.Globs, tc.expected.Globs) {
				t.Errorf("Globs mismatch: got %v, expected %v", got.Globs, tc.expected.Globs)
			}
			if got.OnlyMatching != tc.expected.OnlyMatching {
				t.Errorf("OnlyMatching mismatch: got %v, expected %v", got.OnlyMatching, tc.expected.OnlyMatching)
			}
			if got.Count != tc.expected.Count {
				t.Errorf("Count mismatch: got %v, expected %v", got.Count, tc.expected.Count)
			}
			if got.Quiet != tc.expected.Quiet {
				t.Errorf("Quiet mismatch: got %v, expected %v", got.Quiet, tc.expected.Quiet)
			}
		})
	}
}
