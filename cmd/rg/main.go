package main

import (
	"context"
	"fmt"
	"go-ripgrep"
	"go-ripgrep/pkg/matcher"
	"go-ripgrep/pkg/printer"
	"go-ripgrep/pkg/searcher"
	"io"
	"os"
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

	Help    bool
	Version bool
}

const version = "15.1.0-go"

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
    -g, --glob GLOB            Include or exclude files/directories using globs.
    -A, --after-context NUM    Show NUM lines after each match.
    -B, --before-context NUM   Show NUM lines before each match.
    -C, --context NUM          Show NUM lines before and after each match.
    -m, --max-count NUM        Limit matches per file to NUM.
    -j, --threads NUM          Number of threads to use.
    --hidden                   Search hidden files and directories.
    --no-ignore                Do not respect ignore files (.gitignore, .ignore, etc.).
    -L, --follow               Follow symbolic links.
    --json                     Output newline-delimited JSON.
    --color WHEN               Whether to use color: always, never, auto. [default: auto]
    --heading                  Print heading for matches from each file. [default: when on terminal]
    --no-heading               Do not print heading for matches.
    -n, --line-number          Show line numbers. [default: on]
    -N, --no-line-number       Suppress line numbers.
    -H, --with-filename        Print the file path for each match.
    -I, --no-filename          Suppress file path for each match.
    --column                   Show 1-based column number of first match.
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

	if searchStdin {
		m, err := matcher.BuildMatcher(cli.Pattern, cli.FixedStrings, cli.CaseInsensitive, cli.WordRegexp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error compiling pattern: %v\n", err)
			os.Exit(2)
		}
		s := searcher.NewSearcher(m, cli.BeforeContext, cli.AfterContext, cli.MaxCount, cli.InvertMatch)
		res, err := s.SearchReader(os.Stdin, "<stdin>")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}

		pConfig := printer.Config{
			Group:         false,
			Color:         colorEnabled,
			WithLineNum:   !cli.NoLineNumber,
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
		WithLineNum:   !cli.NoLineNumber,
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
		Color:      "auto",
		LineNumber: true,
	}

	n := len(args)
	for i := 0; i < n; i++ {
		arg := args[i]
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

			switch name {
			case "ignore-case":
				cli.CaseInsensitive = true
			case "case-sensitive":
				cli.CaseInsensitive = false
			case "word-regexp":
				cli.WordRegexp = true
			case "fixed-strings":
				cli.FixedStrings = true
			case "invert-match":
				cli.InvertMatch = true
			case "hidden":
				cli.Hidden = true
			case "no-ignore":
				cli.NoIgnore = true
			case "follow":
				cli.FollowSymlinks = true
			case "json":
				cli.JSON = true
			case "only-matching":
				cli.OnlyMatching = true
			case "count":
				cli.Count = true
			case "quiet":
				cli.Quiet = true
			case "column":
				cli.Column = true
			case "heading":
				cli.Heading = true
			case "no-heading":
				cli.NoHeading = true
			case "line-number":
				cli.LineNumber = true
				cli.NoLineNumber = false
			case "no-line-number":
				cli.NoLineNumber = true
				cli.LineNumber = false
			case "with-filename":
				cli.WithFilename = true
				cli.NoFilename = false
			case "no-filename":
				cli.NoFilename = true
				cli.WithFilename = false
			case "glob":
				if !hasValue {
					if i+1 < n {
						value = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: --glob requires a value")
					}
				}
				cli.Globs = append(cli.Globs, value)
			case "after-context":
				if !hasValue {
					if i+1 < n {
						value = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: --after-context requires a value")
					}
				}
				val, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("error: --after-context invalid integer: %s", value)
				}
				cli.AfterContext = val
			case "before-context":
				if !hasValue {
					if i+1 < n {
						value = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: --before-context requires a value")
					}
				}
				val, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("error: --before-context invalid integer: %s", value)
				}
				cli.BeforeContext = val
			case "context":
				if !hasValue {
					if i+1 < n {
						value = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: --context requires a value")
					}
				}
				val, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("error: --context invalid integer: %s", value)
				}
				cli.BeforeContext = val
				cli.AfterContext = val
			case "max-count":
				if !hasValue {
					if i+1 < n {
						value = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: --max-count requires a value")
					}
				}
				val, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("error: --max-count invalid integer: %s", value)
				}
				cli.MaxCount = val
			case "threads":
				if !hasValue {
					if i+1 < n {
						value = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: --threads requires a value")
					}
				}
				val, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("error: --threads invalid integer: %s", value)
				}
				cli.Threads = val
			case "color":
				if !hasValue {
					if i+1 < n {
						value = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: --color requires a value")
					}
				}
				cli.Color = value
			case "max-depth":
				if !hasValue {
					if i+1 < n {
						value = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: --max-depth requires a value")
					}
				}
				val, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("error: --max-depth invalid integer: %s", value)
				}
				cli.MaxDepth = val
			default:
				return nil, fmt.Errorf("error: unknown flag: %s", arg)
			}
		} else if strings.HasPrefix(arg, "-") && arg != "-" {
			// Parse short flags
			runes := []rune(arg[1:])
			for j := 0; j < len(runes); j++ {
				r := runes[j]
				switch r {
				case 'i':
					cli.CaseInsensitive = true
				case 's':
					cli.CaseInsensitive = false
				case 'w':
					cli.WordRegexp = true
				case 'F':
					cli.FixedStrings = true
				case 'v':
					cli.InvertMatch = true
				case 'L':
					cli.FollowSymlinks = true
				case 'o':
					cli.OnlyMatching = true
				case 'c':
					cli.Count = true
				case 'q':
					cli.Quiet = true
				case 'n':
					cli.LineNumber = true
					cli.NoLineNumber = false
				case 'N':
					cli.NoLineNumber = true
					cli.LineNumber = false
				case 'H':
					cli.WithFilename = true
					cli.NoFilename = false
				case 'I':
					cli.NoFilename = true
					cli.WithFilename = false
				case 'g':
					// Glob needs a value. Check remaining chars first.
					val := ""
					if j+1 < len(runes) {
						val = string(runes[j+1:])
						j = len(runes) // consume the rest
					} else if i+1 < n {
						val = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: -g requires a value")
					}
					cli.Globs = append(cli.Globs, val)
				case 'A':
					valStr := ""
					if j+1 < len(runes) {
						valStr = string(runes[j+1:])
						j = len(runes)
					} else if i+1 < n {
						valStr = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: -A requires a value")
					}
					val, err := strconv.Atoi(valStr)
					if err != nil {
						return nil, fmt.Errorf("error: -A invalid integer: %s", valStr)
					}
					cli.AfterContext = val
				case 'B':
					valStr := ""
					if j+1 < len(runes) {
						valStr = string(runes[j+1:])
						j = len(runes)
					} else if i+1 < n {
						valStr = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: -B requires a value")
					}
					val, err := strconv.Atoi(valStr)
					if err != nil {
						return nil, fmt.Errorf("error: -B invalid integer: %s", valStr)
					}
					cli.BeforeContext = val
				case 'C':
					valStr := ""
					if j+1 < len(runes) {
						valStr = string(runes[j+1:])
						j = len(runes)
					} else if i+1 < n {
						valStr = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: -C requires a value")
					}
					val, err := strconv.Atoi(valStr)
					if err != nil {
						return nil, fmt.Errorf("error: -C invalid integer: %s", valStr)
					}
					cli.BeforeContext = val
					cli.AfterContext = val
				case 'm':
					valStr := ""
					if j+1 < len(runes) {
						valStr = string(runes[j+1:])
						j = len(runes)
					} else if i+1 < n {
						valStr = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: -m requires a value")
					}
					val, err := strconv.Atoi(valStr)
					if err != nil {
						return nil, fmt.Errorf("error: -m invalid integer: %s", valStr)
					}
					cli.MaxCount = val
				case 'j':
					valStr := ""
					if j+1 < len(runes) {
						valStr = string(runes[j+1:])
						j = len(runes)
					} else if i+1 < n {
						valStr = args[i+1]
						i++
					} else {
						return nil, fmt.Errorf("error: -j requires a value")
					}
					val, err := strconv.Atoi(valStr)
					if err != nil {
						return nil, fmt.Errorf("error: -j invalid integer: %s", valStr)
					}
					cli.Threads = val
				default:
					return nil, fmt.Errorf("error: unknown flag: -%c", r)
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

func isStdinTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return true
	}
	return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}
