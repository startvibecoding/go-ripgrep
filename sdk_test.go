package goriggrep

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestSDKSearchPositiveGlobTraversesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	targetFile := filepath.Join(nestedDir, "main.go")
	if err := os.WriteFile(targetFile, []byte("package main\n// TODO: fix me\n"), 0644); err != nil {
		t.Fatalf("failed to write target file: %v", err)
	}

	resultsChan, err := Search(context.Background(), []string{tmpDir}, Options{
		Pattern: "TODO",
		Globs:   []string{"*.go"},
	})
	if err != nil {
		t.Fatalf("search failed to initialize: %v", err)
	}

	var paths []string
	for res := range resultsChan {
		paths = append(paths, res.Path)
	}

	if !reflect.DeepEqual(paths, []string{targetFile}) {
		t.Fatalf("expected only %q, got %v", targetFile, paths)
	}
}

func TestSDKSearchExplicitFileRespectsTypeFilters(t *testing.T) {
	tmpDir := t.TempDir()
	textFile := filepath.Join(tmpDir, "notes.txt")
	if err := os.WriteFile(textFile, []byte("needle\n"), 0644); err != nil {
		t.Fatalf("failed to write text file: %v", err)
	}

	resultsChan, err := Search(context.Background(), []string{textFile}, Options{
		Pattern: "needle",
		Types:   []string{"go"},
	})
	if err != nil {
		t.Fatalf("search failed to initialize: %v", err)
	}

	for res := range resultsChan {
		t.Fatalf("expected no results, got %+v", res)
	}
}

func TestSortDirEntriesReversePath(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"alpha.txt", "beta.txt", "gamma.txt"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(name), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}

	sortDirEntries(tmpDir, entries, "path", true)

	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	expected := []string{"gamma.txt", "beta.txt", "alpha.txt"}
	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("expected %v, got %v", expected, names)
	}
}

func TestSDKSearchCancellationStopsWork(t *testing.T) {
	tmpDir := t.TempDir()
	largeFile := filepath.Join(tmpDir, "large.txt")
	content := strings.Repeat("line without match\n", 200000)
	if err := os.WriteFile(largeFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write large file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	resultsChan, err := Search(ctx, []string{largeFile}, Options{
		Pattern: "never-matches",
		Threads: 1,
	})
	if err != nil {
		t.Fatalf("search failed to initialize: %v", err)
	}

	cancel()

	done := make(chan struct{})
	go func() {
		for range resultsChan {
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("search did not stop promptly after cancellation")
	}
}
