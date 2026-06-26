package ignore

import (
	"path/filepath"
	"strings"
)

// BuiltInTypes maps type names to their list of glob patterns.
var BuiltInTypes = map[string][]string{
	"asm":      {"*.asm", "*.s", "*.S"},
	"c":        {"*.c", "*.h"},
	"cpp":      {"*.cpp", "*.cc", "*.cxx", "*.c++", "*.h", "*.hpp", "*.hxx", "*.h++"},
	"css":      {"*.css"},
	"go":       {"*.go"},
	"html":     {"*.html", "*.htm", "*.xhtml"},
	"java":     {"*.java", "*.jsp"},
	"js":       {"*.js", "*.jsx", "*.mjs", "*.cjs"},
	"json":     {"*.json", "*.ipynb"},
	"markdown": {"*.md", "*.markdown", "*.mdown", "*.mkdn"},
	"python":   {"*.py", "*.pyi"},
	"rust":     {"*.rs"},
	"ts":       {"*.ts", "*.tsx", "*.mts", "*.cts"},
	"yaml":     {"*.yaml", "*.yml"},
}

// MatchType checks if a filename matches any glob pattern associated with a type.
func MatchType(filename string, typeName string) bool {
	globs, ok := BuiltInTypes[strings.ToLower(typeName)]
	if !ok {
		return false
	}
	for _, glob := range globs {
		matched, _ := filepath.Match(glob, filename)
		if matched {
			return true
		}
	}
	return false
}

// ShouldIgnoreByType determines if a file should be ignored based on positive and negative type filters.
func ShouldIgnoreByType(filename string, types []string, typesNot []string) bool {
	// If positive type filters are specified, file must match at least one
	if len(types) > 0 {
		matchedAny := false
		for _, t := range types {
			if MatchType(filename, t) {
				matchedAny = true
				break
			}
		}
		if !matchedAny {
			return true // Ignore because it didn't match any positive types
		}
	}

	// If negative type filters are specified, file must NOT match any
	if len(typesNot) > 0 {
		for _, t := range typesNot {
			if MatchType(filename, t) {
				return true // Ignore because it matched an excluded type
			}
		}
	}

	return false
}
