package matcher

import (
	"reflect"
	"testing"
)

func TestRegexMatcher(t *testing.T) {
	m, err := BuildMatcher("hello", false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	line := []byte("hello world")
	if !m.Match(line) {
		t.Error("expected match")
	}

	spans := m.FindSpans(line)
	expectedSpans := [][2]int{{0, 5}}
	if !reflect.DeepEqual(spans, expectedSpans) {
		t.Errorf("expected spans %v, got %v", expectedSpans, spans)
	}
}

func TestRegexMatcherCaseInsensitive(t *testing.T) {
	m, err := BuildMatcher("hElLo", false, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	line := []byte("hello world")
	if !m.Match(line) {
		t.Error("expected match")
	}

	spans := m.FindSpans(line)
	expectedSpans := [][2]int{{0, 5}}
	if !reflect.DeepEqual(spans, expectedSpans) {
		t.Errorf("expected spans %v, got %v", expectedSpans, spans)
	}
}

func TestRegexMatcherWordRegexp(t *testing.T) {
	m, err := BuildMatcher("cat", false, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.Match([]byte("category")) {
		t.Error("expected no match on category")
	}

	if !m.Match([]byte("a cat is here")) {
		t.Error("expected match on a cat is here")
	}
}

func TestFixedMatcher(t *testing.T) {
	m, err := BuildMatcher("world", true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	line := []byte("hello world world")
	if !m.Match(line) {
		t.Error("expected match")
	}

	spans := m.FindSpans(line)
	expectedSpans := [][2]int{{6, 11}, {12, 17}}
	if !reflect.DeepEqual(spans, expectedSpans) {
		t.Errorf("expected spans %v, got %v", expectedSpans, spans)
	}
}

func TestFixedMatcherCaseInsensitive(t *testing.T) {
	m, err := BuildMatcher("WoRlD", true, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	line := []byte("hello world")
	if !m.Match(line) {
		t.Error("expected match")
	}

	spans := m.FindSpans(line)
	expectedSpans := [][2]int{{6, 11}}
	if !reflect.DeepEqual(spans, expectedSpans) {
		t.Errorf("expected spans %v, got %v", expectedSpans, spans)
	}
}
