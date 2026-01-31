package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/snormore/slackmd"
)

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

	body, _ := json.Marshal(map[string]string{"text": output})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Fprintf(os.Stderr, "%s\n", resp.Status)
}
