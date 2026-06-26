# Architecture

This document describes the internal architecture and design of go-ripgrep.

## Overview

go-ripgrep is structured as a pipeline that transforms file paths into search results:

```
Input Paths → Directory Walker → File Discovery → Pattern Matching → Result Formatting → Output
```

The system is designed for concurrent execution using Go's goroutines and channels.

## Package Structure

```
go-ripgrep/
├── cmd/rg/              # CLI application
├── pkg/
│   ├── matcher/         # Pattern matching engine
│   ├── searcher/        # File reading and line processing
│   ├── printer/         # Output formatting
│   ├── globset/         # Glob pattern compilation
│   └── ignore/          # Ignore file management
├── sdk.go               # Public API (Search function)
└── sdk_test.go          # SDK tests
```

## Data Flow

### 1. CLI Entry Point (`cmd/rg/main.go`)

The CLI parses command-line arguments into a `CliArgs` struct, which is then converted to the SDK's `Options` struct.

```
os.Args → parseArgs() → CliArgs → Options → Search()
```

### 2. SDK Entry Point (`sdk.go`)

The `Search()` function orchestrates the entire search process:

```go
func Search(ctx context.Context, paths []string, opts Options) (<-chan printer.FileResult, error)
```

It creates:
- A **Matcher** (regex or fixed string)
- A **GlobSet** (for `-g` filtering)
- Worker goroutines (one per thread)
- A file discovery goroutine

### 3. Directory Walking

The `walkDir()` function recursively traverses directories:

```
walkDir(dirPath)
  ├── Push ignore rules for dirPath
  ├── Read directory entries
  ├── For each entry:
  │   ├── Check ignore rules (IsIgnored)
  │   ├── Check glob filters (MatchGlobFilter)
  │   ├── If directory: recurse (walkDir)
  │   └── If file: send to filesChan
  └── Pop ignore rules
```

**Concurrency Model:**

```
                    ┌─────────────┐
                    │  Walker     │
                    │  (1 goroutine)
                    └──────┬──────┘
                           │ filesChan
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │ Worker 1 │ │ Worker 2 │ │ Worker N │
        └────┬─────┘ └────┬─────┘ └────┬─────┘
             │             │             │
             └─────────────┼─────────────┘
                           │ outChan
                           ▼
                    ┌─────────────┐
                    │   Consumer  │
                    └─────────────┘
```

### 4. Pattern Matching (`pkg/matcher/`)

Two matcher implementations:

**RegexMatcher:**
- Uses Go's `regexp` package
- Supports all Go regex features
- `FindSpans()` returns byte offsets of matches

**FixedMatcher:**
- Uses `bytes.Contains()` for matching
- More efficient for literal string searches
- Supports case-insensitive matching via `bytes.ToLower()`

**BuildMatcher()** selects the appropriate matcher:

```go
func BuildMatcher(pattern string, isFixed, caseInsensitive, wordRegexp bool) (Matcher, error) {
    if isFixed && !wordRegexp {
        return NewFixedMatcher(pattern, caseInsensitive), nil
    }
    // Build regex pattern...
    return NewRegexMatcher(re), nil
}
```

### 5. File Searching (`pkg/searcher/`)

The `Searcher` processes files line-by-line:

```
Open file → Peek (binary detection) → Read lines → Match → Build result
```

**Binary Detection:**
- Reads first 1024 bytes
- If NUL byte found, skips the file (returns empty result)

**Context Management:**
- `beforeBuf` — Circular buffer of previous lines
- `afterCount` — Counter for remaining after-context lines
- `lastPrintedLineNum` — Prevents duplicate context lines

**Line Processing Flow:**

```
processLine(line):
  ├── Trim trailing newlines
  ├── Match against pattern
  ├── If invertMatch: flip result
  ├── If match:
  │   ├── Emit before-context from buffer
  │   ├── Find submatch spans
  │   ├── Emit match line
  │   └── Reset after-context counter
  └── If no match:
      ├── If after-context active: emit as context
      └── Else: add to before-context buffer
```

### 6. Ignore File Management (`pkg/ignore/`)

Uses a stack-based approach for nested directory ignore rules:

```
IgnoreStack:
  Level 0: /repo/.gitignore
  Level 1: /repo/src/.gitignore
  Level 2: /repo/src/vendor/.ignore
```

**Priority Order** (highest to lowest):
1. `.rgignore`
2. `.ignore`
3. `.gitignore`

**Rule Evaluation:**
- Rules from deeper directories take precedence
- Negated patterns (`!`) can un-ignore files
- Hidden files (starting with `.`) are ignored unless `--hidden` is set

### 7. Glob Pattern Matching (`pkg/globset/`)

Glob patterns are compiled to regular expressions:

```
*.go        → ^[^/]*\.go$
src/**      → ^src/.*$
!vendor/    → (negated) ^vendor/
```

**GlobToRegex()** handles:
- `*` — matches anything except `/`
- `**` — matches anything including `/`
- `?` — matches single character
- `[abc]` — character classes
- `!` — negation prefix

**MatchGlobFilter()** implements ripgrep's `-g` logic:
- Negated patterns exclude matching files
- Positive patterns include matching files
- If only positive patterns exist, non-matching files are excluded

### 8. Output Formatting (`pkg/printer/`)

Two output modes:

**Text Mode (default):**
```
file.go
10:func main() {
11:    // main logic
```

Or non-grouped:
```
file.go:10:func main() {
file.go:11:    // main logic
```

**JSON Mode (`--json`):**
```json
{"type":"begin","data":{"path":{"text":"file.go"}}}
{"type":"match","data":{...}}
{"type":"end","data":{...}}
```

**Color Support:**
- File paths: magenta (`\x1b[35m`)
- Line numbers: green (`\x1b[32m`)
- Matches: red, bold (`\x1b[1;31m`)

## Concurrency Design

### Goroutines

1. **Walker goroutine** — Recursively traverses directories, sends file paths to `filesChan`
2. **Worker goroutines** — Read from `filesChan`, search files, send results to `outChan`
3. **Closer goroutine** — Waits for all workers, then closes `outChan`

### Channels

- `filesChan` — Buffered channel for file paths (capacity: `threads * 4`)
- `outChan` — Buffered channel for results (capacity: `threads * 2`)

### Context Cancellation

All goroutines check `ctx.Done()` at key points:
- Before processing each directory
- Before processing each file
- Between channel operations

This ensures immediate termination when the context is cancelled.

### Synchronization

- `sync.WaitGroup` tracks active worker goroutines
- Channel close signals completion
- No mutexes needed (data flows through channels)

## Performance Considerations

### Memory Usage

- Streaming results via channels (no buffering all results)
- Reusable line buffer in `Searcher`
- Binary detection avoids reading entire files

### CPU Utilization

- One goroutine per CPU core by default
- File I/O and pattern matching are CPU-bound
- Channel operations are efficient with Go runtime

### I/O Optimization

- Small buffer size for reading (line-by-line)
- Early termination on binary files
- Skip ignored directories (no stat calls on children)

## Design Decisions

### Why Pure Go?

- Single binary distribution
- Easy cross-compilation with `GOOS`/`GOARCH`
- No CGO dependencies
- Simple `go install` installation

### Why Not Use ripgrep Directly?

- Embeddable Go SDK for programmatic use
- npm distribution for Node.js ecosystems
- Simpler deployment (no Rust toolchain needed)

### Trade-offs

- **SIMD optimization**: Go Assembly is planned for platform-specific SIMD optimization (see [Alignment Roadmap](alignment-roadmap.md) for details).
- **No PCRE2 support**: Go uses RE2 syntax; PCRE2 features (lookarounds, backreferences) are omitted as they require linking to C libraries (CGO).
- **Performance**: While currently slower on extremely large codebases, ongoing SIMD and walking optimizations aim to bridge the performance gap significantly.
