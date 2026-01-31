# slackmd

Go library and CLI tool for posting Markdown to Slack.

Slack doesn't understand Markdown. It has its own formatting syntax (mrkdwn) which lacks features like nested lists, and a richer block-based format (rich_text) that supports them but requires constructing complex JSON payloads. Neither accepts Markdown directly.

`slackmd` bridges that gap. Write standard Markdown and it handles the conversion — either to mrkdwn for simple cases, or to rich_text blocks for full-fidelity formatting with nested lists, blockquotes, code blocks, headers, dividers, images, and tables. It also handles Slack's payload limits automatically, chunking long messages and splitting large block payloads across multiple posts.

Use it as a Go library to convert or post programmatically, or as a CLI tool to pipe Markdown files straight to a Slack channel.

## Library

```go
import "github.com/snormore/slackmd"

// Convert markdown to Slack mrkdwn (plain text formatting)
output := slackmd.Convert("**bold** and _italic_")
// output: "*bold* and _italic_"

// Convert markdown to Slack rich_text blocks (native formatting)
blocks := slackmd.ConvertBlocks("- a\n  - b\n    - c")

// Post to Slack using rich_text blocks (native nested lists, quotes, etc.)
err := slackmd.Post(webhookURL, "**bold** and _italic_")

// Post pre-built blocks
err = slackmd.PostBlocks(webhookURL, blocks)

// Post pre-converted mrkdwn text
err = slackmd.PostMrkdwn(webhookURL, output)

// Split long text into chunks (for custom posting logic)
chunks := slackmd.Chunk(text, 3900)
```

### Using with slack-go client

If you use [slack-go](https://github.com/slack-go/slack)'s `client.PostMessage`, import the `slackgo` subpackage to get `[]slack.Block` directly:

```go
import "github.com/snormore/slackmd/slackgo"

// Convert and post in one call
ts, err := slackgo.Post(api, "#channel", "**bold** and _italic_")

// Reply in a thread
_, err = slackgo.Post(api, "#channel", "threaded reply",
	slackgo.WithThreadTS(ts),
	slackgo.WithFallbackText("threaded reply"),
)

// Or convert to []slack.Block for custom use
blocks := slackgo.ConvertBlocks("- a\n  - b")
api.PostMessage("#channel", slack.MsgOptionBlocks(blocks...))
```

The `slackgo` subpackage is optional — `github.com/slack-go/slack` is only pulled in as a dependency if you import it.

`Post()` and `PostBlocks()` use Slack's rich_text block format, which supports native nested lists, blockquotes, styled text, headers, dividers, and image blocks. Messages with more than 50 blocks are automatically split into multiple posts. `PostMrkdwn()` sends plain mrkdwn text for simpler use cases. Long mrkdwn messages are automatically chunked at paragraph boundaries, preserving code blocks intact.

## CLI

### Install

```
go install github.com/snormore/slackmd/cmd/slackmd@latest
```

### Usage

Convert markdown and print to stdout:

```
slackmd input.md
echo "**bold** and _italic_" | slackmd
```

Post directly to Slack via incoming webhook:

```
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T.../B.../..."
slackmd -post input.md
echo "# hello" | slackmd -post
```

Or pass the webhook URL directly:

```
slackmd -post -webhook "https://hooks.slack.com/services/T.../B.../..." input.md
```

Stdout prints mrkdwn text. Posting with `-post` uses rich_text blocks for native Slack formatting.

### Setting up a Slack webhook

1. Go to https://api.slack.com/apps and click **Create New App** > **From an app manifest**
2. Select your workspace, then paste the contents of [`slack-app-manifest.yaml`](slack-app-manifest.yaml)
3. Click **Create**
4. Go to **Incoming Webhooks** and click **Add New Webhook to Workspace**
5. Pick a channel and authorize
6. Copy the webhook URL and set it as an environment variable:
   ```
   export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T.../B.../..."
   ```

### Testing it end to end

```
# Convert only (no webhook needed)
echo "**bold** and _italic_" | slackmd

# Convert a sample file
slackmd examples/release-notes.md

# Post to Slack (requires webhook)
echo "hello from slackmd" | slackmd -post

# Post a full example
slackmd -post examples/mixed.md

# Try other examples
slackmd examples/release-notes.md
slackmd examples/long-form.md
slackmd examples/tables.md
```
