package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/joho/godotenv"
	"github.com/snormore/slackmd"
)

func main() {
	_ = godotenv.Load()
	webhook := flag.String("webhook", "", "webhook URL (overrides SLACK_WEBHOOK_URL)")
	flag.Parse()

	url := *webhook
	if url == "" {
		url = os.Getenv("SLACK_WEBHOOK_URL")
	}
	if url == "" {
		fmt.Fprintln(os.Stderr, "error: no webhook URL provided (use -webhook or SLACK_WEBHOOK_URL)")
		os.Exit(1)
	}

	files, err := filepath.Glob("examples/*.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	sort.Strings(files)

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no .md files found in examples/")
		os.Exit(1)
	}

	for i, f := range files {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}

		name := filepath.Base(f)
		fmt.Printf("posting %s...\n", name)

		// Post label
		label := fmt.Sprintf(":page_facing_up: *%s*", name)
		if err := slackmd.PostMrkdwn(url, label); err != nil {
			fmt.Fprintf(os.Stderr, "error posting label for %s: %v\n", name, err)
			os.Exit(1)
		}

		time.Sleep(time.Second)

		// Post content as rich_text blocks
		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", f, err)
			os.Exit(1)
		}

		if err := slackmd.Post(url, string(content)); err != nil {
			fmt.Fprintf(os.Stderr, "error posting %s: %v\n", name, err)
			os.Exit(1)
		}
	}

	fmt.Println("done")
}
