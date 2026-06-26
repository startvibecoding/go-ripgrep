# go-ripgrep

A high-performance line-oriented search tool written in Go — a pure Go port of [ripgrep](https://github.com/BurntSushi/ripgrep). It provides both a CLI tool (`rg`) compatible with ripgrep's interface and a Go SDK for programmatic use.

[![Go Version](https://img.shields.io/badge/Go%201.26+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **Fast recursive search** — multi-threaded file walking and searching using goroutines
- **Regex & fixed-string matching** — full Go `regexp` support plus literal string search (`-F`)
- **Ignore file support** — respects `.gitignore`, `.ignore`, and `.rgignore` with nested directory support
- **Glob filtering** — include/exclude files using `-g`/`--glob` patterns with negation support
- **Context lines** — show lines before/after matches (`-A`, `-B`, `-C`)
- **Color output** — ANSI color highlighting for matches, filenames, and line numbers
- **JSON output** — newline-delimited JSON (NDJSON) format (`--json`) compatible with ripgrep's JSON spec
- **Stdin support** — pipe input from other commands
- **Cross-platform** — builds for Linux (amd64, arm64, loong64), macOS (amd64, arm64), and Windows (amd64, arm64)
- **NPM packages** — distributable via npm for Node.js integration
- **Pure Go SDK** — embed search functionality in your Go applications

## Installation

### From Source

```bash
git clone https://github.com/startvibecoding/go-ripgrep.git
cd go-ripgrep
make build
# Binary will be at ./bin/rg
```

### Via `go install`

```bash
go install github.com/startvibecoding/go-ripgrep/cmd/rg@latest
```

### Via npm

```bash
npm install go-ripgrep
# Binary available at node_modules/.bin/rg
```

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/startvibecoding/go-ripgrep/releases) for your platform.

## Usage

### CLI

```
rg [OPTIONS] PATTERN [PATH...]
rg [OPTIONS] -F PATTERN [PATH...]
cat file | rg [OPTIONS] PATTERN
```

#### Examples

```bash
# Search for a pattern recursively
rg "hello" ./src

# Case-insensitive search
rg -i "error" /var/log

# Fixed string search (no regex)
rg -F "function(" ./src

# Show context lines
rg -C 3 "TODO" .

# Search only specific file types
rg -g "*.go" "func main" .

# Exclude file patterns
rg -g "!*.min.js" "function" ./dist

# JSON output
rg --json "pattern" .

# Pipe from stdin
cat README.md | rg "install"

# Word boundary match
rg -w "test" .

# Limit matches per file
rg -m 5 "error" /var/log

# Show column numbers
rg --column "TODO" .

# Follow symlinks
rg -L "pattern" ./links

# Search hidden files
rg --hidden "secret" .

# Ignore .gitignore rules
rg --no-ignore "node_modules" .
```

### Go SDK

```go
package main

import (
    "context"
    "fmt"
    goriggrep "go-ripgrep"
    "go-ripgrep/pkg/printer"
)

func main() {
    opts := goriggrep.Options{
        Pattern:         "TODO",
        CaseInsensitive: true,
        // MaxDepth:       3,
        // Globs:          []string{"*.go", "!vendor/"},
        // Threads:        8,
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    results, err := goriggrep.Search(ctx, []string{"./src"}, opts)
    if err != nil {
        panic(err)
    }

    for res := range results {
        fmt.Printf("File: %s (%d matches)\n", res.Path, res.Stats.Matches)
        for _, m := range res.Matches {
            if !m.IsContext {
                fmt.Printf("  Line %d: %s\n", m.LineNum, m.Line)
            }
        }
    }
}
```

## CLI Options Reference

| Flag | Short | Description |
|------|-------|-------------|
| `--ignore-case` | `-i` | Case-insensitive search |
| `--case-sensitive` | `-s` | Force case-sensitive search (overrides `-i`) |
| `--word-regexp` | `-w` | Match whole words only |
| `--fixed-strings` | `-F` | Treat pattern as a literal string |
| `--invert-match` | `-v` | Select non-matching lines |
| `--glob GLOB` | `-g` | Include/exclude files by glob pattern |
| `--after-context NUM` | `-A` | Show NUM lines after each match |
| `--before-context NUM` | `-B` | Show NUM lines before each match |
| `--context NUM` | `-C` | Show NUM lines before and after each match |
| `--max-count NUM` | `-m` | Limit matches per file |
| `--threads NUM` | `-j` | Number of worker threads |
| `--hidden` | | Search hidden files and directories |
| `--no-ignore` | | Don't respect ignore files |
| `--follow` | `-L` | Follow symbolic links |
| `--max-depth NUM` | | Maximum directory depth |
| `--json` | | Output newline-delimited JSON |
| `--color WHEN` | | Color output: `always`, `never`, `auto` |
| `--heading` | | Group matches under file headings |
| `--no-heading` | | Don't print file headings |
| `--line-number` | `-n` | Show line numbers (default: on) |
| `--no-line-number` | `-N` | Suppress line numbers |
| `--with-filename` | `-H` | Print file path for each match |
| `--no-filename` | `-I` | Suppress file path |
| `--column` | | Show column number of first match |
| `--help` | `-h` | Print help message |
| `--version` | `-V` | Print version |

## Architecture

```
go-ripgrep/
├── cmd/rg/           # CLI entry point
│   └── main.go       # Argument parsing, search orchestration, output
├── pkg/
│   ├── matcher/      # Pattern matching engine
│   │   ├── matcher.go    # RegexMatcher, FixedMatcher, BuildMatcher
│   │   └── matcher_test.go
│   ├── searcher/     # File reading & line-by-line search
│   │   ├── searcher.go   # Searcher with context support
│   │   └── searcher_test.go
│   ├── printer/      # Output formatting
│   │   ├── printer.go    # CLI text & NDJSON output
│   │   └── printer_test.go
│   ├── globset/      # Glob pattern compilation & matching
│   │   ├── globset.go    # GlobToRegex, GlobSet, MatchGlobFilter
│   │   └── globset_test.go
│   └── ignore/       # Ignore file parsing & stack management
│       ├── ignore.go     # .gitignore, .ignore, .rgignore support
│       └── ignore_test.go
├── sdk.go            # Public Go SDK (Search, Options)
├── sdk_test.go       # SDK unit tests
├── tests/
│   └── integration_test.go  # End-to-end CLI tests
├── npm/              # NPM package distribution
├── scripts/          # Build & packaging scripts
└── Makefile          # Build system
```

### Data Flow

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  CLI / SDK   │────▶│   Walker     │────▶│  Searcher    │
│  (options)   │     │  (dirs/files)│     │  (per file)  │
└──────────────┘     └──────────────┘     └──────────────┘
                           │                     │
                     ┌─────▼─────┐         ┌─────▼─────┐
                     │  Ignore   │         │  Matcher  │
                     │  Stack    │         │  (regex/  │
                     │(.gitignore│         │   fixed)  │
                     │ .ignore)  │         └───────────┘
                     └───────────┘
```

1. **CLI** parses arguments and creates `Options`
2. **Walker** recursively traverses directories, consulting the **Ignore Stack** to respect `.gitignore` rules
3. Files are sent to worker goroutines via a channel
4. **Searcher** reads each file line-by-line, using **Matcher** to find matches
5. Results flow back through a channel to **Printer** for formatted output

## Building

```bash
# Current platform
make build

# All platforms
make build-all

# Specific platforms
make build-linux
make build-darwin
make build-windows

# Static binary (musl)
make build-linux-musl

# Run tests
make test

# Format code
make fmt

# Clean build artifacts
make clean
```

## NPM Distribution

```bash
# Sync version across npm packages
make npm-version

# Build platform-specific npm packages
make npm-packages

# Pack all npm packages
make npm-pack

# Publish all npm packages
make npm-publish-all
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Matches found |
| `1` | No matches found |
| `2` | Error (invalid arguments, pattern error, etc.) |

## Comparison with ripgrep

| Feature | ripgrep (Rust) | go-ripgrep (Go) |
|---------|----------------|-----------------|
| Language | Rust | Go |
| Regex engine | Rust `regex` crate | Go `regexp` stdlib |
| SIMD optimization | Yes | No |
| PCRE2 support | Yes | No |
| .gitignore support | Yes | Yes |
| JSON output | Yes | Yes |
| Go SDK | No | Yes |
| npm distribution | Via community | Built-in |
| Cross-compilation | Rust toolchain | `GOOS`/`GOARCH` |

> **Note:** This project aims for CLI compatibility with ripgrep but may differ in performance and edge-case behavior. For maximum performance on large codebases, consider using the original [ripgrep](https://github.com/BurntSushi/ripgrep).

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [ripgrep](https://github.com/BurntSushi/ripgrep) by Andrew Gallant — the original implementation in Rust
- [Go standard library](https://pkg.go.dev/) — `regexp`, `filepath`, `os` packages
