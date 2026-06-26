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

func TestIgnoreStackParentClimbing(t *testing.T) {
	// Create a nested temporary directory structure
	tmpDir, err := os.MkdirTemp("", "ignore-climb-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested directories: tmpDir/parent/child
	parentDir := filepath.Join(tmpDir, "parent")
	childDir := filepath.Join(parentDir, "child")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create childDir: %v", err)
	}

	// Write .gitignore in parentDir
	parentGitignore := filepath.Join(parentDir, ".gitignore")
	if err := os.WriteFile(parentGitignore, []byte("*.tmp\n"), 0644); err != nil {
		t.Fatalf("failed to write parent .gitignore: %v", err)
	}

	// Write .ignore in childDir
	childIgnore := filepath.Join(childDir, ".ignore")
	if err := os.WriteFile(childIgnore, []byte("!special.tmp\n"), 0644); err != nil {
		t.Fatalf("failed to write child .ignore: %v", err)
	}

	// Setup stack with LoadBaseRules starting from childDir
	stack := NewIgnoreStack(false, false, 0)
	stack.LoadBaseRules(childDir)

	// We also manually push childDir to simulate the walker's push on the search target
	if err := stack.Push(childDir); err != nil {
		t.Fatalf("failed to push childDir: %v", err)
	}

	// Check if a .tmp file in childDir is ignored due to parent's .gitignore
	testTmp := filepath.Join(childDir, "test.tmp")
	if !stack.IsIgnored(testTmp, false) {
		t.Errorf("expected %s to be ignored due to parent's .gitignore", testTmp)
	}

	// Check if special.tmp is NOT ignored due to child's .ignore negation
	specialTmp := filepath.Join(childDir, "special.tmp")
	if stack.IsIgnored(specialTmp, false) {
		t.Errorf("expected %s NOT to be ignored due to child's .ignore negation", specialTmp)
	}
}
