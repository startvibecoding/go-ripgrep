package ignore

import (
	"bufio"
	"bytes"
	"github.com/startvibecoding/go-ripgrep/pkg/globset"
	"os"
	"os/exec"
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

// LoadBaseRules searches for global ignore files and climbs parent directories of startPath
// to pre-populate the ignore stack.
func (s *IgnoreStack) LoadBaseRules(startPath string) {
	if s.NoIgnore {
		return
	}

	// 1. Load global gitignore if available (as the lowest level / Level 0)
	globalPath := getGlobalGitIgnorePath()
	if globalPath != "" {
		if patterns, err := parseIgnoreFile(globalPath); err == nil {
			if gs, err := globset.NewGlobSet(patterns); err == nil {
				s.Levels = append(s.Levels, IgnoreLevel{
					DirPath:   filepath.Dir(globalPath),
					GitIgnore: gs,
				})
			}
		}
	}

	// 2. Climb parents of startPath to find intermediate ignore files
	abs, err := filepath.Abs(startPath)
	if err != nil {
		return
	}

	var dirs []string
	curr := filepath.Dir(abs)
	for {
		dirs = append(dirs, curr)
		parent := filepath.Dir(curr)
		if parent == curr {
			break // hit root
		}
		// Stop climbing if we find a .git directory boundary
		if _, err := os.Stat(filepath.Join(curr, ".git")); err == nil {
			break
		}
		curr = parent
	}

	// Load from shallowest (furthest parent) to deepest (parent of startPath)
	for i := len(dirs) - 1; i >= 0; i-- {
		dir := dirs[i]
		level := IgnoreLevel{DirPath: dir}
		hasRules := false

		gitPath := filepath.Join(dir, ".gitignore")
		if patterns, err := parseIgnoreFile(gitPath); err == nil {
			if gs, err := globset.NewGlobSet(patterns); err == nil {
				level.GitIgnore = gs
				hasRules = true
			}
		}

		ignorePath := filepath.Join(dir, ".ignore")
		if patterns, err := parseIgnoreFile(ignorePath); err == nil {
			if gs, err := globset.NewGlobSet(patterns); err == nil {
				level.Ignore = gs
				hasRules = true
			}
		}

		rgPath := filepath.Join(dir, ".rgignore")
		if patterns, err := parseIgnoreFile(rgPath); err == nil {
			if gs, err := globset.NewGlobSet(patterns); err == nil {
				level.RgIgnore = gs
				hasRules = true
			}
		}

		if hasRules {
			s.Levels = append(s.Levels, level)
		}
	}
}

func getGlobalGitIgnorePath() string {
	// Check if git config --global core.excludesfile is set
	cmd := exec.Command("git", "config", "--global", "core.excludesfile")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		path := strings.TrimSpace(out.String())
		if path != "" {
			if strings.HasPrefix(path, "~") {
				home, _ := os.UserHomeDir()
				path = filepath.Join(home, path[1:])
			}
			return path
		}
	}

	// Default fallbacks
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	p := filepath.Join(home, ".config", "git", "ignore")
	if _, err := os.Stat(p); err == nil {
		return p
	}

	p = filepath.Join(home, ".gitignore")
	if _, err := os.Stat(p); err == nil {
		return p
	}

	return ""
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
