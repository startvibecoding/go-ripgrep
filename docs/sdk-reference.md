# Go SDK Reference

Reference documentation for using go-ripgrep as a Go library in your applications.

## Installation

Add go-ripgrep to your module:

```bash
go get github.com/startvibecoding/go-ripgrep
```

## Import

```go
import (
    goriggrep "go-ripgrep"
    "go-ripgrep/pkg/printer"
)
```

## Types

### Options

The `Options` struct configures the search behavior.

```go
type Options struct {
    // Pattern settings
    Pattern         string   // The search pattern (regex or fixed string)
    IsFixed         bool     // If true, treat Pattern as a literal string
    CaseInsensitive bool     // If true, perform case-insensitive matching
    WordRegexp      bool     // If true, only match whole words
    InvertMatch     bool     // If true, select non-matching lines

    // Filtering settings
    NoIgnore       bool     // If true, ignore .gitignore/.ignore/.rgignore files
    Hidden         bool     // If true, search hidden files and directories
    FollowSymlinks bool     // If true, follow symbolic links
    MaxDepth       int      // Maximum directory depth (0 = unlimited)
    Globs          []string // Glob patterns for filtering files (negated with !)

    // Context settings
    BeforeContext int        // Lines to show before each match
    AfterContext  int        // Lines to show after each match
    MaxCount      int        // Maximum matches per file (0 = unlimited)

    // Execution settings
    Threads int              // Number of worker threads (0 = NumCPU)
}
```

### FileResult

The `FileResult` struct contains search results for a single file.

```go
type FileResult struct {
    Path    string        `json:"path"`     // File path
    Matches []SearchMatch `json:"matches"`  // List of matches
    Stats   FileStats     `json:"stats"`    // File statistics
    Elapsed time.Duration `json:"elapsed"`  // Time spent searching this file
}
```

### SearchMatch

The `SearchMatch` struct represents a single matched line or context line.

```go
type SearchMatch struct {
    Line       string     `json:"line"`        // The line text
    LineNum    int        `json:"line_number"` // 1-based line number
    IsContext  bool       `json:"is_context"`  // true if this is a context line
    Submatches []Submatch `json:"submatches"`  // Matched portions
}
```

### Submatch

The `Submatch` struct represents a matched portion within a line.

```go
type Submatch struct {
    Start int    `json:"start"` // Start byte offset (0-based)
    End   int    `json:"end"`   // End byte offset (0-based, exclusive)
    Text  string `json:"text"`  // Matched text
}
```

### FileStats

The `FileStats` struct contains statistics for a searched file.

```go
type FileStats struct {
    SearchedLines int `json:"searched_lines"` // Total lines read
    Matches       int `json:"matches"`        // Number of matching lines
}
```

## Functions

### Search

```go
func Search(ctx context.Context, paths []string, opts Options) (<-chan printer.FileResult, error)
```

Recursively searches the given paths for the pattern specified in `Options`. Returns a channel that streams `FileResult` values as files are searched.

**Parameters:**
- `ctx` — Context for cancellation. Cancel the context to stop the search early.
- `paths` — List of file or directory paths to search.
- `opts` — Search options.

**Returns:**
- A read-only channel of `FileResult`. The channel is closed when all files have been searched.
- An error if the pattern is invalid or glob compilation fails.

**Example:**

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

results, err := goriggrep.Search(ctx, []string{"./src"}, goriggrep.Options{
    Pattern: "TODO",
})
if err != nil {
    log.Fatal(err)
}

for res := range results {
    fmt.Printf("%s: %d matches\n", res.Path, res.Stats.Matches)
}
```

## Package: matcher

The `matcher` package provides pattern matching functionality.

### Matcher Interface

```go
type Matcher interface {
    Match(line []byte) bool
    FindSpans(line []byte) [][2]int
}
```

### BuildMatcher

```go
func BuildMatcher(pattern string, isFixed, caseInsensitive, wordRegexp bool) (Matcher, error)
```

Creates a `Matcher` based on the given flags. Returns either a `RegexMatcher` or `FixedMatcher`.

### RegexMatcher

Matches lines using a compiled Go regular expression.

```go
re := regexp.MustCompile(`(?i)TODO|FIXME`)
m := matcher.NewRegexMatcher(re)
if m.Match([]byte("// TODO: fix this")) {
    // matched
}
```

### FixedMatcher

Matches lines using substring search. More efficient than regex for literal patterns.

```go
m := matcher.NewFixedMatcher("TODO", true) // case-insensitive
if m.Match([]byte("// todo: fix this")) {
    // matched
}
```

## Package: searcher

The `searcher` package handles file reading and line-by-line searching.

### Searcher

```go
type Searcher struct { /* ... */ }

func NewSearcher(m matcher.Matcher, beforeContext, afterContext, maxCount int, invertMatch bool) *Searcher
```

### SearchFile

```go
func (s *Searcher) SearchFile(path string) (*printer.FileResult, error)
```

Opens and searches a file. Returns the search result.

### SearchReader

```go
func (s *Searcher) SearchReader(r io.Reader, path string) (*printer.FileResult, error)
```

Searches from an `io.Reader`. Useful for searching stdin or in-memory data.

**Example:**

```go
m, _ := matcher.BuildMatcher("pattern", false, false, false)
s := searcher.NewSearcher(m, 0, 0, 0, false)

result, err := s.SearchReader(strings.NewReader("hello world\nfoo pattern bar\n"), "<input>")
if err != nil {
    log.Fatal(err)
}
// result.Matches contains the matching lines
```

## Package: printer

The `printer` package formats and outputs search results.

### Config

```go
type Config struct {
    Group         bool // Group matches under file headings
    Color         bool // Enable ANSI color codes
    JSON          bool // Output NDJSON format
    WithLineNum   bool // Include line numbers
    WithFilename  bool // Include file paths
    WithColumnNum bool // Include column numbers
}
```

### Printer

```go
func NewPrinter(w io.Writer, cfg Config) *Printer
```

### PrintFileResult

```go
func (p *Printer) PrintFileResult(res FileResult) error
```

Formats and writes a `FileResult` to the output writer.

### PrintSummary

```go
func (p *Printer) PrintSummary(totalFiles, totalMatches, totalLines int, elapsed time.Duration) error
```

Prints a summary of the search results (used with JSON output).

### IsTerminal

```go
func IsTerminal() bool
```

Returns `true` if stdout is a terminal (character device).

## Package: globset

The `globset` package handles glob pattern compilation and matching.

### Glob

```go
type Glob struct {
    Original  string
    Regexp    *regexp.Regexp
    IsNegated bool
}

func NewGlob(pattern string) (*Glob, error)
func (g *Glob) Match(path string) bool
```

### GlobSet

```go
type GlobSet struct { /* ... */ }

func NewGlobSet(patterns []string) (*GlobSet, error)
func (gs *GlobSet) Match(path string) (matched bool, isIgnored bool)
func (gs *GlobSet) MatchPath(path string) (matched bool, isIgnored bool)
func (gs *GlobSet) MatchGlobFilter(path string) bool
```

### GlobToRegex

```go
func GlobToRegex(pattern string) (string, error)
```

Converts a `.gitignore`-style glob pattern to a Go regular expression string.

## Package: ignore

The `ignore` package manages ignore file parsing and stack-based rule evaluation.

### IgnoreStack

```go
type IgnoreStack struct { /* ... */ }

func NewIgnoreStack(noIgnore, hidden bool, maxDepth int) *IgnoreStack
func (s *IgnoreStack) Push(dirPath string) error
func (s *IgnoreStack) Pop()
func (s *IgnoreStack) Clone() *IgnoreStack
func (s *IgnoreStack) IsIgnored(path string, isDir bool) bool
```

**Example:**

```go
stack := ignore.NewIgnoreStack(false, false, 0)
stack.Push("/path/to/repo")
defer stack.Pop()

if stack.IsIgnored("/path/to/repo/node_modules/package", true) {
    // directory is ignored
}
```

## Complete Example

Here's a complete example that searches for patterns and formats the output:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    goriggrep "go-ripgrep"
    "go-ripgrep/pkg/printer"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintf(os.Stderr, "Usage: %s PATTERN [PATH...]\n", os.Args[0])
        os.Exit(1)
    }

    pattern := os.Args[1]
    paths := []string{"."}
    if len(os.Args) > 2 {
        paths = os.Args[2:]
    }

    opts := goriggrep.Options{
        Pattern:         pattern,
        CaseInsensitive: false,
        Threads:         0, // auto-detect
    }

    start := time.Now()
    results, err := goriggrep.Search(context.Background(), paths, opts)
    if err != nil {
        log.Fatal(err)
    }

    pConfig := printer.Config{
        Group:        printer.IsTerminal(),
        Color:        printer.IsTerminal(),
        WithLineNum:  true,
        WithFilename: true,
    }
    p := printer.NewPrinter(os.Stdout, pConfig)

    totalFiles := 0
    totalMatches := 0

    for res := range results {
        totalFiles++
        totalMatches += res.Stats.Matches
        _ = p.PrintFileResult(res)
    }

    elapsed := time.Since(start)
    fmt.Fprintf(os.Stderr, "\n%d files, %d matches in %v\n", totalFiles, totalMatches, elapsed)

    if totalMatches > 0 {
        os.Exit(0)
    }
    os.Exit(1)
}
```

## Thread Safety

- `Search()` is safe to call from multiple goroutines
- The returned channel is safe to read from multiple goroutines
- `Matcher` instances are safe for concurrent use
- `Printer` instances are **not** safe for concurrent writes; synchronize access if needed
