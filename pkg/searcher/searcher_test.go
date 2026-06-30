package searcher

import (
	"bytes"
	"context"
	"github.com/startvibecoding/go-ripgrep/pkg/matcher"
	"testing"
)

func TestSearcherBasic(t *testing.T) {
	m, err := matcher.BuildMatcher("world", true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := []byte("hello\nworld\nfoo\nworld bar\n")
	s := NewSearcher(m, 0, 0, 0, false)

	res, err := s.SearchReader(bytes.NewReader(content), "test.txt")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if res.Stats.Matches != 2 {
		t.Errorf("expected 2 matches, got %d", res.Stats.Matches)
	}
	if len(res.Matches) != 2 {
		t.Fatalf("expected 2 match items, got %d", len(res.Matches))
	}

	if res.Matches[0].Line != "world" || res.Matches[0].LineNum != 2 {
		t.Errorf("unexpected match: %+v", res.Matches[0])
	}
	if res.Matches[1].Line != "world bar" || res.Matches[1].LineNum != 4 {
		t.Errorf("unexpected match: %+v", res.Matches[1])
	}
}

func TestSearcherContext(t *testing.T) {
	m, err := matcher.BuildMatcher("world", true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Line 1: hello
	// Line 2: world (match)
	// Line 3: foo
	// Line 4: bar
	content := []byte("hello\nworld\nfoo\nbar\n")
	s := NewSearcher(m, 1, 1, 0, false)

	res, err := s.SearchReader(bytes.NewReader(content), "test.txt")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	// Expected:
	// Line 1 (context)
	// Line 2 (match)
	// Line 3 (context)
	if len(res.Matches) != 3 {
		t.Fatalf("expected 3 items, got %d", len(res.Matches))
	}

	if !res.Matches[0].IsContext || res.Matches[0].LineNum != 1 || res.Matches[0].Line != "hello" {
		t.Errorf("expected line 1 context, got %+v", res.Matches[0])
	}
	if res.Matches[1].IsContext || res.Matches[1].LineNum != 2 || res.Matches[1].Line != "world" {
		t.Errorf("expected line 2 match, got %+v", res.Matches[1])
	}
	if !res.Matches[2].IsContext || res.Matches[2].LineNum != 3 || res.Matches[2].Line != "foo" {
		t.Errorf("expected line 3 context, got %+v", res.Matches[2])
	}
}

func TestSearcherInvertMatch(t *testing.T) {
	m, err := matcher.BuildMatcher("world", true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := []byte("hello\nworld\nfoo\n")
	s := NewSearcher(m, 0, 0, 0, true)

	res, err := s.SearchReader(bytes.NewReader(content), "test.txt")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if res.Stats.Matches != 2 {
		t.Errorf("expected 2 matches, got %d", res.Stats.Matches)
	}
}

func TestSearcherCancellation(t *testing.T) {
	m, err := matcher.BuildMatcher("world", true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewSearcher(m, 0, 0, 0, false)
	s.SetContext(ctx)

	if _, err := s.SearchReader(bytes.NewReader([]byte("hello\nworld\n")), "test.txt"); err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
}

func TestSearcherReplace(t *testing.T) {
	m, err := matcher.BuildMatcher("world", true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := []byte("hello world\nfoo bar\n")
	s := NewSearcher(m, 0, 0, 0, false)
	s.SetReplace("earth")

	res, err := s.SearchReader(bytes.NewReader(content), "test.txt")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(res.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(res.Matches))
	}

	if res.Matches[0].Line != "hello earth" {
		t.Errorf("expected 'hello earth', got %q", res.Matches[0].Line)
	}

	if len(res.Matches[0].Submatches) != 1 {
		t.Fatalf("expected 1 submatch, got %d", len(res.Matches[0].Submatches))
	}

	if res.Matches[0].Submatches[0].Text != "earth" {
		t.Errorf("expected submatch text 'earth', got %q", res.Matches[0].Submatches[0].Text)
	}
}

func TestSearcherBinaryDetection(t *testing.T) {
	m, err := matcher.BuildMatcher("hello", true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Content with a NUL byte in the first 1024 bytes
	content := make([]byte, 100)
	copy(content, "hello\x00world\n")

	s := NewSearcher(m, 0, 0, 0, false)
	res, err := s.SearchReader(bytes.NewReader(content), "binary.bin")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if res.Stats.Matches != 0 {
		t.Errorf("expected 0 matches for binary file, got %d", res.Stats.Matches)
	}
	if len(res.Matches) != 0 {
		t.Errorf("expected 0 match items for binary file, got %d", len(res.Matches))
	}
}

func TestSearcherMaxLineLength(t *testing.T) {
	m, err := matcher.BuildMatcher("hello", true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	longLine := append([]byte("hello"), bytes.Repeat([]byte("a"), maxLineLength+1)...)
	content := append(longLine, '\n')
	content = append(content, []byte("hello world\n")...)

	s := NewSearcher(m, 0, 0, 0, false)
	res, err := s.SearchReader(bytes.NewReader(content), "test.txt")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if res.Stats.Matches != 2 {
		t.Errorf("expected 2 matches, got %d", res.Stats.Matches)
	}
	if len(res.Matches) != 2 {
		t.Fatalf("expected 2 match items, got %d", len(res.Matches))
	}
	if len(res.Matches[0].Line) != maxLineLength {
		t.Errorf("expected first long line to be truncated to %d bytes, got %d", maxLineLength, len(res.Matches[0].Line))
	}
}

func TestSearcherMaxCount(t *testing.T) {
	m, err := matcher.BuildMatcher("match", true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := []byte("match line 1\nmatch line 2\nmatch line 3\nno match\n")
	s := NewSearcher(m, 0, 0, 2, false)

	res, err := s.SearchReader(bytes.NewReader(content), "test.txt")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if res.Stats.Matches != 2 {
		t.Errorf("expected 2 matches (max-count), got %d", res.Stats.Matches)
	}
}
