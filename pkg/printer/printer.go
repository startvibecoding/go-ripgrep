package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Submatch represents a matched portion of a line.
type Submatch struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
}

// SearchMatch represents a single matched line or context line.
type SearchMatch struct {
	Line       string     `json:"line"`
	LineNum    int        `json:"line_number"`
	IsContext  bool       `json:"is_context"`
	Submatches []Submatch `json:"submatches"`
}

// FileStats holds stats for a searched file.
type FileStats struct {
	SearchedLines int `json:"searched_lines"`
	Matches       int `json:"matches"`
}

// FileResult represents the complete search results for one file.
type FileResult struct {
	Path    string        `json:"path"`
	Matches []SearchMatch `json:"matches"`
	Stats   FileStats     `json:"stats"`
	Elapsed time.Duration `json:"elapsed"`
}

// Config controls printer options.
type Config struct {
	Group         bool // Group matches by filename
	Color         bool // Enable ANSI colors
	JSON          bool // Output NDJSON format
	WithLineNum   bool // Show line numbers
	WithFilename  bool // Show filenames
	WithColumnNum bool // Show column number of first match
	OnlyMatching  bool // Show only matching parts of line
	Count         bool // Show only match count per file
}

// Printer formats and outputs search results.
type Printer struct {
	cfg Config
	w   io.Writer
}

// NewPrinter creates a new Printer.
func NewPrinter(w io.Writer, cfg Config) *Printer {
	return &Printer{cfg: cfg, w: w}
}

// PrintFileResult prints the search result of a single file.
func (p *Printer) PrintFileResult(res FileResult) error {
	if p.cfg.JSON {
		return p.printJSON(res)
	}

	if len(res.Matches) == 0 {
		return nil
	}

	if p.cfg.Count {
		if p.cfg.WithFilename {
			if p.cfg.Color {
				fmt.Fprintf(p.w, "\x1b[35m%s\x1b[0m:%d\n", res.Path, res.Stats.Matches)
			} else {
				fmt.Fprintf(p.w, "%s:%d\n", res.Path, res.Stats.Matches)
			}
		} else {
			fmt.Fprintf(p.w, "%d\n", res.Stats.Matches)
		}
		return nil
	}

	if p.cfg.Group {
		// Grouped style:
		// path/to/file
		// 10:match
		// 11-context
		if p.cfg.WithFilename {
			if p.cfg.Color {
				fmt.Fprintf(p.w, "\x1b[35m%s\x1b[0m\n", res.Path)
			} else {
				fmt.Fprintf(p.w, "%s\n", res.Path)
			}
		}

		for _, m := range res.Matches {
			p.printLine(res.Path, m, false)
		}
		fmt.Fprintln(p.w) // Empty line between files
	} else {
		// Non-grouped style:
		// path/to/file:10:match
		for _, m := range res.Matches {
			p.printLine(res.Path, m, true)
		}
	}

	return nil
}

func (p *Printer) printLine(path string, m SearchMatch, printPath bool) {
	if p.cfg.OnlyMatching && len(m.Submatches) > 0 && !m.IsContext {
		for _, sub := range m.Submatches {
			var sb strings.Builder
			if printPath && p.cfg.WithFilename {
				if p.cfg.Color {
					sb.WriteString(fmt.Sprintf("\x1b[35m%s\x1b[0m:", path))
				} else {
					sb.WriteString(path + ":")
				}
			}
			if p.cfg.WithLineNum {
				lineNumStr := fmt.Sprintf("%d", m.LineNum)
				if p.cfg.Color {
					sb.WriteString(fmt.Sprintf("\x1b[32m%s\x1b[0m:", lineNumStr))
				} else {
					sb.WriteString(lineNumStr + ":")
				}
			}
			if p.cfg.WithColumnNum {
				colStr := fmt.Sprintf("%d", sub.Start+1)
				sb.WriteString(colStr + ":")
			}
			if p.cfg.Color {
				sb.WriteString(fmt.Sprintf("\x1b[1;31m%s\x1b[0m", sub.Text))
			} else {
				sb.WriteString(sub.Text)
			}
			fmt.Fprintln(p.w, sb.String())
		}
		return
	}

	var sb strings.Builder

	if printPath && p.cfg.WithFilename {
		if p.cfg.Color {
			sb.WriteString(fmt.Sprintf("\x1b[35m%s\x1b[0m", path))
		} else {
			sb.WriteString(path)
		}
		if m.IsContext {
			sb.WriteRune('-')
		} else {
			sb.WriteRune(':')
		}
	}

	if p.cfg.WithLineNum {
		lineNumStr := fmt.Sprintf("%d", m.LineNum)
		if p.cfg.Color {
			sb.WriteString(fmt.Sprintf("\x1b[32m%s\x1b[0m", lineNumStr))
		} else {
			sb.WriteString(lineNumStr)
		}
		if m.IsContext {
			sb.WriteRune('-')
		} else {
			sb.WriteRune(':')
		}
	}

	if p.cfg.WithColumnNum && len(m.Submatches) > 0 && !m.IsContext {
		colStr := fmt.Sprintf("%d", m.Submatches[0].Start+1)
		sb.WriteString(colStr)
		sb.WriteRune(':')
	}

	// Print match line text with coloring if active
	if p.cfg.Color && len(m.Submatches) > 0 && !m.IsContext {
		// Highlight matched spans in red and bold
		lastIdx := 0
		lineBytes := []byte(m.Line)
		for _, sub := range m.Submatches {
			if sub.Start >= lastIdx && sub.End <= len(lineBytes) {
				sb.Write(lineBytes[lastIdx:sub.Start])
				sb.WriteString("\x1b[1;31m")
				sb.Write(lineBytes[sub.Start:sub.End])
				sb.WriteString("\x1b[0m")
				lastIdx = sub.End
			}
		}
		if lastIdx < len(lineBytes) {
			sb.Write(lineBytes[lastIdx:])
		}
	} else {
		sb.WriteString(m.Line)
	}

	// Strip trailing newline if any, and print
	lineOut := strings.TrimRight(sb.String(), "\r\n")
	fmt.Fprintln(p.w, lineOut)
}

// Ripgrep JSON types
type jsonMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type jsonDataText struct {
	Text string `json:"text"`
}

type jsonBeginData struct {
	Path jsonDataText `json:"path"`
}

type jsonMatchData struct {
	Path       jsonDataText   `json:"path"`
	Lines      jsonDataText   `json:"lines"`
	LineNumber int            `json:"line_number"`
	Submatches []jsonSubmatch `json:"submatches"`
}

type jsonSubmatch struct {
	Match jsonDataText `json:"match"`
	Start int          `json:"start"`
	End   int          `json:"end"`
}

type jsonEndStats struct {
	SearchedLines int `json:"searched_lines"`
	Matches       int `json:"matches"`
}

type jsonEndData struct {
	Path   jsonDataText `json:"path"`
	Binary bool         `json:"binary"`
	Stats  jsonEndStats `json:"stats"`
}

func (p *Printer) printJSON(res FileResult) error {
	enc := json.NewEncoder(p.w)

	// 1. Emit "begin" message
	begin := jsonMessage{
		Type: "begin",
		Data: jsonBeginData{
			Path: jsonDataText{Text: res.Path},
		},
	}
	if err := enc.Encode(begin); err != nil {
		return err
	}

	// 2. Emit "match" or "context" messages
	for _, m := range res.Matches {
		if m.IsContext {
			contextMsg := jsonMessage{
				Type: "context",
				Data: jsonMatchData{
					Path:       jsonDataText{Text: res.Path},
					Lines:      jsonDataText{Text: m.Line},
					LineNumber: m.LineNum,
					Submatches: []jsonSubmatch{},
				},
			}
			if err := enc.Encode(contextMsg); err != nil {
				return err
			}
		} else {
			submatches := make([]jsonSubmatch, len(m.Submatches))
			for i, sub := range m.Submatches {
				submatches[i] = jsonSubmatch{
					Match: jsonDataText{Text: sub.Text},
					Start: sub.Start,
					End:   sub.End,
				}
			}

			matchMsg := jsonMessage{
				Type: "match",
				Data: jsonMatchData{
					Path:       jsonDataText{Text: res.Path},
					Lines:      jsonDataText{Text: m.Line},
					LineNumber: m.LineNum,
					Submatches: submatches,
				},
			}
			if err := enc.Encode(matchMsg); err != nil {
				return err
			}
		}
	}

	// 3. Emit "end" message
	end := jsonMessage{
		Type: "end",
		Data: jsonEndData{
			Path:   jsonDataText{Text: res.Path},
			Binary: false,
			Stats: jsonEndStats{
				SearchedLines: res.Stats.SearchedLines,
				Matches:       res.Stats.Matches,
			},
		},
	}
	return enc.Encode(end)
}

// PrintSummary prints overall execution summary in JSON or CLI format if required.
func (p *Printer) PrintSummary(totalFiles, totalMatches, totalLines int, elapsed time.Duration) error {
	if p.cfg.JSON {
		enc := json.NewEncoder(p.w)
		type jsonSummaryStats struct {
			Elapsed       interface{} `json:"elapsed"`
			SearchedLines int         `json:"searched_lines"`
			Matches       int         `json:"matches"`
		}
		type jsonSummaryData struct {
			Stats jsonSummaryStats `json:"stats"`
		}
		type elapsedData struct {
			Secs  int64 `json:"secs"`
			Nanos int64 `json:"nanos"`
		}

		summary := jsonMessage{
			Type: "summary",
			Data: jsonSummaryData{
				Stats: jsonSummaryStats{
					Elapsed: elapsedData{
						Secs:  int64(elapsed.Seconds()),
						Nanos: int64(elapsed.Nanoseconds() % 1e9),
					},
					SearchedLines: totalLines,
					Matches:       totalMatches,
				},
			},
		}
		return enc.Encode(summary)
	}
	return nil
}

// IsTerminal returns true if fd is a terminal.
func IsTerminal() bool {
	o, _ := os.Stdout.Stat()
	return (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}
