package goriggrep

import (
	"context"
	"github.com/startvibecoding/go-ripgrep/pkg/globset"
	"github.com/startvibecoding/go-ripgrep/pkg/ignore"
	"github.com/startvibecoding/go-ripgrep/pkg/matcher"
	"github.com/startvibecoding/go-ripgrep/pkg/printer"
	"github.com/startvibecoding/go-ripgrep/pkg/searcher"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Options specifies configurations for ripgrep search.
type Options struct {
	// Pattern settings
	Pattern         string
	IsFixed         bool
	CaseInsensitive bool
	WordRegexp      bool
	InvertMatch     bool

	// Replacement settings
	Replace    string
	HasReplace bool

	// Filtering settings
	NoIgnore       bool
	Hidden         bool
	FollowSymlinks bool
	MaxDepth       int
	Globs          []string // patterns for -g/--glob (negated ones exclude)
	Types          []string // patterns for -t/--type
	TypesNot       []string // patterns for -T/--type-not
	SearchZip      bool     // search inside compressed files

	// Sorting settings
	SortBy      string // "path", "modified", "size", or "none"
	SortReverse bool   // reverse sorting order

	// Context settings
	BeforeContext int
	AfterContext  int
	MaxCount      int // max matches per file

	// Execution settings
	Threads int // number of worker threads (0 or negative defaults to runtime.NumCPU())
}

// Search recursively searches the paths for the given options and streams the FileResults.
// The search is context-aware and terminates immediately if the context is cancelled.
func Search(ctx context.Context, paths []string, opts Options) (<-chan printer.FileResult, error) {
	// 1. Build pattern matcher
	m, err := matcher.BuildMatcher(opts.Pattern, opts.IsFixed, opts.CaseInsensitive, opts.WordRegexp)
	if err != nil {
		return nil, err
	}

	// 2. Build extra glob set (for -g/--glob filters)
	var globSet *globset.GlobSet
	if len(opts.Globs) > 0 {
		var err error
		globSet, err = globset.NewGlobSet(opts.Globs)
		if err != nil {
			return nil, err
		}
	}

	// Determine threads count
	threads := opts.Threads
	if opts.SortBy != "" && opts.SortBy != "none" {
		threads = 1
	} else if threads <= 0 {
		threads = runtime.NumCPU()
	}

	outChan := make(chan printer.FileResult, threads*2)
	filesChan := make(chan string, threads*4)

	// Start workers to search files
	var wg sync.WaitGroup
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := searcher.NewSearcher(m, opts.BeforeContext, opts.AfterContext, opts.MaxCount, opts.InvertMatch)
			s.SetContext(ctx)
			if opts.HasReplace {
				s.SetReplace(opts.Replace)
			}
			if opts.SearchZip {
				s.SetSearchZip(true)
			}
			for {
				select {
				case <-ctx.Done():
					return
				case path, ok := <-filesChan:
					if !ok {
						return
					}
					startTime := time.Now()
					results, err := s.SearchFile(path)
					if err == nil && len(results) > 0 {
						elapsed := time.Since(startTime)
						for _, res := range results {
							if res != nil {
								res.Elapsed = elapsed
								if len(res.Matches) > 0 {
									select {
									case <-ctx.Done():
										return
									case outChan <- *res:
									}
								}
							}
						}
					}
				}
			}
		}()
	}

	// Close outChan when all workers are done
	go func() {
		wg.Wait()
		close(outChan)
	}()

	// Start walking paths in a separate goroutine
	go func() {
		defer close(filesChan)

		for _, path := range paths {
			select {
			case <-ctx.Done():
				return
			default:
			}

			info, err := os.Lstat(path)
			if err != nil {
				continue
			}

			isSymlink := (info.Mode() & os.ModeSymlink) == os.ModeSymlink
			isDir := info.IsDir()

			if isSymlink && opts.FollowSymlinks {
				resolved, err := filepath.EvalSymlinks(path)
				if err == nil {
					stat, err := os.Stat(resolved)
					if err == nil {
						isDir = stat.IsDir()
						path = resolved
					}
				}
			}

			// If explicitly passed file, we bypass the walk ignore stack but still respect global -g glob filters
			if !isDir {
				if ignore.ShouldIgnoreByType(filepath.Base(path), opts.Types, opts.TypesNot) {
					continue
				}
				if globSet != nil {
					if globSet.MatchGlobFilter(path) {
						continue
					}
				}
				select {
				case <-ctx.Done():
					return
				case filesChan <- path:
				}
				continue
			}

			// Walk directory
			stack := ignore.NewIgnoreStack(opts.NoIgnore, opts.Hidden, opts.MaxDepth)
			stack.LoadBaseRules(path)
			walkDir(ctx, path, stack, 1, opts.MaxDepth, opts.FollowSymlinks, globSet, filesChan, opts.Types, opts.TypesNot, opts.SortBy, opts.SortReverse)
		}
	}()

	return outChan, nil
}

func walkDir(
	ctx context.Context,
	dirPath string,
	stack *ignore.IgnoreStack,
	depth int,
	maxDepth int,
	followSymlinks bool,
	globSet *globset.GlobSet,
	filesChan chan<- string,
	types []string,
	typesNot []string,
	sortBy string,
	sortReverse bool,
) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	if maxDepth > 0 && depth > maxDepth {
		return
	}

	// Push current directory's ignore rules to the stack
	stack.Push(dirPath)
	defer stack.Pop()

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return
	}
	sortDirEntries(dirPath, entries, sortBy, sortReverse)

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return
		default:
		}

		name := entry.Name()
		path := filepath.Join(dirPath, name)

		isDir := entry.IsDir()
		isSymlink := (entry.Type() & os.ModeSymlink) == os.ModeSymlink

		if isSymlink && followSymlinks {
			resolved, err := filepath.EvalSymlinks(path)
			if err == nil {
				stat, err := os.Stat(resolved)
				if err == nil {
					isDir = stat.IsDir()
					path = resolved
				}
			}
		}

		// 1. Check ignore files (.gitignore, .ignore, .rgignore) & hidden rules
		if stack.IsIgnored(path, isDir) {
			continue
		}

		// 2. Check -g/--glob option filters if active
		if globSet != nil {
			shouldSkip := globSet.MatchGlobFilter(path)
			if isDir {
				shouldSkip = globSet.MatchGlobFilterDir(path)
			}
			if shouldSkip {
				continue
			}
		}

		if isDir {
			// Clone stack for subdirectories so changes to stack are scoped
			subStack := stack.Clone()
			walkDir(ctx, path, subStack, depth+1, maxDepth, followSymlinks, globSet, filesChan, types, typesNot, sortBy, sortReverse)
		} else {
			if ignore.ShouldIgnoreByType(entry.Name(), types, typesNot) {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case filesChan <- path:
			}
		}
	}
}

func sortDirEntries(dirPath string, entries []os.DirEntry, sortBy string, reverse bool) {
	if len(entries) <= 1 || sortBy == "" || sortBy == "none" {
		return
	}

	switch sortBy {
	case "path":
		sort.Slice(entries, func(i, j int) bool {
			if reverse {
				return strings.Compare(entries[i].Name(), entries[j].Name()) > 0
			}
			return strings.Compare(entries[i].Name(), entries[j].Name()) < 0
		})
	case "modified", "size":
		type entryWithInfo struct {
			entry os.DirEntry
			info  os.FileInfo
		}
		list := make([]entryWithInfo, len(entries))
		for i, entry := range entries {
			list[i].entry = entry
			info, err := entry.Info()
			if err == nil {
				list[i].info = info
			}
		}

		sort.Slice(list, func(i, j int) bool {
			infoI, infoJ := list[i].info, list[j].info
			var cmp int
			if infoI == nil && infoJ == nil {
				cmp = strings.Compare(list[i].entry.Name(), list[j].entry.Name())
			} else if infoI == nil {
				cmp = 1
			} else if infoJ == nil {
				cmp = -1
			} else {
				if sortBy == "modified" {
					switch {
					case infoI.ModTime().Before(infoJ.ModTime()):
						cmp = -1
					case infoI.ModTime().After(infoJ.ModTime()):
						cmp = 1
					default:
						cmp = strings.Compare(list[i].entry.Name(), list[j].entry.Name())
					}
				} else { // "size"
					switch {
					case infoI.Size() < infoJ.Size():
						cmp = -1
					case infoI.Size() > infoJ.Size():
						cmp = 1
					default:
						cmp = strings.Compare(list[i].entry.Name(), list[j].entry.Name())
					}
				}
			}
			if reverse {
				return cmp > 0
			}
			return cmp < 0
		})

		for i := range list {
			entries[i] = list[i].entry
		}
	}
}
