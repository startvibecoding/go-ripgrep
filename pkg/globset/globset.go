package globset

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Glob represents a single compiled glob pattern.
type Glob struct {
	Original  string
	Regexp    *regexp.Regexp
	IsNegated bool
}

// NewGlob compiles a glob pattern into a Glob structure.
func NewGlob(pattern string) (*Glob, error) {
	isNegated := false
	if strings.HasPrefix(pattern, "!") {
		isNegated = true
		pattern = pattern[1:]
	}

	// Normalize windows separators
	pattern = filepath.ToSlash(pattern)

	regexStr, err := GlobToRegex(pattern)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, err
	}

	return &Glob{
		Original:  pattern,
		Regexp:    re,
		IsNegated: isNegated,
	}, nil
}

// Match checks if the path matches the glob pattern.
func (g *Glob) Match(path string) bool {
	path = filepath.ToSlash(path)
	return g.Regexp.MatchString(path)
}

// GlobToRegex translates a gitignore-style glob pattern into a Go regex pattern.
func GlobToRegex(pattern string) (string, error) {
	var sb strings.Builder

	isAnchored := strings.HasPrefix(pattern, "/")
	trimmed := pattern
	if isAnchored {
		trimmed = pattern[1:]
		pattern = trimmed
	}
	// Check for middle slash (after stripping leading/trailing slashes)
	if strings.Contains(strings.TrimSuffix(trimmed, "/"), "/") {
		isAnchored = true
	}

	if !isAnchored {
		// If not anchored, it matches any component of the path.
		sb.WriteString(`(?:^|/)`)
	} else {
		// If anchored, it starts matching from the root.
		sb.WriteString(`^`)
	}

	runes := []rune(pattern)
	n := len(runes)
	inBracket := false
	for i := 0; i < n; i++ {
		r := runes[i]
		switch r {
		case '*':
			if i+1 < n && runes[i+1] == '*' {
				i++
				if i+1 < n && runes[i+1] == '/' {
					i++
					// '**/': match zero or more directory levels
					sb.WriteString(`(?:.*/)?`)
				} else {
					// '**': match everything
					sb.WriteString(`.*`)
				}
			} else {
				// '*': match anything within a single directory level
				sb.WriteString(`[^/]*`)
			}
		case '?':
			sb.WriteString(`[^/]`)
		case '[':
			inBracket = true
			sb.WriteRune('[')
			if i+1 < n && runes[i+1] == '!' {
				sb.WriteRune('^')
				i++
			}
		case ']':
			inBracket = false
			sb.WriteRune(']')
		case '\\':
			if i+1 < n {
				i++
				sb.WriteString(regexp.QuoteMeta(string(runes[i])))
			} else {
				sb.WriteString(`\\`)
			}
		case '.', '+', '$', '^', '(', ')', '|', '{', '}':
			if inBracket {
				sb.WriteRune(r)
			} else {
				sb.WriteString(`\` + string(r))
			}
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteString(`$`)
	return sb.String(), nil
}

// GlobSet is a collection of compiled Glob patterns.
type GlobSet struct {
	globs []*Glob
}

// NewGlobSet compiles a list of glob patterns.
func NewGlobSet(patterns []string) (*GlobSet, error) {
	var globs []*Glob
	for _, pat := range patterns {
		if pat == "" || strings.HasPrefix(pat, "#") {
			// skip empty lines and comments
			continue
		}
		g, err := NewGlob(pat)
		if err != nil {
			return nil, err
		}
		globs = append(globs, g)
	}
	return &GlobSet{globs: globs}, nil
}

// Match checks the path against all glob patterns in the set.
// If a negated pattern matches, it overrides normal matches.
// Returns (matched, isIgnored). isIgnored is true if a match dictates the path should be ignored.
// In gitignore logic, the last matching pattern in the file takes precedence.
func (gs *GlobSet) Match(path string) (matched bool, isIgnored bool) {
	for i := len(gs.globs) - 1; i >= 0; i-- {
		g := gs.globs[i]
		if g.Match(path) {
			if g.IsNegated {
				// Negated match: means do NOT ignore/exclude
				return true, false
			}
			// Regular match: means DO ignore/exclude
			return true, true
		}
	}
	return false, false
}

// MatchPath checks if a path is matched, including all of its parent directories
// (which allows directory-level ignore/exclude patterns to work properly).
func (gs *GlobSet) MatchPath(path string) (matched bool, isIgnored bool) {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")

	var current string
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "" {
			continue
		}
		if current == "" {
			current = parts[i]
		} else {
			current = current + "/" + parts[i]
		}

		// Check with trailing slash (e.g. "bin/")
		if m, ig := gs.Match(current + "/"); m {
			if ig {
				return true, true
			}
		}
		// Check without trailing slash (e.g. "bin")
		if m, ig := gs.Match(current); m {
			if ig {
				return true, true
			}
		}
	}

	return gs.Match(path)
}

// MatchGlobFilter checks if a path should be ignored according to ripgrep's -g/--glob rules.
// 1. If there are negated globs (starting with '!') and the path matches one, it is ignored (returns true).
// 2. If there are positive globs:
//   - If the path matches a positive glob, it is NOT ignored (returns false).
//   - If it does not match any positive glob, it IS ignored (returns true).
//
// 3. Otherwise, it is NOT ignored (returns false).
func (gs *GlobSet) MatchGlobFilter(path string) bool {
	if len(gs.globs) == 0 {
		return false
	}

	path = filepath.ToSlash(path)

	// Check if there are any positive globs in the set
	hasPositive := false
	for _, g := range gs.globs {
		if !g.IsNegated {
			hasPositive = true
			break
		}
	}

	// 1. If path matches a negated glob, it is ignored/excluded
	for _, g := range gs.globs {
		if g.IsNegated {
			if g.Match(path) {
				return true
			}
		}
	}

	// 2. If there are positive globs, path must match at least one to be included
	if hasPositive {
		matchedPositive := false
		for _, g := range gs.globs {
			if !g.IsNegated {
				if g.Match(path) {
					matchedPositive = true
					break
				}
			}
		}
		if !matchedPositive {
			return true // ignored because it didn't match any positive glob
		}
	}

	return false
}
