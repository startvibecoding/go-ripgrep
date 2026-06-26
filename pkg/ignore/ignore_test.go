package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIgnoreStack(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "ignore-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Write gitignore in root
	rootGitignore := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(rootGitignore, []byte("*.log\n!important.log\n"), 0644); err != nil {
		t.Fatalf("failed to write .gitignore: %v", err)
	}

	// Write .rgignore in subdir
	subdirRgignore := filepath.Join(subDir, ".rgignore")
	if err := os.WriteFile(subdirRgignore, []byte("important.log\n"), 0644); err != nil {
		t.Fatalf("failed to write subdir .rgignore: %v", err)
	}

	// Setup stack
	stack := NewIgnoreStack(false, false, 0)

	// Push root
	if err := stack.Push(tmpDir); err != nil {
		t.Fatalf("failed to push root: %v", err)
	}

	// Check standard ignore
	normalLog := filepath.Join(tmpDir, "normal.log")
	if !stack.IsIgnored(normalLog, false) {
		t.Errorf("expected %s to be ignored by root .gitignore", normalLog)
	}

	// Check negation (override) in root
	importantLog := filepath.Join(tmpDir, "important.log")
	if stack.IsIgnored(importantLog, false) {
		t.Errorf("expected %s NOT to be ignored due to negation in root .gitignore", importantLog)
	}

	// Check hidden files
	hiddenFile := filepath.Join(tmpDir, ".hidden")
	if !stack.IsIgnored(hiddenFile, false) {
		t.Errorf("expected %s to be ignored because it is hidden", hiddenFile)
	}

	// Push subdir
	subStack := stack.Clone()
	if err := subStack.Push(subDir); err != nil {
		t.Fatalf("failed to push subdir: %v", err)
	}

	// Check that subdir's .rgignore overrides root .gitignore's negation
	subdirImportantLog := filepath.Join(subDir, "important.log")
	if !subStack.IsIgnored(subdirImportantLog, false) {
		t.Errorf("expected %s to be ignored due to subdir .rgignore override", subdirImportantLog)
	}
}
