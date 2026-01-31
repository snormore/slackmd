package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/snormore/slackmd"
)

const maxChunkRunes = 3900

func main() {
	post := flag.Bool("post", false, "post to Slack webhook instead of printing")
	webhook := flag.String("webhook", "", "webhook URL (overrides SLACK_WEBHOOK_URL)")
	flag.Parse()

	var input []byte
	var err error

	if flag.NArg() > 0 {
		input, err = os.ReadFile(flag.Arg(0))
	} else {
		input, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output := slackmd.Convert(string(input))

	if !*post {
		fmt.Print(output)
		return
	}

	url := *webhook
	if url == "" {
		url = os.Getenv("SLACK_WEBHOOK_URL")
	}
	if url == "" {
		fmt.Fprintln(os.Stderr, "error: no webhook URL provided (use -webhook or SLACK_WEBHOOK_URL)")
		os.Exit(1)
	}

	chunks := chunk(output, maxChunkRunes)
	for i, c := range chunks {
		if i > 0 {
			time.Sleep(time.Second)
		}
		if err := postToSlack(url, c); err != nil {
			fmt.Fprintf(os.Stderr, "error posting chunk %d/%d: %v\n", i+1, len(chunks), err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "posted chunk %d/%d\n", i+1, len(chunks))
	}
}

// chunk splits text into pieces of at most maxLen runes, avoiding splits inside
// code blocks. It prefers breaking at paragraph boundaries (\n\n), then single
// newlines, then rune boundaries as a last resort. Empty chunks are skipped.
func chunk(text string, maxLen int) []string {
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

func postToSlack(url, text string) error {
	body, _ := json.Marshal(map[string]string{"text": text})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack returned %s", resp.Status)
	}
	return nil
}
