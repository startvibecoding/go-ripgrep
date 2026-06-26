# Getting Started

This guide covers installation and basic usage of go-ripgrep.

## Prerequisites

- Go 1.26 or later (for building from source)
- Node.js 14+ (for npm installation)

## Installation

### Building from Source

```bash
git clone https://github.com/startvibecoding/go-ripgrep.git
cd go-ripgrep
make build
```

The binary will be created at `./bin/rg`.

To install it to your `$GOPATH/bin`:

```bash
make install
```

### Using `go install`

```bash
go install github.com/startvibecoding/go-ripgrep/cmd/rg@latest
```

### Using npm

```bash
# Global installation
npm install -g go-ripgrep

# Local project dependency
npm install go-ripgrep
```

When installed via npm, the binary is available at:
- Global: `rg` (in your PATH)
- Local: `node_modules/.bin/rg`

### Pre-built Binaries

Download pre-built binaries from [GitHub Releases](https://github.com/startvibecoding/go-ripgrep/releases):

| Platform | Architecture | File |
|----------|-------------|------|
| Linux | x86_64 | `rg-linux-amd64` |
| Linux | ARM64 | `rg-linux-arm64` |
| Linux | LoongArch64 | `rg-linux-loong64` |
| Linux (static) | x86_64 | `rg-linux-musl-amd64` |
| macOS | x86_64 | `rg-darwin-amd64` |
| macOS | ARM64 (Apple Silicon) | `rg-darwin-arm64` |
| Windows | x86_64 | `rg-windows-amd64.exe` |
| Windows | ARM64 | `rg-windows-arm64.exe` |

## Quick Start

### Basic Search

```bash
# Search for "error" in the current directory
rg "error"

# Search in a specific directory
rg "TODO" ./src

# Search in multiple directories
rg "pattern" ./src ./lib ./test
```

### Case-Insensitive Search

```bash
rg -i "hello"
# Matches: Hello, HELLO, hello, etc.
```

### Fixed String Search

Use `-F` when your pattern contains regex special characters:

```bash
rg -F "user.name = 'admin'"
# Treats the pattern as a literal string, not a regex
```

### File Filtering

```bash
# Search only Go files
rg -g "*.go" "func main"

# Exclude test files
rg -g "!*_test.go" "TODO"

# Multiple glob patterns
rg -g "*.go" -g "*.rs" "function"

# Exclude directories
rg -g "!vendor/" -g "!node_modules/" "pattern"
```

### Context Lines

```bash
# Show 2 lines before and after each match
rg -C 2 "error"

# Show 3 lines after each match
rg -A 3 "function"

# Show 1 line before each match
rg -B 1 "return"
```

### Output Formats

```bash
# Standard output (default)
rg "pattern"

# JSON output (NDJSON)
rg --json "pattern"

# Suppress filenames
rg -I "pattern"

# Show column numbers
rg --column "pattern"
```

### Piping

```bash
# Search stdin
cat logfile.txt | rg "ERROR"

# Use with other commands
ps aux | rg "nginx"

# Search from stdin with explicit dash
cat file.txt | rg - "pattern"
```

## Verifying Installation

```bash
# Check version
rg --version

# Run a simple test
echo "hello world" | rg "hello"
# Output: hello world
```

## Next Steps

- Read the [CLI Reference](cli-reference.md) for all available options
- Check the [SDK Reference](sdk-reference.md) for programmatic usage
- See the [Architecture Guide](architecture.md) to understand how it works
