package slackgo

import (
	"testing"

	"github.com/slack-go/slack"
	"github.com/snormore/slackmd"
)

func TestConvertBlocks_Paragraph(t *testing.T) {
	blocks := ConvertBlocks("Hello **world**")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	rt, ok := blocks[0].(*slack.RichTextBlock)
	if !ok {
		t.Fatalf("expected RichTextBlock, got %T", blocks[0])
	}
	if rt.Type != slack.MBTRichText {
		t.Errorf("expected type %s, got %s", slack.MBTRichText, rt.Type)
	}
	if len(rt.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(rt.Elements))
	}
	section, ok := rt.Elements[0].(*slack.RichTextSection)
	if !ok {
		t.Fatalf("expected RichTextSection, got %T", rt.Elements[0])
	}
	if len(section.Elements) != 2 {
		t.Fatalf("expected 2 inlines, got %d", len(section.Elements))
	}
	text0, ok := section.Elements[0].(*slack.RichTextSectionTextElement)
	if !ok {
		t.Fatalf("expected TextElement, got %T", section.Elements[0])
	}
	if text0.Text != "Hello " {
		t.Errorf("expected 'Hello ', got %q", text0.Text)
	}
	text1, ok := section.Elements[1].(*slack.RichTextSectionTextElement)
	if !ok {
		t.Fatalf("expected TextElement, got %T", section.Elements[1])
	}
	if text1.Text != "world" {
		t.Errorf("expected 'world', got %q", text1.Text)
	}
	if text1.Style == nil || !text1.Style.Bold {
		t.Errorf("expected bold style")
	}
}

func TestConvertBlocks_Header(t *testing.T) {
	blocks := ConvertBlocks("# Title")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	hdr, ok := blocks[0].(*slack.HeaderBlock)
	if !ok {
		t.Fatalf("expected HeaderBlock, got %T", blocks[0])
	}
	if hdr.Text.Text != "Title" {
		t.Errorf("expected 'Title', got %q", hdr.Text.Text)
	}
}

func TestConvertBlocks_Divider(t *testing.T) {
	blocks := ConvertBlocks("---")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	_, ok := blocks[0].(*slack.DividerBlock)
	if !ok {
		t.Fatalf("expected DividerBlock, got %T", blocks[0])
	}
}

func TestConvertBlocks_CodeBlock(t *testing.T) {
	blocks := ConvertBlocks("```\nfoo\n```")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	rt, ok := blocks[0].(*slack.RichTextBlock)
	if !ok {
		t.Fatalf("expected RichTextBlock, got %T", blocks[0])
	}
	pre, ok := rt.Elements[0].(*slack.RichTextPreformatted)
	if !ok {
		t.Fatalf("expected RichTextPreformatted, got %T", rt.Elements[0])
	}
	if len(pre.Elements) == 0 {
		t.Fatal("expected at least one element in preformatted block")
	}
}

func TestConvertBlocks_Link(t *testing.T) {
	blocks := ConvertBlocks("[click](https://example.com)")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	rt := blocks[0].(*slack.RichTextBlock)
	section := rt.Elements[0].(*slack.RichTextSection)
	link, ok := section.Elements[0].(*slack.RichTextSectionLinkElement)
	if !ok {
		t.Fatalf("expected LinkElement, got %T", section.Elements[0])
	}
	if link.URL != "https://example.com" {
		t.Errorf("expected URL https://example.com, got %q", link.URL)
	}
	if link.Text != "click" {
		t.Errorf("expected text 'click', got %q", link.Text)
	}
}

func TestToSlackBlocks_Empty(t *testing.T) {
	result := ToSlackBlocks(nil)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d blocks", len(result))
	}
}

func TestToSlackBlocks_List(t *testing.T) {
	blocks := ConvertBlocks("- item one\n- item two")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	rt := blocks[0].(*slack.RichTextBlock)
	list, ok := rt.Elements[0].(*slack.RichTextList)
	if !ok {
		t.Fatalf("expected RichTextList, got %T", rt.Elements[0])
	}
	if list.Style != slack.RTEListBullet {
		t.Errorf("expected bullet style, got %s", list.Style)
	}
	if len(list.Elements) != 2 {
		t.Errorf("expected 2 list items, got %d", len(list.Elements))
	}
}

func TestConvertBlocks_Quote(t *testing.T) {
	blocks := ConvertBlocks("> quoted text")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	rt := blocks[0].(*slack.RichTextBlock)
	q, ok := rt.Elements[0].(*slack.RichTextQuote)
	if !ok {
		t.Fatalf("expected RichTextQuote, got %T", rt.Elements[0])
	}
	if len(q.Elements) == 0 {
		t.Fatal("expected at least one element in quote")
	}
	text, ok := q.Elements[0].(*slack.RichTextSectionTextElement)
	if !ok {
		t.Fatalf("expected TextElement, got %T", q.Elements[0])
	}
	if text.Text != "quoted text" {
		t.Errorf("expected 'quoted text', got %q", text.Text)
	}
}

func TestConvertBlocks_OrderedList(t *testing.T) {
	blocks := ConvertBlocks("1. first\n2. second")
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	rt := blocks[0].(*slack.RichTextBlock)
	list, ok := rt.Elements[0].(*slack.RichTextList)
	if !ok {
		t.Fatalf("expected RichTextList, got %T", rt.Elements[0])
	}
	if list.Style != slack.RTEListOrdered {
		t.Errorf("expected ordered style, got %s", list.Style)
	}
}

func TestPostOptions(t *testing.T) {
	var o postOptions

	WithThreadTS("1234.5678")(&o)
	if o.threadTS != "1234.5678" {
		t.Errorf("expected threadTS '1234.5678', got %q", o.threadTS)
	}

	WithFallbackText("fallback")(&o)
	if o.fallbackText != "fallback" {
		t.Errorf("expected fallbackText 'fallback', got %q", o.fallbackText)
	}
}

func TestToSlackBlocks_Image(t *testing.T) {
	blocks := slackmd.ConvertBlocks("![alt text](https://example.com/img.png)")
	result := ToSlackBlocks(blocks)
	if len(result) != 1 {
		t.Fatalf("expected 1 block, got %d", len(result))
	}
	img, ok := result[0].(*slack.ImageBlock)
	if !ok {
		t.Fatalf("expected ImageBlock, got %T", result[0])
	}
	if img.ImageURL != "https://example.com/img.png" {
		t.Errorf("unexpected image URL: %s", img.ImageURL)
	}
}
