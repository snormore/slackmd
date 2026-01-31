// Package slackgo converts Markdown to slack-go compatible block types.
// Import this package only if you use github.com/slack-go/slack's client.PostMessage.
package slackgo

import (
	"context"
	"math"
	"time"

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

// PostOption configures Post behavior.
type PostOption func(*postOptions)

type postOptions struct {
	threadTS     string
	fallbackText string
	retry        *RetryConfig
}

// RetryConfig controls retry behavior for Post.
type RetryConfig struct {
	MaxAttempts int           // maximum number of attempts (default 3)
	InitialWait time.Duration // initial backoff duration (default 1s)
	MaxWait     time.Duration // maximum backoff duration (default 30s)
}

var defaultRetryConfig = RetryConfig{
	MaxAttempts: 3,
	InitialWait: time.Second,
	MaxWait:     30 * time.Second,
}

// WithThreadTS sets the thread timestamp to reply in a thread.
func WithThreadTS(ts string) PostOption {
	return func(o *postOptions) { o.threadTS = ts }
}

// WithFallbackText sets fallback plain text for notifications and accessibility.
func WithFallbackText(text string) PostOption {
	return func(o *postOptions) { o.fallbackText = text }
}

// WithRetry enables retry with exponential backoff. Pass nil for defaults.
func WithRetry(cfg *RetryConfig) PostOption {
	return func(o *postOptions) {
		if cfg != nil {
			o.retry = cfg
		} else {
			c := defaultRetryConfig
			o.retry = &c
		}
	}
}

// poster abstracts the Slack API methods used by Post and Update.
type poster interface {
	PostMessageContext(ctx context.Context, channel string, opts ...slack.MsgOption) (string, string, error)
	UpdateMessageContext(ctx context.Context, channel, timestamp string, opts ...slack.MsgOption) (string, string, string, error)
}

// Post converts markdown to slack-go blocks and posts via client.PostMessageContext.
// Returns the response timestamp of the posted message.
func Post(ctx context.Context, api *slack.Client, channel, markdown string, opts ...PostOption) (string, error) {
	return post(ctx, api, channel, markdown, opts...)
}

func post(ctx context.Context, api poster, channel, markdown string, opts ...PostOption) (string, error) {
	var o postOptions
	for _, opt := range opts {
		opt(&o)
	}

	blocks := ConvertBlocks(markdown)
	msgOpts := []slack.MsgOption{slack.MsgOptionBlocks(blocks...)}

	if o.threadTS != "" {
		msgOpts = append(msgOpts, slack.MsgOptionTS(o.threadTS))
	}
	if o.fallbackText != "" {
		msgOpts = append(msgOpts, slack.MsgOptionText(o.fallbackText, false))
	}

	if o.retry == nil {
		_, ts, err := api.PostMessageContext(ctx, channel, msgOpts...)
		return ts, err
	}

	var ts string
	var err error
	for attempt := range o.retry.MaxAttempts {
		_, ts, err = api.PostMessageContext(ctx, channel, msgOpts...)
		if err == nil {
			return ts, nil
		}
		if rateLimitedErr, ok := err.(*slack.RateLimitedError); ok {
			wait := rateLimitedErr.RetryAfter
			if wait == 0 {
				wait = backoff(attempt, o.retry)
			}
			if err := sleep(ctx, wait); err != nil {
				return "", err
			}
			continue
		}
		if attempt < o.retry.MaxAttempts-1 {
			if err := sleep(ctx, backoff(attempt, o.retry)); err != nil {
				return "", err
			}
		}
	}
	return ts, err
}

// Update converts markdown to slack-go blocks and updates an existing message.
func Update(ctx context.Context, api *slack.Client, channel, timestamp, markdown string, opts ...PostOption) error {
	return update(ctx, api, channel, timestamp, markdown, opts...)
}

func update(ctx context.Context, api poster, channel, timestamp, markdown string, opts ...PostOption) error {
	var o postOptions
	for _, opt := range opts {
		opt(&o)
	}

	blocks := ConvertBlocks(markdown)
	msgOpts := []slack.MsgOption{slack.MsgOptionBlocks(blocks...)}

	if o.fallbackText != "" {
		msgOpts = append(msgOpts, slack.MsgOptionText(o.fallbackText, false))
	}

	if o.retry == nil {
		_, _, _, err := api.UpdateMessageContext(ctx, channel, timestamp, msgOpts...)
		return err
	}

	var err error
	for attempt := range o.retry.MaxAttempts {
		_, _, _, err = api.UpdateMessageContext(ctx, channel, timestamp, msgOpts...)
		if err == nil {
			return nil
		}
		if rateLimitedErr, ok := err.(*slack.RateLimitedError); ok {
			wait := rateLimitedErr.RetryAfter
			if wait == 0 {
				wait = backoff(attempt, o.retry)
			}
			if err := sleep(ctx, wait); err != nil {
				return err
			}
			continue
		}
		if attempt < o.retry.MaxAttempts-1 {
			if err := sleep(ctx, backoff(attempt, o.retry)); err != nil {
				return err
			}
		}
	}
	return err
}

func backoff(attempt int, cfg *RetryConfig) time.Duration {
	d := time.Duration(float64(cfg.InitialWait) * math.Pow(2, float64(attempt)))
	if d > cfg.MaxWait {
		d = cfg.MaxWait
	}
	return d
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
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
	case "emoji":
		return slack.NewRichTextSectionEmojiElement(in.Name, 0, convertInlineStyle(in.Style))
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
