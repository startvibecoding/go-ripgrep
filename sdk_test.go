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

func TestSDKSearchSortByPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in unsorted order
	files := []string{"c.txt", "a.txt", "b.txt"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("needle\n"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	resultsChan, err := Search(context.Background(), []string{tmpDir}, Options{
		Pattern: "needle",
		SortBy:  "path",
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	var paths []string
	for res := range resultsChan {
		paths = append(paths, filepath.Base(res.Path))
	}

	if len(paths) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(paths), paths)
	}

	// Should be sorted alphabetically
	if paths[0] != "a.txt" || paths[1] != "b.txt" || paths[2] != "c.txt" {
		t.Errorf("expected sorted order [a.txt, b.txt, c.txt], got %v", paths)
	}
}

func TestSDKSearchSortReverse(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{"c.txt", "a.txt", "b.txt"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("needle\n"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	resultsChan, err := Search(context.Background(), []string{tmpDir}, Options{
		Pattern:     "needle",
		SortBy:      "path",
		SortReverse: true,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	var paths []string
	for res := range resultsChan {
		paths = append(paths, filepath.Base(res.Path))
	}

	if len(paths) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(paths), paths)
	}

	if paths[0] != "c.txt" || paths[1] != "b.txt" || paths[2] != "a.txt" {
		t.Errorf("expected reverse sorted order [c.txt, b.txt, a.txt], got %v", paths)
	}
}

func TestSDKSearchReplace(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello world\nfoo bar\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	resultsChan, err := Search(context.Background(), []string{tmpDir}, Options{
		Pattern:    "world",
		IsFixed:    true,
		Replace:    "earth",
		HasReplace: true,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	var found bool
	for res := range resultsChan {
		found = true
		if len(res.Matches) != 1 {
			t.Errorf("expected 1 match, got %d", len(res.Matches))
		}
		if res.Matches[0].Line != "hello earth" {
			t.Errorf("expected 'hello earth', got %q", res.Matches[0].Line)
		}
	}

	if !found {
		t.Error("expected to find a match")
	}
}

func TestSDKSearchMaxDepth(t *testing.T) {
	tmpDir := t.TempDir()

	nestedDir := filepath.Join(tmpDir, "level1", "level2", "level3")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dirs: %v", err)
	}

	// Write a matching file at each level
	for _, dir := range []string{tmpDir, filepath.Join(tmpDir, "level1"), nestedDir} {
		if err := os.WriteFile(filepath.Join(dir, "match.txt"), []byte("needle\n"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	resultsChan, err := Search(context.Background(), []string{tmpDir}, Options{
		Pattern:  "needle",
		MaxDepth: 1,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	count := 0
	for range resultsChan {
		count++
	}

	// With MaxDepth=1, we should only find the file in tmpDir root
	// (level1/matches.txt is at depth 2, so it's skipped)
	if count != 1 {
		t.Errorf("expected 1 file with MaxDepth=1, got %d", count)
	}

	// With MaxDepth=2, we should also find the file in level1
	resultsChan2, err := Search(context.Background(), []string{tmpDir}, Options{
		Pattern:  "needle",
		MaxDepth: 2,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	count2 := 0
	for range resultsChan2 {
		count2++
	}

	if count2 != 2 {
		t.Errorf("expected 2 files with MaxDepth=2, got %d", count2)
	}
}
