package main

import (
	"context"
	"fmt"

	goriggrep "github.com/startvibecoding/go-ripgrep"
	"github.com/startvibecoding/go-ripgrep/pkg/ignore"
	"github.com/startvibecoding/go-ripgrep/pkg/matcher"
	"github.com/startvibecoding/go-ripgrep/pkg/printer"
	"github.com/startvibecoding/go-ripgrep/pkg/searcher"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CliArgs struct {
	Pattern string
	Paths   []string

	CaseInsensitive bool
	WordRegexp      bool
	FixedStrings    bool
	InvertMatch     bool

	NoIgnore       bool
	Hidden         bool
	FollowSymlinks bool
	MaxDepth       int
	Globs          []string
	Types          []string
	TypesNot       []string
	TypeList       bool
	SearchZip      bool

	BeforeContext int
	AfterContext  int
	MaxCount      int
	Threads       int

	JSON         bool
	Color        string // "always", "never", "auto"
	Heading      bool
	NoHeading    bool
	LineNumber   bool
	NoLineNumber bool
	WithFilename bool
	NoFilename   bool
	Column       bool
	OnlyMatching bool
	Count        bool
	Quiet        bool
	SortBy       string // "path", "modified", "size", or "none"
	SortReverse  bool   // reverse sorting order
	Replace      string
	HasReplace   bool

	Help    bool
	Version bool
}

const version = "0.0.1"

const helpMessage = `go-ripgrep (rg) recursively searches your directory for a regex pattern.

USAGE:
    rg [OPTIONS] PATTERN [PATH...]
    rg [OPTIONS] -F PATTERN [PATH...]
    cat file | rg [OPTIONS] PATTERN

ARGS:
    <PATTERN>      A regular expression or fixed string to search for.
    <PATH...>      The directories or files to search. [default: .]

OPTIONS:
    -i, --ignore-case          Case-insensitive search.
    -s, --case-sensitive       Case-sensitive search (overrides -i).
    -w, --word-regexp          Match whole words only.
    -F, --fixed-strings        Treat PATTERN as a literal string.
    -v, --invert-match         Invert match: select non-matching lines.
    -r, --replace STR          Replace matches with STR.
    -g, --glob GLOB            Include or exclude files/directories using globs.
    -t, --type TYPE            Only search files matching TYPE.
    -T, --type-not TYPE        Do not search files matching TYPE.
        --type-list            Show all supported file types.
    -A, --after-context NUM    Show NUM lines after each match.
    -B, --before-context NUM   Show NUM lines before each match.
    -C, --context NUM          Show NUM lines before and after each match.
    -m, --max-count NUM        Limit matches per file to NUM.
    -j, --threads NUM          Number of threads to use.
        --hidden               Search hidden files and directories.
        --no-ignore            Do not respect ignore files (.gitignore, .ignore, etc.).
    -L, --follow               Follow symbolic links.
        --json                 Output newline-delimited JSON.
        --color WHEN           Whether to use color: always, never, auto. [default: auto]
        --heading              Print heading for matches from each file. [default: when on terminal]
        --no-heading           Do not print heading for matches.
    -n, --line-number          Show line numbers. [default: on]
    -N, --no-line-number       Suppress line numbers.
    -H, --with-filename        Print the file path for each match.
    -I, --no-filename          Suppress file path for each match.
        --column               Show 1-based column number of first match.
    -o, --only-matching        Show only matching parts of lines.
    -c, --count                Show only match count per file.
    -q, --quiet                Suppress output; exit 0 if match found.
    -z, --search-zip           Search inside compressed files (gz, bz2, zip).
        --sort SORT            Sort results by: path, modified, size, none.
        --sortr SORT           Sort results in reverse order.
        --max-depth NUM        Limit directory traversal depth.
    -h, --help                 Print this help message.
    -V, --version              Print version information.
`

func main() {
	cli, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	if cli.Help {
		fmt.Print(helpMessage)
		os.Exit(0)
	}

	if cli.Version {
		fmt.Printf("go-ripgrep %s\n", version)
		os.Exit(0)
	}

	if cli.TypeList {
		var keys []string
		for k := range ignore.BuiltInTypes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("%s: %s\n", k, strings.Join(ignore.BuiltInTypes[k], ", "))
		}
		os.Exit(0)
	}

	if cli.Pattern == "" {
		fmt.Fprintln(os.Stderr, "error: PATTERN is required. See 'rg --help'.")
		os.Exit(2)
	}

	// Determine if we should search stdin
	searchStdin := false
	if len(cli.Paths) == 0 {
		// If no paths specified and stdin is a pipe, search stdin
		if !isStdinTerminal() {
			searchStdin = true
		} else {
			// Otherwise default to "."
			cli.Paths = []string{"."}
		}
	} else if len(cli.Paths) == 1 && cli.Paths[0] == "-" {
		searchStdin = true
	}

	// Configure color output
	colorEnabled := false
	switch cli.Color {
	case "always":
		colorEnabled = true
	case "never":
		colorEnabled = false
	default: // "auto"
		colorEnabled = printer.IsTerminal() && !cli.JSON
	}

	// Configure line numbers
	lineNumbersEnabled := false
	if cli.LineNumber {
		lineNumbersEnabled = true
	} else if cli.NoLineNumber {
		lineNumbersEnabled = false
	} else {
		// Default: on if stdout is terminal, off otherwise
		lineNumbersEnabled = printer.IsTerminal()
	}

	if searchStdin {
		m, err := matcher.BuildMatcher(cli.Pattern, cli.FixedStrings, cli.CaseInsensitive, cli.WordRegexp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error compiling pattern: %v\n", err)
			os.Exit(2)
		}
		s := searcher.NewSearcher(m, cli.BeforeContext, cli.AfterContext, cli.MaxCount, cli.InvertMatch)
		if cli.HasReplace {
			s.SetReplace(cli.Replace)
		}
		res, err := s.SearchReader(os.Stdin, "<stdin>")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}

		pConfig := printer.Config{
			Group:         false,
			Color:         colorEnabled,
			WithLineNum:   lineNumbersEnabled,
			WithFilename:  cli.WithFilename && !cli.NoFilename,
			WithColumnNum: cli.Column,
			JSON:          cli.JSON,
		}
		p := printer.NewPrinter(os.Stdout, pConfig)
		_ = p.PrintFileResult(*res)

		if res.Stats.Matches > 0 {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	// Directory / Multi-file search
	opts := goriggrep.Options{
		Replace:         cli.Replace,
		HasReplace:      cli.HasReplace,
		Pattern:         cli.Pattern,
		IsFixed:         cli.FixedStrings,
		CaseInsensitive: cli.CaseInsensitive,
		WordRegexp:      cli.WordRegexp,
		InvertMatch:     cli.InvertMatch,

		NoIgnore:       cli.NoIgnore,
		Hidden:         cli.Hidden,
		FollowSymlinks: cli.FollowSymlinks,
		MaxDepth:       cli.MaxDepth,
		Globs:          cli.Globs,
		Types:          cli.Types,
		TypesNot:       cli.TypesNot,
		SearchZip:      cli.SearchZip,
		SortBy:         cli.SortBy,
		SortReverse:    cli.SortReverse,

		BeforeContext: cli.BeforeContext,
		AfterContext:  cli.AfterContext,
		MaxCount:      cli.MaxCount,
		Threads:       cli.Threads,
	}

	startTime := time.Now()
	resultsChan, err := goriggrep.Search(context.Background(), cli.Paths, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing search: %v\n", err)
		os.Exit(2)
	}

	// Printer configuration
	withFilename := false
	if cli.WithFilename {
		withFilename = true
	} else if cli.NoFilename {
		withFilename = false
	} else {
		if searchStdin {
			withFilename = false
		} else if len(cli.Paths) > 1 {
			withFilename = true
		} else if len(cli.Paths) == 1 {
			info, err := os.Stat(cli.Paths[0])
			if err == nil && info.IsDir() {
				withFilename = true
			} else {
				withFilename = false
			}
		}
	}

	group := false
	if cli.Heading {
		group = true
	} else if cli.NoHeading {
		group = false
	} else {
		group = printer.IsTerminal() && withFilename
	}

	pConfig := printer.Config{
		Group:         group,
		Color:         colorEnabled,
		WithLineNum:   lineNumbersEnabled,
		WithFilename:  withFilename,
		WithColumnNum: cli.Column,
		JSON:          cli.JSON,
		OnlyMatching:  cli.OnlyMatching,
		Count:         cli.Count,
	}

	var outputWriter io.Writer = os.Stdout
	if cli.Quiet {
		outputWriter = io.Discard
	}
	p := printer.NewPrinter(outputWriter, pConfig)

	totalFiles := 0
	totalMatches := 0
	totalLines := 0

	for res := range resultsChan {
		totalFiles++
		totalMatches += res.Stats.Matches
		totalLines += res.Stats.SearchedLines
		_ = p.PrintFileResult(res)
	}

	elapsed := time.Since(startTime)
	_ = p.PrintSummary(totalFiles, totalMatches, totalLines, elapsed)

	if totalMatches > 0 {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func parseArgs(args []string) (*CliArgs, error) {
	cli := &CliArgs{
		Color: "auto",
	}

	// Flag definitions: each entry describes a long flag, optional short flag,
	// whether it needs a value, and how to apply it.
	type flagDef struct {
		long       string
		short      rune
		needsValue bool
		apply      func(cli *CliArgs, value string) error
	}

	flags := []flagDef{
		// Boolean flags (no value)
		{long: "ignore-case", short: 'i', apply: func(c *CliArgs, _ string) error { c.CaseInsensitive = true; return nil }},
		{long: "case-sensitive", short: 's', apply: func(c *CliArgs, _ string) error { c.CaseInsensitive = false; return nil }},
		{long: "word-regexp", short: 'w', apply: func(c *CliArgs, _ string) error { c.WordRegexp = true; return nil }},
		{long: "fixed-strings", short: 'F', apply: func(c *CliArgs, _ string) error { c.FixedStrings = true; return nil }},
		{long: "invert-match", short: 'v', apply: func(c *CliArgs, _ string) error { c.InvertMatch = true; return nil }},
		{long: "hidden", apply: func(c *CliArgs, _ string) error { c.Hidden = true; return nil }},
		{long: "search-zip", short: 'z', apply: func(c *CliArgs, _ string) error { c.SearchZip = true; return nil }},
		{long: "no-ignore", apply: func(c *CliArgs, _ string) error { c.NoIgnore = true; return nil }},
		{long: "follow", short: 'L', apply: func(c *CliArgs, _ string) error { c.FollowSymlinks = true; return nil }},
		{long: "json", apply: func(c *CliArgs, _ string) error { c.JSON = true; return nil }},
		{long: "only-matching", short: 'o', apply: func(c *CliArgs, _ string) error { c.OnlyMatching = true; return nil }},
		{long: "count", short: 'c', apply: func(c *CliArgs, _ string) error { c.Count = true; return nil }},
		{long: "quiet", short: 'q', apply: func(c *CliArgs, _ string) error { c.Quiet = true; return nil }},
		{long: "column", apply: func(c *CliArgs, _ string) error { c.Column = true; return nil }},
		{long: "heading", apply: func(c *CliArgs, _ string) error { c.Heading = true; return nil }},
		{long: "no-heading", apply: func(c *CliArgs, _ string) error { c.NoHeading = true; return nil }},
		{long: "type-list", apply: func(c *CliArgs, _ string) error { c.TypeList = true; return nil }},

		// Mutually-exclusive boolean pairs
		{long: "line-number", short: 'n', apply: func(c *CliArgs, _ string) error { c.LineNumber = true; c.NoLineNumber = false; return nil }},
		{long: "no-line-number", short: 'N', apply: func(c *CliArgs, _ string) error { c.NoLineNumber = true; c.LineNumber = false; return nil }},
		{long: "with-filename", short: 'H', apply: func(c *CliArgs, _ string) error { c.WithFilename = true; c.NoFilename = false; return nil }},
		{long: "no-filename", short: 'I', apply: func(c *CliArgs, _ string) error { c.NoFilename = true; c.WithFilename = false; return nil }},

		// String value flags
		{long: "replace", short: 'r', needsValue: true, apply: func(c *CliArgs, v string) error { c.Replace = v; c.HasReplace = true; return nil }},
		{long: "type", short: 't', needsValue: true, apply: func(c *CliArgs, v string) error { c.Types = append(c.Types, v); return nil }},
		{long: "type-not", short: 'T', needsValue: true, apply: func(c *CliArgs, v string) error { c.TypesNot = append(c.TypesNot, v); return nil }},
		{long: "sort", needsValue: true, apply: func(c *CliArgs, v string) error { c.SortBy = v; return nil }},
		{long: "sortr", needsValue: true, apply: func(c *CliArgs, v string) error { c.SortBy = v; c.SortReverse = true; return nil }},
		{long: "glob", short: 'g', needsValue: true, apply: func(c *CliArgs, v string) error { c.Globs = append(c.Globs, v); return nil }},
		{long: "color", needsValue: true, apply: func(c *CliArgs, v string) error { c.Color = v; return nil }},

		// Integer value flags
		{long: "after-context", short: 'A', needsValue: true, apply: intFlag(func(c *CliArgs, v int) { c.AfterContext = v })},
		{long: "before-context", short: 'B', needsValue: true, apply: intFlag(func(c *CliArgs, v int) { c.BeforeContext = v })},
		{long: "context", short: 'C', needsValue: true, apply: intFlag(func(c *CliArgs, v int) { c.BeforeContext = v; c.AfterContext = v })},
		{long: "max-count", short: 'm', needsValue: true, apply: intFlag(func(c *CliArgs, v int) { c.MaxCount = v })},
		{long: "threads", short: 'j', needsValue: true, apply: intFlag(func(c *CliArgs, v int) { c.Threads = v })},
		{long: "max-depth", needsValue: true, apply: intFlag(func(c *CliArgs, v int) { c.MaxDepth = v })},
	}

	// Build lookup maps
	longMap := make(map[string]*flagDef)
	shortMap := make(map[rune]*flagDef)
	for i := range flags {
		f := &flags[i]
		longMap[f.long] = f
		if f.short != 0 {
			shortMap[f.short] = f
		}
	}

	// Helper: extract a value either from remaining runes or the next arg
	extractValue := func(runes []rune, pos int, argIdx int) (string, int, int) {
		if pos+1 < len(runes) {
			return string(runes[pos+1:]), len(runes), argIdx
		}
		if argIdx+1 < len(args) {
			return args[argIdx+1], pos, argIdx + 1
		}
		return "", pos, argIdx
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Special early-exit flags
		if arg == "-h" || arg == "--help" {
			cli.Help = true
			return cli, nil
		}
		if arg == "-V" || arg == "--version" {
			cli.Version = true
			return cli, nil
		}

		if strings.HasPrefix(arg, "--") {
			name := arg[2:]
			value := ""
			hasValue := false
			if idx := strings.Index(name, "="); idx != -1 {
				value = name[idx+1:]
				name = name[:idx]
				hasValue = true
			}

			f, ok := longMap[name]
			if !ok {
				return nil, fmt.Errorf("error: unknown flag: %s", arg)
			}

			if f.needsValue {
				if !hasValue {
					if i+1 < len(args) {
						value = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: --%s requires a value", f.long)
					}
				}
				if err := f.apply(cli, value); err != nil {
					return nil, err
				}
			} else {
				_ = f.apply(cli, "")
			}
		} else if strings.HasPrefix(arg, "-") && arg != "-" {
			runes := []rune(arg[1:])
			for j := 0; j < len(runes); j++ {
				r := runes[j]
				f, ok := shortMap[r]
				if !ok {
					return nil, fmt.Errorf("error: unknown flag: -%c", r)
				}

				if f.needsValue {
					val, newJ, newI := extractValue(runes, j, i)
					if val == "" {
						return nil, fmt.Errorf("error: -%c requires a value", r)
					}
					j = newJ
					i = newI
					if err := f.apply(cli, val); err != nil {
						return nil, err
					}
				} else {
					_ = f.apply(cli, "")
				}
			}
		} else {
			// Positional arguments
			if cli.Pattern == "" {
				cli.Pattern = arg
			} else {
				cli.Paths = append(cli.Paths, arg)
			}
		}
	}

	return cli, nil
}

// intFlag wraps a func(*CliArgs, int) as a func(*CliArgs, string) error for use in flagDef.
func intFlag(set func(*CliArgs, int)) func(*CliArgs, string) error {
	return func(c *CliArgs, v string) error {
		val, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("error: invalid integer: %s", v)
		}
		set(c, val)
		return nil
	}
}

func isStdinTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return true
	}
	return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}
