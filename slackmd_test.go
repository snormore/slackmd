package slackmd

import (
	"testing"
)

func TestConvert(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic formatting
		{
			name:     "bold",
			input:    "**bold**",
			expected: "*bold*\n",
		},
		{
			name:     "italic with asterisks",
			input:    "*italic*",
			expected: "_italic_\n",
		},
		{
			name:     "italic with underscores",
			input:    "_italic_",
			expected: "_italic_\n",
		},
		{
			name:     "strikethrough",
			input:    "~~strike~~",
			expected: "~strike~\n",
		},
		{
			name:     "inline code",
			input:    "`code`",
			expected: "`code`\n",
		},
		{
			name:     "bold italic",
			input:    "***bold italic***",
			expected: "_*bold italic*_\n",
		},

		// Headings
		{
			name:     "h1",
			input:    "# Heading",
			expected: "*Heading*\n",
		},
		{
			name:     "h2",
			input:    "## Heading 2",
			expected: "*Heading 2*\n",
		},

		// Links
		{
			name:     "link",
			input:    "[text](https://example.com)",
			expected: "<https://example.com|text>\n",
		},
		{
			name:     "image",
			input:    "![alt text](https://example.com/img.png)",
			expected: "<https://example.com/img.png|alt text>\n",
		},
		{
			name:     "autolink",
			input:    "<https://example.com>",
			expected: "<https://example.com>\n",
		},

		// Lists
		{
			name:     "unordered list",
			input:    "- one\n- two\n- three",
			expected: "• one\n• two\n• three\n",
		},
		{
			name:     "ordered list",
			input:    "1. one\n2. two\n3. three",
			expected: "1. one\n2. two\n3. three\n",
		},
		{
			name:  "nested list",
			input: "- a\n  - b\n    - c",
			expected: "• a\n" +
				"  • b\n" +
				"    • c\n",
		},

		// Task list
		{
			name:     "task list unchecked",
			input:    "- [ ] todo",
			expected: "• ☐ todo\n",
		},
		{
			name:     "task list checked",
			input:    "- [x] done",
			expected: "• ☑ done\n",
		},

		// Code blocks
		{
			name:     "fenced code block",
			input:    "```go\nfmt.Println(\"hello\")\n```",
			expected: "```\nfmt.Println(\"hello\")\n```\n",
		},
		{
			name:     "indented code block",
			input:    "    code line",
			expected: "```\ncode line\n```\n",
		},

		// Thematic break
		{
			name:     "horizontal rule",
			input:    "---",
			expected: "———\n",
		},

		// Blockquote
		{
			name:     "blockquote",
			input:    "> quoted text",
			expected: "&gt; quoted text\n",
		},
		{
			name:     "nested blockquote flattened",
			input:    "> > nested",
			expected: "&gt; nested\n",
		},

		// Escaping
		{
			name:     "escape angle brackets and ampersand",
			input:    "a < b & c > d",
			expected: "a &lt; b &amp; c &gt; d\n",
		},
		{
			name:     "no escape in code span",
			input:    "`a < b & c > d`",
			expected: "`a < b & c > d`\n",
		},

		// Edge cases
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "unicode and emoji",
			input:    "Hello 🌍 世界",
			expected: "Hello 🌍 世界\n",
		},

		// Tables
		{
			name:     "simple table",
			input:    "| A | B |\n|---|---|\n| 1 | 2 |",
			expected: "```\n+---+---+\n| A | B |\n+---+---+\n| 1 | 2 |\n+---+---+\n```\n",
		},

		// Multiple blocks
		{
			name:     "paragraph then list",
			input:    "Hello\n\n- one\n- two",
			expected: "Hello\n\n• one\n• two\n",
		},

		// Link with formatting
		{
			name:     "link with bold text",
			input:    "[**bold link**](https://example.com)",
			expected: "<https://example.com|*bold link*>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Convert(tt.input)
			if got != tt.expected {
				t.Errorf("Convert(%q)\n  got:  %q\n  want: %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestConvertWithOptions(t *testing.T) {
	got := ConvertWithOptions("- item", WithBulletChar('-'))
	if got != "- item\n" {
		t.Errorf("WithBulletChar: got %q, want %q", got, "- item\n")
	}
}

func BenchmarkConvert(b *testing.B) {
	input := "# Title\n\nSome **bold** and *italic* text.\n\n- item 1\n- item 2\n\n```go\nfmt.Println()\n```\n\n[link](https://example.com)\n"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Convert(input)
	}
}
