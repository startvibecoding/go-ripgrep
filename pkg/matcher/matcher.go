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
	// Replace returns the line with matches replaced, along with the new spans [start, end] for each replacement.
	Replace(line []byte, replacement []byte) (replacedLine []byte, newSpans [][2]int)
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

func (m *RegexMatcher) Replace(line []byte, replacement []byte) ([]byte, [][2]int) {
	indices := m.re.FindAllSubmatchIndex(line, -1)
	if len(indices) == 0 {
		return line, nil
	}

	var result []byte
	var newSpans [][2]int
	lastIdx := 0
	currOffset := 0

	for _, match := range indices {
		result = append(result, line[lastIdx:match[0]]...)

		expanded := m.re.Expand(nil, replacement, line, match)
		startInReplaced := match[0] + currOffset
		endInReplaced := startInReplaced + len(expanded)
		newSpans = append(newSpans, [2]int{startInReplaced, endInReplaced})

		result = append(result, expanded...)
		currOffset += len(expanded) - (match[1] - match[0])
		lastIdx = match[1]
	}
	result = append(result, line[lastIdx:]...)
	return result, newSpans
}

// FixedMatcher matches lines using sub-string search.
type FixedMatcher struct {
	pattern         []byte
	patternStr      string
	caseInsensitive bool

	isPatASCII      bool
	lowerFirst      byte
	upperFirst      byte
	lowerPattern    []byte
}

func isASCII(b []byte) bool {
	for _, c := range b {
		if c >= 128 {
			return false
		}
	}
	return true
}

func NewFixedMatcher(pattern string, caseInsensitive bool) *FixedMatcher {
	patBytes := []byte(pattern)
	fm := &FixedMatcher{
		pattern:         patBytes,
		patternStr:      pattern,
		caseInsensitive: caseInsensitive,
	}

	if caseInsensitive {
		fm.isPatASCII = isASCII(patBytes)
		if len(patBytes) > 0 {
			c := patBytes[0]
			if c >= 'a' && c <= 'z' {
				fm.lowerFirst = c
				fm.upperFirst = c - 32
			} else if c >= 'A' && c <= 'Z' {
				fm.lowerFirst = c + 32
				fm.upperFirst = c
			} else {
				fm.lowerFirst = c
				fm.upperFirst = c
			}
		}
		fm.lowerPattern = bytes.ToLower(patBytes)
	}
	return fm
}

func asciiEqualFold(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca |= 0x20
		}
		if cb >= 'A' && cb <= 'Z' {
			cb |= 0x20
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func (m *FixedMatcher) Match(line []byte) bool {
	if !m.caseInsensitive {
		return bytes.Contains(line, m.pattern)
	}

	if len(m.pattern) == 0 {
		return true
	}

	// Optimize for ASCII matching
	if m.isPatASCII && isASCII(line) {
		start := 0
		patLen := len(m.pattern)
		for {
			idx := IndexByte2(line[start:], m.lowerFirst, m.upperFirst)
			if idx == -1 {
				break
			}
			absStart := start + idx
			if absStart+patLen > len(line) {
				break
			}
			if asciiEqualFold(line[absStart:absStart+patLen], m.lowerPattern) {
				return true
			}
			start = absStart + 1
		}
		return false
	}

	// Fallback to standard Unicode case conversion if non-ASCII detected
	return bytes.Contains(bytes.ToLower(line), m.lowerPattern)
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
		return spans
	}

	if len(m.pattern) == 0 {
		return [][2]int{{0, 0}}
	}

	// Optimize for ASCII matching
	if m.isPatASCII && isASCII(line) {
		start := 0
		patLen := len(m.pattern)
		for {
			idx := IndexByte2(line[start:], m.lowerFirst, m.upperFirst)
			if idx == -1 {
				break
			}
			absStart := start + idx
			absEnd := absStart + patLen
			if absEnd > len(line) {
				break
			}
			if asciiEqualFold(line[absStart:absEnd], m.lowerPattern) {
				spans = append(spans, [2]int{absStart, absEnd})
				start = absEnd
			} else {
				start = absStart + 1
			}
		}
		return spans
	}

	// Fallback to standard Unicode case conversion
	lowerLine := bytes.ToLower(line)
	start := 0
	for {
		idx := bytes.Index(lowerLine[start:], m.lowerPattern)
		if idx == -1 {
			break
		}
		absStart := start + idx
		absEnd := absStart + len(m.pattern)
		spans = append(spans, [2]int{absStart, absEnd})
		start = absEnd
	}
	return spans
}

func (m *FixedMatcher) Replace(line []byte, replacement []byte) ([]byte, [][2]int) {
	spans := m.FindSpans(line)
	if len(spans) == 0 {
		return line, nil
	}

	var result []byte
	var newSpans [][2]int
	lastIdx := 0
	currOffset := 0

	for _, span := range spans {
		result = append(result, line[lastIdx:span[0]]...)

		startInReplaced := span[0] + currOffset
		endInReplaced := startInReplaced + len(replacement)
		newSpans = append(newSpans, [2]int{startInReplaced, endInReplaced})

		result = append(result, replacement...)
		currOffset += len(replacement) - (span[1] - span[0])
		lastIdx = span[1]
	}
	result = append(result, line[lastIdx:]...)
	return result, newSpans
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
