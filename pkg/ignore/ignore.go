package ignore

import (
	"bufio"
	"go-ripgrep/pkg/globset"
	"os"
	"path/filepath"
	"strings"
)

// IgnoreLevel represents the compiled ignore patterns for a single directory.
type IgnoreLevel struct {
	DirPath   string
	GitIgnore *globset.GlobSet
	Ignore    *globset.GlobSet
	RgIgnore  *globset.GlobSet
}

// IgnoreStack manages nested ignore levels.
type IgnoreStack struct {
	Levels   []IgnoreLevel
	NoIgnore bool // If true, ignore files (.gitignore, etc.) are bypassed
	Hidden   bool // If true, search hidden files/directories
	MaxDepth int  // Maximum depth to search (0 means infinite)
}

// NewIgnoreStack creates an empty IgnoreStack.
func NewIgnoreStack(noIgnore, hidden bool, maxDepth int) *IgnoreStack {
	return &IgnoreStack{
		Levels:   nil,
		NoIgnore: noIgnore,
		Hidden:   hidden,
		MaxDepth: maxDepth,
	}
}

// Clone creates a copy of the stack, useful when walking different subdirectories.
func (s *IgnoreStack) Clone() *IgnoreStack {
	levels := make([]IgnoreLevel, len(s.Levels))
	copy(levels, s.Levels)
	return &IgnoreStack{
		Levels:   levels,
		NoIgnore: s.NoIgnore,
		Hidden:   s.Hidden,
		MaxDepth: s.MaxDepth,
	}
}

// Push adds ignore rules for a directory to the stack.
func (s *IgnoreStack) Push(dirPath string) error {
	if s.NoIgnore {
		return nil
	}

	level := IgnoreLevel{DirPath: dirPath}

	// Read .gitignore
	gitPath := filepath.Join(dirPath, ".gitignore")
	if patterns, err := parseIgnoreFile(gitPath); err == nil {
		if gs, err := globset.NewGlobSet(patterns); err == nil {
			level.GitIgnore = gs
		}
	}

	// Read .ignore
	ignorePath := filepath.Join(dirPath, ".ignore")
	if patterns, err := parseIgnoreFile(ignorePath); err == nil {
		if gs, err := globset.NewGlobSet(patterns); err == nil {
			level.Ignore = gs
		}
	}

	// Read .rgignore
	rgPath := filepath.Join(dirPath, ".rgignore")
	if patterns, err := parseIgnoreFile(rgPath); err == nil {
		if gs, err := globset.NewGlobSet(patterns); err == nil {
			level.RgIgnore = gs
		}
	}

	s.Levels = append(s.Levels, level)
	return nil
}

// Pop removes the deepest ignore level.
func (s *IgnoreStack) Pop() {
	if len(s.Levels) > 0 {
		s.Levels = s.Levels[:len(s.Levels)-1]
	}
}

// IsIgnored checks if the given path should be ignored.
// It checks rules from the deepest level to the root level.
func (s *IgnoreStack) IsIgnored(path string, isDir bool) bool {
	filename := filepath.Base(path)

	// 1. Check hidden rules
	if !s.Hidden && strings.HasPrefix(filename, ".") && filename != "." && filename != ".." {
		return true
	}

	if s.NoIgnore {
		return false
	}

	// 2. Check each level of ignore stack, deepest first
	for i := len(s.Levels) - 1; i >= 0; i-- {
		level := s.Levels[i]

		// Compute relative path from the ignore file's directory
		rel, err := filepath.Rel(level.DirPath, path)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if isDir {
			rel = rel + "/"
		}

		// Ripgrep priority: .rgignore > .ignore > .gitignore
		if level.RgIgnore != nil {
			if matched, isIgnored := level.RgIgnore.MatchPath(rel); matched {
				return isIgnored
			}
		}

		if level.Ignore != nil {
			if matched, isIgnored := level.Ignore.MatchPath(rel); matched {
				return isIgnored
			}
		}

		if level.GitIgnore != nil {
			if matched, isIgnored := level.GitIgnore.MatchPath(rel); matched {
				return isIgnored
			}
		}
	}

	return false
}

// parseIgnoreFile reads an ignore file and returns valid patterns.
func parseIgnoreFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, scanner.Err()
}
