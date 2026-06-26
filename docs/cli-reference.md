# CLI Reference

Complete reference for the go-ripgrep command-line interface.

## Synopsis

```
rg [OPTIONS] PATTERN [PATH...]
rg [OPTIONS] -e PATTERN [PATH...]
rg [OPTIONS] -F STRING [PATH...]
rg [OPTIONS] -f PATTERNFILE [PATH...]
cat FILE | rg [OPTIONS] PATTERN
```

## Arguments

### `PATTERN`

The search pattern. By default, this is interpreted as a regular expression. Use `-F` to treat it as a literal string.

### `PATH...`

One or more files or directories to search. Defaults to the current directory (`.`) if not specified. Use `-` to read from stdin.

## Options

### Pattern Options

#### `-i, --ignore-case`

Perform case-insensitive matching. By default, searches are case-sensitive.

```bash
rg -i "hello"  # Matches: Hello, HELLO, hello, hElLo
```

#### `-s, --case-sensitive`

Force case-sensitive matching. This overrides `-i` if both are specified.

```bash
rg -s -i "Hello"  # Only matches "Hello", not "hello" or "HELLO"
```

#### `-w, --word-regexp`

Only match whole words. The match must be surrounded by word boundaries (letters, digits, underscores, or start/end of line).

```bash
rg -w "test"      # Matches "test" but not "testing" or "test123"
rg -w "func"      # Matches "func" but not "function"
```

#### `-F, --fixed-strings`

Treat the pattern as a literal string rather than a regular expression. Special regex characters are not interpreted.

```bash
rg -F "user.name"     # Matches literal "user.name", not "userXname"
rg -F "$var + 1"      # Matches literal "$var + 1"
```

#### `-v, --invert-match`

Select lines that do not match the pattern.

```bash
rg -v "TODO"      # Show all lines that don't contain "TODO"
rg -v "^$" file   # Show all non-empty lines
```

### File Filtering Options

#### `-g, --glob GLOB`

Include or exclude files and directories matching the given glob pattern. Can be specified multiple times.

Glob patterns follow `.gitignore` syntax:
- `*` matches anything except `/`
- `**` matches anything including `/`
- `?` matches any single character except `/`
- `[abc]` matches any character in the set
- `[!abc]` matches any character not in the set
- `!` prefix negates the pattern (excludes matching files)

```bash
# Include only Go files
rg -g "*.go" "pattern"

# Exclude minified files
rg -g "!*.min.js" "pattern"

# Include specific directory
rg -g "src/**" "pattern"

# Exclude directories
rg -g "!vendor/" -g "!node_modules/" "pattern"

# Combine include and exclude
rg -g "*.go" -g "!*_test.go" "func"
```

#### `--hidden`

Search hidden files and directories (those starting with `.`). By default, hidden files are skipped.

```bash
rg --hidden "secret" .env
```

#### `--no-ignore`

Don't respect ignore files (`.gitignore`, `.ignore`, `.rgignore`). By default, files matching patterns in these files are skipped.

```bash
rg --no-ignore "node_modules"  # Search even if node_modules is gitignored
```

#### `-L, --follow`

Follow symbolic links. By default, symbolic links are not followed.

```bash
rg -L "pattern" ./links
```

#### `--max-depth NUM`

Maximum directory depth to search. `0` means only the specified paths (no recursion).

```bash
rg --max-depth 2 "pattern"    # Search at most 2 levels deep
rg --max-depth 1 "pattern" .  # Only current directory, no subdirectories
```

### Context Options

#### `-A, --after-context NUM`

Show NUM lines after each matching line.

```bash
rg -A 3 "error"  # Show 3 lines after each "error" match
```

#### `-B, --before-context NUM`

Show NUM lines before each matching line.

```bash
rg -B 2 "error"  # Show 2 lines before each "error" match
```

#### `-C, --context NUM`

Show NUM lines before and after each matching line. Equivalent to `-A NUM -B NUM`.

```bash
rg -C 5 "TODO"  # Show 5 lines of context around each "TODO"
```

#### `-m, --max-count NUM`

Limit the number of matching lines per file. After NUM matches, the file is skipped.

```bash
rg -m 1 "error"  # Show only the first match per file
```

### Output Options

#### `-n, --line-number`

Show line numbers for each match. This is enabled by default.

#### `-N, --no-line-number`

Suppress line numbers in output.

```bash
rg -N "pattern"  # Output without line numbers
```

#### `-H, --with-filename`

Show the file path for each match. This is enabled by default when searching multiple files.

#### `-I, --no-filename`

Suppress the file path in output.

```bash
rg -I "pattern" ./single-file.txt
```

#### `--column`

Show the 1-based column number of the first match on each line.

```bash
rg --column "TODO"
# Output: file.go:10:5:// TODO: fix this
#              ^line ^column
```

#### `--heading`

Group matches under file path headings. When enabled, the file path is printed once on its own line, followed by matching lines.

```
src/main.go
10:func main() {
15:    // main logic
```

This is the default when outputting to a terminal.

#### `--no-heading`

Don't print file headings. Each match line includes the file path.

```
src/main.go:10:func main() {
src/main.go:15:    // main logic
```

#### `--color WHEN`

Control when to use ANSI color codes in output.

- `always` — Always use colors
- `never` — Never use colors
- `auto` — Use colors when outputting to a terminal (default)

```bash
rg --color always "pattern" | less -R  # Force colors for piping to less
rg --color never "pattern" > results.txt  # No colors for file output
```

Colors used:
- **Magenta** — File paths
- **Green** — Line numbers
- **Red, bold** — Matched text

#### `--json`

Output results as newline-delimited JSON (NDJSON). Each line is a complete JSON object.

Output format:

```json
{"type":"begin","data":{"path":{"text":"file.go"}}}
{"type":"match","data":{"path":{"text":"file.go"},"lines":{"text":"matched line"},"line_number":10,"submatches":[{"match":{"text":"pattern"},"start":5,"end":12}]}}
{"type":"context","data":{"path":{"text":"file.go"},"lines":{"text":"context line"},"line_number":11,"submatches":[]}}
{"type":"end","data":{"path":{"text":"file.go"},"binary":false,"stats":{"searched_lines":100,"matches":1}}}
{"type":"summary","data":{"stats":{"elapsed":{"secs":0,"nanos":123456},"searched_lines":1000,"matches":10}}}
```

### Performance Options

#### `-j, --threads NUM`

Number of worker threads to use for searching. Defaults to the number of CPU cores.

```bash
rg -j 4 "pattern"  # Use 4 threads
```

### General Options

#### `-h, --help`

Print the help message and exit.

#### `-V, --version`

Print the version information and exit.

```
go-ripgrep 15.1.0-go
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | One or more matches were found |
| `1` | No matches were found |
| `2` | An error occurred (invalid arguments, pattern compilation error, etc.) |

## Environment Variables

Currently, go-ripgrep does not read environment variables for configuration. All options must be specified via command-line flags.

## Ignore File Priority

When determining whether to ignore a file, the following priority is used (highest to lowest):

1. `.rgignore` — ripgrep-specific ignore rules
2. `.ignore` — generic ignore rules
3. `.gitignore` — Git ignore rules

Rules in deeper directories override rules in parent directories. A negated pattern (`!`) in a deeper file can un-ignore a file that was ignored by a parent file.

## Glob Pattern Syntax

Glob patterns follow `.gitignore` conventions:

| Pattern | Meaning |
|---------|---------|
| `*` | Match anything except `/` |
| `**` | Match anything including `/` |
| `?` | Match any single character except `/` |
| `[abc]` | Match any character in the set |
| `[a-z]` | Match any character in the range |
| `[!abc]` | Match any character not in the set |
| `!pattern` | Negate the pattern |

### Anchoring

- Patterns without `/` are matched against the filename only
- Patterns with `/` are matched against the full path
- Patterns starting with `/` are anchored to the search root

```bash
rg -g "*.go" "pattern"      # Matches any .go file in any directory
rg -g "/src/*.go" "pattern" # Matches .go files only in src/ at root
rg -g "src/**" "pattern"    # Matches anything under src/
```
