package searcher

import (
	"bufio"
	"bytes"
	"go-ripgrep/pkg/matcher"
	"go-ripgrep/pkg/printer"
	"io"
	"os"
)

// Searcher performs line-by-line searching of a single file/reader.
type Searcher struct {
	m             matcher.Matcher
	beforeContext int
	afterContext  int
	maxCount      int
	invertMatch   bool
}

// NewSearcher creates a Searcher configured with the given options.
func NewSearcher(m matcher.Matcher, beforeContext, afterContext, maxCount int, invertMatch bool) *Searcher {
	return &Searcher{
		m:             m,
		beforeContext: beforeContext,
		afterContext:  afterContext,
		maxCount:      maxCount,
		invertMatch:   invertMatch,
	}
}

type bufferedLine struct {
	num  int
	text string
}

// SearchReader searches from an io.Reader (like os.File or standard input).
func (s *Searcher) SearchReader(r io.Reader, path string) (*printer.FileResult, error) {
	// Read a small prefix to detect if it's binary
	br := bufio.NewReader(r)
	prefix, err := br.Peek(1024)
	if err != nil && err != io.EOF {
		return nil, err
	}

	// Heuristic: if contains a NUL byte, consider it binary and skip
	if bytes.IndexByte(prefix, 0) != -1 {
		return &printer.FileResult{
			Path:  path,
			Stats: printer.FileStats{},
		}, nil
	}

	var matches []printer.SearchMatch
	var lineCount int
	var matchCount int

	// Context tracking
	beforeBuf := make([]bufferedLine, 0, s.beforeContext)
	afterCount := 0
	lastPrintedLineNum := 0

	// Custom reader to avoid string allocations for every line
	var lineBuf []byte
	for {
		chunk, err := br.ReadSlice('\n')
		if len(chunk) > 0 {
			lineBuf = append(lineBuf, chunk...)
			if err == bufio.ErrBufferFull {
				// Line too long for ReadSlice buffer, read more in next iterations
				continue
			}
		}

		if err != nil {
			if err == io.EOF {
				if len(lineBuf) > 0 {
					lineCount++
					s.processLine(path, lineBuf, lineCount, &matches, &matchCount, &beforeBuf, &afterCount, &lastPrintedLineNum)
				}
				break
			}
			return nil, err
		}

		lineCount++
		s.processLine(path, lineBuf, lineCount, &matches, &matchCount, &beforeBuf, &afterCount, &lastPrintedLineNum)
		lineBuf = lineBuf[:0] // reuse buffer

		if s.maxCount > 0 && matchCount >= s.maxCount {
			break
		}
	}

	return &printer.FileResult{
		Path:    path,
		Matches: matches,
		Stats: printer.FileStats{
			SearchedLines: lineCount,
			Matches:       matchCount,
		},
	}, nil
}

// SearchFile opens a file and searches it.
func (s *Searcher) SearchFile(path string) (*printer.FileResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return s.SearchReader(f, path)
}

func (s *Searcher) processLine(
	path string,
	lineBytes []byte,
	lineNum int,
	matches *[]printer.SearchMatch,
	matchCount *int,
	beforeBuf *[]bufferedLine,
	afterCount *int,
	lastPrintedLineNum *int,
) {
	// Strip trailing newline characters for matching and storage
	trimmedLineBytes := bytes.TrimRight(lineBytes, "\r\n")
	lineStr := string(trimmedLineBytes)

	hasMatch := s.m.Match(trimmedLineBytes)
	isReportedMatch := hasMatch

	if s.invertMatch {
		isReportedMatch = !hasMatch
	}

	if isReportedMatch {
		*matchCount++

		// 1. Print before context if needed
		if s.beforeContext > 0 {
			for _, bl := range *beforeBuf {
				if bl.num > *lastPrintedLineNum {
					*matches = append(*matches, printer.SearchMatch{
						Line:      bl.text,
						LineNum:   bl.num,
						IsContext: true,
					})
					*lastPrintedLineNum = bl.num
				}
			}
			*beforeBuf = (*beforeBuf)[:0] // clear before context
		}

		// 2. Compile submatches
		var subs []printer.Submatch
		if !s.invertMatch {
			spans := s.m.FindSpans(trimmedLineBytes)
			subs = make([]printer.Submatch, len(spans))
			for i, span := range spans {
				subs[i] = printer.Submatch{
					Start: span[0],
					End:   span[1],
					Text:  string(trimmedLineBytes[span[0]:span[1]]),
				}
			}
		}

		// 3. Print match line
		if lineNum > *lastPrintedLineNum {
			*matches = append(*matches, printer.SearchMatch{
				Line:       lineStr,
				LineNum:    lineNum,
				IsContext:  false,
				Submatches: subs,
			})
			*lastPrintedLineNum = lineNum
		}

		// 4. Reset after context count
		*afterCount = s.afterContext

	} else {
		// No match on this line
		if *afterCount > 0 {
			// This line serves as after-context
			if lineNum > *lastPrintedLineNum {
				*matches = append(*matches, printer.SearchMatch{
					Line:      lineStr,
					LineNum:   lineNum,
					IsContext: true,
				})
				*lastPrintedLineNum = lineNum
			}
			*afterCount--
		} else if s.beforeContext > 0 {
			// Save in before context buffer
			*beforeBuf = append(*beforeBuf, bufferedLine{
				num:  lineNum,
				text: lineStr,
			})
			if len(*beforeBuf) > s.beforeContext {
				*beforeBuf = (*beforeBuf)[1:]
			}
		}
	}
}
