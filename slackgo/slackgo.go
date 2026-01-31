// Package slackgo converts Markdown to slack-go compatible block types.
// Import this package only if you use github.com/slack-go/slack's client.PostMessage.
package slackgo

import (
	"github.com/slack-go/slack"
	"github.com/snormore/slackmd"
)

// ConvertBlocks converts Markdown to slack-go Block types.
func ConvertBlocks(markdown string) []slack.Block {
	return ToSlackBlocks(slackmd.ConvertBlocks(markdown))
}

// ToSlackBlocks converts slackmd blocks to slack-go Block types.
func ToSlackBlocks(blocks []slackmd.Block) []slack.Block {
	out := make([]slack.Block, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, convertBlock(b))
	}
	return out
}

// Post converts markdown to slack-go blocks and posts via client.PostMessage.
func Post(api *slack.Client, channel, markdown string) error {
	blocks := ConvertBlocks(markdown)
	_, _, err := api.PostMessage(channel, slack.MsgOptionBlocks(blocks...))
	return err
}

func convertBlock(b slackmd.Block) slack.Block {
	switch b.Type {
	case "header":
		text := ""
		if b.Text != nil {
			text = b.Text.Text
		}
		return slack.NewHeaderBlock(slack.NewTextBlockObject(slack.PlainTextType, text, false, false))
	case "divider":
		return slack.NewDividerBlock()
	case "image":
		alt := b.AltText
		if alt == "" {
			alt = "image"
		}
		img := slack.NewImageBlock(b.ImageURL, alt, "", nil)
		if b.Title != nil {
			img.Title = slack.NewTextBlockObject(slack.PlainTextType, b.Title.Text, false, false)
		}
		return img
	case "rich_text":
		return &slack.RichTextBlock{
			Type:     slack.MBTRichText,
			Elements: convertElements(b.Elements),
		}
	default:
		// Fallback: treat as rich_text
		return &slack.RichTextBlock{
			Type:     slack.MBTRichText,
			Elements: convertElements(b.Elements),
		}
	}
}

func convertElements(elems []slackmd.Element) []slack.RichTextElement {
	out := make([]slack.RichTextElement, 0, len(elems))
	for _, e := range elems {
		out = append(out, convertElement(e))
	}
	return out
}

func convertElement(e slackmd.Element) slack.RichTextElement {
	switch e.Type {
	case "rich_text_section":
		return &slack.RichTextSection{
			Type:     slack.RTESection,
			Elements: convertInlines(extractInlines(e.Elements)),
		}
	case "rich_text_preformatted":
		return &slack.RichTextPreformatted{
			RichTextSection: slack.RichTextSection{
				Type:     slack.RTEPreformatted,
				Elements: convertInlines(extractInlines(e.Elements)),
			},
			Border: e.Border,
		}
	case "rich_text_quote":
		return (*slack.RichTextQuote)(&slack.RichTextSection{
			Type:     slack.RTEQuote,
			Elements: convertInlines(extractInlines(e.Elements)),
		})
	case "rich_text_list":
		style := slack.RTEListBullet
		if e.Style == "ordered" {
			style = slack.RTEListOrdered
		}
		return &slack.RichTextList{
			Type:     slack.RTEList,
			Style:    style,
			Elements: convertListSections(e.Elements),
			Indent:   e.Indent,
		}
	default:
		return &slack.RichTextSection{
			Type:     slack.RTESection,
			Elements: convertInlines(extractInlines(e.Elements)),
		}
	}
}

// extractInlines extracts []slackmd.Inline from an Element's Elements field,
// which uses type any.
func extractInlines(v any) []slackmd.Inline {
	if v == nil {
		return nil
	}
	if inlines, ok := v.([]slackmd.Inline); ok {
		return inlines
	}
	return nil
}

// convertListSections extracts []slackmd.Element (list item sections) from
// a list element's Elements field.
func convertListSections(v any) []slack.RichTextElement {
	if v == nil {
		return nil
	}
	if elems, ok := v.([]slackmd.Element); ok {
		out := make([]slack.RichTextElement, 0, len(elems))
		for _, e := range elems {
			out = append(out, convertElement(e))
		}
		return out
	}
	return nil
}

func convertInlines(inlines []slackmd.Inline) []slack.RichTextSectionElement {
	out := make([]slack.RichTextSectionElement, 0, len(inlines))
	for _, in := range inlines {
		out = append(out, convertInline(in))
	}
	return out
}

func convertInline(in slackmd.Inline) slack.RichTextSectionElement {
	switch in.Type {
	case "link":
		el := &slack.RichTextSectionLinkElement{
			Type: slack.RTSELink,
			URL:  in.URL,
			Text: in.Text,
		}
		if in.Style != nil {
			el.Style = convertInlineStyle(in.Style)
		}
		return el
	default:
		el := &slack.RichTextSectionTextElement{
			Type: slack.RTSEText,
			Text: in.Text,
		}
		if in.Style != nil {
			el.Style = convertInlineStyle(in.Style)
		}
		return el
	}
}

func convertInlineStyle(s *slackmd.InlineStyle) *slack.RichTextSectionTextStyle {
	if s == nil {
		return nil
	}
	return &slack.RichTextSectionTextStyle{
		Bold:   s.Bold,
		Italic: s.Italic,
		Strike: s.Strike,
		Code:   s.Code,
	}
}
