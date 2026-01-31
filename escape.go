package slackmd

import "strings"

// escapeText escapes &, <, > for Slack mrkdwn text content.
// This must NOT be applied inside code blocks/spans or link URLs.
func escapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
