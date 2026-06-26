package goriggrep

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestSDKSearch(t *testing.T) {
	// Create a temp workspace
	tmpDir, err := os.MkdirTemp("", "goriggrep-sdk-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some files
	fileA := filepath.Join(tmpDir, "fileA.txt")
	if err := os.WriteFile(fileA, []byte("hello world\nlearning golang\n"), 0644); err != nil {
		t.Fatalf("failed to write fileA: %v", err)
	}

	fileB := filepath.Join(tmpDir, "fileB.log")
	if err := os.WriteFile(fileB, []byte("error: connection reset\nhello server\n"), 0644); err != nil {
		t.Fatalf("failed to write fileB: %v", err)
	}

	// Create an ignored file
	fileC := filepath.Join(tmpDir, "fileC.tmp")
	if err := os.WriteFile(fileC, []byte("hello hidden\n"), 0644); err != nil {
		t.Fatalf("failed to write fileC: %v", err)
	}

	// Create .gitignore
	gitignore := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("*.tmp\n"), 0644); err != nil {
		t.Fatalf("failed to write .gitignore: %v", err)
	}

	// Run SDK Search
	opts := Options{
		Pattern: "hello",
	}

	resultsChan, err := Search(context.Background(), []string{tmpDir}, opts)
	if err != nil {
		t.Fatalf("search failed to initialize: %v", err)
	}

	var results []string
	for res := range resultsChan {
		results = append(results, res.Path)
		if len(res.Matches) == 0 {
			t.Errorf("file result had no matches: %s", res.Path)
		}
	}

	// We expect fileA.txt and fileB.log to be found. fileC.tmp should be ignored.
	if len(results) != 2 {
		t.Errorf("expected 2 files with matches, got %d: %v", len(results), results)
	}

	// Run search with glob exclusion
	optsWithGlob := Options{
		Pattern: "hello",
		Globs:   []string{"!*.log"},
	}

	resultsChan2, err := Search(context.Background(), []string{tmpDir}, optsWithGlob)
	if err != nil {
		t.Fatalf("search with glob failed to initialize: %v", err)
	}

	results2 := 0
	for range resultsChan2 {
		results2++
	}

	// With !*.log, only fileA.txt should be matched (fileC.tmp still ignored by gitignore).
	if results2 != 1 {
		t.Errorf("expected 1 file with matches under glob exclusion, got %d", results2)
	}
}

func BenchmarkSearch(b *testing.B) {
	// Create a temp workspace with 50 files
	tmpDir, err := os.MkdirTemp("", "goriggrep-bench-")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for i := 0; i < 50; i++ {
		filePath := filepath.Join(tmpDir, "file_"+strconv.Itoa(i)+".txt")
		content := strings.Repeat("hello world and standard golang package\nthis is a test line\nanother line with match here\n", 100)
		_ = os.WriteFile(filePath, []byte(content), 0644)
	}

	opts := Options{
		Pattern: "golang",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultsChan, err := Search(context.Background(), []string{tmpDir}, opts)
		if err != nil {
			b.Fatalf("failed: %v", err)
		}
		for range resultsChan {
		}
	}
}
