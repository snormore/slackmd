package slackmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// SlackRenderer renders goldmark AST nodes to Slack mrkdwn format.
type SlackRenderer struct {
	bulletChar rune
	listIndent int

	// state
	listDepth        int
	listCounter      []int
	orderedListStack []bool
	inCodeBlock      bool
	inLink           bool
	linkBuf          *bytes.Buffer
	inBlockquote     bool
	blockquoteDepth  int
}

func newSlackRenderer(opts *options) *SlackRenderer {
	return &SlackRenderer{
		bulletChar: opts.bulletChar,
		listIndent: opts.listIndent,
	}
}

// RegisterFuncs implements renderer.NodeRenderer.
func (r *SlackRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// Block nodes
	reg.Register(ast.KindDocument, r.renderDocument)
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindParagraph, r.renderParagraph)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
	reg.Register(ast.KindList, r.renderList)
	reg.Register(ast.KindListItem, r.renderListItem)
	reg.Register(ast.KindThematicBreak, r.renderThematicBreak)
	reg.Register(ast.KindTextBlock, r.renderTextBlock)
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)

	// Inline nodes
	reg.Register(ast.KindText, r.renderText)
	reg.Register(ast.KindEmphasis, r.renderEmphasis)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)

	// GFM nodes
	reg.Register(east.KindStrikethrough, r.renderStrikethrough)
	reg.Register(east.KindTable, r.renderTable)
	reg.Register(east.KindTableHeader, r.renderTableHeader)
	reg.Register(east.KindTableRow, r.renderTableRow)
	reg.Register(east.KindTableCell, r.renderTableCell)
	reg.Register(east.KindTaskCheckBox, r.renderTaskCheckBox)
}

func (r *SlackRenderer) renderDocument(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.writeBlockSeparator(w, node)
		w.WriteString("*")
	} else {
		w.WriteString("*")
		w.WriteByte('\n')
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.writeBlockSeparator(w, node)
	} else {
		w.WriteByte('\n')
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.inCodeBlock = true
		r.writeBlockSeparator(w, node)
		w.WriteString("```\n")
		r.writeLines(w, source, node)
		w.WriteString("```\n")
		r.inCodeBlock = false
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.inCodeBlock = true
		r.writeBlockSeparator(w, node)
		w.WriteString("```\n")
		r.writeLines(w, source, node)
		w.WriteString("```\n")
		r.inCodeBlock = false
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.blockquoteDepth++
		if r.blockquoteDepth == 1 {
			r.inBlockquote = true
			r.writeBlockSeparator(w, node)

			// Render inner content to a buffer, then prefix each line with "> "
			var buf bytes.Buffer
			innerRenderer := *r
			innerRenderer.inBlockquote = false
			innerRenderer.blockquoteDepth = 0

			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				r.renderBlockquoteChild(&buf, source, child, &innerRenderer)
			}

			content := strings.TrimRight(buf.String(), "\n")
			lines := strings.Split(content, "\n")
			for i, line := range lines {
				w.WriteString("&gt; ")
				w.WriteString(strings.TrimRight(line, " \t"))
				if i < len(lines)-1 {
					w.WriteByte('\n')
				}
			}
			w.WriteByte('\n')

			return ast.WalkSkipChildren, nil
		}
	} else {
		r.blockquoteDepth--
		if r.blockquoteDepth == 0 {
			r.inBlockquote = false
		}
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderBlockquoteChild(w *bytes.Buffer, source []byte, node ast.Node, sr *SlackRenderer) {
	// Walk the subtree manually using a temporary goldmark-compatible writer
	bw := &bufWriterAdapter{buf: w}
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n.Kind() {
		case ast.KindParagraph:
			return sr.renderParagraph(bw, source, n, entering)
		case ast.KindHeading:
			return sr.renderHeading(bw, source, n, entering)
		case ast.KindText:
			return sr.renderText(bw, source, n, entering)
		case ast.KindEmphasis:
			return sr.renderEmphasis(bw, source, n, entering)
		case ast.KindCodeSpan:
			return sr.renderCodeSpan(bw, source, n, entering)
		case ast.KindLink:
			return sr.renderLink(bw, source, n, entering)
		case ast.KindImage:
			return sr.renderImage(bw, source, n, entering)
		case ast.KindAutoLink:
			return sr.renderAutoLink(bw, source, n, entering)
		case ast.KindTextBlock:
			return sr.renderTextBlock(bw, source, n, entering)
		case ast.KindCodeBlock:
			return sr.renderCodeBlock(bw, source, n, entering)
		case ast.KindFencedCodeBlock:
			return sr.renderFencedCodeBlock(bw, source, n, entering)
		case ast.KindList:
			return sr.renderList(bw, source, n, entering)
		case ast.KindListItem:
			return sr.renderListItem(bw, source, n, entering)
		case ast.KindThematicBreak:
			return sr.renderThematicBreak(bw, source, n, entering)
		case ast.KindBlockquote:
			// Flatten nested blockquotes
			return ast.WalkContinue, nil
		case ast.KindRawHTML:
			return sr.renderRawHTML(bw, source, n, entering)
		case ast.KindHTMLBlock:
			return sr.renderHTMLBlock(bw, source, n, entering)
		case ast.KindDocument:
			return ast.WalkContinue, nil
		}
		// GFM
		switch n.Kind() {
		case east.KindStrikethrough:
			return sr.renderStrikethrough(bw, source, n, entering)
		case east.KindTaskCheckBox:
			return sr.renderTaskCheckBox(bw, source, n, entering)
		case east.KindTable:
			return sr.renderTable(bw, source, n, entering)
		case east.KindTableHeader:
			return sr.renderTableHeader(bw, source, n, entering)
		case east.KindTableRow:
			return sr.renderTableRow(bw, source, n, entering)
		case east.KindTableCell:
			return sr.renderTableCell(bw, source, n, entering)
		}
		return ast.WalkContinue, nil
	})
}

func (r *SlackRenderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.List)
	if entering {
		if r.listDepth == 0 {
			r.writeBlockSeparator(w, node)
		}
		r.listDepth++
		r.orderedListStack = append(r.orderedListStack, n.IsOrdered())
		if n.IsOrdered() {
			r.listCounter = append(r.listCounter, n.Start)
		} else {
			r.listCounter = append(r.listCounter, 0)
		}
	} else {
		r.listDepth--
		r.orderedListStack = r.orderedListStack[:len(r.orderedListStack)-1]
		r.listCounter = r.listCounter[:len(r.listCounter)-1]
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderListItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		indent := strings.Repeat(" ", (r.listDepth-1)*r.listIndent)
		w.WriteString(indent)

		idx := len(r.orderedListStack) - 1
		if r.orderedListStack[idx] {
			fmt.Fprintf(w, "%d. ", r.listCounter[idx])
			r.listCounter[idx]++
		} else {
			w.WriteRune(r.bulletChar)
			w.WriteByte(' ')
		}
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderThematicBreak(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.writeBlockSeparator(w, node)
		w.WriteString("———\n")
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderTextBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		w.WriteByte('\n')
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.writeBlockSeparator(w, node)
		r.writeLines(w, source, node)
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Text)
	segment := n.Segment
	text := string(segment.Value(source))

	if r.inLink {
		r.linkBuf.WriteString(escapeText(text))
	} else if r.inCodeBlock {
		w.WriteString(text)
	} else {
		w.WriteString(escapeText(text))
	}

	if n.HardLineBreak() || n.SoftLineBreak() {
		if r.inLink {
			r.linkBuf.WriteByte('\n')
		} else {
			w.WriteByte('\n')
		}
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Emphasis)
	target := w
	if r.inLink {
		target = &bufWriterAdapter{buf: r.linkBuf}
	}

	if n.Level == 2 {
		target.WriteString("*")
	} else {
		target.WriteString("_")
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	target := w
	if r.inLink {
		target = &bufWriterAdapter{buf: r.linkBuf}
	}
	if entering {
		target.WriteString("`")
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			segment := child.(*ast.Text).Segment
			target.WriteString(string(segment.Value(source)))
		}
		target.WriteString("`")
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		r.inLink = true
		r.linkBuf = &bytes.Buffer{}
	} else {
		r.inLink = false
		text := r.linkBuf.String()
		url := string(n.Destination)
		if text == "" || text == url {
			fmt.Fprintf(w, "<%s>", url)
		} else {
			fmt.Fprintf(w, "<%s|%s>", url, text)
		}
		r.linkBuf = nil
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.Image)
		url := string(n.Destination)
		alt := r.renderInlineChildren(source, node)
		if alt == "" {
			fmt.Fprintf(w, "<%s>", url)
		} else {
			fmt.Fprintf(w, "<%s|%s>", url, escapeText(alt))
		}
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.AutoLink)
		url := string(n.URL(source))
		fmt.Fprintf(w, "<%s>", url)
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.RawHTML)
		for i := 0; i < n.Segments.Len(); i++ {
			seg := n.Segments.At(i)
			w.WriteString(escapeText(string(seg.Value(source))))
		}
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderStrikethrough(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	w.WriteString("~")
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderTaskCheckBox(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*east.TaskCheckBox)
		if n.IsChecked {
			w.WriteString("☑ ")
		} else {
			w.WriteString("☐ ")
		}
	}
	return ast.WalkContinue, nil
}

// --- Table rendering ---

func (r *SlackRenderer) renderTable(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.writeBlockSeparator(w, node)
		// Collect all rows and cells, then render as code block
		table := node.(*east.Table)
		rows := r.collectTableRows(source, table)
		if len(rows) == 0 {
			return ast.WalkSkipChildren, nil
		}

		w.WriteString("```\n")
		w.WriteString(formatTable(rows))
		w.WriteByte('\n')
		w.WriteString("```\n")
		return ast.WalkSkipChildren, nil
	}
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) collectTableRows(source []byte, table *east.Table) [][]string {
	var rows [][]string
	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		// TableHeader or TableRow
		if child.Kind() == east.KindTableHeader || child.Kind() == east.KindTableRow {
			row := r.collectCells(source, child)
			if row != nil {
				rows = append(rows, row)
			}
		}
	}
	return rows
}

func (r *SlackRenderer) collectCells(source []byte, rowNode ast.Node) []string {
	var cells []string
	for cell := rowNode.FirstChild(); cell != nil; cell = cell.NextSibling() {
		cells = append(cells, strings.TrimSpace(r.renderInlineChildren(source, cell)))
	}
	return cells
}

func (r *SlackRenderer) renderInlineChildren(source []byte, node ast.Node) string {
	var buf bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		r.renderInlineNode(&buf, source, child)
	}
	return buf.String()
}

func (r *SlackRenderer) renderInlineNode(buf *bytes.Buffer, source []byte, node ast.Node) {
	switch node.Kind() {
	case ast.KindText:
		n := node.(*ast.Text)
		buf.WriteString(string(n.Segment.Value(source)))
	case ast.KindEmphasis:
		n := node.(*ast.Emphasis)
		if n.Level == 2 {
			buf.WriteString("*")
		} else {
			buf.WriteString("_")
		}
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			r.renderInlineNode(buf, source, child)
		}
		if n.Level == 2 {
			buf.WriteString("*")
		} else {
			buf.WriteString("_")
		}
	case ast.KindCodeSpan:
		buf.WriteString("`")
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if t, ok := child.(*ast.Text); ok {
				buf.WriteString(string(t.Segment.Value(source)))
			}
		}
		buf.WriteString("`")
	case ast.KindLink:
		n := node.(*ast.Link)
		text := r.renderInlineChildren(source, node)
		url := string(n.Destination)
		if text == "" || text == url {
			fmt.Fprintf(buf, "<%s>", url)
		} else {
			fmt.Fprintf(buf, "<%s|%s>", url, text)
		}
	default:
		// Fallback: render children
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			r.renderInlineNode(buf, source, child)
		}
	}
}


func (r *SlackRenderer) renderTableHeader(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderTableRow(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *SlackRenderer) renderTableCell(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

// --- Helpers ---

func (r *SlackRenderer) writeLines(w util.BufWriter, source []byte, node ast.Node) {
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		w.WriteString(string(line.Value(source)))
	}
}

func (r *SlackRenderer) writeBlockSeparator(w util.BufWriter, node ast.Node) {
	if node.PreviousSibling() != nil {
		// Inside a list item, paragraphs shouldn't double-space
		if node.Parent() != nil && node.Parent().Kind() == ast.KindListItem && node.Kind() == ast.KindParagraph {
			return
		}
		w.WriteByte('\n')
	}
}

// bufWriterAdapter wraps a *bytes.Buffer to satisfy util.BufWriter.
type bufWriterAdapter struct {
	buf *bytes.Buffer
}

func (b *bufWriterAdapter) Write(p []byte) (int, error)       { return b.buf.Write(p) }
func (b *bufWriterAdapter) WriteByte(c byte) error             { return b.buf.WriteByte(c) }
func (b *bufWriterAdapter) WriteRune(r rune) (int, error)      { return b.buf.WriteRune(r) }
func (b *bufWriterAdapter) WriteString(s string) (int, error)  { return b.buf.WriteString(s) }
func (b *bufWriterAdapter) Flush() error                       { return nil }
func (b *bufWriterAdapter) Available() int                     { return 0 }
func (b *bufWriterAdapter) Buffered() int                      { return b.buf.Len() }
func (b *bufWriterAdapter) Bytes() []byte                      { return b.buf.Bytes() }
