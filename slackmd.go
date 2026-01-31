// Package slackmd converts standard Markdown to Slack's mrkdwn format.
package slackmd

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Convert converts standard Markdown to Slack mrkdwn.
func Convert(markdown string) string {
	return defaultConverter.Convert(markdown)
}

// ConvertWithOptions converts Markdown to Slack mrkdwn with the given options.
func ConvertWithOptions(markdown string, opts ...Option) string {
	return NewConverter(opts...).Convert(markdown)
}

// Converter converts Markdown to Slack mrkdwn.
type Converter struct {
	md goldmark.Markdown
}

// NewConverter creates a new Converter with the given options.
func NewConverter(opts ...Option) *Converter {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	sr := newSlackRenderer(o)

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRenderer(
			renderer.NewRenderer(
				renderer.WithNodeRenderers(
					util.Prioritized(sr, 1),
				),
			),
		),
	)

	return &Converter{md: md}
}

// Convert converts Markdown to Slack mrkdwn.
func (c *Converter) Convert(markdown string) string {
	var buf bytes.Buffer
	source := []byte(markdown)
	_ = c.md.Convert(source, &buf)
	return cleanOutput(buf.String())
}

func cleanOutput(s string) string {
	// Strip trailing whitespace from each line
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	s = strings.Join(lines, "\n")
	// Trim trailing newlines, leave one
	s = strings.TrimRight(s, "\n")
	if s != "" {
		s += "\n"
	}
	return s
}

var defaultConverter = NewConverter()

// Options

type options struct {
	bulletChar rune
	listIndent int
}

func defaultOptions() *options {
	return &options{
		bulletChar: '•',
		listIndent: 2,
	}
}

// Option configures a Converter.
type Option func(*options)

// WithBulletChar sets the bullet character for unordered lists.
func WithBulletChar(c rune) Option {
	return func(o *options) { o.bulletChar = c }
}

// WithListIndent sets the number of spaces per list indentation level.
func WithListIndent(n int) Option {
	return func(o *options) { o.listIndent = n }
}
