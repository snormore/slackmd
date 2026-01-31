package slackmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const maxChunkRunes = 3900

// Post converts markdown to Slack rich_text blocks and posts them to the given webhook URL.
func Post(webhookURL, markdown string) error {
	blocks := ConvertBlocks(markdown)
	return PostBlocks(webhookURL, blocks)
}

const maxBlocksPerMessage = 50

// PostBlocks posts rich_text blocks to the given webhook URL.
// Long messages are automatically split into multiple posts of up to 50 blocks each.
func PostBlocks(webhookURL string, blocks []Block) error {
	for i := 0; i < len(blocks); i += maxBlocksPerMessage {
		end := min(i+maxBlocksPerMessage, len(blocks))
		if i > 0 {
			time.Sleep(time.Second)
		}
		if err := postBlocks(webhookURL, blocks[i:end]); err != nil {
			return fmt.Errorf("posting blocks %d–%d of %d: %w", i+1, end, len(blocks), err)
		}
	}
	return nil
}

func postBlocks(webhookURL string, blocks []Block) error {
	payload := map[string]any{
		"blocks": blocks,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack returned %s: %s", resp.Status, string(body))
	}
	return nil
}

// PostMrkdwn posts pre-converted mrkdwn text to the given webhook URL.
// Long messages are automatically split into multiple posts.
func PostMrkdwn(webhookURL, mrkdwn string) error {
	chunks := Chunk(mrkdwn, maxChunkRunes)
	for i, c := range chunks {
		if i > 0 {
			time.Sleep(time.Second)
		}
		if err := postMessage(webhookURL, c); err != nil {
			return fmt.Errorf("posting chunk %d/%d: %w", i+1, len(chunks), err)
		}
	}
	return nil
}

func postMessage(url, text string) error {
	body, _ := json.Marshal(map[string]string{"text": text})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack returned %s: %s", resp.Status, string(body))
	}
	return nil
}

// Chunk splits text into pieces of at most maxLen runes, avoiding splits inside
// code blocks. It prefers breaking at paragraph boundaries (\n\n), then single
// newlines, then rune boundaries as a last resort. Empty chunks are skipped.
func Chunk(text string, maxLen int) []string {
	if utf8.RuneCountInString(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if utf8.RuneCountInString(text) <= maxLen {
			chunks = append(chunks, text)
			break
		}

		// Convert rune limit to byte offset.
		cut := runeOffset(text, maxLen)

		// Avoid splitting inside a code block.
		cut = adjustForCodeBlock(text, cut)

		if cut == runeOffset(text, maxLen) {
			// No code block adjustment — prefer paragraph/line boundaries.
			if i := strings.LastIndex(text[:cut], "\n\n"); i > 0 {
				cut = i + 2
			} else if i := strings.LastIndex(text[:cut], "\n"); i > 0 {
				cut = i + 1
			}
		}

		c := strings.TrimRight(text[:cut], "\n")
		if len(c) > 0 {
			chunks = append(chunks, c)
		}
		text = strings.TrimLeft(text[cut:], "\n")
	}
	return chunks
}

// runeOffset returns the byte offset of the nth rune in s, or len(s) if
// s has fewer than n runes.
func runeOffset(s string, n int) int {
	off := 0
	for i := 0; i < n && off < len(s); i++ {
		_, size := utf8.DecodeRuneInString(s[off:])
		off += size
	}
	return off
}

// adjustForCodeBlock checks if cutting at byteLimit would split inside a code
// block. If so, it tries two strategies:
//  1. Include the entire code block (extend cut to after the closing fence) if
//     the closing fence exists in the remaining text.
//  2. Break before the opening fence if there's meaningful content before it.
//
// Returns the original byteLimit if no adjustment is needed.
func adjustForCodeBlock(text string, byteLimit int) int {
	region := text[:byteLimit]
	fenceCount := 0
	lastFenceStart := 0
	idx := 0
	for {
		pos := strings.Index(region[idx:], "```")
		if pos < 0 {
			break
		}
		fenceCount++
		lastFenceStart = idx + pos
		idx += pos + 3
	}

	if fenceCount%2 == 0 {
		return byteLimit // not inside a code block
	}

	// We're inside a code block. Try to find the closing fence after the cut.
	closingPos := strings.Index(text[idx:], "```")
	if closingPos >= 0 {
		// Include through the closing fence and its line.
		endOfFence := idx + closingPos + 3
		// Include trailing newline if present.
		if endOfFence < len(text) && text[endOfFence] == '\n' {
			endOfFence++
		}
		return endOfFence
	}

	// No closing fence found. Break before the opening fence if possible.
	if lastFenceStart > 0 {
		return lastFenceStart
	}

	return byteLimit
}
