package slackmd

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Block represents a Slack block (rich_text, header, divider, or image).
type Block struct {
	Type     string    `json:"type"`
	Elements []Element `json:"elements,omitempty"`
	Text     *TextObj  `json:"text,omitempty"`
	ImageURL string    `json:"image_url,omitempty"`
	AltText  string    `json:"alt_text,omitempty"`
	Title    *TextObj  `json:"title,omitempty"`
}

// TextObj represents a plain_text object used by header blocks.
type TextObj struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Element represents an element within a rich_text block (section, list, preformatted, quote).
// For sections/preformatted/quotes, Elements contains Inline values.
// For lists, Elements contains Element values (rich_text_section items).
type Element struct {
	Type     string `json:"type"`
	Elements any    `json:"elements,omitempty"`
	Style    string `json:"style,omitempty"`
	Indent   int    `json:"indent,omitempty"`
	Border   int    `json:"border,omitempty"`
}

// Inline represents an inline element (text, link, emoji).
type Inline struct {
	Type  string       `json:"type"`
	Text  string       `json:"text,omitempty"`
	Name  string       `json:"name,omitempty"`
	URL   string       `json:"url,omitempty"`
	Style *InlineStyle `json:"style,omitempty"`
}

// InlineStyle represents styling for an inline element.
type InlineStyle struct {
	Bold   bool `json:"bold,omitempty"`
	Italic bool `json:"italic,omitempty"`
	Strike bool `json:"strike,omitempty"`
	Code   bool `json:"code,omitempty"`
}

// ConvertBlocks converts Markdown to Slack rich_text blocks.
func ConvertBlocks(markdown string) []Block {
	return defaultBlockConverter.ConvertBlocks(markdown)
}

// BlockConverter converts Markdown to Slack rich_text blocks.
type BlockConverter struct{}

var defaultBlockConverter = &BlockConverter{}

// ConvertBlocks converts Markdown to Slack rich_text blocks.
func (bc *BlockConverter) ConvertBlocks(markdown string) []Block {
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	source := []byte(markdown)
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var blocks []Block
	var accum []Element

	spacer := Element{Type: "rich_text_section", Elements: []Inline{{Type: "text", Text: "\n"}}}

	flush := func() {
		if len(accum) > 0 {
			blocks = append(blocks, Block{Type: "rich_text", Elements: accum})
			accum = nil
		}
	}

	appendRichText := func(elements []Element) {
		if len(elements) == 0 {
			return
		}
		if len(accum) > 0 {
			accum = append(accum, spacer)
		}
		accum = append(accum, elements...)
	}

	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case ast.KindHeading:
			flush()
			text := renderInlineChildrenPlain(source, child)
			if len(text) > 150 {
				text = text[:150]
			}
			blocks = append(blocks, Block{
				Type: "header",
				Text: &TextObj{Type: "plain_text", Text: text},
			})
		case ast.KindThematicBreak:
			flush()
			blocks = append(blocks, Block{Type: "divider"})
		case ast.KindParagraph:
			if img := soleImage(child); img != nil {
				flush()
				url := string(img.Destination)
				alt := renderInlineChildrenPlain(source, img)
				if alt == "" {
					alt = "image"
				}
				b := Block{Type: "image", ImageURL: url, AltText: alt}
				title := renderInlineChildrenPlain(source, img)
				if title != "" {
					b.Title = &TextObj{Type: "plain_text", Text: title}
				}
				blocks = append(blocks, b)
			} else {
				appendRichText(renderBlockNode(source, child))
			}
		default:
			appendRichText(renderBlockNode(source, child))
		}
	}

	flush()
	return blocks
}

// soleImage returns the *ast.Image if the paragraph contains only a single
// image (with no other content), otherwise nil.
func soleImage(para ast.Node) *ast.Image {
	child := para.FirstChild()
	if child == nil || child.NextSibling() != nil {
		return nil
	}
	img, ok := child.(*ast.Image)
	if !ok {
		return nil
	}
	return img
}

func renderBlockNode(source []byte, node ast.Node) []Element {
	switch node.Kind() {
	case ast.KindParagraph:
		return []Element{renderSection(source, node)}
	case ast.KindCodeBlock, ast.KindFencedCodeBlock:
		return []Element{renderPreformatted(source, node)}
	case ast.KindBlockquote:
		return []Element{renderQuote(source, node)}
	case ast.KindList:
		return renderList(source, node)
	case ast.KindHTMLBlock:
		return []Element{renderSection(source, node)}
	}
	// GFM table
	if node.Kind() == east.KindTable {
		return []Element{renderTableBlock(source, node)}
	}
	return nil
}

func renderSection(source []byte, node ast.Node) Element {
	inlines := collectInlines(source, node, InlineStyle{})
	return Element{Type: "rich_text_section", Elements: inlines}
}

func renderPreformatted(source []byte, node ast.Node) Element {
	var buf bytes.Buffer
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		buf.Write(line.Value(source))
	}
	text := strings.TrimRight(buf.String(), "\n")
	return Element{
		Type:     "rich_text_preformatted",
		Elements: []Inline{{Type: "text", Text: text}},
	}
}

func renderQuote(source []byte, node ast.Node) Element {
	var inlines []Inline
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		childInlines := collectInlines(source, child, InlineStyle{})
		if len(inlines) > 0 && len(childInlines) > 0 {
			inlines = append(inlines, Inline{Type: "text", Text: "\n"})
		}
		inlines = append(inlines, childInlines...)
	}
	return Element{Type: "rich_text_quote", Elements: inlines}
}

func renderList(source []byte, node ast.Node) []Element {
	n := node.(*ast.List)
	style := "bullet"
	if n.IsOrdered() {
		style = "ordered"
	}
	return collectListItems(source, node, style, 0)
}

// listEntry holds the data for a single list item before grouping.
type listEntry struct {
	style   string
	indent  int
	section Element // rich_text_section for this item
}

func collectListItems(source []byte, node ast.Node, style string, indent int) []Element {
	var entries []listEntry
	collectListEntries(source, node, style, indent, &entries)
	return groupListEntries(entries)
}

func collectListEntries(source []byte, node ast.Node, style string, indent int, entries *[]listEntry) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() != ast.KindListItem {
			continue
		}
		// Collect inline content from the list item (non-list children)
		var inlines []Inline
		for itemChild := child.FirstChild(); itemChild != nil; itemChild = itemChild.NextSibling() {
			if itemChild.Kind() == ast.KindList {
				continue
			}
			childInlines := collectInlines(source, itemChild, InlineStyle{})
			// Check for task checkbox
			if itemChild.Kind() == ast.KindTextBlock || itemChild.Kind() == ast.KindParagraph {
				if fc := itemChild.FirstChild(); fc != nil && fc.Kind() == east.KindTaskCheckBox {
					cb := fc.(*east.TaskCheckBox)
					prefix := "☐ "
					if cb.IsChecked {
						prefix = "☑ "
					}
					childInlines = append([]Inline{{Type: "text", Text: prefix}}, childInlines...)
				}
			}
			inlines = append(inlines, childInlines...)
		}

		*entries = append(*entries, listEntry{
			style:  style,
			indent: indent,
			section: Element{
				Type:     "rich_text_section",
				Elements: inlines,
			},
		})

		// Process nested lists
		for itemChild := child.FirstChild(); itemChild != nil; itemChild = itemChild.NextSibling() {
			if itemChild.Kind() == ast.KindList {
				nestedList := itemChild.(*ast.List)
				nestedStyle := "bullet"
				if nestedList.IsOrdered() {
					nestedStyle = "ordered"
				}
				collectListEntries(source, itemChild, nestedStyle, indent+1, entries)
			}
		}
	}
}

// groupListEntries groups consecutive entries with the same style+indent into
// single rich_text_list elements.
func groupListEntries(entries []listEntry) []Element {
	var elements []Element
	i := 0
	for i < len(entries) {
		cur := entries[i]
		elem := Element{
			Type:  "rich_text_list",
			Style: cur.style,
		}
		if cur.indent > 0 {
			elem.Indent = cur.indent
		}
		// Group consecutive entries with same style and indent
		var sections []Element
		for i < len(entries) && entries[i].style == cur.style && entries[i].indent == cur.indent {
			sections = append(sections, entries[i].section)
			i++
		}
		elem.Elements = sections
		elements = append(elements, elem)
	}
	return elements
}

func collectInlines(source []byte, node ast.Node, baseStyle InlineStyle) []Inline {
	var inlines []Inline
	collectInlinesWalk(source, node, baseStyle, &inlines)
	return mergeInlines(inlines)
}

// mergeInlines merges adjacent text inlines with the same style.
func mergeInlines(inlines []Inline) []Inline {
	if len(inlines) <= 1 {
		return inlines
	}
	merged := []Inline{inlines[0]}
	for _, in := range inlines[1:] {
		last := &merged[len(merged)-1]
		if last.Type == "text" && in.Type == "text" && stylesEqual(last.Style, in.Style) {
			last.Text += in.Text
		} else {
			merged = append(merged, in)
		}
	}
	// Trim trailing whitespace from last text inline
	if len(merged) > 0 {
		last := &merged[len(merged)-1]
		if last.Type == "text" {
			last.Text = strings.TrimRight(last.Text, "\n")
		}
	}
	return merged
}

func stylesEqual(a, b *InlineStyle) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func collectInlinesWalk(source []byte, node ast.Node, style InlineStyle, inlines *[]Inline) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case ast.KindText:
			n := child.(*ast.Text)
			seg := n.Segment
			t := string(seg.Value(source))
			addTextInline(inlines, t, style)
			if n.HardLineBreak() || n.SoftLineBreak() {
				addTextInline(inlines, "\n", style)
			}
		case ast.KindEmphasis:
			n := child.(*ast.Emphasis)
			childStyle := style
			if n.Level == 2 {
				childStyle.Bold = true
			} else {
				childStyle.Italic = true
			}
			collectInlinesWalk(source, child, childStyle, inlines)
		case ast.KindCodeSpan:
			var buf bytes.Buffer
			for c := child.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					buf.Write(t.Segment.Value(source))
				}
			}
			codeStyle := style
			codeStyle.Code = true
			addTextInline(inlines, buf.String(), codeStyle)
		case ast.KindLink:
			n := child.(*ast.Link)
			linkText := renderInlineChildrenPlain(source, child)
			url := string(n.Destination)
			inline := Inline{Type: "link", URL: url}
			if linkText != "" && linkText != url {
				inline.Text = linkText
			}
			if hasStyle(style) {
				inline.Style = stylePtr(style)
			}
			*inlines = append(*inlines, inline)
		case ast.KindImage:
			n := child.(*ast.Image)
			url := string(n.Destination)
			alt := renderInlineChildrenPlain(source, child)
			if alt != "" {
				addTextInline(inlines, alt+" ", style)
			}
			*inlines = append(*inlines, Inline{Type: "link", URL: url})
		case ast.KindAutoLink:
			n := child.(*ast.AutoLink)
			url := string(n.URL(source))
			*inlines = append(*inlines, Inline{Type: "link", URL: url})
		case ast.KindRawHTML:
			n := child.(*ast.RawHTML)
			for i := 0; i < n.Segments.Len(); i++ {
				seg := n.Segments.At(i)
				addTextInline(inlines, string(seg.Value(source)), style)
			}
		case east.KindStrikethrough:
			childStyle := style
			childStyle.Strike = true
			collectInlinesWalk(source, child, childStyle, inlines)
		case east.KindTaskCheckBox:
			// Handled at list item level
			continue
		default:
			// Recurse for unknown nodes
			collectInlinesWalk(source, child, style, inlines)
		}
	}
}

var emojiPattern = regexp.MustCompile(`:([a-zA-Z0-9_+-]+):`)

func addTextInline(inlines *[]Inline, text string, style InlineStyle) {
	if text == "" {
		return
	}
	var sp *InlineStyle
	if hasStyle(style) {
		sp = stylePtr(style)
	}
	matches := emojiPattern.FindAllStringIndex(text, -1)
	if matches == nil {
		*inlines = append(*inlines, Inline{Type: "text", Text: text, Style: sp})
		return
	}
	pos := 0
	for _, m := range matches {
		if m[0] > pos {
			*inlines = append(*inlines, Inline{Type: "text", Text: text[pos:m[0]], Style: sp})
		}
		name := text[m[0]+1 : m[1]-1]
		*inlines = append(*inlines, Inline{Type: "emoji", Name: name, Style: sp})
		pos = m[1]
	}
	if pos < len(text) {
		*inlines = append(*inlines, Inline{Type: "text", Text: text[pos:], Style: sp})
	}
}

func hasStyle(s InlineStyle) bool {
	return s.Bold || s.Italic || s.Strike || s.Code
}

func stylePtr(s InlineStyle) *InlineStyle {
	return &s
}

func renderInlineChildrenPlain(source []byte, node ast.Node) string {
	var buf bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		renderPlainText(&buf, source, child)
	}
	return buf.String()
}

func renderPlainText(buf *bytes.Buffer, source []byte, node ast.Node) {
	if node.Kind() == ast.KindText {
		n := node.(*ast.Text)
		buf.Write(n.Segment.Value(source))
		return
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		renderPlainText(buf, source, child)
	}
}

// renderTableBlock renders a table as a preformatted block.
func renderTableBlock(source []byte, node ast.Node) Element {
	table := node.(*east.Table)
	rows := collectTableRowsRT(source, table)
	text := formatTable(rows)
	return Element{Type: "rich_text_preformatted", Elements: []Inline{{Type: "text", Text: text}}}
}

func collectTableRowsRT(source []byte, table *east.Table) [][]string {
	var rows [][]string
	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == east.KindTableHeader || child.Kind() == east.KindTableRow {
			var cells []string
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				cells = append(cells, strings.TrimSpace(renderInlineChildrenPlain(source, cell)))
			}
			if cells != nil {
				rows = append(rows, cells)
			}
		}
	}
	return rows
}
