package printer

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrinterGrouped(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Group:        true,
		Color:        false,
		WithLineNum:  true,
		WithFilename: true,
	}
	p := NewPrinter(&buf, cfg)

	res := FileResult{
		Path: "test.txt",
		Matches: []SearchMatch{
			{
				Line:      "hello world\n",
				LineNum:   1,
				IsContext: false,
				Submatches: []Submatch{
					{Start: 0, End: 5, Text: "hello"},
				},
			},
		},
		Stats: FileStats{SearchedLines: 10, Matches: 1},
	}

	if err := p.PrintFileResult(res); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "test.txt\n1:hello world\n\n"
	if buf.String() != expected {
		t.Errorf("expected output:\n%q\ngot:\n%q", expected, buf.String())
	}
}

func TestPrinterNonGrouped(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Group:        false,
		Color:        false,
		WithLineNum:  true,
		WithFilename: true,
	}
	p := NewPrinter(&buf, cfg)

	res := FileResult{
		Path: "test.txt",
		Matches: []SearchMatch{
			{
				Line:      "hello world\n",
				LineNum:   1,
				IsContext: false,
				Submatches: []Submatch{
					{Start: 0, End: 5, Text: "hello"},
				},
			},
		},
		Stats: FileStats{SearchedLines: 10, Matches: 1},
	}

	if err := p.PrintFileResult(res); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "test.txt:1:hello world\n"
	if buf.String() != expected {
		t.Errorf("expected output:\n%q\ngot:\n%q", expected, buf.String())
	}
}

func TestPrinterJSON(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{JSON: true}
	p := NewPrinter(&buf, cfg)

	res := FileResult{
		Path: "test.txt",
		Matches: []SearchMatch{
			{
				Line:      "hello world\n",
				LineNum:   1,
				IsContext: false,
				Submatches: []Submatch{
					{Start: 0, End: 5, Text: "hello"},
				},
			},
		},
		Stats: FileStats{SearchedLines: 10, Matches: 1},
	}

	if err := p.PrintFileResult(res); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 JSON messages (begin, match, end), got %d lines: %v", len(lines), lines)
	}

	// Verify "begin"
	var beginMsg jsonMessage
	if err := json.Unmarshal([]byte(lines[0]), &beginMsg); err != nil {
		t.Fatalf("failed to decode begin: %v", err)
	}
	if beginMsg.Type != "begin" {
		t.Errorf("expected first message type 'begin', got %q", beginMsg.Type)
	}

	// Verify "match"
	var matchMsg jsonMessage
	if err := json.Unmarshal([]byte(lines[1]), &matchMsg); err != nil {
		t.Fatalf("failed to decode match: %v", err)
	}
	if matchMsg.Type != "match" {
		t.Errorf("expected second message type 'match', got %q", matchMsg.Type)
	}

	// Verify "end"
	var endMsg jsonMessage
	if err := json.Unmarshal([]byte(lines[2]), &endMsg); err != nil {
		t.Fatalf("failed to decode end: %v", err)
	}
	if endMsg.Type != "end" {
		t.Errorf("expected third message type 'end', got %q", endMsg.Type)
	}
}
