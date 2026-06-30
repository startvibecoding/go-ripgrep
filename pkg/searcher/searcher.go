package searcher

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"github.com/startvibecoding/go-ripgrep/pkg/matcher"
	"github.com/startvibecoding/go-ripgrep/pkg/printer"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// maxLineLength is the maximum line length (in bytes) that will be read.
	// Lines longer than this are silently truncated to avoid unbounded memory growth.
	maxLineLength = 10 * 1024 * 1024 // 10 MB
)

// Searcher performs line-by-line searching of a single file/reader.
type Searcher struct {
	m             matcher.Matcher
	beforeContext int
	afterContext  int
	maxCount      int
	invertMatch   bool

	replace    string
	hasReplace bool
	searchZip  bool
	ctx        context.Context
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

func (s *Searcher) SetReplace(replace string) {
	s.replace = replace
	s.hasReplace = true
}

func (s *Searcher) SetSearchZip(active bool) {
	s.searchZip = active
}

func (s *Searcher) SetContext(ctx context.Context) {
	s.ctx = ctx
}

type bufferedLine struct {
	num  int
	text string
}

// SearchReader searches from an io.Reader (like os.File or standard input).
func (s *Searcher) SearchReader(r io.Reader, path string) (*printer.FileResult, error) {
	if err := s.checkCancelled(); err != nil {
		return nil, err
	}

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
		if err := s.checkCancelled(); err != nil {
			return nil, err
		}

		chunk, err := br.ReadSlice('\n')
		if len(chunk) > 0 {
			if len(lineBuf)+len(chunk) > maxLineLength {
				remaining := maxLineLength - len(lineBuf)
				if remaining > 0 {
					lineBuf = append(lineBuf, chunk[:remaining]...)
				}
				for err == bufio.ErrBufferFull {
					_, err = br.ReadSlice('\n')
				}
				lineCount++
				s.processLine(path, lineBuf, lineCount, &matches, &matchCount, &beforeBuf, &afterCount, &lastPrintedLineNum)
				lineBuf = lineBuf[:0]
				if err == io.EOF {
					break
				}
				if err == nil {
					if s.maxCount > 0 && matchCount >= s.maxCount {
						break
					}
					continue
				}
			}
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

// SearchFile opens a file and searches it, potentially decompressing or unpacking it.
func (s *Searcher) SearchFile(path string) ([]*printer.FileResult, error) {
	if err := s.checkCancelled(); err != nil {
		return nil, err
	}

	if s.searchZip {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".gz":
			f, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			gr, err := gzip.NewReader(f)
			if err != nil {
				return nil, err
			}
			defer gr.Close()
			res, err := s.SearchReader(gr, path)
			if err != nil {
				return nil, err
			}
			return []*printer.FileResult{res}, nil

		case ".bz2":
			f, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			br := bzip2.NewReader(f)
			res, err := s.SearchReader(br, path)
			if err != nil {
				return nil, err
			}
			return []*printer.FileResult{res}, nil

		case ".zip":
			zr, err := zip.OpenReader(path)
			if err != nil {
				return nil, err
			}
			defer zr.Close()

			var results []*printer.FileResult
			for _, file := range zr.File {
				if err := s.checkCancelled(); err != nil {
					return nil, err
				}
				if file.FileInfo().IsDir() {
					continue
				}
				rc, err := file.Open()
				if err != nil {
					continue
				}
				innerPath := path + "//" + file.Name
				res, err := s.SearchReader(rc, innerPath)
				rc.Close()
				if err == nil && res != nil {
					results = append(results, res)
				}
			}
			return results, nil
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	res, err := s.SearchReader(f, path)
	if err != nil {
		return nil, err
	}
	return []*printer.FileResult{res}, nil
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

		// 2. Compile submatches and perform replacement if needed
		var subs []printer.Submatch
		if !s.invertMatch {
			if s.hasReplace {
				replacedBytes, newSpans := s.m.Replace(trimmedLineBytes, []byte(s.replace))
				lineStr = string(replacedBytes)
				subs = make([]printer.Submatch, len(newSpans))
				for i, span := range newSpans {
					subs[i] = printer.Submatch{
						Start: span[0],
						End:   span[1],
						Text:  string(replacedBytes[span[0]:span[1]]),
					}
				}
			} else {
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

func (s *Searcher) checkCancelled() error {
	if s.ctx == nil {
		return nil
	}
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
		return nil
	}
}
