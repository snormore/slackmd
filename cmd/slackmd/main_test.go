package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestChunk_ShortText(t *testing.T) {
	text := "hello world"
	chunks := chunk(text, 100)
	if len(chunks) != 1 || chunks[0] != text {
		t.Fatalf("expected single chunk %q, got %v", text, chunks)
	}
}

func TestChunk_ExactLimit(t *testing.T) {
	text := "abcde"
	chunks := chunk(text, 5)
	if len(chunks) != 1 || chunks[0] != text {
		t.Fatalf("expected single chunk, got %v", chunks)
	}
}

func TestChunk_SplitsOnParagraphBoundary(t *testing.T) {
	text := "aaa\n\nbbb\n\nccc"
	chunks := chunk(text, 7)
	expected := []string{"aaa", "bbb", "ccc"}
	if len(chunks) != len(expected) {
		t.Fatalf("expected %d chunks, got %d: %v", len(expected), len(chunks), chunks)
	}
	for i, c := range chunks {
		if c != expected[i] {
			t.Errorf("chunk %d: got %q, want %q", i, c, expected[i])
		}
	}
}

func TestChunk_SplitsOnNewline(t *testing.T) {
	text := "aaa\nbbb\nccc"
	chunks := chunk(text, 6)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %v", chunks)
	}
	for _, c := range chunks {
		if utf8.RuneCountInString(c) > 6 {
			t.Errorf("chunk exceeds max rune length: %q (%d)", c, utf8.RuneCountInString(c))
		}
	}
}

func TestChunk_HardSplitWhenNoNewlines(t *testing.T) {
	text := "abcdefghij"
	chunks := chunk(text, 4)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d: %v", len(chunks), chunks)
	}
	reassembled := strings.Join(chunks, "")
	if reassembled != text {
		t.Errorf("reassembled text doesn't match:\ngot:  %q\nwant: %q", reassembled, text)
	}
}

func TestChunk_PreservesAllContent(t *testing.T) {
	var parts []string
	for range 50 {
		parts = append(parts, strings.Repeat("x", 100))
	}
	text := strings.Join(parts, "\n\n")

	chunks := chunk(text, 500)
	for i, c := range chunks {
		if utf8.RuneCountInString(c) > 500 {
			t.Errorf("chunk %d exceeds max: %d runes", i, utf8.RuneCountInString(c))
		}
	}
}

func TestChunk_NoEmptyChunks(t *testing.T) {
	text := "aaa\n\n\n\n\n\nbbb"
	chunks := chunk(text, 5)
	for i, c := range chunks {
		if len(c) == 0 {
			t.Errorf("chunk %d is empty", i)
		}
	}
}

func TestChunk_Unicode(t *testing.T) {
	// Each emoji is multiple bytes but 1 rune (well, some are 1 codepoint).
	// Use simple multi-byte chars.
	text := "café\n\nnaïve\n\nrésumé"
	chunks := chunk(text, 8)
	for i, c := range chunks {
		runeCount := utf8.RuneCountInString(c)
		if runeCount > 8 {
			t.Errorf("chunk %d has %d runes (max 8): %q", i, runeCount, c)
		}
	}
	// Verify no content lost.
	reassembled := strings.Join(chunks, "\n\n")
	if reassembled != text {
		t.Errorf("content lost:\ngot:  %q\nwant: %q", reassembled, text)
	}
}

func TestChunk_UnicodeHardSplit(t *testing.T) {
	// 8 multi-byte runes, limit 4 runes — should split cleanly on rune boundary.
	text := "àèìòùáéí"
	chunks := chunk(text, 4)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunks)
	}
	for i, c := range chunks {
		if utf8.RuneCountInString(c) > 4 {
			t.Errorf("chunk %d exceeds 4 runes: %q", i, c)
		}
		if !utf8.ValidString(c) {
			t.Errorf("chunk %d is not valid UTF-8: %q", i, c)
		}
	}
}

func TestChunk_CodeBlockNotSplit(t *testing.T) {
	before := "some text here"
	code := "\n\n```\nline1\nline2\nline3\n```"
	text := before + code
	// Set limit so code block would be split if we didn't protect it.
	limit := len(before) + 10
	chunks := chunk(text, limit)

	// The code block should be entirely in one chunk, not split.
	for _, c := range chunks {
		fences := strings.Count(c, "```")
		if fences%2 != 0 {
			t.Errorf("chunk has unmatched code fence (count=%d): %q", fences, c)
		}
	}
}

func TestChunk_CodeBlockAtStart(t *testing.T) {
	// Code block at the very start that fits when we extend past the limit.
	text := "```\nshort\n```\n\nafter"
	chunks := chunk(text, 10)

	found := false
	for _, c := range chunks {
		if strings.Contains(c, "```") {
			fences := strings.Count(c, "```")
			if fences%2 != 0 {
				t.Errorf("chunk has unmatched code fence (count=%d): %q", fences, c)
			}
			found = true
		}
	}
	if !found {
		t.Error("no chunk contained code fences")
	}
}

func TestChunk_CodeBlockFitsWhenExtended(t *testing.T) {
	// Code block extends slightly past the limit but is included whole.
	text := "hello\n\n```\ncode\n```\n\nworld"
	// limit of 12 runes would cut inside the code block, but extending
	// to include the closing fence keeps it intact.
	chunks := chunk(text, 12)
	for _, c := range chunks {
		fences := strings.Count(c, "```")
		if fences%2 != 0 {
			t.Errorf("chunk has unmatched code fence (count=%d): %q", fences, c)
		}
	}
}

func TestChunk_LargeCodeBlockExceedsLimit(t *testing.T) {
	// A single code block larger than the limit — can't avoid splitting it.
	// Just verify we don't panic or produce empty chunks.
	code := "```\n" + strings.Repeat("x", 200) + "\n```"
	chunks := chunk(code, 50)
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	for i, c := range chunks {
		if len(c) == 0 {
			t.Errorf("chunk %d is empty", i)
		}
	}
}

func TestRuneOffset(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want int
	}{
		{"hello", 3, 3},
		{"café", 3, 3},   // c-a-f are 1 byte each
		{"café", 4, 5},   // é is 2 bytes
		{"hello", 10, 5}, // beyond length
		{"", 0, 0},
	}
	for _, tt := range tests {
		got := runeOffset(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("runeOffset(%q, %d) = %d, want %d", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestAdjustForCodeBlock(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		limit int
		want  int
	}{
		{
			name:  "no code blocks",
			text:  "hello world this is text",
			limit: 15,
			want:  15, // unchanged
		},
		{
			name:  "closed code block before limit",
			text:  "before ```code``` after more text",
			limit: 25,
			want:  25, // unchanged
		},
		{
			name:  "open block with closing fence after limit",
			text:  "before\n```\ncode here\nmore code\n```\nafter",
			limit: 25,
			want:  35, // extended to include closing fence + newline
		},
		{
			name:  "open block at start with closing fence",
			text:  "```\ncode\n```\nmore",
			limit: 8,
			want:  13, // extended to include closing fence + newline
		},
		{
			name:  "open block no closing fence, break before",
			text:  "before\n```\ncode here\nmore code",
			limit: 25,
			want:  7, // before the opening fence
		},
		{
			name:  "open block at start no closing fence",
			text:  "```\ncode\nmore",
			limit: 8,
			want:  8, // can't break before, unchanged
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adjustForCodeBlock(tt.text, tt.limit)
			if got != tt.want {
				t.Errorf("adjustForCodeBlock() = %d, want %d", got, tt.want)
			}
		})
	}
}
