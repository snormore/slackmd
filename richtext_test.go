package slackmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func inlines(t *testing.T, elem Element) []Inline {
	t.Helper()
	v, ok := elem.Elements.([]Inline)
	if !ok {
		t.Fatalf("expected []Inline elements, got %T", elem.Elements)
	}
	return v
}

func sections(t *testing.T, elem Element) []Element {
	t.Helper()
	v, ok := elem.Elements.([]Element)
	if !ok {
		t.Fatalf("expected []Element elements, got %T", elem.Elements)
	}
	return v
}

func TestConvertBlocks_BasicText(t *testing.T) {
	blocks := ConvertBlocks("Hello world")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Type != "rich_text" {
		t.Fatalf("expected rich_text block, got %s", blocks[0].Type)
	}
	if len(blocks[0].Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(blocks[0].Elements))
	}
	elem := blocks[0].Elements[0]
	if elem.Type != "rich_text_section" {
		t.Fatalf("expected rich_text_section, got %s", elem.Type)
	}
	il := inlines(t, elem)
	if len(il) != 1 {
		t.Fatalf("expected 1 inline, got %d", len(il))
	}
	if il[0].Text != "Hello world" {
		t.Fatalf("expected 'Hello world', got %q", il[0].Text)
	}
}

func TestConvertBlocks_Bold(t *testing.T) {
	blocks := ConvertBlocks("**bold text**")
	il := inlines(t, blocks[0].Elements[0])
	if len(il) != 1 {
		t.Fatalf("expected 1 inline, got %d", len(il))
	}
	if il[0].Style == nil || !il[0].Style.Bold {
		t.Fatal("expected bold style")
	}
	if il[0].Text != "bold text" {
		t.Fatalf("expected 'bold text', got %q", il[0].Text)
	}
}

func TestConvertBlocks_Italic(t *testing.T) {
	blocks := ConvertBlocks("*italic text*")
	il := inlines(t, blocks[0].Elements[0])
	if il[0].Style == nil || !il[0].Style.Italic {
		t.Fatal("expected italic style")
	}
}

func TestConvertBlocks_Strikethrough(t *testing.T) {
	blocks := ConvertBlocks("~~struck~~")
	il := inlines(t, blocks[0].Elements[0])
	if il[0].Style == nil || !il[0].Style.Strike {
		t.Fatal("expected strike style")
	}
}

func TestConvertBlocks_InlineCode(t *testing.T) {
	blocks := ConvertBlocks("`code`")
	il := inlines(t, blocks[0].Elements[0])
	if il[0].Style == nil || !il[0].Style.Code {
		t.Fatal("expected code style")
	}
	if il[0].Text != "code" {
		t.Fatalf("expected 'code', got %q", il[0].Text)
	}
}

func TestConvertBlocks_CodeBlock(t *testing.T) {
	blocks := ConvertBlocks("```\nfoo\nbar\n```")
	elem := blocks[0].Elements[0]
	if elem.Type != "rich_text_preformatted" {
		t.Fatalf("expected rich_text_preformatted, got %s", elem.Type)
	}
	il := inlines(t, elem)
	if il[0].Text != "foo\nbar" {
		t.Fatalf("expected 'foo\\nbar', got %q", il[0].Text)
	}
}

func TestConvertBlocks_Heading(t *testing.T) {
	blocks := ConvertBlocks("# Heading")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Type != "header" {
		t.Fatalf("expected header block, got %s", blocks[0].Type)
	}
	if blocks[0].Text == nil || blocks[0].Text.Text != "Heading" {
		t.Fatalf("expected text 'Heading', got %+v", blocks[0].Text)
	}
	if blocks[0].Text.Type != "plain_text" {
		t.Fatalf("expected plain_text type, got %s", blocks[0].Text.Type)
	}
}

func TestConvertBlocks_Blockquote(t *testing.T) {
	blocks := ConvertBlocks("> quoted text")
	elem := blocks[0].Elements[0]
	if elem.Type != "rich_text_quote" {
		t.Fatalf("expected rich_text_quote, got %s", elem.Type)
	}
}

func TestConvertBlocks_BulletList(t *testing.T) {
	blocks := ConvertBlocks("- a\n- b\n- c")
	elems := blocks[0].Elements
	// Should be grouped into 1 rich_text_list with 3 sections
	if len(elems) != 1 {
		t.Fatalf("expected 1 list element, got %d", len(elems))
	}
	if elems[0].Type != "rich_text_list" {
		t.Fatalf("expected rich_text_list, got %s", elems[0].Type)
	}
	if elems[0].Style != "bullet" {
		t.Fatalf("expected bullet style, got %s", elems[0].Style)
	}
	secs := sections(t, elems[0])
	if len(secs) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(secs))
	}
}

func TestConvertBlocks_OrderedList(t *testing.T) {
	blocks := ConvertBlocks("1. first\n2. second")
	elems := blocks[0].Elements
	if len(elems) != 1 {
		t.Fatalf("expected 1 list element, got %d", len(elems))
	}
	if elems[0].Style != "ordered" {
		t.Fatalf("expected ordered style, got %s", elems[0].Style)
	}
	secs := sections(t, elems[0])
	if len(secs) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(secs))
	}
}

func TestConvertBlocks_OrderedListRepeatedNumbers(t *testing.T) {
	blocks := ConvertBlocks("1. first\n1. second\n1. third")
	elems := blocks[0].Elements
	if len(elems) != 1 {
		t.Fatalf("expected 1 list element, got %d", len(elems))
	}
	if elems[0].Style != "ordered" {
		t.Fatalf("expected ordered style, got %s", elems[0].Style)
	}
	secs := sections(t, elems[0])
	if len(secs) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(secs))
	}
}

func TestConvertBlocks_NestedList(t *testing.T) {
	blocks := ConvertBlocks("- a\n  - b\n    - c")
	elems := blocks[0].Elements
	// Should be 3 separate list elements with increasing indent
	if len(elems) != 3 {
		t.Fatalf("expected 3 list elements, got %d", len(elems))
	}
	if elems[0].Indent != 0 {
		t.Fatalf("expected indent 0, got %d", elems[0].Indent)
	}
	if elems[1].Indent != 1 {
		t.Fatalf("expected indent 1, got %d", elems[1].Indent)
	}
	if elems[2].Indent != 2 {
		t.Fatalf("expected indent 2, got %d", elems[2].Indent)
	}
}

func TestConvertBlocks_Link(t *testing.T) {
	blocks := ConvertBlocks("[click](https://example.com)")
	il := inlines(t, blocks[0].Elements[0])
	if il[0].Type != "link" {
		t.Fatalf("expected link type, got %s", il[0].Type)
	}
	if il[0].URL != "https://example.com" {
		t.Fatalf("expected URL, got %q", il[0].URL)
	}
	if il[0].Text != "click" {
		t.Fatalf("expected 'click', got %q", il[0].Text)
	}
}

func TestConvertBlocks_JSONOutput(t *testing.T) {
	blocks := ConvertBlocks("**hello**")
	data, err := json.Marshal(blocks)
	if err != nil {
		t.Fatal(err)
	}
	var parsed []map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed[0]["type"] != "rich_text" {
		t.Fatal("expected rich_text type in JSON")
	}
}

func TestConvertBlocks_Table(t *testing.T) {
	md := "| A | B |\n|---|---|\n| 1 | 2 |"
	blocks := ConvertBlocks(md)
	elem := blocks[0].Elements[0]
	if elem.Type != "rich_text_preformatted" {
		t.Fatalf("expected rich_text_preformatted for table, got %s", elem.Type)
	}
}

func TestConvertBlocks_TaskCheckbox(t *testing.T) {
	blocks := ConvertBlocks("- [x] done\n- [ ] todo")
	elems := blocks[0].Elements
	if len(elems) != 1 {
		t.Fatalf("expected 1 list element, got %d", len(elems))
	}
}

func TestConvertBlocks_Empty(t *testing.T) {
	blocks := ConvertBlocks("")
	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestConvertBlocks_ThematicBreak(t *testing.T) {
	blocks := ConvertBlocks("above\n\n---\n\nbelow")
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	if blocks[0].Type != "rich_text" {
		t.Fatalf("expected rich_text, got %s", blocks[0].Type)
	}
	if blocks[1].Type != "divider" {
		t.Fatalf("expected divider, got %s", blocks[1].Type)
	}
	if blocks[2].Type != "rich_text" {
		t.Fatalf("expected rich_text, got %s", blocks[2].Type)
	}
}

func TestConvertBlocks_Image(t *testing.T) {
	blocks := ConvertBlocks("![alt text](https://example.com/img.png)")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Type != "image" {
		t.Fatalf("expected image block, got %s", blocks[0].Type)
	}
	if blocks[0].ImageURL != "https://example.com/img.png" {
		t.Fatalf("expected image URL, got %q", blocks[0].ImageURL)
	}
	if blocks[0].AltText != "alt text" {
		t.Fatalf("expected alt text 'alt text', got %q", blocks[0].AltText)
	}
}

func TestConvertBlocks_HeadingTruncation(t *testing.T) {
	long := strings.Repeat("A", 200)
	blocks := ConvertBlocks("# " + long)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if len(blocks[0].Text.Text) != 150 {
		t.Fatalf("expected 150 chars, got %d", len(blocks[0].Text.Text))
	}
}

func TestConvertBlocks_HeadingBetweenContent(t *testing.T) {
	blocks := ConvertBlocks("above\n\n## Title\n\nbelow")
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	if blocks[0].Type != "rich_text" {
		t.Fatalf("expected rich_text, got %s", blocks[0].Type)
	}
	if blocks[1].Type != "header" {
		t.Fatalf("expected header, got %s", blocks[1].Type)
	}
	if blocks[2].Type != "rich_text" {
		t.Fatalf("expected rich_text, got %s", blocks[2].Type)
	}
}

func TestConvertBlocks_ImageInlineWithText(t *testing.T) {
	// Image mixed with text in a paragraph should stay as rich_text, not become an image block
	blocks := ConvertBlocks("see ![pic](https://example.com/img.png) here")
	if blocks[0].Type != "rich_text" {
		t.Fatalf("expected rich_text for inline image, got %s", blocks[0].Type)
	}
}

func TestConvertBlocks_AutoLink(t *testing.T) {
	blocks := ConvertBlocks("visit https://example.com for info")
	il := inlines(t, blocks[0].Elements[0])
	found := false
	for _, in := range il {
		if in.Type == "link" && in.URL == "https://example.com" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected autolink inline")
	}
}

func TestConvertBlocks_MultiParagraphQuote(t *testing.T) {
	blocks := ConvertBlocks("> line one\n>\n> line two")
	elem := blocks[0].Elements[0]
	if elem.Type != "rich_text_quote" {
		t.Fatalf("expected rich_text_quote, got %s", elem.Type)
	}
	il := inlines(t, elem)
	// Should have a newline separator between paragraphs
	hasNewline := false
	for _, in := range il {
		if in.Text == "\n" {
			hasNewline = true
		}
	}
	if !hasNewline {
		t.Fatal("expected newline separator between quote paragraphs")
	}
}

func TestConvertBlocks_BoldItalicCombined(t *testing.T) {
	blocks := ConvertBlocks("***bold and italic***")
	il := inlines(t, blocks[0].Elements[0])
	if len(il) != 1 {
		t.Fatalf("expected 1 inline, got %d", len(il))
	}
	if il[0].Style == nil || !il[0].Style.Bold || !il[0].Style.Italic {
		t.Fatal("expected bold+italic style")
	}
}

func TestConvertBlocks_LinkWithFormatting(t *testing.T) {
	blocks := ConvertBlocks("[**bold link**](https://example.com)")
	il := inlines(t, blocks[0].Elements[0])
	if il[0].Type != "link" {
		t.Fatalf("expected link, got %s", il[0].Type)
	}
	if il[0].URL != "https://example.com" {
		t.Fatalf("expected URL, got %q", il[0].URL)
	}
}

func TestConvertBlocks_MixedList(t *testing.T) {
	// Bullet list with nested ordered list
	md := "- a\n  1. one\n  2. two\n- b"
	blocks := ConvertBlocks(md)
	elems := blocks[0].Elements
	// Should have bullet items and ordered sub-items
	foundBullet := false
	foundOrdered := false
	for _, e := range elems {
		if e.Type == "rich_text_list" && e.Style == "bullet" {
			foundBullet = true
		}
		if e.Type == "rich_text_list" && e.Style == "ordered" {
			foundOrdered = true
		}
	}
	if !foundBullet {
		t.Fatal("expected bullet list")
	}
	if !foundOrdered {
		t.Fatal("expected ordered nested list")
	}
}

func TestConvertBlocks_IndentedCodeBlock(t *testing.T) {
	// Indented code block (4 spaces)
	blocks := ConvertBlocks("    code line 1\n    code line 2")
	elem := blocks[0].Elements[0]
	if elem.Type != "rich_text_preformatted" {
		t.Fatalf("expected rich_text_preformatted, got %s", elem.Type)
	}
}

func TestConvertBlocks_HTMLBlock(t *testing.T) {
	blocks := ConvertBlocks("<div>hello</div>")
	if len(blocks) == 0 {
		t.Fatal("expected blocks for HTML content")
	}
}

func TestConvertBlocks_RawHTMLInline(t *testing.T) {
	blocks := ConvertBlocks("text <br> more")
	il := inlines(t, blocks[0].Elements[0])
	if len(il) == 0 {
		t.Fatal("expected inlines for raw HTML")
	}
}

func TestConvertBlocks_StyledMerging(t *testing.T) {
	// Bold text next to non-bold should not merge
	blocks := ConvertBlocks("**bold** normal")
	il := inlines(t, blocks[0].Elements[0])
	if len(il) != 2 {
		t.Fatalf("expected 2 inlines, got %d", len(il))
	}
	if il[0].Style == nil || !il[0].Style.Bold {
		t.Fatal("first inline should be bold")
	}
	if il[1].Style != nil {
		t.Fatal("second inline should have no style")
	}
}
