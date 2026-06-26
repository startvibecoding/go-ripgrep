package matcher

import (
	"bytes"
	"regexp"
)

// Matcher represents a text search matcher.
type Matcher interface {
	// Match returns true if there is any match in the line.
	Match(line []byte) bool
	// FindSpans returns all matched byte offsets [start, end] in the line.
	FindSpans(line []byte) [][2]int
}

// RegexMatcher matches lines using a regular expression.
type RegexMatcher struct {
	re *regexp.Regexp
}

func NewRegexMatcher(re *regexp.Regexp) *RegexMatcher {
	return &RegexMatcher{re: re}
}

func (m *RegexMatcher) Match(line []byte) bool {
	return m.re.Match(line)
}

func (m *RegexMatcher) FindSpans(line []byte) [][2]int {
	matches := m.re.FindAllSubmatchIndex(line, -1)
	if len(matches) == 0 {
		return nil
	}
	spans := make([][2]int, len(matches))
	for i, match := range matches {
		spans[i] = [2]int{match[0], match[1]}
	}
	return spans
}

// FixedMatcher matches lines using sub-string search.
type FixedMatcher struct {
	pattern         []byte
	patternStr      string
	caseInsensitive bool
}

func NewFixedMatcher(pattern string, caseInsensitive bool) *FixedMatcher {
	return &FixedMatcher{
		pattern:         []byte(pattern),
		patternStr:      pattern,
		caseInsensitive: caseInsensitive,
	}
}

func (m *FixedMatcher) Match(line []byte) bool {
	if !m.caseInsensitive {
		return bytes.Contains(line, m.pattern)
	}
	return bytes.Contains(bytes.ToLower(line), bytes.ToLower(m.pattern))
}

func (m *FixedMatcher) FindSpans(line []byte) [][2]int {
	var spans [][2]int
	if !m.caseInsensitive {
		start := 0
		for {
			idx := bytes.Index(line[start:], m.pattern)
			if idx == -1 {
				break
			}
			absStart := start + idx
			absEnd := absStart + len(m.pattern)
			spans = append(spans, [2]int{absStart, absEnd})
			start = absEnd
			if len(m.pattern) == 0 {
				break
			}
		}
	} else {
		lowerLine := bytes.ToLower(line)
		lowerPattern := bytes.ToLower(m.pattern)
		start := 0
		for {
			idx := bytes.Index(lowerLine[start:], lowerPattern)
			if idx == -1 {
				break
			}
			absStart := start + idx
			absEnd := absStart + len(m.pattern)
			spans = append(spans, [2]int{absStart, absEnd})
			start = absEnd
			if len(m.pattern) == 0 {
				break
			}
		}
	}
	return spans
}

// BuildMatcher builds a matcher according to flags.
func BuildMatcher(pattern string, isFixed, caseInsensitive, wordRegexp bool) (Matcher, error) {
	if isFixed && !wordRegexp {
		return NewFixedMatcher(pattern, caseInsensitive), nil
	}

	pat := pattern
	if isFixed {
		pat = regexp.QuoteMeta(pat)
	}
	if wordRegexp {
		// Wrap word boundaries. In Go regexp, \b is supported.
		// Note that \b works on word characters (letters, digits, underscore).
		pat = `\b` + pat + `\b`
	}
	if caseInsensitive {
		pat = `(?i)` + pat
	}

	re, err := regexp.Compile(pat)
	if err != nil {
		return nil, err
	}
	return NewRegexMatcher(re), nil
}
