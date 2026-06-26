package searcher

import (
	"bytes"
	"go-ripgrep/pkg/matcher"
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
